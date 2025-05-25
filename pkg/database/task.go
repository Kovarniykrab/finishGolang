package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Kovarniykrab/finishGolang/pkg/domain"
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
