package server

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/Kovarniykrab/finishGolang/internal/api"
)

func Start(port int, db *sql.DB) error {
	mux := http.NewServeMux()

	// API обработчики
	api.RegisterHandlers(mux, db)

	// Статические файлы (только для корневого пути)
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	fmt.Printf("Сервер запущен на порту %d\n", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}
