package domain

import "time"

// SatisfactionSurvey representa una encuesta de satisfacción
type SatisfactionSurvey struct {
	SurveyID           int       `json:"surveyId"`
	ReservationID      int       `json:"reservationId"`
	ClientID           int       `json:"clientId"`
	GeneralExperience  int       `json:"generalExperience"`  // 1-5
	Cleanliness        int       `json:"cleanliness"`        // 1-5
	StaffAttention     int       `json:"staffAttention"`     // 1-5
	Comfort            int       `json:"comfort"`            // 1-5
	Recommendation     int       `json:"recommendation"`     // 1-5
	AdditionalServices int       `json:"additionalServices"` // 1-5
	Comments           *string   `json:"comments,omitempty"` // Optional
	ResponseDate       time.Time `json:"responseDate"`
	CreatedAt          time.Time `json:"createdAt"`
}

// SatisfactionSurveyRepository define las operaciones con encuestas
type SatisfactionSurveyRepository interface {
	// Create crea una nueva encuesta de satisfacción
	Create(survey *SatisfactionSurvey) error
	// GetByReservationID obtiene la encuesta de una reserva
	GetByReservationID(reservationID int) (*SatisfactionSurvey, error)
	// GetByClientID obtiene todas las encuestas de un cliente
	GetByClientID(clientID int) ([]SatisfactionSurvey, error)
	// GetAll obtiene todas las encuestas con paginación
	GetAll(limit, offset int) ([]SatisfactionSurvey, error)
	// GetAverageScores obtiene los promedios de todas las respuestas
	GetAverageScores() (map[string]float64, error)
}
