package repository

import (
	"database/sql"
	"fmt"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

type reservaRepository struct {
	db *sql.DB
}

// NewReservaRepository crea una nueva instancia del repositorio de reservas
func NewReservaRepository(db *sql.DB) domain.ReservaRepository {
	return &reservaRepository{db: db}
}

// GetReservaByID obtiene una reserva por su ID con sus habitaciones
func (r *reservaRepository) GetReservaByID(id int) (*domain.Reserva, error) {
	query := `
		SELECT 
			r.reservation_id,
			r.adults_count,
			r.children_count,
			r.status,
			r.client_id,
			r.subtotal,
			r.discount,
			r.confirmation_date
		FROM reservation r
		WHERE r.reservation_id = $1
	`

	reserva := &domain.Reserva{}
	err := r.db.QueryRow(query, id).Scan(
		&reserva.ID,
		&reserva.CantidadAdultos,
		&reserva.CantidadNinhos,
		&reserva.Estado,
		&reserva.ClienteID,
		&reserva.Subtotal,
		&reserva.Descuento,
		&reserva.FechaConfirmacion,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("reserva con ID %d no encontrada", id)
		}
		return nil, fmt.Errorf("error al obtener reserva: %w", err)
	}

	// Obtener las habitaciones de la reserva
	habitacionesQuery := `
		SELECT 
			rh.reservation_id,
			rh.room_id,
			rh.price,
			rh.check_in_date,
			rh.check_out_date,
			rh.status,
			h.name,
			h.capacity,
			h.number
		FROM reservation_room rh
		INNER JOIN room h ON h.room_id = rh.room_id
		WHERE rh.reservation_id = $1 AND rh.status = 1
	`

	rows, err := r.db.Query(habitacionesQuery, id)
	if err != nil {
		return nil, fmt.Errorf("error al obtener habitaciones de la reserva: %w", err)
	}
	defer rows.Close()

	var habitaciones []domain.ReservaHabitacion
	for rows.Next() {
		var rh domain.ReservaHabitacion
		var habitacion domain.Habitacion

		err := rows.Scan(
			&rh.ReservaID,
			&rh.HabitacionID,
			&rh.Precio,
			&rh.FechaEntrada,
			&rh.FechaSalida,
			&rh.Estado,
			&habitacion.Nombre,
			&habitacion.Capacidad,
			&habitacion.Numero,
		)
		if err != nil {
			return nil, fmt.Errorf("error al escanear habitación: %w", err)
		}

		habitacion.ID = rh.HabitacionID
		rh.Habitacion = &habitacion
		habitaciones = append(habitaciones, rh)
	}

	reserva.Habitaciones = habitaciones
	return reserva, nil
}

// CreateReserva crea una nueva reserva
func (r *reservaRepository) CreateReserva(reserva *domain.Reserva) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error al iniciar transacción: %w", err)
	}
	defer tx.Rollback()

	// Insertar la reserva principal
	query := `
		INSERT INTO reservation (
			adults_count,
			children_count,
			status,
			client_id,
			subtotal,
			discount,
			confirmation_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING reservation_id
	`

	err = tx.QueryRow(
		query,
		reserva.CantidadAdultos,
		reserva.CantidadNinhos,
		reserva.Estado,
		reserva.ClienteID,
		reserva.Subtotal,
		reserva.Descuento,
		reserva.FechaConfirmacion,
	).Scan(&reserva.ID)

	if err != nil {
		return fmt.Errorf("error al crear reserva: %w", err)
	}

	// Insertar las habitaciones de la reserva
	for i := range reserva.Habitaciones {
		habitacionQuery := `
			INSERT INTO reservation_room (
				reservation_id,
				room_id,
				price,
				check_in_date,
				check_out_date,
				status
			) VALUES ($1, $2, $3, $4, $5, $6)
		`

		_, err = tx.Exec(
			habitacionQuery,
			reserva.ID,
			reserva.Habitaciones[i].HabitacionID,
			reserva.Habitaciones[i].Precio,
			reserva.Habitaciones[i].FechaEntrada,
			reserva.Habitaciones[i].FechaSalida,
			1, // status activo
		)

		if err != nil {
			return fmt.Errorf("error al crear reserva de habitación: %w", err)
		}

		reserva.Habitaciones[i].ReservaID = reserva.ID
		reserva.Habitaciones[i].Estado = 1
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error al confirmar transacción: %w", err)
	}

	return nil
}

// UpdateReservastatus actualiza el status de una reserva
func (r *reservaRepository) UpdateReservaEstado(id int, status domain.EstadoReserva) error {
	query := `
		UPDATE reservation 
		SET status = $1 
		WHERE reservation_id = $2
	`

	result, err := r.db.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("error al actualizar status de reserva: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error al verificar filas afectadas: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("reserva con ID %d no encontrada", id)
	}

	return nil
}

// GetReservasCliente obtiene todas las reservas de un cliente
func (r *reservaRepository) GetReservasCliente(clientID int) ([]domain.Reserva, error) {
	query := `
		SELECT 
			r.reservation_id,
			r.adults_count,
			r.cantidadniños,
			r.status,
			r.client_id,
			r.subtotal,
			r.discount,
			r.confirmation_date
		FROM reservation r
		WHERE r.client_id = $1
		ORDER BY r.confirmation_date DESC
	`

	rows, err := r.db.Query(query, clientID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener reservas del cliente: %w", err)
	}
	defer rows.Close()

	var reservas []domain.Reserva
	for rows.Next() {
		var reserva domain.Reserva
		err := rows.Scan(
			&reserva.ID,
			&reserva.CantidadAdultos,
			&reserva.CantidadNinhos,
			&reserva.Estado,
			&reserva.ClienteID,
			&reserva.Subtotal,
			&reserva.Descuento,
			&reserva.FechaConfirmacion,
		)
		if err != nil {
			return nil, fmt.Errorf("error al escanear reserva: %w", err)
		}

		// Obtener las habitaciones de cada reserva
		habitacionesQuery := `
			SELECT 
				rh.reservation_id,
				rh.room_id,
				rh.price,
				rh.check_in_date,
				rh.check_out_date,
				rh.status,
				h.name,
				h.capacity,
				h.number
			FROM reservation_room rh
			INNER JOIN room h ON h.room_id = rh.room_id
			WHERE rh.reservation_id = $1 AND rh.status = 1
		`

		habRows, err := r.db.Query(habitacionesQuery, reserva.ID)
		if err != nil {
			return nil, fmt.Errorf("error al obtener habitaciones: %w", err)
		}

		var habitaciones []domain.ReservaHabitacion
		for habRows.Next() {
			var rh domain.ReservaHabitacion
			var habitacion domain.Habitacion

			err := habRows.Scan(
				&rh.ReservaID,
				&rh.HabitacionID,
				&rh.Precio,
				&rh.FechaEntrada,
				&rh.FechaSalida,
				&rh.Estado,
				&habitacion.Nombre,
				&habitacion.Capacidad,
				&habitacion.Numero,
			)
			if err != nil {
				habRows.Close()
				return nil, fmt.Errorf("error al escanear habitación: %w", err)
			}

			habitacion.ID = rh.HabitacionID
			rh.Habitacion = &habitacion
			habitaciones = append(habitaciones, rh)
		}
		habRows.Close()

		reserva.Habitaciones = habitaciones
		reservas = append(reservas, reserva)
	}

	return reservas, nil
}
