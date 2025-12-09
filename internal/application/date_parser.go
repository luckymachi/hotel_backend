package application

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DateParser maneja el parseo de fechas en lenguaje natural
type DateParser struct{}

// ParseNaturalDate intenta parsear una fecha en lenguaje natural
func (dp *DateParser) ParseNaturalDate(input string, baseDate time.Time) (*time.Time, error) {
	input = strings.ToLower(strings.TrimSpace(input))

	// Intentar formatos estándar primero
	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"02-01-2006",
		"2/1/2006",
		"2-1-2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, input); err == nil {
			return &t, nil
		}
	}

	// Palabras clave para "hoy"
	if strings.Contains(input, "hoy") {
		return &baseDate, nil
	}

	// Palabras clave para "mañana"
	if strings.Contains(input, "mañana") || strings.Contains(input, "manana") {
		tomorrow := baseDate.AddDate(0, 0, 1)
		return &tomorrow, nil
	}

	// Palabras clave para "pasado mañana"
	if strings.Contains(input, "pasado mañana") || strings.Contains(input, "pasado manana") {
		dayAfterTomorrow := baseDate.AddDate(0, 0, 2)
		return &dayAfterTomorrow, nil
	}

	// Siguiente semana
	if strings.Contains(input, "próxima semana") || strings.Contains(input, "proxima semana") ||
		strings.Contains(input, "siguiente semana") || strings.Contains(input, "la semana que viene") {
		nextWeek := baseDate.AddDate(0, 0, 7)
		// Ajustar al lunes de esa semana
		for nextWeek.Weekday() != time.Monday {
			nextWeek = nextWeek.AddDate(0, 0, 1)
		}
		return &nextWeek, nil
	}

	// Próximo mes
	if strings.Contains(input, "próximo mes") || strings.Contains(input, "proximo mes") ||
		strings.Contains(input, "siguiente mes") || strings.Contains(input, "mes que viene") {
		nextMonth := baseDate.AddDate(0, 1, 0)
		// Primer día del próximo mes
		nextMonth = time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, nextMonth.Location())
		return &nextMonth, nil
	}

	// Fin de semana (próximo sábado)
	if strings.Contains(input, "fin de semana") || strings.Contains(input, "este fin de semana") {
		nextSaturday := baseDate
		for nextSaturday.Weekday() != time.Saturday {
			nextSaturday = nextSaturday.AddDate(0, 0, 1)
		}
		return &nextSaturday, nil
	}

	// En X días
	re := regexp.MustCompile(`en\s+(\d+)\s+d[íi]as?`)
	if matches := re.FindStringSubmatch(input); len(matches) > 1 {
		days, _ := strconv.Atoi(matches[1])
		future := baseDate.AddDate(0, 0, days)
		return &future, nil
	}

	// Dentro de X semanas
	re = regexp.MustCompile(`(?:dentro de|en)\s+(\d+)\s+semanas?`)
	if matches := re.FindStringSubmatch(input); len(matches) > 1 {
		weeks, _ := strconv.Atoi(matches[1])
		future := baseDate.AddDate(0, 0, weeks*7)
		return &future, nil
	}

	// Días de la semana (próximo lunes, martes, etc.)
	weekdays := map[string]time.Weekday{
		"lunes":     time.Monday,
		"martes":    time.Tuesday,
		"miércoles": time.Wednesday,
		"miercoles": time.Wednesday,
		"jueves":    time.Thursday,
		"viernes":   time.Friday,
		"sábado":    time.Saturday,
		"sabado":    time.Saturday,
		"domingo":   time.Sunday,
	}

	for dayName, weekday := range weekdays {
		if strings.Contains(input, dayName) {
			nextDay := baseDate
			// Avanzar hasta el próximo día de la semana
			for nextDay.Weekday() != weekday || nextDay.Equal(baseDate) {
				nextDay = nextDay.AddDate(0, 0, 1)
			}
			return &nextDay, nil
		}
	}

	return nil, fmt.Errorf("no se pudo parsear la fecha: %s", input)
}

