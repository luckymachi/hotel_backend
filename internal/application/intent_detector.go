package application

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

// IntentDetector detecta intenciones del usuario y extrae información
type IntentDetector struct {
	reservationTools *ReservationTools
}

func NewIntentDetector(tools *ReservationTools) *IntentDetector {
	return &IntentDetector{
		reservationTools: tools,
	}
}

// DetectedIntent representa una intención detectada con datos extraídos
type DetectedIntent struct {
	Intent           string                        // "check_availability", "book_room", etc.
	FechaEntrada     *string
	FechaSalida      *string
	CantidadAdultos  *int
	CantidadNinhos   *int
	TipoHabitacionID *int
	PersonalData     *domain.PersonalDataInput
	ToolResults      []ToolResult
}

type ToolResult struct {
	ToolName string
	Result   string
	Error    error
}

// DetectAndProcess detecta intenciones del mensaje y ejecuta herramientas automáticamente
func (d *IntentDetector) DetectAndProcess(message string, reservation *domain.ReservationInProgress) (*DetectedIntent, error) {
	detected := &DetectedIntent{
		ToolResults: []ToolResult{},
	}

	msgLower := strings.ToLower(message)

	log.Printf("[IntentDetector] Processing message: %s", message)

	// 1. Detectar y extraer fechas
	fechaEntrada, fechaSalida := d.extractDates(message)
	if fechaEntrada != nil {
		detected.FechaEntrada = fechaEntrada
		log.Printf("[IntentDetector] Detected check-in date: %s", *fechaEntrada)
	}
	if fechaSalida != nil {
		detected.FechaSalida = fechaSalida
		log.Printf("[IntentDetector] Detected check-out date: %s", *fechaSalida)
	}

	// 2. Detectar cantidad de personas
	adultos, ninhos := d.extractGuestCounts(message)
	if adultos != nil {
		detected.CantidadAdultos = adultos
		log.Printf("[IntentDetector] Detected adults: %d", *adultos)
	}
	if ninhos != nil {
		detected.CantidadNinhos = ninhos
		log.Printf("[IntentDetector] Detected children: %d", *ninhos)
	}

	// 3. Si tenemos fechas (del mensaje o del estado), verificar disponibilidad automáticamente
	checkInDate := detected.FechaEntrada
	checkOutDate := detected.FechaSalida

	if reservation != nil {
		if checkInDate == nil && reservation.FechaEntrada != nil {
			checkInDate = reservation.FechaEntrada
		}
		if checkOutDate == nil && reservation.FechaSalida != nil {
			checkOutDate = reservation.FechaSalida
		}
	}

	if checkInDate != nil && checkOutDate != nil {
		log.Printf("[IntentDetector] Checking availability for %s to %s", *checkInDate, *checkOutDate)

		args := fmt.Sprintf(`{"fechaEntrada": "%s", "fechaSalida": "%s"}`, *checkInDate, *checkOutDate)
		result, err := d.reservationTools.CheckAvailability(args)

		detected.ToolResults = append(detected.ToolResults, ToolResult{
			ToolName: "check_availability",
			Result:   result,
			Error:    err,
		})

		detected.Intent = "check_availability"
	}

	// 4. Detectar solicitud de tipos de habitaciones
	if strings.Contains(msgLower, "tipos de habitacion") ||
		strings.Contains(msgLower, "que habitaciones") ||
		strings.Contains(msgLower, "qué habitaciones") ||
		strings.Contains(msgLower, "habitaciones disponibles") ||
		strings.Contains(msgLower, "opciones") ||
		(strings.Contains(msgLower, "mostrar") && strings.Contains(msgLower, "habitacion")) {

		log.Printf("[IntentDetector] Fetching room types")

		result, err := d.reservationTools.GetRoomTypes("{}")
		detected.ToolResults = append(detected.ToolResults, ToolResult{
			ToolName: "get_room_types",
			Result:   result,
			Error:    err,
		})

		if detected.Intent == "" {
			detected.Intent = "show_room_types"
		}
	}

	// 5. Detectar selección de tipo de habitación
	tipoID := d.extractRoomTypeSelection(message)
	if tipoID != nil {
		detected.TipoHabitacionID = tipoID
		log.Printf("[IntentDetector] Detected room type selection: %d", *tipoID)

		// Si tenemos fechas, calcular precio automáticamente
		if checkInDate != nil && checkOutDate != nil {
			log.Printf("[IntentDetector] Calculating price for room type %d", *tipoID)

			args := fmt.Sprintf(`{"tipoHabitacionId": %d, "fechaEntrada": "%s", "fechaSalida": "%s"}`,
				*tipoID, *checkInDate, *checkOutDate)
			result, err := d.reservationTools.CalculatePrice(args)

			detected.ToolResults = append(detected.ToolResults, ToolResult{
				ToolName: "calculate_price",
				Result:   result,
				Error:    err,
			})

			detected.Intent = "calculate_price"
		}
	}

	// 6. Detectar datos personales
	personalData := d.extractPersonalData(message)
	if personalData != nil {
		detected.PersonalData = personalData
		log.Printf("[IntentDetector] Detected personal data for: %s %s", personalData.Nombre, personalData.PrimerApellido)
		detected.Intent = "provide_personal_data"
	}

	// 7. Detectar confirmación para crear reserva
	if d.isConfirmation(message) && reservation != nil && reservation.PersonalData != nil {
		log.Printf("[IntentDetector] Confirmation detected, creating reservation")

		// Crear la reserva automáticamente
		reservaData := map[string]interface{}{
			"fechaEntrada":     reservation.FechaEntrada,
			"fechaSalida":      reservation.FechaSalida,
			"cantidadAdultos":  reservation.CantidadAdultos,
			"cantidadNinhos":   reservation.CantidadNinhos,
			"tipoHabitacionId": reservation.TipoHabitacionID,
			"personalData":     reservation.PersonalData,
		}

		args, _ := json.Marshal(reservaData)
		log.Printf("[IntentDetector] Ejecutando create_reservation con datos: %s", string(args))
		result, err := d.reservationTools.CreateReservation(string(args))

		if err != nil {
			log.Printf("[IntentDetector] ERROR en create_reservation: %v", err)
		} else {
			log.Printf("[IntentDetector] create_reservation exitoso: %s", result)
		}

		detected.ToolResults = append(detected.ToolResults, ToolResult{
			ToolName: "create_reservation",
			Result:   result,
			Error:    err,
		})

		detected.Intent = "create_reservation"
	}

	log.Printf("[IntentDetector] Detected intent: %s, executed %d tools", detected.Intent, len(detected.ToolResults))

	return detected, nil
}

