package application

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

// Tool representa una herramienta/acción que el chatbot puede ejecutar
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Execute     func(args string) (string, error)
}

// ReservationTools contiene todas las herramientas relacionadas con reservas
type ReservationTools struct {
	habitacionRepo domain.HabitacionRepository
	reservaService *ReservaService
	personRepo     domain.PersonRepository
	clientRepo     domain.ClientRepository
}

func NewReservationTools(
	habitacionRepo domain.HabitacionRepository,
	reservaService *ReservaService,
	personRepo domain.PersonRepository,
	clientRepo domain.ClientRepository,
) *ReservationTools {
	return &ReservationTools{
		habitacionRepo: habitacionRepo,
		reservaService: reservaService,
		personRepo:     personRepo,
		clientRepo:     clientRepo,
	}
}

// GetAvailableTools retorna todas las herramientas disponibles
func (rt *ReservationTools) GetAvailableTools() []Tool {
	return []Tool{
		{
			Name:        "get_room_types",
			Description: "Obtiene todos los tipos de habitaciones disponibles en el hotel con sus precios y características",
			Execute:     rt.GetRoomTypes,
		},
		{
			Name:        "check_availability",
			Description: "Verifica la disponibilidad de habitaciones para fechas específicas. Args: {\"fechaEntrada\": \"YYYY-MM-DD\", \"fechaSalida\": \"YYYY-MM-DD\"}",
			Execute:     rt.CheckAvailability,
		},
		{
			Name:        "calculate_price",
			Description: "Calcula el precio total de una reserva. Args: {\"tipoHabitacionId\": 1, \"fechaEntrada\": \"YYYY-MM-DD\", \"fechaSalida\": \"YYYY-MM-DD\"}",
			Execute:     rt.CalculatePrice,
		},
		{
			Name:        "create_reservation",
			Description: "Crea una nueva reserva. Args: JSON con todos los datos de la reserva incluyendo fechas, habitación, datos personales del cliente",
			Execute:     rt.CreateReservation,
		},
	}
}

// GetRoomTypes obtiene todos los tipos de habitaciones
func (rt *ReservationTools) GetRoomTypes(args string) (string, error) {
	tipos, err := rt.habitacionRepo.GetRoomTypes()
	if err != nil {
		return "", fmt.Errorf("error al obtener tipos de habitaciones: %w", err)
	}

	var result strings.Builder
	result.WriteString("Tipos de Habitaciones Disponibles:\n\n")

	for _, tipo := range tipos {
		result.WriteString(fmt.Sprintf("• %s (ID: %d)\n", tipo.Titulo, tipo.ID))
		result.WriteString(fmt.Sprintf("  Precio: S/%.2f por noche\n", tipo.Precio))
		result.WriteString(fmt.Sprintf("  Capacidad: %d adultos, %d niños\n", tipo.CapacidadAdultos, tipo.CapacidadNinhos))
		result.WriteString(fmt.Sprintf("  Camas: %d\n", tipo.CantidadCamas))
		result.WriteString(fmt.Sprintf("  Descripción: %s\n\n", tipo.Descripcion))
	}

	return result.String(), nil
}

// CheckAvailability verifica disponibilidad para fechas específicas
func (rt *ReservationTools) CheckAvailability(args string) (string, error) {
	var input struct {
		FechaEntrada string `json:"fechaEntrada"`
		FechaSalida  string `json:"fechaSalida"`
	}

	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", fmt.Errorf("argumentos inválidos: %w", err)
	}

	fechaEntrada, err := time.Parse("2006-01-02", input.FechaEntrada)
	if err != nil {
		return "", fmt.Errorf("fecha de entrada inválida: %w", err)
	}

	fechaSalida, err := time.Parse("2006-01-02", input.FechaSalida)
	if err != nil {
		return "", fmt.Errorf("fecha de salida inválida: %w", err)
	}

	disponibles, err := rt.habitacionRepo.GetAvailableRooms(fechaEntrada, fechaSalida)
	if err != nil {
		return "", fmt.Errorf("error al verificar disponibilidad: %w", err)
	}

	if len(disponibles) == 0 {
		return fmt.Sprintf("No hay habitaciones disponibles para las fechas %s a %s", input.FechaEntrada, input.FechaSalida), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Habitaciones disponibles para %s - %s:\n\n", input.FechaEntrada, input.FechaSalida))

	for _, tipo := range disponibles {
		result.WriteString(fmt.Sprintf("✅ %s (ID: %d)\n", tipo.Titulo, tipo.ID))
		result.WriteString(fmt.Sprintf("   Precio: S/%.2f por noche\n", tipo.Precio))
		result.WriteString(fmt.Sprintf("   Capacidad: %d adultos, %d niños\n\n", tipo.CapacidadAdultos, tipo.CapacidadNinhos))
	}

	return result.String(), nil
}

