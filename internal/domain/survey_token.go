package domain

import "time"

// SurveyToken representa un token para acceder a la encuesta
type SurveyToken struct {
	TokenID       int       `json:"tokenId"`
	Token         string    `json:"token"`
	ReservationID int       `json:"reservationId"`
	ClientID      int       `json:"clientId"`
	ExpiresAt     time.Time `json:"expiresAt"`
	Used          bool      `json:"used"`
	CreatedAt     time.Time `json:"createdAt"`
}

// SurveyTokenRepository define las operaciones con tokens
type SurveyTokenRepository interface {
	// Create crea un nuevo token
	Create(token *SurveyToken) error
	// GetByToken obtiene un token por su valor
	GetByToken(token string) (*SurveyToken, error)
	// MarkAsUsed marca un token como usado
	MarkAsUsed(token string) error
	// DeleteExpired elimina tokens expirados
	DeleteExpired() error
}
