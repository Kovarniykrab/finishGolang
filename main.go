package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/yourusername/finishGolang/pkg/api"
	"github.com/yourusername/finishGolang/pkg/db"
	"github.com/yourusername/finishGolang/pkg/server"
)

const defaultPort = 7540

func main() {
	// Инициализация БД
	database, err := db.InitDB()
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	defer database.Close()

	// Регистрация API обработчиков
	api.RegisterHandlers(database)

	// Получаем порт и запускаем сервер
	port := getPort()
	if err := server.Start(port); err != nil {
		fmt.Printf("Ошибка при запуске сервера: %v\n", err)
		os.Exit(1)
	}
}

func getPort() int {
	portStr := os.Getenv("TODO_PORT")
	if portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err == nil && port > 0 {
			return port
		}
	}
	return defaultPort
}
