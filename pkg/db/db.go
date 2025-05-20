package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite" // Драйвер SQLite
)

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
		db.Close() // Важно закрыть соединение при ошибке
		return nil, fmt.Errorf("не удалось создать таблицу: %v", err)
	}

	// Проверяем соединение с БД
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("проверка соединения с БД не удалась: %v", err)
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
