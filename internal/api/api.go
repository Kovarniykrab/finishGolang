package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Kovarniykrab/finishGolang/internal/database"
	"github.com/Kovarniykrab/finishGolang/internal/domain"
	"github.com/Kovarniykrab/finishGolang/internal/util"
)

// TasksResp структура для ответа списка задач
type TasksResp struct {
	Tasks []*domain.Task `json:"tasks"`
}

// RegisterHandlers регистрирует все API обработчики
func RegisterHandlers(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("/util/nextdate", util.NextDateHandler)
	mux.HandleFunc("/api/tasks", tasksHandler(db))

	mux.HandleFunc("/api/task", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			addTaskHandler(db, w, r)
		default:
			sendJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})

	mux.HandleFunc("/api/task/", func(w http.ResponseWriter, r *http.Request) {
		idStr := strings.TrimPrefix(r.URL.Path, "/api/task/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			sendJSONError(w, http.StatusBadRequest, "Invalid task ID")
			return
		}

		switch r.Method {
		case http.MethodGet:
			getTaskHandler(db, w, r, id)
		case http.MethodPut:
			updateTaskHandler(db, w, r, id)
		case http.MethodDelete:
			deleteTaskHandler(db, w, r, id)
		default:
			sendJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})
}

func tasksHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sendJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		search := r.URL.Query().Get("search")
		limitStr := r.URL.Query().Get("limit")
		limit := 50

		if limitStr != "" {
			l, err := strconv.Atoi(limitStr)
			if err != nil || l < 1 {
				sendJSONError(w, http.StatusBadRequest, "Invalid limit")
				return
			}
			limit = l
		}

		tasks, err := database.GetTasks(db, search, limit)
		if err != nil {
			sendJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
			return
		}

		if tasks == nil {
			tasks = make([]*domain.Task, 0)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(TasksResp{Tasks: tasks}); err != nil {
			log.Printf("Failed to encode tasks response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

func addTaskHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var task domain.Task

	if r.Method != http.MethodPost {
		sendJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		sendJSONError(w, http.StatusBadRequest, "Invalid JSON data")
		return
	}

	// Валидация заголовка
	task.Title = strings.TrimSpace(task.Title)
	if task.Title == "" {
		sendJSONError(w, http.StatusBadRequest, "Title is required")
		return
	}
	if len(task.Title) > 100 {
		sendJSONError(w, http.StatusBadRequest, "Title is too long (max 100 characters)")
		return
	}

	// Валидация комментария
	task.Comment = strings.TrimSpace(task.Comment)
	if len(task.Comment) > 500 {
		sendJSONError(w, http.StatusBadRequest, "Comment is too long (max 500 characters)")
		return
	}

	// Обработка даты
	now := time.Now().UTC()
	today := now.Format("20060102")

	if task.Date == "" {
		task.Date = today
	} else {
		// Проверка формата даты
		if _, err := time.Parse("20060102", task.Date); err != nil {
			sendJSONError(w, http.StatusBadRequest, "Invalid date format (expected YYYYMMDD)")
			return
		}
	}

	// Обработка повторяющихся задач
	if task.Repeat != "" {
		task.Repeat = strings.TrimSpace(task.Repeat)
		next, err := util.NextDate(now, task.Date, task.Repeat)
		if err != nil {
			sendJSONError(w, http.StatusBadRequest, "Invalid repeat rule: "+err.Error())
			return
		}
		task.Date = next
	} else {
		// Для неповторяющихся задач проверяем, что дата не в прошлом
		parsedDate, _ := time.Parse("20060102", task.Date)
		if parsedDate.Before(now) {
			task.Date = today
		}
	}

	// Добавление задачи в БД
	result, err := db.Exec(
		"INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)",
		task.Date, task.Title, task.Comment, task.Repeat,
	)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "Failed to get task ID")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": strconv.FormatInt(id, 10)})
}

func sendJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		log.Printf("Failed to send JSON error: %v", err)
	}
}

func getTaskHandler(db *sql.DB, w http.ResponseWriter, r *http.Request, id int64) {
	var task domain.Task
	err := db.QueryRow(
		"SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?",
		id,
	).Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)

	switch {
	case err == sql.ErrNoRows:
		sendJSONError(w, http.StatusNotFound, "Task not found")
		return
	case err != nil:
		sendJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(task); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func updateTaskHandler(db *sql.DB, w http.ResponseWriter, r *http.Request, id int64) {
	var newValues struct {
		Title   *string `json:"title"`
		Date    *string `json:"date"`
		Comment *string `json:"comment"`
		Repeat  *string `json:"repeat"`
	}

	// Декодируем JSON
	if err := json.NewDecoder(r.Body).Decode(&newValues); err != nil {
		sendJSONError(w, http.StatusBadRequest, "Invalid JSON data")
		return
	}

	// Валидация Title
	if newValues.Title != nil {
		*newValues.Title = strings.TrimSpace(*newValues.Title)
		if *newValues.Title == "" {
			sendJSONError(w, http.StatusBadRequest, "Title cannot be empty")
			return
		}
		if len(*newValues.Title) > 100 {
			sendJSONError(w, http.StatusBadRequest, "Title too long (max 100 chars)")
			return
		}
	}

	// Валидация Comment
	if newValues.Comment != nil {
		*newValues.Comment = strings.TrimSpace(*newValues.Comment)
		if len(*newValues.Comment) > 500 {
			sendJSONError(w, http.StatusBadRequest, "Comment too long (max 500 chars)")
			return
		}
	}

	// Валидация Repeat и Date
	if newValues.Repeat != nil {
		now := time.Now().UTC()
		currentDate := now.Format("20060102")

		// Если обновляется дата - используем новое значение
		if newValues.Date != nil {
			currentDate = *newValues.Date
		}

		// Проверяем правило повтора
		if _, err := util.NextDate(now, currentDate, *newValues.Repeat); err != nil {
			sendJSONError(w, http.StatusBadRequest, "Invalid repeat rule: "+err.Error())
			return
		}
	}

	// Формируем SQL запрос
	clauses := []string{}
	params := []interface{}{}

	if newValues.Title != nil {
		clauses = append(clauses, "title=?")
		params = append(params, *newValues.Title)
	}

	if newValues.Date != nil {
		clauses = append(clauses, "date=?")
		params = append(params, *newValues.Date)
	}

	if newValues.Comment != nil {
		clauses = append(clauses, "comment=?")
		params = append(params, *newValues.Comment)
	}

	if newValues.Repeat != nil {
		clauses = append(clauses, "repeat=?")
		params = append(params, *newValues.Repeat)
	}

	if len(clauses) == 0 {
		sendJSONError(w, http.StatusBadRequest, "Nothing to update")
		return
	}

	// Добавляем ID в параметры
	params = append(params, id)

	// Выполняем запрос
	query := fmt.Sprintf("UPDATE scheduler SET %s WHERE id=?", strings.Join(clauses, ", "))
	result, err := db.Exec(query, params...)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
		return
	}

	// Проверяем обновление
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Update check failed: %v", err))
		return
	}

	if rowsAffected == 0 {
		sendJSONError(w, http.StatusNotFound, "Task not found")
		return
	}

	// Возвращаем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct{ Success bool }{true})
}

func deleteTaskHandler(db *sql.DB, w http.ResponseWriter, r *http.Request, id int64) {
	result, err := db.Exec("DELETE FROM scheduler WHERE id = ?", id)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Database delete error: %v", err))
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to check deletion: %v", err))
		return
	}

	if rowsAffected == 0 {
		sendJSONError(w, http.StatusNotFound, "Task not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
