package api

import (
	"database/sql"
	"encoding/json"

	"errors"
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

type TasksResp struct {
	Tasks []*domain.Task `json:"tasks"`
}

type DB struct {
	*sql.DB
}

func RegisterHandlers(mux *http.ServeMux, db *sql.DB) {
	dbs := &DB{db}
	mux.HandleFunc("/api/nextdate", util.NextDateHandler)
	mux.HandleFunc("/api/tasks", tasksHandler(db))
	mux.HandleFunc("/api/task", taskAll(dbs))

	http.HandleFunc("/api/task/done", dbs.completedTaskHandler)
	// http.HandleFunc("/api/signin"

}
func taskAll(d *DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			d.addTaskHandler(w, r)
		case http.MethodGet:
			d.getTaskHandler(w, r)
		case http.MethodPut:
			d.updateTaskHandler(w, r)
		case http.MethodDelete:
			d.deleteHandler(w, r)
		default:
			sendJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	}
}

func (d *DB) getTaskHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL)
	idS := r.URL.Query().Get("id")
	if idS == "" {
		sendJSONError(w, http.StatusBadRequest, "id is required")
		return
	}
	id, err := strconv.ParseInt(idS, 10, 64)
	if err != nil {
		sendJSONError(w, http.StatusBadRequest, "id is required")
		return
	}
	task, err := database.GetTaskStory(d.DB, id)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	fmt.Println(task)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"id":      strconv.FormatInt(task.ID, 10),
		"date":    task.Date,
		"title":   task.Title,
		"comment": task.Comment,
		"repeat":  task.Repeat,
	}); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (d *DB) deleteHandler(w http.ResponseWriter, r *http.Request) {
	idS := r.URL.Query().Get("id")
	if idS == "" {
		sendJSONError(w, http.StatusBadRequest, "id is required")
		return // Добавлен return после ошибки
	}
	id, err := strconv.ParseInt(idS, 10, 64)
	if err != nil {
		sendJSONError(w, http.StatusBadRequest, "invalid id format")
		return // Добавлен return после ошибки
	}
	if err := database.DeleteTaskStory(d.DB, id); err != nil {
		sendJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Возвращаем пустой JSON {}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct{}{}); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (d *DB) updateTaskHandler(w http.ResponseWriter, r *http.Request) {

	var t domain.Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		sendJSONError(w, http.StatusBadRequest, "Invalid JSON data")
		return
	}

	if t.Title == "" {
		sendJSONError(w, http.StatusBadRequest, "Название не может быть пустым")
		return
	}

	// Проверяем, что ID задан
	if t.ID == 0 {
		sendJSONError(w, http.StatusBadRequest, "ошибка id is required")
		return
	}

	task, err := database.UpdateTaskStory(d.DB, t, t.ID)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"id":      strconv.FormatInt(task.ID, 10),
		"date":    task.Date,
		"title":   task.Title,
		"comment": task.Comment,
		"repeat":  task.Repeat,
	}); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
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

		tasks, err := database.GetTasksStory(db, search, limit)
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

func (d *DB) addTaskHandler(w http.ResponseWriter, r *http.Request) {
	var task domain.Task

	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		sendJSONError(w, http.StatusBadRequest, "Invalid JSON data")
		return
	}

	task.Title = strings.TrimSpace(task.Title)
	if task.Title == "" {
		sendJSONError(w, http.StatusBadRequest, "Title is required")
		return
	}
	if len(task.Title) > 100 {
		sendJSONError(w, http.StatusBadRequest, "Title is too long (max 100 characters)")
		return
	}

	task.Comment = strings.TrimSpace(task.Comment)
	if len(task.Comment) > 500 {
		sendJSONError(w, http.StatusBadRequest, "Comment is too long (max 500 characters)")
		return
	}

	Now := time.Now()

	if task.Date == "" {
		task.Date = Now.Format(util.DateFormat)
	}

	dateTime, err := time.Parse(util.DateFormat, task.Date)
	if err != nil {
		sendJSONError(w, http.StatusBadRequest, "дата представлена в формате, отличном от 20060102")
		log.Println(err)
		return
	}

	if dateTime.Format(util.DateFormat) < Now.Format(util.DateFormat) {
		if task.Repeat == "" {
			task.Date = Now.Format(util.DateFormat)
		} else {
			nextDate, err := util.NextDate(Now, task.Date, task.Repeat)
			if err != nil {
				sendJSONError(w, http.StatusInternalServerError, "ошибка работы NextDate")
				log.Println(err)
				return
			}
			task.Date = nextDate
		}
	}

	id, err := database.AddTaskStory(d.DB, task)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": strconv.FormatInt(id, 10)})
}

func sendJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		log.Printf("Failed to send JSON error: %v", err)
	}
}

func (d *DB) completedTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		sendJSONError(w, http.StatusBadRequest, "ID parameter is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		sendJSONError(w, http.StatusBadRequest, "Invalid ID format")
		return
	}

	task, err := database.GetTaskStory(d.DB, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			sendJSONError(w, http.StatusNotFound, "Task not found")
		} else {
			sendJSONError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	now := time.Now().UTC().Truncate(24 * time.Hour)

	if task.Repeat == "" {
		if err := database.DeleteTaskStory(d.DB, id); err != nil {
			sendJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		nextDate, err := util.NextDate(now, task.Date, task.Repeat)
		if err != nil {
			sendJSONError(w, http.StatusBadRequest, "Invalid repeat rule: "+err.Error())
			return
		}

		// Создаем объект только с обновленной датой
		updateData := domain.Task{
			Date: nextDate,
		}

		// Обновляем только дату
		if _, err = database.UpdateTaskStory(d.DB, updateData, id); err != nil {
			sendJSONError(w, http.StatusInternalServerError, "Failed to update task date: "+err.Error())
			return
		}
	}

	// Возвращаем пустой JSON {}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct{}{}); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}
