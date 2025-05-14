package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite" // Драйвер SQLite
)

const dbFile = "shleduler.db"

func InitDB() (*sql.DB, error) {
	_, err := os.Stat(dbFile)
	install := os.IsNotExist(err)

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("Не удалось открыть базу данных: %v\n", err)
	}

	if install {
		if err := createTable(db); err != nil {
			return nil, fmt.Errorf("Не удалось создать таблицу: %v\n, err")
		}
		log.Println("База данных успешно создана")
	}

	return db, nil
}

func createTable(db *sql.DB) error {
	query := `
	CREATE TABLE scheduler (
	        id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL,
			title TEXT NOT NULL,
			comment TEXT,
			repeat TEXT
			);
			CREATE INDEX idx_date ON scheduler(date);`

	_, err := db.Exec(query)
	return err
}
