package server

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/Kovarniykrab/finishGolang/pkg/api"
)

func Start(port int, db *sql.DB) error {
	// Регистрируем API обработчики
	api.RegisterHandlers(db)

	// Статические файлы
	http.Handle("/", http.FileServer(http.Dir("./web")))

	fmt.Printf("Сервер запущен на порту %d\n", port)
	fmt.Printf("Откройте http://localhost:%d в браузере\n", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
