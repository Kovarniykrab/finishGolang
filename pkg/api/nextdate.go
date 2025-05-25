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
func NextDate(now time.Time, startDate string, rule string) (string, error) {

	const dateLayout = "20060102"

	// Парсинг начальной даты
	parsedDate, err := time.Parse(dateLayout, startDate)
	if err != nil {
		return "", fmt.Errorf("invalid date format: %v", err)
	}

	// Обработка пустого правила
	if strings.TrimSpace(rule) == "" {
		return "", errors.New("repeat rule is required")
	}

	// Определение типа правила
	switch {
	case rule == "y":
		next := parsedDate
		maxIterations := 1000
		for i := 0; i < maxIterations; i++ {
			next = next.AddDate(1, 0, 0)
			if next.Month() == time.February && next.Day() == 29 {
				if !isLeap(next.Year()) {
					next = time.Date(next.Year(), time.March, 1, 0, 0, 0, 0, time.UTC)
				}
			}
			if next.After(now) {
				return next.Format(dateLayout), nil
			}
		}
		return "", errors.New("max iterations reached")
	case strings.HasPrefix(rule, "d "):
		// Ежедневное повторение
		parts := strings.Split(rule, " ")
		if len(parts) != 2 {
			return "", errors.New("invalid daily rule format")
		}

		interval, err := strconv.Atoi(parts[1])
		if err != nil || interval < 1 || interval > 400 {
			return "", errors.New("invalid days interval (1-400)")
		}

		// Используем начальную дату как отправную точку
		next := parsedDate
		for {
			if next.After(now) {
				break
			}
			next = next.AddDate(0, 0, interval)
		}

		// Если начальная дата уже в будущем, возвращаем её
		if parsedDate.After(now) {
			return parsedDate.Format(dateLayout), nil
		}

		return next.Format(dateLayout), nil

	case strings.HasPrefix(rule, "m "):
		// Ежемесячное повторение
		parts := strings.Split(rule, " ")
		if len(parts) < 2 {
			return "", errors.New("invalid monthly rule")
		}

		days := make([]int, 0)
		for _, s := range parts[1:] {
			day, err := strconv.Atoi(s)
			if err != nil || day < 1 || day > 31 {
				continue // Пропускаем невалидные дни
			}
			days = append(days, day)
		}

		if len(days) == 0 {
			return "", errors.New("no valid days in monthly rule")
		}

		next := parsedDate
		for {
			next = next.AddDate(0, 1, 0)
			lastDay := time.Date(next.Year(), next.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()

			for _, d := range days {
				currentDay := d
				if currentDay > lastDay {
					currentDay = lastDay
				}

				candidate := time.Date(next.Year(), next.Month(), currentDay, 0, 0, 0, 0, time.UTC)
				if candidate.After(now) {
					return candidate.Format(dateLayout), nil
				}
			}
		}

	default:
		return "", errors.New("unsupported repeat rule")
	}
}

// Вспомогательная функция для определения високосного года
func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
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