// CalculatePrice calcula el precio total de una reserva
func (rt *ReservationTools) CalculatePrice(args string) (string, error) {
	var input struct {
		TipoHabitacionID int    `json:"tipoHabitacionId"`
		FechaEntrada     string `json:"fechaEntrada"`
		FechaSalida      string `json:"fechaSalida"`
	}

	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", fmt.Errorf("argumentos inválidos: %w", err)
	}

	// Obtener el tipo de habitación
	tipo, err := rt.habitacionRepo.GetRoomTypeByID(input.TipoHabitacionID)
	if err != nil {
		return "", fmt.Errorf("error al obtener tipo de habitación: %w", err)
	}

	// Parsear fechas
	fechaEntrada, err := time.Parse("2006-01-02", input.FechaEntrada)
	if err != nil {
		return "", fmt.Errorf("fecha de entrada inválida: %w", err)
	}

	fechaSalida, err := time.Parse("2006-01-02", input.FechaSalida)
	if err != nil {
		return "", fmt.Errorf("fecha de salida inválida: %w", err)
	}

	// Calcular noches
	noches := int(fechaSalida.Sub(fechaEntrada).Hours() / 24)
	if noches < 1 {
		noches = 1
	}

	total := tipo.Precio * float64(noches)

	result := fmt.Sprintf("Cálculo de Precio:\n\n"+
		"Habitación: %s\n"+
		"Precio por noche: S/%.2f\n"+
		"Número de noches: %d\n"+
		"Total: S/%.2f\n",
		tipo.Titulo, tipo.Precio, noches, total)

	return result, nil
}

