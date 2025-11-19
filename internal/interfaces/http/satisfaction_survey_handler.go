package http

import (
	"strconv"

	"github.com/Maxito7/hotel_backend/internal/application"
	"github.com/Maxito7/hotel_backend/internal/domain"
	"github.com/gofiber/fiber/v2"
)

type SatisfactionSurveyHandler struct {
	service *application.SatisfactionSurveyService
}

// NewSatisfactionSurveyHandler crea una nueva instancia del handler
func NewSatisfactionSurveyHandler(service *application.SatisfactionSurveyService) *SatisfactionSurveyHandler {
	return &SatisfactionSurveyHandler{
		service: service,
	}
}

// CreateSurveyRequest representa la petición para crear una encuesta
type CreateSurveyRequest struct {
	Token              string  `json:"token"`              // Token de acceso a la encuesta
	GeneralExperience  int     `json:"generalExperience"`  // 1-5
	Cleanliness        int     `json:"cleanliness"`        // 1-5
	StaffAttention     int     `json:"staffAttention"`     // 1-5
	Comfort            int     `json:"comfort"`            // 1-5
	Recommendation     int     `json:"recommendation"`     // 1-5
	AdditionalServices int     `json:"additionalServices"` // 1-5
	Comments           *string `json:"comments,omitempty"`
}

// ValidateTokenResponse representa la respuesta de validación de token
type ValidateTokenResponse struct {
	Valid         bool `json:"valid"`
	ReservationID int  `json:"reservationId,omitempty"`
	ClientID      int  `json:"clientId,omitempty"`
}

// CreateSurvey crea una nueva encuesta de satisfacción usando un token
func (h *SatisfactionSurveyHandler) CreateSurvey(c *fiber.Ctx) error {
	var req CreateSurveyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de solicitud inválido",
		})
	}

	// Validar que el token esté presente
	if req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El token es requerido",
		})
	}

	// Crear el objeto de encuesta
	survey := &domain.SatisfactionSurvey{
		GeneralExperience:  req.GeneralExperience,
		Cleanliness:        req.Cleanliness,
		StaffAttention:     req.StaffAttention,
		Comfort:            req.Comfort,
		Recommendation:     req.Recommendation,
		AdditionalServices: req.AdditionalServices,
		Comments:           req.Comments,
	}

	// Crear la encuesta con el token (el servicio validará el token y asignará los IDs)
	if err := h.service.CreateSurveyWithToken(req.Token, survey); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Encuesta creada exitosamente",
		"data":    survey,
	})
}

// GetSurveyByReservation obtiene la encuesta de una reserva
func (h *SatisfactionSurveyHandler) GetSurveyByReservation(c *fiber.Ctx) error {
	reservationIDStr := c.Params("reservationId")
	reservationID, err := strconv.Atoi(reservationIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de reserva inválido",
		})
	}

	survey, err := h.service.GetSurveyByReservation(reservationID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": survey,
	})
}

// GetSurveysByClient obtiene todas las encuestas de un cliente
func (h *SatisfactionSurveyHandler) GetSurveysByClient(c *fiber.Ctx) error {
	clientIDStr := c.Params("clientId")
	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de cliente inválido",
		})
	}

	surveys, err := h.service.GetSurveysByClient(clientID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": surveys,
	})
}

// GetAllSurveys obtiene todas las encuestas con paginación
func (h *SatisfactionSurveyHandler) GetAllSurveys(c *fiber.Ctx) error {
	// Parámetros de paginación
	limitStr := c.Query("limit", "50")
	offsetStr := c.Query("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Parámetro 'limit' inválido",
		})
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Parámetro 'offset' inválido",
		})
	}

	surveys, err := h.service.GetAllSurveys(limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": surveys,
	})
}

// GetAverageScores obtiene los promedios de todas las respuestas
func (h *SatisfactionSurveyHandler) GetAverageScores(c *fiber.Ctx) error {
	averages, err := h.service.GetAverageScores()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": averages,
	})
}

// ValidateToken valida un token y retorna los IDs de reserva y cliente
func (h *SatisfactionSurveyHandler) ValidateToken(c *fiber.Ctx) error {
	token := c.Params("token")

	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token es requerido",
		})
	}

	reservationID, clientID, valid, err := h.service.ValidateToken(token)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ValidateTokenResponse{
			Valid: false,
		})
	}

	if !valid {
		return c.Status(fiber.StatusBadRequest).JSON(ValidateTokenResponse{
			Valid: false,
		})
	}

	return c.JSON(ValidateTokenResponse{
		Valid:         true,
		ReservationID: reservationID,
		ClientID:      clientID,
	})
}

// GetTopRatedSurveys obtiene las mejores encuestas para mostrar en el landing page
func (h *SatisfactionSurveyHandler) GetTopRatedSurveys(c *fiber.Ctx) error {
	// Parámetro de límite (cuántas encuestas mostrar)
	limitStr := c.Query("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Parámetro 'limit' inválido",
		})
	}

	surveys, err := h.service.GetTopRatedSurveys(limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": surveys,
	})
}
