package http

import (
	"fmt"
	"strconv"
	"time"

	"github.com/Maxito7/hotel_backend/internal/application"
	"github.com/Maxito7/hotel_backend/internal/domain"
	"github.com/gofiber/fiber/v2"
)

type ReservaHandler struct {
	service *application.ReservaService
}

// NewReservaHandler crea una nueva instancia del handler de reservas
func NewReservaHandler(service *application.ReservaService) *ReservaHandler {
	return &ReservaHandler{
		service: service,
	}
}

// CreateReservaRequest representa la petición para crear una reserva
type CreateReservaRequest struct {
	CantidadAdultos int                       `json:"cantidadAdultos"`
	CantidadNinhos  int                       `json:"cantidadNinhos"`
	Descuento       float64                   `json:"descuento"`
	Cliente         ClienteData               `json:"cliente"`
	Huespedes       []HuespedData             `json:"huespedes,omitempty"` // Huéspedes adicionales
	Habitaciones    []CreateHabitacionReserva `json:"habitaciones"`
	Servicios       []int                     `json:"servicios,omitempty"` // Array de IDs de servicios
	Pago            *PaymentData              `json:"pago,omitempty"`      // Opcional
}

// PaymentData representa los datos del pago
type PaymentData struct {
	Amount        float64 `json:"amount"`
	PaymentMethod string  `json:"paymentMethod"` // "card", "transfer", "cash"
	Status        string  `json:"status"`        // "pending", "completed", "failed"
}

// ClienteData representa los datos del cliente para crear/buscar la persona
type ClienteData struct {
	Name             string  `json:"name"`
	FirstSurname     string  `json:"firstSurname"`
	SecondSurname    *string `json:"secondSurname,omitempty"`
	DocumentNumber   string  `json:"documentNumber"`
	Gender           string  `json:"gender"`
	Email            string  `json:"email"`
	Phone1           string  `json:"phone1"`
	Phone2           *string `json:"phone2,omitempty"`
	ReferenceCity    string  `json:"referenceCity"`
	ReferenceCountry string  `json:"referenceCountry"`
	BirthDate        string  `json:"birthDate"` // Formato: YYYY-MM-DD
}

// HuespedData representa los datos de un huésped adicional
type HuespedData struct {
	Name           string  `json:"name"`
	FirstSurname   string  `json:"firstSurname"`
	SecondSurname  *string `json:"secondSurname,omitempty"`
	DocumentNumber string  `json:"documentNumber"`
	Gender         string  `json:"gender"`    // M, F, O
	Email          string  `json:"email"`     // Email del huésped
	Phone1         string  `json:"phone1"`    // Teléfono del huésped
	BirthDate      string  `json:"birthDate"` // Formato: YYYY-MM-DD
}

// CreateHabitacionReserva representa una habitación a reservar
type CreateHabitacionReserva struct {
	RoomTypeID   int     `json:"roomTypeId"` // ID del tipo de habitación
	Precio       float64 `json:"precio"`
	FechaEntrada string  `json:"fechaEntrada"` // Formato: YYYY-MM-DD
	FechaSalida  string  `json:"fechaSalida"`  // Formato: YYYY-MM-DD
}

// UpdateEstadoRequest representa la petición para actualizar el estado de una reserva
type UpdateEstadoRequest struct {
	Estado string `json:"estado"`
}

// VerificarDisponibilidadRequest representa la petición para verificar disponibilidad
type VerificarDisponibilidadRequest struct {
	HabitacionID int    `json:"habitacionId"`
	FechaEntrada string `json:"fechaEntrada"` // Formato: YYYY-MM-DD
	FechaSalida  string `json:"fechaSalida"`  // Formato: YYYY-MM-DD
}

