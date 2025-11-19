package application

import (
	"fmt"
	"time"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

type SatisfactionSurveyService struct {
	surveyRepo  domain.SatisfactionSurveyRepository
	reservaRepo domain.ReservaRepository
	tokenRepo   domain.SurveyTokenRepository
}

// NewSatisfactionSurveyService crea una nueva instancia del servicio
func NewSatisfactionSurveyService(
	surveyRepo domain.SatisfactionSurveyRepository,
	reservaRepo domain.ReservaRepository,
	tokenRepo domain.SurveyTokenRepository,
) *SatisfactionSurveyService {
	return &SatisfactionSurveyService{
		surveyRepo:  surveyRepo,
		reservaRepo: reservaRepo,
		tokenRepo:   tokenRepo,
	}
}

// CreateSurvey crea una nueva encuesta de satisfacción
func (s *SatisfactionSurveyService) CreateSurvey(survey *domain.SatisfactionSurvey) error {
	// Validar que la reserva exista
	reserva, err := s.reservaRepo.GetReservaByID(survey.ReservationID)
	if err != nil {
		return fmt.Errorf("reserva no encontrada: %w", err)
	}

	// Validar que el cliente coincida con la reserva
	if reserva.ClienteID != survey.ClientID {
		return fmt.Errorf("el cliente no coincide con la reserva")
	}

	// Validar las puntuaciones (1-5)
	if err := s.validateScores(survey); err != nil {
		return err
	}

	// Establecer fecha de respuesta
	if survey.ResponseDate.IsZero() {
		survey.ResponseDate = time.Now()
	}

	// Crear la encuesta
	if err := s.surveyRepo.Create(survey); err != nil {
		return fmt.Errorf("error al crear encuesta: %w", err)
	}

	return nil
}

// validateScores valida que todas las puntuaciones estén entre 1 y 5
func (s *SatisfactionSurveyService) validateScores(survey *domain.SatisfactionSurvey) error {
	scores := map[string]int{
		"generalExperience":  survey.GeneralExperience,
		"cleanliness":        survey.Cleanliness,
		"staffAttention":     survey.StaffAttention,
		"comfort":            survey.Comfort,
		"recommendation":     survey.Recommendation,
		"additionalServices": survey.AdditionalServices,
	}

	for field, score := range scores {
		if score < 1 || score > 5 {
			return fmt.Errorf("la puntuación de '%s' debe estar entre 1 y 5, recibido: %d", field, score)
		}
	}

	return nil
}

// GetSurveyByReservation obtiene la encuesta de una reserva
func (s *SatisfactionSurveyService) GetSurveyByReservation(reservationID int) (*domain.SatisfactionSurvey, error) {
	return s.surveyRepo.GetByReservationID(reservationID)
}

// GetSurveysByClient obtiene todas las encuestas de un cliente
func (s *SatisfactionSurveyService) GetSurveysByClient(clientID int) ([]domain.SatisfactionSurvey, error) {
	return s.surveyRepo.GetByClientID(clientID)
}

// GetAllSurveys obtiene todas las encuestas con paginación
func (s *SatisfactionSurveyService) GetAllSurveys(limit, offset int) ([]domain.SatisfactionSurvey, error) {
	if limit <= 0 {
		limit = 50 // Default
	}
	if offset < 0 {
		offset = 0
	}
	return s.surveyRepo.GetAll(limit, offset)
}

// GetAverageScores obtiene los promedios de todas las respuestas
func (s *SatisfactionSurveyService) GetAverageScores() (map[string]float64, error) {
	return s.surveyRepo.GetAverageScores()
}

// GetTopRatedSurveys obtiene las encuestas con mejor puntuación para el landing page
func (s *SatisfactionSurveyService) GetTopRatedSurveys(limit int) ([]domain.SurveyResponse, error) {
	if limit <= 0 {
		limit = 10 // Default: 10 mejores encuestas
	}
	return s.surveyRepo.GetTopRatedSurveys(limit)
}

// ValidateToken valida un token y retorna los IDs de reserva y cliente
func (s *SatisfactionSurveyService) ValidateToken(tokenValue string) (reservationID, clientID int, valid bool, err error) {
	token, err := s.tokenRepo.GetByToken(tokenValue)
	if err != nil {
		return 0, 0, false, err
	}

	// Verificar si el token ya fue usado
	if token.Used {
		return 0, 0, false, fmt.Errorf("el token ya fue utilizado")
	}

	// Verificar si el token expiró
	if time.Now().After(token.ExpiresAt) {
		return 0, 0, false, fmt.Errorf("el token ha expirado")
	}

	return token.ReservationID, token.ClientID, true, nil
}

// CreateSurveyWithToken crea una encuesta usando un token
func (s *SatisfactionSurveyService) CreateSurveyWithToken(tokenValue string, survey *domain.SatisfactionSurvey) error {
	// Validar el token
	reservationID, clientID, valid, err := s.ValidateToken(tokenValue)
	if err != nil {
		return err
	}

	if !valid {
		return fmt.Errorf("token inválido")
	}

	// Asignar los IDs de la token a la encuesta
	survey.ReservationID = reservationID
	survey.ClientID = clientID

	// Crear la encuesta
	if err := s.CreateSurvey(survey); err != nil {
		return err
	}

	// Marcar el token como usado
	if err := s.tokenRepo.MarkAsUsed(tokenValue); err != nil {
		return fmt.Errorf("encuesta creada pero error al marcar token: %w", err)
	}

	return nil
}

// CreateTokenForReservation crea un token para una reserva
func (s *SatisfactionSurveyService) CreateTokenForReservation(reservationID, clientID int) (*domain.SurveyToken, error) {
	token := &domain.SurveyToken{
		ReservationID: reservationID,
		ClientID:      clientID,
	}

	if err := s.tokenRepo.Create(token); err != nil {
		return nil, fmt.Errorf("error al crear token: %w", err)
	}

	return token, nil
}
