package domain

import (
	"time"
)

type EstadoReserva string

const (
	ReservaPendiente  EstadoReserva = "Pendiente"
	ReservaConfirmada EstadoReserva = "Confirmada"
	ReservaCancelada  EstadoReserva = "Cancelada"
	ReservaCompletada EstadoReserva = "Completada"
)

// Reserva representa una reserva principal
type Reserva struct {
	ID                int                 `json:"id"`
	CantidadAdultos   int                 `json:"cantidadAdultos"`
	CantidadNinhos    int                 `json:"cantidadNinhos"`
	Estado            EstadoReserva       `json:"estado"`
	ClienteID         int                 `json:"clienteId"`
	Subtotal          float64             `json:"subtotal"`
	Descuento         float64             `json:"descuento"`
	FechaConfirmacion time.Time           `json:"fechaConfirmacion"`
	Habitaciones      []ReservaHabitacion `json:"habitaciones"`
}

// ReservaRepository define las operaciones disponibles con las reservas
type ReservaRepository interface {
	// GetReservaByID obtiene una reserva por su ID
	GetReservaByID(id int) (*Reserva, error)
	// CreateReserva crea una nueva reserva
	CreateReserva(reserva *Reserva) error
	// UpdateReservaEstado actualiza el estado de una reserva
	UpdateReservaEstado(id int, estado EstadoReserva) error
	// GetReservasCliente obtiene todas las reservas de un cliente
	GetReservasCliente(clienteID int) ([]Reserva, error)
}