// CreateReserva crea una nueva reserva
func (h *ReservaHandler) CreateReserva(c *fiber.Ctx) error {
	var req CreateReservaRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de solicitud inválido",
		})
	}

	// Validaciones básicas
	if req.Cliente.DocumentNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El número de documento del cliente es requerido",
		})
	}

	if len(req.Habitaciones) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Debe incluir al menos una habitación",
		})
	}

	// Parsear fecha de nacimiento
	birthDate, err := time.Parse("2006-01-02", req.Cliente.BirthDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fecha de nacimiento inválido. Use YYYY-MM-DD",
		})
	}

	// Convertir habitaciones y buscar habitación disponible de cada tipo
	habitaciones := make([]domain.ReservaHabitacion, len(req.Habitaciones))
	for i, hab := range req.Habitaciones {
		// Validar que roomTypeId sea válido
		if hab.RoomTypeID <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("roomTypeId inválido en habitación %d. Debe ser mayor a 0. Valor recibido: %d", i+1, hab.RoomTypeID),
			})
		}

		fechaEntrada, err := time.Parse("2006-01-02", hab.FechaEntrada)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Formato de fechaEntrada inválido. Use YYYY-MM-DD",
			})
		}

		fechaSalida, err := time.Parse("2006-01-02", hab.FechaSalida)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Formato de fechaSalida inválido. Use YYYY-MM-DD",
			})
		}

		// Buscar una habitación disponible del tipo especificado
		habitacionID, err := h.service.FindAvailableRoomByType(hab.RoomTypeID, fechaEntrada, fechaSalida)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("No hay habitaciones disponibles del tipo %d para las fechas seleccionadas: %v", hab.RoomTypeID, err),
			})
		}

		habitaciones[i] = domain.ReservaHabitacion{
			HabitacionID: habitacionID,
			Precio:       hab.Precio,
			FechaEntrada: fechaEntrada,
			FechaSalida:  fechaSalida,
			Estado:       1, // Activa
		}
	}

	// Convertir servicios (si se enviaron)
	var servicios []domain.ReservaServicio
	if len(req.Servicios) > 0 {
		// Usar las fechas de la primera habitación para los servicios
		fechaEntrada := habitaciones[0].FechaEntrada
		fechaSalida := habitaciones[0].FechaSalida

		servicios = make([]domain.ReservaServicio, len(req.Servicios))
		for i, servicioID := range req.Servicios {
			servicios[i] = domain.ReservaServicio{
				ServiceID: servicioID,
				StartDate: fechaEntrada,
				EndDate:   fechaSalida,
				Status:    1, // Activo
			}
		}
	}

	// Crear el objeto Person con los datos del cliente
	// Convertir el género del frontend (Masculino/Femenino/Otro) a BD (M/F/O)
	person := &domain.Person{
		Name:             req.Cliente.Name,
		FirstSurname:     req.Cliente.FirstSurname,
		SecondSurname:    req.Cliente.SecondSurname,
		DocumentNumber:   req.Cliente.DocumentNumber,
		Gender:           convertGenderToDatabase(req.Cliente.Gender),
		Email:            req.Cliente.Email,
		Phone1:           req.Cliente.Phone1,
		Phone2:           req.Cliente.Phone2,
		ReferenceCity:    req.Cliente.ReferenceCity,
		ReferenceCountry: req.Cliente.ReferenceCountry,
		Active:           true,
		CreationDate:     time.Now(),
		BirthDate:        birthDate,
	}

	// Calcular subtotal desde el amount del pago o desde habitaciones
	subtotal := 0.0
	if req.Pago != nil && req.Pago.Amount > 0 {
		// Si hay pago, usar el amount como subtotal (ya incluye servicios)
		subtotal = req.Pago.Amount
	} else {
		// Si no hay pago, calcular desde habitaciones (fallback)
		for _, hab := range habitaciones {
			dias := hab.FechaSalida.Sub(hab.FechaEntrada).Hours() / 24
			if dias < 1 {
				dias = 1
			}
			subtotal += hab.Precio * dias
		}
	}

	// Crear la reserva con los datos del cliente
	reserva := &domain.Reserva{
		CantidadAdultos:   req.CantidadAdultos,
		CantidadNinhos:    req.CantidadNinhos,
		Descuento:         req.Descuento,
		Subtotal:          subtotal,
		Estado:            domain.ReservaPendiente,
		FechaConfirmacion: time.Now(),
		Habitaciones:      habitaciones,
		Servicios:         servicios,
	}

	// Crear el pago si se proporcionó
	var payment *domain.Payment
	if req.Pago != nil {
		payment = &domain.Payment{
			Amount:        req.Pago.Amount,
			Date:          time.Now(),
			PaymentMethod: domain.PaymentMethod(req.Pago.PaymentMethod),
			Status:        domain.PaymentStatus(req.Pago.Status),
		}
	}

	// Convertir huéspedes adicionales (si existen)
	var huespedes []domain.Person
	if len(req.Huespedes) > 0 {
		huespedes = make([]domain.Person, len(req.Huespedes))
		for i, h := range req.Huespedes {
			hBirthDate, err := time.Parse("2006-01-02", h.BirthDate)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("Formato de birthDate inválido para huésped %d. Use YYYY-MM-DD", i+1),
				})
			}

			huespedes[i] = domain.Person{
				Name:             h.Name,
				FirstSurname:     h.FirstSurname,
				SecondSurname:    h.SecondSurname,
				DocumentNumber:   h.DocumentNumber,
				Gender:           h.Gender, // Ya viene en formato correcto: M, F, O
				Email:            h.Email,  // Email del huésped
				Phone1:           h.Phone1, // Teléfono del huésped
				Phone2:           nil,
				ReferenceCity:    "",
				ReferenceCountry: "",
				Active:           true,
				CreationDate:     time.Now(),
				BirthDate:        hBirthDate,
			}
		}
	}

	// Llamar al servicio para crear la reserva con el cliente, huéspedes y el pago
	if err := h.service.CreateReservaWithClientAndPayment(person, reserva, huespedes, payment); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Reserva creada exitosamente",
		"data":    reserva,
	})
}

