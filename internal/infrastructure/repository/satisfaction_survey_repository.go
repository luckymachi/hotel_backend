package repository

import (
	"database/sql"
	"fmt"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

type satisfactionSurveyRepository struct {
	db *sql.DB
}

// NewSatisfactionSurveyRepository crea una nueva instancia del repositorio
func NewSatisfactionSurveyRepository(db *sql.DB) domain.SatisfactionSurveyRepository {
	return &satisfactionSurveyRepository{db: db}
}

// Create crea una nueva encuesta de satisfacción
func (r *satisfactionSurveyRepository) Create(survey *domain.SatisfactionSurvey) error {
	query := `
		INSERT INTO satisfaction_survey (
			reservation_id,
			client_id,
			general_experience,
			cleanliness,
			staff_attention,
			comfort,
			recommendation,
			additional_services,
			comments,
			response_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING survey_id, created_at
	`

	// Convertir *string a sql.NullString para comments
	var comments sql.NullString
	if survey.Comments != nil {
		comments = sql.NullString{String: *survey.Comments, Valid: true}
	}

	err := r.db.QueryRow(
		query,
		survey.ReservationID,
		survey.ClientID,
		survey.GeneralExperience,
		survey.Cleanliness,
		survey.StaffAttention,
		survey.Comfort,
		survey.Recommendation,
		survey.AdditionalServices,
		comments,
		survey.ResponseDate,
	).Scan(&survey.SurveyID, &survey.CreatedAt)

	if err != nil {
		// Verificar si es un error de duplicado
		if err.Error() == "pq: duplicate key value violates unique constraint \"unique_survey_per_reservation\"" {
			return fmt.Errorf("ya existe una encuesta para esta reserva")
		}
		return fmt.Errorf("error al crear encuesta: %w", err)
	}

	return nil
}

// GetByReservationID obtiene la encuesta de una reserva
func (r *satisfactionSurveyRepository) GetByReservationID(reservationID int) (*domain.SatisfactionSurvey, error) {
	query := `
		SELECT 
			survey_id,
			reservation_id,
			client_id,
			general_experience,
			cleanliness,
			staff_attention,
			comfort,
			recommendation,
			additional_services,
			comments,
			response_date,
			created_at
		FROM satisfaction_survey
		WHERE reservation_id = $1
	`

	survey := &domain.SatisfactionSurvey{}
	var comments sql.NullString

	err := r.db.QueryRow(query, reservationID).Scan(
		&survey.SurveyID,
		&survey.ReservationID,
		&survey.ClientID,
		&survey.GeneralExperience,
		&survey.Cleanliness,
		&survey.StaffAttention,
		&survey.Comfort,
		&survey.Recommendation,
		&survey.AdditionalServices,
		&comments,
		&survey.ResponseDate,
		&survey.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no existe encuesta para la reserva %d", reservationID)
	}

	if err != nil {
		return nil, fmt.Errorf("error al obtener encuesta: %w", err)
	}

	// Convertir sql.NullString a *string
	if comments.Valid {
		survey.Comments = &comments.String
	}

	return survey, nil
}

// GetByClientID obtiene todas las encuestas de un cliente
func (r *satisfactionSurveyRepository) GetByClientID(clientID int) ([]domain.SatisfactionSurvey, error) {
	query := `
		SELECT 
			survey_id,
			reservation_id,
			client_id,
			general_experience,
			cleanliness,
			staff_attention,
			comfort,
			recommendation,
			additional_services,
			comments,
			response_date,
			created_at
		FROM satisfaction_survey
		WHERE client_id = $1
		ORDER BY response_date DESC
	`

	rows, err := r.db.Query(query, clientID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener encuestas del cliente: %w", err)
	}
	defer rows.Close()

	var surveys []domain.SatisfactionSurvey
	for rows.Next() {
		var survey domain.SatisfactionSurvey
		var comments sql.NullString

		err := rows.Scan(
			&survey.SurveyID,
			&survey.ReservationID,
			&survey.ClientID,
			&survey.GeneralExperience,
			&survey.Cleanliness,
			&survey.StaffAttention,
			&survey.Comfort,
			&survey.Recommendation,
			&survey.AdditionalServices,
			&comments,
			&survey.ResponseDate,
			&survey.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error al escanear encuesta: %w", err)
		}

		if comments.Valid {
			survey.Comments = &comments.String
		}

		surveys = append(surveys, survey)
	}

	return surveys, nil
}

// GetAll obtiene todas las encuestas con paginación
func (r *satisfactionSurveyRepository) GetAll(limit, offset int) ([]domain.SatisfactionSurvey, error) {
	query := `
		SELECT 
			survey_id,
			reservation_id,
			client_id,
			general_experience,
			cleanliness,
			staff_attention,
			comfort,
			recommendation,
			additional_services,
			comments,
			response_date,
			created_at
		FROM satisfaction_survey
		ORDER BY response_date DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error al obtener encuestas: %w", err)
	}
	defer rows.Close()

	var surveys []domain.SatisfactionSurvey
	for rows.Next() {
		var survey domain.SatisfactionSurvey
		var comments sql.NullString

		err := rows.Scan(
			&survey.SurveyID,
			&survey.ReservationID,
			&survey.ClientID,
			&survey.GeneralExperience,
			&survey.Cleanliness,
			&survey.StaffAttention,
			&survey.Comfort,
			&survey.Recommendation,
			&survey.AdditionalServices,
			&comments,
			&survey.ResponseDate,
			&survey.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error al escanear encuesta: %w", err)
		}

		if comments.Valid {
			survey.Comments = &comments.String
		}

		surveys = append(surveys, survey)
	}

	return surveys, nil
}

// GetAverageScores obtiene los promedios de todas las respuestas
func (r *satisfactionSurveyRepository) GetAverageScores() (map[string]float64, error) {
	query := `
		SELECT 
			AVG(general_experience) as avg_general_experience,
			AVG(cleanliness) as avg_cleanliness,
			AVG(staff_attention) as avg_staff_attention,
			AVG(comfort) as avg_comfort,
			AVG(recommendation) as avg_recommendation,
			AVG(additional_services) as avg_additional_services,
			COUNT(*) as total_surveys
		FROM satisfaction_survey
	`

	var avgGeneralExperience, avgCleanliness, avgStaffAttention, avgComfort, avgRecommendation, avgAdditionalServices float64
	var totalSurveys int

	err := r.db.QueryRow(query).Scan(
		&avgGeneralExperience,
		&avgCleanliness,
		&avgStaffAttention,
		&avgComfort,
		&avgRecommendation,
		&avgAdditionalServices,
		&totalSurveys,
	)

	if err != nil {
		return nil, fmt.Errorf("error al calcular promedios: %w", err)
	}

	averages := map[string]float64{
		"generalExperience":  avgGeneralExperience,
		"cleanliness":        avgCleanliness,
		"staffAttention":     avgStaffAttention,
		"comfort":            avgComfort,
		"recommendation":     avgRecommendation,
		"additionalServices": avgAdditionalServices,
		"totalSurveys":       float64(totalSurveys),
		"overallAverage":     (avgGeneralExperience + avgCleanliness + avgStaffAttention + avgComfort + avgRecommendation + avgAdditionalServices) / 6.0,
	}

	return averages, nil
}

// GetTopRatedSurveys obtiene las encuestas con mejor puntuación para el landing page
func (r *satisfactionSurveyRepository) GetTopRatedSurveys(limit int) ([]domain.SurveyResponse, error) {
	query := `
		SELECT 
			survey_id,
			general_experience,
			cleanliness,
			staff_attention,
			comfort,
			recommendation,
			additional_services,
			comments,
			response_date,
			(general_experience + cleanliness + staff_attention + comfort + recommendation + additional_services) / 6.0 as average_score
		FROM satisfaction_survey
		WHERE comments IS NOT NULL AND comments != ''
		ORDER BY average_score DESC, response_date DESC
		LIMIT $1
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("error al obtener encuestas mejor puntuadas: %w", err)
	}
	defer rows.Close()

	var surveys []domain.SurveyResponse
	for rows.Next() {
		var survey domain.SurveyResponse
		var comments sql.NullString

		err := rows.Scan(
			&survey.SurveyID,
			&survey.GeneralExperience,
			&survey.Cleanliness,
			&survey.StaffAttention,
			&survey.Comfort,
			&survey.Recommendation,
			&survey.AdditionalServices,
			&comments,
			&survey.ResponseDate,
			&survey.AverageScore,
		)
		if err != nil {
			return nil, fmt.Errorf("error al escanear encuesta: %w", err)
		}

		if comments.Valid {
			survey.Comments = &comments.String
		}

		surveys = append(surveys, survey)
	}

	return surveys, nil
}
