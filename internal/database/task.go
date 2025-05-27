package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Kovarniykrab/finishGolang/internal/domain"
	"github.com/Kovarniykrab/finishGolang/internal/util"
)

func GetTasks(db *sql.DB, search string, limit int) ([]*domain.Task, error) {
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
func GetTask(db *sql.DB, id int64) (*domain.Task, error) {
	var task domain.Task
	err := db.QueryRow(
		"SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?",
		id,
	).Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func UpdateTask(db *sql.DB, task *domain.Task) error {
	// Валидация заголовка
	task.Title = strings.TrimSpace(task.Title)
	if task.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if len(task.Title) > 100 {
		return fmt.Errorf("title too long (max 100 characters)")
	}

	// Валидация комментария
	task.Comment = strings.TrimSpace(task.Comment)
	if len(task.Comment) > 500 {
		return fmt.Errorf("comment too long (max 500 characters)")
	}

	// Валидация даты
	if _, err := time.Parse("20060102", task.Date); err != nil {
		return fmt.Errorf("invalid date format: %v", err)
	}

	// Обработка повторяющихся задач
	now := time.Now().UTC()
	if task.Repeat != "" {
		nextDate, err := util.NextDate(now, task.Date, task.Repeat)
		if err != nil {
			return fmt.Errorf("invalid repeat rule: %v", err)
		}
		task.Date = nextDate
	} else {
		parsedDate, _ := time.Parse("20060102", task.Date)
		// Проверяем дату только если она не была явно установлена
		if parsedDate.Before(now) {
			task.Date = now.Format("20060102")
		}
	}

	// Обновление в базе данных
	result, err := db.Exec(
		"UPDATE scheduler SET date=?, title=?, comment=?, repeat=? WHERE id=?",
		task.Date, task.Title, task.Comment, task.Repeat, task.ID,
	)
	if err != nil {
		return fmt.Errorf("database error: %v", err)
	}

	// Проверка что запись была обновлена
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update check failed: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
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
