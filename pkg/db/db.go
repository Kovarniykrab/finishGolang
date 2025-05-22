package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite" // Драйвер SQLite
)

var DB *sql.DB

const dbFile = "scheduler.db" // Исправлено имя файла

func InitDB() (*sql.DB, error) {
	// Удалим старый файл БД, если он существует (для тестов)
	if os.Getenv("GO_TEST") == "1" {
		os.Remove(dbFile)
	}

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть БД: %v", err) // Убрал \n
	}

	// Всегда пытаемся создать таблицу (используем IF NOT EXISTS)
	if err := createTable(db); err != nil {
		return nil, fmt.Errorf("failed to create table: %v", err)
	}

	// Проверяем соединение с БД
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database connection failed: %v", err)
	}

	log.Println("База данных успешно инициализирована")

	DB = db
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
