package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Kovarniykrab/finishGolang/internal/domain"
	"github.com/Kovarniykrab/finishGolang/internal/util"
	_ "modernc.org/sqlite"
)

const dbFile = "scheduler.db"

func InitDB() (*sql.DB, error) {
	if os.Getenv("GO_TEST") == "1" {
		os.Remove(dbFile)
	}

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть БД: %v", err)
	}

	if err := createTable(db); err != nil {
		return nil, fmt.Errorf("failed to create table: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database connection failed: %v", err)
	}

	log.Println("База данных успешно инициализирована")
	return db, nil
}

func createTable(db *sql.DB) error {
	query := `
    CREATE TABLE IF NOT EXISTS scheduler (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        date TEXT NOT NULL,
        title TEXT NOT NULL,
        comment TEXT,
        repeat TEXT
    );
    CREATE INDEX IF NOT EXISTS idx_date ON scheduler(date);`

	_, err := db.Exec(query)
	return err
}

func GetTaskStory(db *sql.DB, id int64) (task domain.Task, err error) {
	err = db.QueryRow(
		"SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?",
		id,
	).Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return task, err
	case err != nil:
		return task, err
	}

	fmt.Println("db", task)

	return task, nil

}

func DeleteTaskStory(db *sql.DB, id int64) error {
	result, err := db.Exec("DELETE FROM scheduler WHERE id = ?", id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err

	}

	if rowsAffected == 0 {
		return errors.New("task not found")
	}
	return nil

}

func UpdateTaskStory(db *sql.DB, newValues domain.Task, id int64) (domain.Task, error) {
	if newValues.Date != "" {
		if len(newValues.Date) != 8 {
			return domain.Task{}, errors.New("invalid date format (expected YYYYMMDD)")
		}

		_, err := time.Parse("20060102", newValues.Date)
		if err != nil {
			return domain.Task{}, errors.New("invalid date (does not exist)")
		}
	}
	if newValues.Title != "" {
		newValues.Title = strings.TrimSpace(newValues.Title)
		if newValues.Title == "" {
			return domain.Task{}, errors.New("title cannot be empty")
		}
		if len(newValues.Title) > 100 {
			return domain.Task{}, errors.New("title is too long (max 100 chars)")
		}
	}

	if newValues.Comment != "" && len(newValues.Comment) > 500 {
		return domain.Task{}, errors.New("comment is too long (max 500 chars)")
	}

	if newValues.Repeat != "" {
		now := time.Now().UTC()
		date := newValues.Date
		if date == "" {
			// Если дата не указана, получаем текущую дату из БД
			var currentDate string
			err := db.QueryRow("SELECT date FROM scheduler WHERE id = ?", id).Scan(&currentDate)
			if err != nil {
				return domain.Task{}, fmt.Errorf("failed to get current date: %w", err)
			}
			date = currentDate
		}

		if _, err := util.NextDate(now, date, newValues.Repeat); err != nil {
			return domain.Task{}, fmt.Errorf("invalid repeat rule: %w", err)
		}
	}

	query := "UPDATE scheduler SET "
	var args []interface{}
	var updates []string

	if newValues.Date != "" {
		updates = append(updates, "date = ?")
		args = append(args, newValues.Date)
	}
	if newValues.Title != "" {
		updates = append(updates, "title = ?")
		args = append(args, newValues.Title)
	}
	if newValues.Comment != "" {
		updates = append(updates, "comment = ?")
		args = append(args, newValues.Comment)
	}
	if newValues.Repeat != "" {
		updates = append(updates, "repeat = ?")
		args = append(args, newValues.Repeat)
	}

	if len(updates) == 0 {
		return domain.Task{}, errors.New("nothing to update")
	}

	query += strings.Join(updates, ", ") + " WHERE id = ?"
	args = append(args, id)

	_, err := db.Exec(query, args...)
	if err != nil {
		return domain.Task{}, fmt.Errorf("database error: %w", err)
	}

	var task domain.Task
	err = db.QueryRow(
		"SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?",
		id,
	).Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		return domain.Task{}, fmt.Errorf("failed to get updated task: %w", err)
	}

	return task, nil
}

func AddTaskStory(db *sql.DB, task domain.Task) (int64, error) {
	result, err := db.Exec(
		"INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)",
		task.Date, task.Title, task.Comment, task.Repeat,
	)
	if err != nil {
		return 0, fmt.Errorf("database error: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get task ID: %w", err)
	}

	return id, nil
}

func GetTasksStory(db *sql.DB, search string, limit int) ([]*domain.Task, error) {
	query, args := buildQuery(search, limit)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("database query error: %v", err)
	}
	defer rows.Close()

	tasks := make([]*domain.Task, 0)
	for rows.Next() {
		var t domain.Task
		if err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat); err != nil {
			return nil, fmt.Errorf("row scan error: %v", err)
		}
		tasks = append(tasks, &t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %v", err)
	}

	return tasks, nil
}

func buildQuery(search string, limit int) (string, []interface{}) {
	baseQuery := "SELECT id, date, title, comment, repeat FROM scheduler"
	where := []string{}
	args := []interface{}{}

	if search != "" {
		if t, err := time.Parse("02.01.2006", search); err == nil {
			where = append(where, "date = ?")
			args = append(args, t.Format("20060102"))
		} else {
			searchTerm := "%" + search + "%"
			where = append(where, "(title LIKE ? OR comment LIKE ?)")
			args = append(args, searchTerm, searchTerm)
		}
	}

	if len(where) > 0 {
		baseQuery += " WHERE " + strings.Join(where, " AND ")
	}

	baseQuery += " ORDER BY date ASC LIMIT ?"
	args = append(args, limit)

	return baseQuery, args
}
