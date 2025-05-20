package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const dateFormat = "20060102"

// NextDate вычисляет следующую дату выполнения задачи
func NextDate(now time.Time, dstart string, repeat string) (string, error) {
	// Проверка на пустое правило
	if repeat == "" {
		return "", errors.New("пустое правило повторения")
	}

	// Парсинг начальной даты
	date, err := time.Parse(dateFormat, dstart)
	if err != nil {
		return "", fmt.Errorf("неверный формат начальной даты: %v", err)
	}

	// Разбиваем правило на части
	parts := strings.Fields(repeat)
	if len(parts) == 0 {
		return "", errors.New("неверный формат правила")
	}

	// Обработка разных типов правил
	switch parts[0] {
	case "d":
		// Обработка ежедневного правила
		if len(parts) != 2 {
			return "", errors.New("неверный формат ежедневного правила")
		}

		days, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", fmt.Errorf("неверное количество дней: %v", err)
		}

		if days <= 0 || days > 400 {
			return "", errors.New("количество дней должно быть от 1 до 400")
		}

		// Добавляем дни, пока не превысим текущую дату
		for !date.After(now) {
			date = date.AddDate(0, 0, days)
		}

	case "y":
		// Обработка ежегодного правила
		if len(parts) != 1 {
			return "", errors.New("неверный формат ежегодного правила")
		}

		// Добавляем год, пока не превысим текущую дату
		for !date.After(now) {
			date = date.AddDate(1, 0, 0)
		}

	default:
		// Для остальных правил пока возвращаем ошибку
		return "", errors.New("неподдерживаемый формат правила")
	}

	return date.Format(dateFormat), nil
}

// NextDateHandler обработчик для /api/nextdate
func NextDateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры запроса
	nowStr := r.FormValue("now")
	date := r.FormValue("date")
	repeat := r.FormValue("repeat")

	// Если now не указан, используем текущую дату
	var now time.Time
	if nowStr == "" {
		now = time.Now()
	} else {
		var err error
		now, err = time.Parse(dateFormat, nowStr)
		if err != nil {
			http.Error(w, "Invalid now date format", http.StatusBadRequest)
			return
		}
	}

	// Вычисляем следующую дату
	nextDate, err := NextDate(now, date, repeat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Возвращаем результат
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(nextDate))
}