// extractDates extrae fechas del mensaje
func (d *IntentDetector) extractDates(message string) (*string, *string) {
	// Patrón para fechas en formato DD/MM/YYYY, DD-MM-YYYY, YYYY-MM-DD
	datePatterns := []string{
		`\d{4}-\d{2}-\d{2}`,      // YYYY-MM-DD
		`\d{2}/\d{2}/\d{4}`,      // DD/MM/YYYY
		`\d{2}-\d{2}-\d{4}`,      // DD-MM-YYYY
	}

	var dates []string
	for _, pattern := range datePatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(message, -1)
		dates = append(dates, matches...)
	}

	if len(dates) >= 2 {
		// Normalizar fechas a YYYY-MM-DD
		entrada := d.normalizeDate(dates[0])
		salida := d.normalizeDate(dates[1])
		return &entrada, &salida
	} else if len(dates) == 1 {
		entrada := d.normalizeDate(dates[0])
		return &entrada, nil
	}

	// Intentar detectar rangos de fechas en texto natural
	// "del 15 al 20 de diciembre"
	re := regexp.MustCompile(`del?\s+(\d{1,2})\s+al?\s+(\d{1,2})\s+de\s+(\w+)`)
	matches := re.FindStringSubmatch(strings.ToLower(message))
	if len(matches) == 4 {
		day1 := matches[1]
		day2 := matches[2]
		month := d.monthNameToNumber(matches[3])

		if month != "" {
			year := time.Now().Year()
			entrada := fmt.Sprintf("%d-%s-%02s", year, month, day1)
			salida := fmt.Sprintf("%d-%s-%02s", year, month, day2)
			return &entrada, &salida
		}
	}

	return nil, nil
}

// normalizeDate convierte cualquier formato de fecha a YYYY-MM-DD
func (d *IntentDetector) normalizeDate(dateStr string) string {
	// Si ya está en formato YYYY-MM-DD
	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, dateStr); matched {
		return dateStr
	}

	// Convertir DD/MM/YYYY o DD-MM-YYYY a YYYY-MM-DD
	re := regexp.MustCompile(`^(\d{2})[-/](\d{2})[-/](\d{4})$`)
	matches := re.FindStringSubmatch(dateStr)
	if len(matches) == 4 {
		return fmt.Sprintf("%s-%s-%s", matches[3], matches[2], matches[1])
	}

	return dateStr
}

// monthNameToNumber convierte nombre de mes a número
func (d *IntentDetector) monthNameToNumber(month string) string {
	months := map[string]string{
		"enero": "01", "febrero": "02", "marzo": "03", "abril": "04",
		"mayo": "05", "junio": "06", "julio": "07", "agosto": "08",
		"septiembre": "09", "octubre": "10", "noviembre": "11", "diciembre": "12",
	}
	return months[strings.ToLower(month)]
}

