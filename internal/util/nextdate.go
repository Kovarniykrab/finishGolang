package util

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	DateFormat string = "20060102"
)

func NextDate(now time.Time, date string, repeat string) (string, error) {
	Start, err := time.Parse(DateFormat, date)
	if err != nil {
		return "", fmt.Errorf("неверный формат начальной даты: %v", err)
	}

	Repeat := strings.Split(repeat, " ")

	if len(Repeat) != 0 {
		switch Repeat[0] {
		case "d":
			if len(Repeat) < 2 {
				return "", fmt.Errorf("неверный формат repeat для d: %s", repeat)
			}
			days, err := strconv.Atoi(Repeat[1])
			if err != nil || days > 400 || days < 1 {
				return "", fmt.Errorf("неверное количество дней: %v", err)
			}
			Start = Start.AddDate(0, 0, days)
			for Start.Format(DateFormat) <= now.Format(DateFormat) {
				Start = Start.AddDate(0, 0, days)
			}
			return Start.Format(DateFormat), nil
		case "y":
			if len(Repeat) != 1 {
				return "", fmt.Errorf("неверный формат repeat для y: %s", repeat)
			}
			Start = Start.AddDate(1, 0, 0)
			for Start.Format(DateFormat) <= now.Format(DateFormat) {
				Start = Start.AddDate(1, 0, 0)
			}
			return Start.Format(DateFormat), nil
		case "w":
			if len(Repeat) != 2 {
				return "", fmt.Errorf("неверный формат repeat для w: %s", repeat)
			}
			week := strings.Split(Repeat[1], ",")

			var targetDays []int

			for _, w := range week {
				day, err := strconv.Atoi(w)
				if err != nil || day < 1 || day > 7 {
					return "", fmt.Errorf("неверный день недели: %v", err)
				}
				targetDays = append(targetDays, day)
			}

			for {
				weekDay := int(Start.Weekday())
				if weekDay == 0 {
					weekDay = 7
				}

				for _, day := range targetDays {
					if weekDay == day && Start.Format(DateFormat) > now.Format(DateFormat) {
						return Start.Format(DateFormat), nil
					}
				}
				Start = Start.AddDate(0, 0, 1)
			}
		case "m":
			if len(Repeat) < 2 {
				return "", fmt.Errorf("неверный формат repeat для m: %s", repeat)
			}

			var months []int
			if len(Repeat) > 2 {
				monthStr := strings.Split(Repeat[2], ",")
				for _, m := range monthStr {
					month, err := strconv.Atoi(m)
					if err != nil || month < 1 || month > 12 {
						return "", fmt.Errorf("неверный месяц: %v", err)
					}
					months = append(months, month)
				}
			}

			days := strings.Split(Repeat[1], ",")
			for {
				dayMatch := false
				for _, d := range days {
					day, err := strconv.Atoi(d)
					if err != nil || day > 31 || day < -2 {
						return "", fmt.Errorf("неверный день: %v", err)
					}
					if day < 0 {
						lastDay := time.Date(Start.Year(), Start.Month()+1, 0, 0, 0, 0, 0, Start.Location()).Day()
						calculatedDay := lastDay + day + 1
						if calculatedDay < 1 {
							return "", fmt.Errorf("некорректный день месяца: %d", calculatedDay)
						}
						day = calculatedDay
					}
					if Start.Day() == day {
						dayMatch = true
						break
					}
				}

				monthMatched := len(months) == 0
				for _, m := range months {
					if int(Start.Month()) == m {
						monthMatched = true
						break
					}
				}

				if dayMatch && monthMatched && Start.Format(DateFormat) > now.Format(DateFormat) {
					return Start.Format(DateFormat), nil
				}
				Start = Start.AddDate(0, 0, 1)

			}

		default:
			return "", fmt.Errorf("правило повторения указано в неправильном формате: %v", repeat)
		}
	}

	return now.Format(DateFormat), nil
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
		now, err = time.Parse(DateFormat, nowStr)
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