// CreateReservation crea una nueva reserva
func (rt *ReservationTools) CreateReservation(args string) (string, error) {
	log.Printf("CreateReservation called with args: %s", args)

	var input struct {
		FechaEntrada     string                      `json:"fechaEntrada"`
		FechaSalida      string                      `json:"fechaSalida"`
		CantidadAdultos  int                         `json:"cantidadAdultos"`
		CantidadNinhos   int                         `json:"cantidadNinhos"`
		TipoHabitacionID int                         `json:"tipoHabitacionId"`
		PersonalData     domain.PersonalDataInput    `json:"personalData"`
	}

	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", fmt.Errorf("argumentos inválidos: %w", err)
	}

	// Validaciones básicas
	if input.FechaEntrada == "" || input.FechaSalida == "" {
		return "", fmt.Errorf("fechas de entrada y salida son requeridas")
	}

	if input.CantidadAdultos < 1 {
		return "", fmt.Errorf("debe haber al menos 1 adulto")
	}

	if input.TipoHabitacionID < 1 {
		return "", fmt.Errorf("tipo de habitación inválido")
	}

	// Parsear fechas
	fechaEntrada, err := time.Parse("2006-01-02", input.FechaEntrada)
	if err != nil {
		return "", fmt.Errorf("fecha de entrada inválida: %w", err)
	}

	fechaSalida, err := time.Parse("2006-01-02", input.FechaSalida)
	if err != nil {
		return "", fmt.Errorf("fecha de salida inválida: %w", err)
	}

	// Verificar que la fecha de entrada no sea en el pasado
	if fechaEntrada.Before(time.Now().Truncate(24 * time.Hour)) {
		return "", fmt.Errorf("la fecha de entrada no puede ser en el pasado")
	}

	// Buscar una habitación disponible del tipo especificado
	habitacionID, err := rt.habitacionRepo.FindAvailableRoomByType(input.TipoHabitacionID, fechaEntrada, fechaSalida)
	if err != nil {
		return "", fmt.Errorf("no hay habitaciones disponibles del tipo seleccionado para esas fechas: %w", err)
	}

	// Obtener el precio del tipo de habitación
	tipo, err := rt.habitacionRepo.GetRoomTypeByID(input.TipoHabitacionID)
	if err != nil {
		return "", fmt.Errorf("error al obtener tipo de habitación: %w", err)
	}

	// Calcular noches y subtotal
	noches := int(fechaSalida.Sub(fechaEntrada).Hours() / 24)
	if noches < 1 {
		noches = 1
	}
	subtotal := tipo.Precio * float64(noches)

	// Crear la persona
	person := &domain.Person{
		Name:             input.PersonalData.Nombre,
		FirstSurname:     input.PersonalData.PrimerApellido,
		SecondSurname:    input.PersonalData.SegundoApellido,
		DocumentNumber:   input.PersonalData.NumeroDocumento,
		Gender:           input.PersonalData.Genero,
		Email:            input.PersonalData.Correo,
		Phone1:           input.PersonalData.Telefono1,
		Phone2:           input.PersonalData.Telefono2,
		ReferenceCity:    input.PersonalData.CiudadReferencia,
		ReferenceCountry: input.PersonalData.PaisReferencia,
	}

	// Crear la reserva
	reserva := &domain.Reserva{
		CantidadAdultos:   input.CantidadAdultos,
		CantidadNinhos:    input.CantidadNinhos,
		Estado:            domain.ReservaPendiente,
		Subtotal:          subtotal,
		Descuento:         0,
		FechaConfirmacion: time.Now(),
		Habitaciones: []domain.ReservaHabitacion{
			{
				HabitacionID: habitacionID,
				FechaEntrada: fechaEntrada,
				FechaSalida:  fechaSalida,
				Precio:       tipo.Precio,
				Estado:       1, // Activo
			},
		},
	}

	// Crear la reserva con el cliente
	if err := rt.reservaService.CreateReservaWithClient(person, reserva); err != nil {
		return "", fmt.Errorf("error al crear la reserva: %w", err)
	}

	result := fmt.Sprintf("✅ Reserva creada exitosamente!\n\n"+
		"Número de Reserva: #%d\n"+
		"Cliente: %s %s\n"+
		"Email: %s\n"+
		"Tipo de Habitación: %s\n"+
		"Check-in: %s\n"+
		"Check-out: %s\n"+
		"Noches: %d\n"+
		"Adultos: %d\n"+
		"Niños: %d\n"+
		"Total: S/%.2f\n"+
		"Estado: %s\n\n"+
		"Se ha enviado un email de confirmación a %s",
		reserva.ID,
		person.Name, person.FirstSurname,
		person.Email,
		tipo.Titulo,
		input.FechaEntrada,
		input.FechaSalida,
		noches,
		input.CantidadAdultos,
		input.CantidadNinhos,
		subtotal,
		reserva.Estado,
		person.Email,
	)

	return result, nil
}

// ExecuteTool ejecuta una herramienta por nombre
func (rt *ReservationTools) ExecuteTool(toolName string, args string) (string, error) {
	tools := rt.GetAvailableTools()

	for _, tool := range tools {
		if tool.Name == toolName {
			return tool.Execute(args)
		}
	}

	return "", fmt.Errorf("herramienta '%s' no encontrada", toolName)
}

// GetToolDescriptions retorna descripciones de las herramientas para incluir en el prompt
func (rt *ReservationTools) GetToolDescriptions() string {
	var sb strings.Builder

	sb.WriteString("\n=== HERRAMIENTAS DISPONIBLES ===\n\n")
	sb.WriteString("Puedes usar las siguientes herramientas para ayudar al usuario:\n\n")

	tools := rt.GetAvailableTools()
	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("• %s: %s\n", tool.Name, tool.Description))
	}

	sb.WriteString("\nPara usar una herramienta, responde EXACTAMENTE en este formato:\n")
	sb.WriteString("[USE_TOOL: nombre_herramienta]\n")
	sb.WriteString("{\"arg1\": \"value1\", \"arg2\": \"value2\"}\n")
	sb.WriteString("[END_TOOL]\n\n")

	sb.WriteString("Ejemplos:\n")
	sb.WriteString("[USE_TOOL: get_room_types]\n")
	sb.WriteString("{}\n")
	sb.WriteString("[END_TOOL]\n\n")

	sb.WriteString("[USE_TOOL: check_availability]\n")
	sb.WriteString("{\"fechaEntrada\": \"2025-12-01\", \"fechaSalida\": \"2025-12-05\"}\n")
	sb.WriteString("[END_TOOL]\n\n")

	return sb.String()
}
