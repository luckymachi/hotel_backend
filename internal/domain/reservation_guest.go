package domain

// ReservationGuest representa la relación entre una reserva y un huésped (person)
type ReservationGuest struct {
	ReservationID int `json:"reservationId"`
	PersonID      int `json:"personId"`
}

// ReservationGuestRepository define las operaciones con huéspedes de reserva
type ReservationGuestRepository interface {
	// Create crea una relación entre reserva y huésped
	Create(reservationID int, personID int) error
	// CreateMultiple crea múltiples relaciones para una reserva
	CreateMultiple(reservationID int, personIDs []int) error
	// GetByReservationID obtiene todos los person_id de una reserva
	GetByReservationID(reservationID int) ([]int, error)
	// Delete elimina una relación
	Delete(reservationID int, personID int) error
	// DeleteByReservationID elimina todas las relaciones de una reserva
	DeleteByReservationID(reservationID int) error
}
