package repository

import (
	"database/sql"
	"fmt"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

type paymentRepository struct {
	db *sql.DB
}

// NewPaymentRepository crea una nueva instancia del repositorio de pagos
func NewPaymentRepository(db *sql.DB) domain.PaymentRepository {
	return &paymentRepository{db: db}
}

// Create crea un nuevo pago
func (r *paymentRepository) Create(payment *domain.Payment) error {
	query := `
		INSERT INTO payment (
			amount,
			date,
			payment_method,
			status,
			reservation_id
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING payment_id
	`

	err := r.db.QueryRow(
		query,
		payment.Amount,
		payment.Date,
		payment.PaymentMethod,
		payment.Status,
		payment.ReservationID,
	).Scan(&payment.PaymentID)

	if err != nil {
		return fmt.Errorf("error al crear pago: %w", err)
	}

	return nil
}

// GetByReservationID obtiene el pago de una reserva
func (r *paymentRepository) GetByReservationID(reservationID int) (*domain.Payment, error) {
	query := `
		SELECT 
			payment_id,
			amount,
			date,
			payment_method,
			status,
			reservation_id
		FROM payment
		WHERE reservation_id = $1
	`

	payment := &domain.Payment{}
	err := r.db.QueryRow(query, reservationID).Scan(
		&payment.PaymentID,
		&payment.Amount,
		&payment.Date,
		&payment.PaymentMethod,
		&payment.Status,
		&payment.ReservationID,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no se encontró pago para la reserva %d", reservationID)
	}

	if err != nil {
		return nil, fmt.Errorf("error al obtener pago: %w", err)
	}

	return payment, nil
}

// UpdateStatus actualiza el estado de un pago
func (r *paymentRepository) UpdateStatus(paymentID int, status domain.PaymentStatus) error {
	query := `
		UPDATE payment 
		SET status = $1 
		WHERE payment_id = $2
	`

	result, err := r.db.Exec(query, status, paymentID)
	if err != nil {
		return fmt.Errorf("error al actualizar estado del pago: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error al verificar filas afectadas: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pago con ID %d no encontrado", paymentID)
	}

	return nil
}