// GetReservaByID obtiene una reserva por su ID
func (h *ReservaHandler) GetReservaByID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de reserva inválido",
		})
	}

	reserva, err := h.service.GetReservaByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": reserva,
	})
}

// GetReservasCliente obtiene todas las reservas de un cliente
func (h *ReservaHandler) GetReservasCliente(c *fiber.Ctx) error {
	clienteIDStr := c.Params("clienteId")
	if clienteIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "clienteId es requerido",
		})
	}

	clienteID, err := strconv.Atoi(clienteIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "clienteId debe ser un número",
		})
	}

	reservas, err := h.service.GetReservasCliente(clienteID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": reservas,
	})
}

// UpdateReservaEstado actualiza el estado de una reserva
func (h *ReservaHandler) UpdateReservaEstado(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de reserva inválido",
		})
	}

	var req UpdateEstadoRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de solicitud inválido",
		})
	}

	// Convertir el estado a EstadoReserva
	estado := domain.EstadoReserva(req.Estado)

	if err := h.service.UpdateReservaEstado(id, estado); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Estado de reserva actualizado exitosamente",
	})
}

// CancelarReserva cancela una reserva
func (h *ReservaHandler) CancelarReserva(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de reserva inválido",
		})
	}

	if err := h.service.CancelarReserva(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Reserva cancelada exitosamente",
	})
}

// ConfirmarReserva confirma una reserva pendiente (sin enviar email)
func (h *ReservaHandler) ConfirmarReserva(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de reserva inválido",
		})
	}

	if err := h.service.ConfirmarReserva(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Reserva confirmada exitosamente",
	})
}

// ConfirmarPago confirma el pago de una reserva y envía email automáticamente
func (h *ReservaHandler) ConfirmarPago(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de reserva inválido",
		})
	}

	if err := h.service.ConfirmarReserva(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Pago confirmado y email enviado exitosamente",
	})
}

// VerificarDisponibilidad verifica si una habitación está disponible
func (h *ReservaHandler) VerificarDisponibilidad(c *fiber.Ctx) error {
	var req VerificarDisponibilidadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de solicitud inválido",
		})
	}

	fechaEntrada, err := time.Parse("2006-01-02", req.FechaEntrada)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fechaEntrada inválido. Use YYYY-MM-DD",
		})
	}

	fechaSalida, err := time.Parse("2006-01-02", req.FechaSalida)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fechaSalida inválido. Use YYYY-MM-DD",
		})
	}

	disponible, err := h.service.VerificarDisponibilidad(req.HabitacionID, fechaEntrada, fechaSalida)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"disponible": disponible,
	})
}

// GetReservasEnRango obtiene todas las reservas en un rango de fechas
func (h *ReservaHandler) GetReservasEnRango(c *fiber.Ctx) error {
	fechaInicioStr := c.Query("fechaInicio")
	fechaFinStr := c.Query("fechaFin")

	if fechaInicioStr == "" || fechaFinStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "fechaInicio y fechaFin son requeridos",
		})
	}

	fechaInicio, err := time.Parse("2006-01-02", fechaInicioStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fechaInicio inválido. Use YYYY-MM-DD",
		})
	}

	fechaFin, err := time.Parse("2006-01-02", fechaFinStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fechaFin inválido. Use YYYY-MM-DD",
		})
	}

	reservas, err := h.service.GetReservasEnRango(fechaInicio, fechaFin)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": reservas,
	})
}
