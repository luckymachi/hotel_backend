package domain

import "time"

type PaymentMethod string
type PaymentStatus string

const (
	PaymentMethodTarjeta        PaymentMethod = "Tarjeta"
	PaymentMethodDeposito       PaymentMethod = "Deposito"
	PaymentMethodBilleteraMovil PaymentMethod = "Billetera movil"
)

const (
	PaymentStatusPendiente PaymentStatus = "Pendiente"
	PaymentStatusAprobado  PaymentStatus = "Aprobado"
	PaymentStatusRechazado PaymentStatus = "Rechazado"
	PaymentStatusReembolso PaymentStatus = "Reembolso"
)

// Payment representa un pago de una reserva
type Payment struct {
	PaymentID     int           `json:"paymentId"`
	Amount        float64       `json:"amount"`
	Date          time.Time     `json:"date"`
	PaymentMethod PaymentMethod `json:"paymentMethod"`
	Status        PaymentStatus `json:"status"`
	ReservationID int           `json:"reservationId"`
}

// PaymentRepository define las operaciones con pagos
type PaymentRepository interface {
	// Create crea un nuevo pago
	Create(payment *Payment) error
	// GetByReservationID obtiene el pago de una reserva
	GetByReservationID(reservationID int) (*Payment, error)
	// UpdateStatus actualiza el estado de un pago
	UpdateStatus(paymentID int, status PaymentStatus) error
}