// ExtractDateRange intenta extraer un rango de fechas del mensaje
func (dp *DateParser) ExtractDateRange(message string) (*time.Time, *time.Time, error) {
	message = strings.ToLower(message)
	now := time.Now().Truncate(24 * time.Hour)

	// Patrones como "del 15 al 20 de diciembre"
	re := regexp.MustCompile(`del\s+(\d+)\s+al\s+(\d+)\s+de\s+(\w+)`)
	if matches := re.FindStringSubmatch(message); len(matches) > 3 {
		startDay, _ := strconv.Atoi(matches[1])
		endDay, _ := strconv.Atoi(matches[2])
		monthName := matches[3]

		month := dp.parseMonthName(monthName)
		if month != 0 {
			year := now.Year()
			// Si el mes ya pasó este año, usar el próximo año
			if int(month) < int(now.Month()) {
				year++
			}

			startDate := time.Date(year, month, startDay, 0, 0, 0, 0, now.Location())
			endDate := time.Date(year, month, endDay, 0, 0, 0, 0, now.Location())

			return &startDate, &endDate, nil
		}
	}

	// Patrones como "desde el 15/12 hasta el 20/12"
	re = regexp.MustCompile(`desde\s+(?:el\s+)?(\d+)/(\d+)\s+hasta\s+(?:el\s+)?(\d+)/(\d+)`)
	if matches := re.FindStringSubmatch(message); len(matches) > 4 {
		startDay, _ := strconv.Atoi(matches[1])
		startMonth, _ := strconv.Atoi(matches[2])
		endDay, _ := strconv.Atoi(matches[3])
		endMonth, _ := strconv.Atoi(matches[4])

		year := now.Year()
		if startMonth < int(now.Month()) {
			year++
		}

		startDate := time.Date(year, time.Month(startMonth), startDay, 0, 0, 0, 0, now.Location())
		endDate := time.Date(year, time.Month(endMonth), endDay, 0, 0, 0, 0, now.Location())

		return &startDate, &endDate, nil
	}

	// Patrón como "3 noches desde mañana"
	re = regexp.MustCompile(`(\d+)\s+noches?\s+desde\s+(.+)`)
	if matches := re.FindStringSubmatch(message); len(matches) > 2 {
		nights, _ := strconv.Atoi(matches[1])
		startDateStr := matches[2]

		startDate, err := dp.ParseNaturalDate(startDateStr, now)
		if err == nil {
			endDate := startDate.AddDate(0, 0, nights)
			return startDate, &endDate, nil
		}
	}

	// Patrón como "del 15 al 20" (mes actual o siguiente)
	re = regexp.MustCompile(`del\s+(\d+)\s+al\s+(\d+)`)
	if matches := re.FindStringSubmatch(message); len(matches) > 2 {
		startDay, _ := strconv.Atoi(matches[1])
		endDay, _ := strconv.Atoi(matches[2])

		month := now.Month()
		year := now.Year()

		// Si el día de inicio ya pasó este mes, usar el próximo mes
		if startDay < now.Day() {
			month++
			if month > 12 {
				month = 1
				year++
			}
		}

		startDate := time.Date(year, month, startDay, 0, 0, 0, 0, now.Location())
		endDate := time.Date(year, month, endDay, 0, 0, 0, 0, now.Location())

		return &startDate, &endDate, nil
	}

	return nil, nil, fmt.Errorf("no se pudo extraer rango de fechas")
}

// parseMonthName convierte nombre de mes en español a time.Month
func (dp *DateParser) parseMonthName(name string) time.Month {
	months := map[string]time.Month{
		"enero":      time.January,
		"febrero":    time.February,
		"marzo":      time.March,
		"abril":      time.April,
		"mayo":       time.May,
		"junio":      time.June,
		"julio":      time.July,
		"agosto":     time.August,
		"septiembre": time.September,
		"setiembre":  time.September,
		"octubre":    time.October,
		"noviembre":  time.November,
		"diciembre":  time.December,
	}

	return months[strings.ToLower(name)]
}

// ExtractNumbers extrae números del texto
func (dp *DateParser) ExtractNumbers(text string) []int {
	re := regexp.MustCompile(`\d+`)
	matches := re.FindAllString(text, -1)

	numbers := make([]int, 0, len(matches))
	for _, match := range matches {
		if num, err := strconv.Atoi(match); err == nil {
			numbers = append(numbers, num)
		}
	}

	return numbers
}