// extractGuestCounts extrae cantidad de adultos y niños
func (d *IntentDetector) extractGuestCounts(message string) (*int, *int) {
	var adultos, ninhos *int

	msgLower := strings.ToLower(message)

	// Buscar adultos
	reAdultos := regexp.MustCompile(`(\d+)\s*adult[oa]s?`)
	if matches := reAdultos.FindStringSubmatch(msgLower); len(matches) > 1 {
		if num, err := strconv.Atoi(matches[1]); err == nil {
			adultos = &num
		}
	}

	// Buscar niños
	reNinhos := regexp.MustCompile(`(\d+)\s*niñ[oa]s?`)
	if matches := reNinhos.FindStringSubmatch(msgLower); len(matches) > 1 {
		if num, err := strconv.Atoi(matches[1]); err == nil {
			ninhos = &num
		}
	}

	// Si dice "2 personas" y no especifica niños, asumir que son adultos
	if adultos == nil && ninhos == nil {
		rePersonas := regexp.MustCompile(`(\d+)\s*personas?`)
		if matches := rePersonas.FindStringSubmatch(msgLower); len(matches) > 1 {
			if num, err := strconv.Atoi(matches[1]); err == nil {
				adultos = &num
				cero := 0
				ninhos = &cero
			}
		}
	}

	return adultos, ninhos
}

// extractRoomTypeSelection detecta selección de tipo de habitación
func (d *IntentDetector) extractRoomTypeSelection(message string) *int {
	msgLower := strings.ToLower(message)

	// Buscar "habitación número X" o "tipo X" o "opción X"
	patterns := []string{
		`tipo\s+(\d+)`,
		`habitaci[oó]n\s+(\d+)`,
		`opci[oó]n\s+(\d+)`,
		`n[uú]mero\s+(\d+)`,
		`id\s+(\d+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(msgLower); len(matches) > 1 {
			if num, err := strconv.Atoi(matches[1]); err == nil {
				return &num
			}
		}
	}

	// Buscar menciones específicas de tipos de habitación
	if strings.Contains(msgLower, "doble") {
		// Asumir que tipo 2 es habitación doble (ajustar según tu BD)
		id := 2
		return &id
	}
	if strings.Contains(msgLower, "suite") || strings.Contains(msgLower, "presidencial") {
		id := 1
		return &id
	}

	return nil
}

// extractPersonalData intenta extraer datos personales del mensaje
func (d *IntentDetector) extractPersonalData(message string) *domain.PersonalDataInput {
	// Este es un detector básico - en producción usarías NLP más sofisticado
	// Por ahora, detectamos si el mensaje contiene múltiples campos personales

	hasEmail := strings.Contains(message, "@")
	hasPhone := regexp.MustCompile(`\d{9,10}`).MatchString(message)
	hasDocument := regexp.MustCompile(`(?i)dni|documento.*\d{8}`).MatchString(message)

	// Si tiene al menos 2 de estos campos, probablemente son datos personales
	count := 0
	if hasEmail {
		count++
	}
	if hasPhone {
		count++
	}
	if hasDocument {
		count++
	}

	if count >= 2 {
		// Extraer los datos (implementación básica)
		data := &domain.PersonalDataInput{}

		// Email
		if hasEmail {
			reEmail := regexp.MustCompile(`[\w\.-]+@[\w\.-]+\.\w+`)
			if email := reEmail.FindString(message); email != "" {
				data.Correo = email
			}
		}

		// Teléfono
		if hasPhone {
			rePhone := regexp.MustCompile(`\d{9,10}`)
			if phone := rePhone.FindString(message); phone != "" {
				data.Telefono1 = phone
			}
		}

		// Documento
		reDoc := regexp.MustCompile(`\d{8}`)
		if doc := reDoc.FindString(message); doc != "" {
			data.NumeroDocumento = doc
		}

		// Nombres (buscar palabras capitalizadas)
		words := strings.Fields(message)
		var nombres []string
		for _, word := range words {
			if len(word) > 2 && word[0] >= 'A' && word[0] <= 'Z' && !strings.Contains(word, "@") {
				nombres = append(nombres, word)
			}
		}

		if len(nombres) >= 2 {
			data.Nombre = nombres[0]
			data.PrimerApellido = nombres[1]
			if len(nombres) >= 3 {
				data.SegundoApellido = &nombres[2]
			}
		}

		// Género (básico)
		if strings.Contains(strings.ToLower(message), " m ") || strings.Contains(strings.ToLower(message), "masculino") {
			data.Genero = "M"
		} else if strings.Contains(strings.ToLower(message), " f ") || strings.Contains(strings.ToLower(message), "femenino") {
			data.Genero = "F"
		} else {
			data.Genero = "M" // Default
		}

		// Solo retornar si tenemos al menos nombre, documento y email
		if data.Nombre != "" && data.NumeroDocumento != "" && data.Correo != "" {
			return data
		}
	}

	return nil
}

// isConfirmation detecta si el mensaje es una confirmación
func (d *IntentDetector) isConfirmation(message string) bool {
	msgLower := strings.ToLower(message)

	confirmations := []string{
		"sí", "si", "confirmo", "confirmar", "ok", "okay",
		"adelante", "procede", "proceder", "correcto",
		"de acuerdo", "acepto", "está bien", "esta bien",
	}

	for _, conf := range confirmations {
		if strings.Contains(msgLower, conf) {
			return true
		}
	}

	return false
}
