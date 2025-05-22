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

	"github.com/Kovarniykrab/finishGolang/pkg/db"
)

// TasksResp структура для ответа списка задач
type TasksResp struct {
	Tasks []*db.Task `json:"tasks"`
}

// RegisterHandlers регистрирует все API обработчики
func RegisterHandlers(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("/api/nextdate", NextDateHandler)
	mux.HandleFunc("/api/tasks", tasksHandler(db))
	mux.HandleFunc("/api/task", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getTasksHandler(db, w, r)
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

// tasksHandler обработчик для /api/tasks
func tasksHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sendJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		search := r.URL.Query().Get("search")
		limit := 50

		tasks, err := db.GetTasks(search, limit)
		if err != nil {
			sendJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
			return
		}

		if tasks == nil {
			tasks = make([]*db.Task, 0)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(TasksResp{Tasks: tasks}); err != nil {
			log.Printf("Failed to encode tasks response: %v", err)
		}
	}
}

func getTasksHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date")
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
		return
	}
	defer rows.Close()

	var tasks []*db.Task
	for rows.Next() {
		var t db.Task
		if err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat); err != nil {
			sendJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Data scan error: %v", err))
			return
		}
		tasks = append(tasks, &t)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(TasksResp{Tasks: tasks}); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func addTaskHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var task db.Task
	var err error

	contentType := r.Header.Get("Content-Type")
	now := time.Now().UTC()
	today := now.Format("20060102")

	if strings.Contains(contentType, "application/json") {
		if err = json.NewDecoder(r.Body).Decode(&task); err != nil {
			sendJSONError(w, http.StatusBadRequest, "Invalid JSON data")
			return
		}
	} else {
		if err = r.ParseForm(); err != nil {
			sendJSONError(w, http.StatusBadRequest, "Invalid form data")
			return
		}

		dateInput := r.FormValue("date")
		if dateInput == "today" {
			task.Date = today
		} else {
			task.Date = dateInput
		}

		task.Title = r.FormValue("title")
		task.Comment = r.FormValue("comment")
		task.Repeat = r.FormValue("repeat")
	}

	if task.Title == "" {
		sendJSONError(w, http.StatusBadRequest, "Title is required")
		return
	}
	if len(task.Title) > 100 {
		sendJSONError(w, http.StatusBadRequest, "Title is too long (max 100 characters)")
		return
	}
	if len(task.Comment) > 500 {
		sendJSONError(w, http.StatusBadRequest, "Comment is too long (max 500 characters)")
		return
	}

	if task.Date == "" {
		task.Date = today
	}

	parsedDate, err := time.Parse("20060102", task.Date)
	if err != nil {
		sendJSONError(w, http.StatusBadRequest, "Invalid date format (expected YYYYMMDD)")
		return
	}

	if task.Repeat != "" {
		startDate := task.Date
		if r.FormValue("date") == "today" {
			startDate = today
		}

		parsedStartDate, err := time.Parse("20060102", startDate)
		if err != nil {
			sendJSONError(w, http.StatusBadRequest, "Invalid start date format")
			return
		}

		nowTruncated := now.Truncate(24 * time.Hour)
		startTruncated := parsedStartDate.Truncate(24 * time.Hour)

		if startTruncated.Before(nowTruncated) {
			next, err := NextDate(now, startDate, task.Repeat)
			if err != nil {
				sendJSONError(w, http.StatusBadRequest, "Invalid repeat rule: "+err.Error())
				return
			}
			task.Date = next
		} else {
			task.Date = startDate
		}
	} else {
		if parsedDate.Before(now) {
			task.Date = today
		}
	}

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
		sendJSONError(w, http.StatusInternalServerError, "Failed to get ID")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

func sendJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		log.Printf("Failed to send JSON error: %v", err)
	}
}

func getTaskHandler(db *sql.DB, w http.ResponseWriter, r *http.Request, id int64) {
	var task db.Task
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
	var task db.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		sendJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM scheduler WHERE id=?)", id).Scan(&exists)
	if err != nil || !exists {
		sendJSONError(w, http.StatusNotFound, "Task not found")
		return
	}

	_, err = db.Exec(
		"UPDATE scheduler SET date=?, title=?, comment=?, repeat=? WHERE id=?",
		task.Date, task.Title, task.Comment, task.Repeat, id,
	)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Database update error: %v", err))
		return
	}

	w.WriteHeader(http.StatusOK)
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
