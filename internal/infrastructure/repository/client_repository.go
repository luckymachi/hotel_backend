package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

type clientRepository struct {
	db *sql.DB
}

// NewClientRepository crea una nueva instancia del repositorio de clientes
func NewClientRepository(db *sql.DB) domain.ClientRepository {
	return &clientRepository{db: db}
}

// GetClientIDByPersonID obtiene el client_id dado un person_id
func (r *clientRepository) GetClientIDByPersonID(personID int) (int, error) {
	query := `
		SELECT client_id
		FROM client
		WHERE person_id = $1
	`

	var clientID int
	err := r.db.QueryRow(query, personID).Scan(&clientID)

	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("no existe cliente para person_id %d", personID)
	}

	if err != nil {
		return 0, fmt.Errorf("error al buscar cliente: %w", err)
	}

	return clientID, nil
}

// Create crea un nuevo cliente
func (r *clientRepository) Create(personID int, captureChannel string, captureStatus string, travelsWithChildren int) (int, error) {
	// Limpiar inputs
	captureChannel = strings.TrimSpace(captureChannel)
	captureStatus = strings.TrimSpace(captureStatus)

	query := `
		INSERT INTO client (person_id, capture_channel, capture_status, travels_with_children)
		VALUES ($1, $2, $3, $4)
		RETURNING client_id
	`

	var clientID int
	err := r.db.QueryRow(query, personID, captureChannel, captureStatus, travelsWithChildren).Scan(&clientID)

	if err != nil {
		return 0, fmt.Errorf("error al crear cliente: %w", err)
	}

	return clientID, nil
}

// GetPersonEmailByClientID obtiene el email de la persona asociada a un cliente
func (r *clientRepository) GetPersonEmailByClientID(clientID int) (string, error) {
	query := `
		SELECT p.email
		FROM person p
		INNER JOIN client c ON p.person_id = c.person_id
		WHERE c.client_id = $1
	`

	var email string
	err := r.db.QueryRow(query, clientID).Scan(&email)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("no existe cliente con client_id %d", clientID)
	}

	if err != nil {
		return "", fmt.Errorf("error al obtener email del cliente: %w", err)
	}

	return email, nil
}
