package server

import (
	"fmt"
	"net/http"
)

func Start(port int) error {
	http.Handle("/", http.FileServer(http.Dir("./web")))

	fmt.Printf("Сервер запущен на порту %d\n", port)
	fmt.Printf("Откройте http://localhost:%d в браузере\n", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
