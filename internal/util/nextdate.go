package util

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	dateFormatYMD    = "20060102"
	dateFormatDMYDot = "02.01.2006"
	dateFormatYMDash = "2006-01-02"
)

func NextDate(now time.Time, startDate string, rule string) (string, error) {
	parsedDate, err := time.Parse(dateFormatYMD, startDate)
	if err != nil {
		return "", fmt.Errorf("invalid date format: %w", err)
	}

	if strings.TrimSpace(rule) == "" {
		return "", errors.New("repeat rule is required")
	}

	ruleParts := strings.SplitN(rule, " ", 2)
	switch ruleParts[0] {
	case "y":
		next := parsedDate
		maxIterations := 1000
		for i := 0; i < maxIterations; i++ {
			next = next.AddDate(1, 0, 0)
			if next.Month() == time.February && next.Day() == 29 && !isLeap(next.Year()) {
				next = time.Date(next.Year(), time.March, 1, 0, 0, 0, 0, time.UTC)
			}
			if next.After(now) {
				return next.Format(dateFormatYMD), nil
			}
		}
		return "", errors.New("max iterations reached")

	case "d":
		if len(ruleParts) < 2 {
			return "", errors.New("invalid daily rule format")
		}
		interval, err := strconv.Atoi(ruleParts[1])
		if err != nil || interval < 1 || interval > 400 {
			return "", errors.New("invalid days interval (1-400)")
		}

		next := parsedDate
		for i := 0; i < 400; i++ {
			next = next.AddDate(0, 0, interval)
			if next.After(now) {
				return next.Format(dateFormatYMD), nil
			}
		}
		return "", errors.New("max iterations reached")

	case "w":
		if len(ruleParts) < 2 {
			return "", errors.New("invalid weekly rule format")
		}
		days, err := parseWeekdays(ruleParts[1])
		if err != nil {
			return "", err
		}

		next := parsedDate
		for i := 0; i < 400; i++ {
			next = next.AddDate(0, 0, 1)
			if days[next.Weekday()] && next.After(now) {
				return next.Format(dateFormatYMD), nil
			}
		}
		return "", errors.New("next date not found")

	case "m":
		if len(ruleParts) < 2 {
			return "", errors.New("invalid monthly rule format")
		}
		days, months, err := parseMonthlyRule(ruleParts[1])
		if err != nil {
			return "", err
		}

		next := parsedDate
		for i := 0; i < 365*3; i++ {
			next = next.AddDate(0, 1, 0)
			if len(months) > 0 && !contains(months, int(next.Month())) {
				continue
			}

			lastDay := time.Date(next.Year(), next.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
			for _, d := range days {
				targetDay := d
				if targetDay < 0 {
					targetDay = lastDay + targetDay + 1
					if targetDay < 1 {
						targetDay = 1
					}
				}

				if targetDay > lastDay {
					targetDay = lastDay
				}

				candidate := time.Date(next.Year(), next.Month(), targetDay, 0, 0, 0, 0, time.UTC)
				if candidate.After(now) {
					return candidate.Format(dateFormatYMD), nil
				}
			}
		}
		return "", errors.New("next date not found")

	default:
		return "", errors.New("unsupported repeat rule")
	}
}

func parseWeekdays(input string) (map[time.Weekday]bool, error) {
	days := make(map[time.Weekday]bool)
	for _, s := range strings.Split(input, ",") {
		day, err := strconv.Atoi(s)
		if err != nil || day < 0 || day > 6 {
			return nil, fmt.Errorf("invalid weekday: %s (0-6, 0=Sunday)", s)
		}
		days[time.Weekday(day)] = true
	}
	return days, nil
}

func parseMonthlyRule(input string) ([]int, []int, error) {
	parts := strings.Split(input, " ")
	if len(parts) == 0 {
		return nil, nil, errors.New("no days specified")
	}

	// Парсинг дней
	var days []int
	for _, s := range strings.Split(parts[0], ",") {
		day, err := strconv.Atoi(s)
		if err != nil || day < -31 || day > 31 || day == 0 {
			return nil, nil, fmt.Errorf("invalid day value: %s", s)
		}
		days = append(days, day)
	}

	// Парсинг месяцев (опционально)
	var months []int
	if len(parts) > 1 {
		for _, s := range strings.Split(parts[1], ",") {
			month, err := strconv.Atoi(s)
			if err != nil || month < 1 || month > 12 {
				return nil, nil, fmt.Errorf("invalid month value: %s", s)
			}
			months = append(months, month)
		}
	}

	return days, months, nil
}

func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func contains(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func NextDateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nowStr := r.URL.Query().Get("now")
	date := r.URL.Query().Get("date")
	repeat := r.URL.Query().Get("repeat")

	var now time.Time
	if nowStr == "" {
		now = time.Now().UTC()
	} else {
		var err error
		now, err = time.Parse(dateFormatYMD, nowStr)
		if err != nil {
			http.Error(w, "Invalid 'now' date format", http.StatusBadRequest)
			return
		}
	}

	nextDate, err := NextDate(now, date, repeat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(nextDate))
}
