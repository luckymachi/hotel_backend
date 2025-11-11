package repository

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

type surveyTokenRepository struct {
	db *sql.DB
}

// NewSurveyTokenRepository crea una nueva instancia del repositorio
func NewSurveyTokenRepository(db *sql.DB) domain.SurveyTokenRepository {
	return &surveyTokenRepository{db: db}
}

// generateToken genera un token aleatorio seguro
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Create crea un nuevo token
func (r *surveyTokenRepository) Create(token *domain.SurveyToken) error {
	// Generar token si no está presente
	if token.Token == "" {
		generatedToken, err := generateToken()
		if err != nil {
			return fmt.Errorf("error al generar token: %w", err)
		}
		token.Token = generatedToken
	}

	// Establecer expiración si no está presente (30 días por defecto)
	if token.ExpiresAt.IsZero() {
		token.ExpiresAt = time.Now().AddDate(0, 0, 30) // 30 días
	}

	query := `
		INSERT INTO survey_token (
			token,
			reservation_id,
			client_id,
			expires_at
		) VALUES ($1, $2, $3, $4)
		RETURNING token_id, created_at
	`

	err := r.db.QueryRow(
		query,
		token.Token,
		token.ReservationID,
		token.ClientID,
		token.ExpiresAt,
	).Scan(&token.TokenID, &token.CreatedAt)

	if err != nil {
		return fmt.Errorf("error al crear token: %w", err)
	}

	return nil
}

// GetByToken obtiene un token por su valor
func (r *surveyTokenRepository) GetByToken(tokenValue string) (*domain.SurveyToken, error) {
	query := `
		SELECT 
			token_id,
			token,
			reservation_id,
			client_id,
			expires_at,
			used,
			created_at
		FROM survey_token
		WHERE token = $1
	`

	token := &domain.SurveyToken{}
	err := r.db.QueryRow(query, tokenValue).Scan(
		&token.TokenID,
		&token.Token,
		&token.ReservationID,
		&token.ClientID,
		&token.ExpiresAt,
		&token.Used,
		&token.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token no encontrado")
	}

	if err != nil {
		return nil, fmt.Errorf("error al obtener token: %w", err)
	}

	return token, nil
}

// MarkAsUsed marca un token como usado
func (r *surveyTokenRepository) MarkAsUsed(tokenValue string) error {
	query := `
		UPDATE survey_token
		SET used = TRUE
		WHERE token = $1
	`

	result, err := r.db.Exec(query, tokenValue)
	if err != nil {
		return fmt.Errorf("error al marcar token como usado: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error al verificar actualización: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("token no encontrado")
	}

	return nil
}

// DeleteExpired elimina tokens expirados
func (r *surveyTokenRepository) DeleteExpired() error {
	query := `
		DELETE FROM survey_token
		WHERE expires_at < NOW()
	`

	_, err := r.db.Exec(query)
	if err != nil {
		return fmt.Errorf("error al eliminar tokens expirados: %w", err)
	}

	return nil
}
