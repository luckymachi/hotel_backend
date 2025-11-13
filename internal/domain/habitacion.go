package domain

import "time"

// TipoHabitacion represents the room type
type TipoHabitacion struct {
	ID               int     `json:"id"`
	Titulo           string  `json:"titulo"`
	Descripcion      string  `json:"descripcion"`
	CapacidadAdultos int     `json:"capacidadAdultos"`
	CapacidadNinhos  int     `json:"capacidadNinhos"`
	CantidadCamas    int     `json:"cantidadCamas"`
	Precio           float64 `json:"precio"`
	Area             float64 `json:"area"`
	// Amenities related to this room type
	Amenities []Amenity `json:"amenities,omitempty"`
}

// Amenity represents an amenity that can be assigned to a room type
type Amenity struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Habitacion represents a room in the hotel with its type information
type Habitacion struct {
	ID                 int            `json:"id"`
	Nombre             string         `json:"nombre"`
	Numero             string         `json:"numero"`
	Capacidad          int            `json:"capacidad"`
	Estado             string         `json:"estado"`
	DescripcionGeneral string         `json:"descripcionGeneral"`
	TipoHabitacion     TipoHabitacion `json:"tipoHabitacion"`
	MediaID            int            `json:"-"` // El tag "-" hace que este campo se omita en la serialización JSON
}

// FechasBloqueadas representa las fechas donde no hay disponibilidad
type FechasBloqueadas struct {
	FechasNoDisponibles []time.Time `json:"fechasNoDisponibles"`
}

// DisponibilidadFecha representa la disponibilidad de habitaciones para una fecha específica
type DisponibilidadFecha struct {
	Fecha        time.Time `json:"fecha"`
	Disponible   bool      `json:"disponible"`
	Habitaciones int       `json:"habitaciones"`
}

// HabitacionRepository defines the interface for room data operations
type HabitacionRepository interface {
	// GetAllRooms returns all rooms in the system
	GetAllRooms() ([]Habitacion, error)
	// GetAvailableRooms returns rooms that are available for the given date range
	GetAvailableRooms(fechaEntrada, fechaSalida time.Time) ([]Habitacion, error)
	// GetFechasBloqueadas returns dates where there are no rooms available
	GetFechasBloqueadas(desde time.Time, hasta time.Time) (*FechasBloqueadas, error)
	// GetDisponibilidadFechas returns the availability status for each date in the given range
	GetDisponibilidadFechas(desde time.Time, hasta time.Time) ([]DisponibilidadFecha, error)
	// GetRoomTypes returns all room types in the system
	GetRoomTypes() ([]TipoHabitacion, error)
}
