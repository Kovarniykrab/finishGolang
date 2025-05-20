package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Task структура для представления задачи
type Task struct {
	ID      int64  `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

// RegisterHandlers регистрирует все API обработчики
func RegisterHandlers(db *sql.DB) {
	// Обработчик для /api/nextdate
	http.HandleFunc("/api/nextdate", NextDateHandler)

	// Обработчик для /api/tasks
	http.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getTasksHandler(db, w, r)
		case http.MethodPost:
			addTaskHandler(db, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Обработчик для /api/tasks/{id}
	http.HandleFunc("/api/tasks/", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Path[len("/api/tasks/"):]
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid task ID", http.StatusBadRequest)
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
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// Остальные функции-обработчики остаются без изменений
func getTasksHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date")
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat); err != nil {
			http.Error(w, fmt.Sprintf("Data scan error: %v", err), http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]Task{"tasks": tasks})
}

func addTaskHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if task.Date == "" {
		task.Date = time.Now().Format(dateFormat)
	}

	result, err := db.Exec(
		"INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)",
		task.Date, task.Title, task.Comment, task.Repeat,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database insert error: %v", err), http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get task ID: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

func getTaskHandler(db *sql.DB, w http.ResponseWriter, r *http.Request, id int64) {
	var task Task
	err := db.QueryRow(
		"SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?",
		id,
	).Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)

	switch {
	case err == sql.ErrNoRows:
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func updateTaskHandler(db *sql.DB, w http.ResponseWriter, r *http.Request, id int64) {
	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM scheduler WHERE id = ?)", id).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	_, err = db.Exec(
		"UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat = ? WHERE id = ?",
		task.Date, task.Title, task.Comment, task.Repeat, id,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database update error: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func deleteTaskHandler(db *sql.DB, w http.ResponseWriter, r *http.Request, id int64) {
	result, err := db.Exec("DELETE FROM scheduler WHERE id = ?", id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database delete error: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check deletion: %v", err), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
