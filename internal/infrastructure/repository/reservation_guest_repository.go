package repository

import (
	"database/sql"
	"fmt"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

type reservationGuestRepository struct {
	db *sql.DB
}

// NewReservationGuestRepository crea una nueva instancia del repositorio de huéspedes
func NewReservationGuestRepository(db *sql.DB) domain.ReservationGuestRepository {
	return &reservationGuestRepository{db: db}
}

// Create crea una relación entre reserva y huésped
func (r *reservationGuestRepository) Create(reservationID int, personID int) error {
	query := `
		INSERT INTO reservation_guest (reservation_id, person_id)
		VALUES ($1, $2)
	`

	_, err := r.db.Exec(query, reservationID, personID)
	if err != nil {
		return fmt.Errorf("error al crear relación reserva-huésped: %w", err)
	}

	return nil
}

// CreateMultiple crea múltiples relaciones para una reserva
func (r *reservationGuestRepository) CreateMultiple(reservationID int, personIDs []int) error {
	if len(personIDs) == 0 {
		return nil
	}

	// Usar una transacción para insertar todas las relaciones
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error al iniciar transacción: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO reservation_guest (reservation_id, person_id)
		VALUES ($1, $2)
	`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("error al preparar statement: %w", err)
	}
	defer stmt.Close()

	for i, personID := range personIDs {
		_, err := stmt.Exec(reservationID, personID)
		if err != nil {
			return fmt.Errorf("error al crear relación %d: %w", i+1, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error al confirmar transacción: %w", err)
	}

	return nil
}

// GetByReservationID obtiene todos los person_id de una reserva
func (r *reservationGuestRepository) GetByReservationID(reservationID int) ([]int, error) {
	query := `
		SELECT person_id
		FROM reservation_guest
		WHERE reservation_id = $1
		ORDER BY person_id
	`

	rows, err := r.db.Query(query, reservationID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener huéspedes: %w", err)
	}
	defer rows.Close()

	var personIDs []int
	for rows.Next() {
		var personID int
		if err := rows.Scan(&personID); err != nil {
			return nil, fmt.Errorf("error al escanear person_id: %w", err)
		}
		personIDs = append(personIDs, personID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error al iterar resultados: %w", err)
	}

	return personIDs, nil
}

// Delete elimina una relación
func (r *reservationGuestRepository) Delete(reservationID int, personID int) error {
	query := `DELETE FROM reservation_guest WHERE reservation_id = $1 AND person_id = $2`

	result, err := r.db.Exec(query, reservationID, personID)
	if err != nil {
		return fmt.Errorf("error al eliminar relación: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error al verificar eliminación: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("relación no encontrada")
	}

	return nil
}

// DeleteByReservationID elimina todas las relaciones de una reserva
func (r *reservationGuestRepository) DeleteByReservationID(reservationID int) error {
	query := `DELETE FROM reservation_guest WHERE reservation_id = $1`

	_, err := r.db.Exec(query, reservationID)
	if err != nil {
		return fmt.Errorf("error al eliminar relaciones de reserva: %w", err)
	}

	return nil
}
