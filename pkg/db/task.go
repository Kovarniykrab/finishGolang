package db

import (
	"fmt"
	"strings"
	"time"
)

type Task struct {
	ID      int64  `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

func GetTasks(search string, limit int) ([]*Task, error) {
	query, args := buildQuery(search, limit)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("database query error: %v", err)
	}
	defer rows.Close()

	tasks := make([]*Task, 0)
	for rows.Next() {
		var t Task
		err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
		if err != nil {
			return nil, fmt.Errorf("row scan error: %v", err)
		}
		tasks = append(tasks, &t)
	}

	// Проверяем ошибки после итерации
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
		// Проверяем формат даты DD.MM.YYYY
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
