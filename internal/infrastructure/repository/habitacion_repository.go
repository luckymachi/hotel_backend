package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

type habitacionRepository struct {
	db *sql.DB
}

// NewHabitacionRepository creates a new instance of habitacionRepository
func NewHabitacionRepository(db *sql.DB) domain.HabitacionRepository {
	return &habitacionRepository{
		db: db,
	}
}

// GetAllRooms implements domain.HabitacionRepository
func (r *habitacionRepository) GetAllRooms() ([]domain.Habitacion, error) {
	query := `
		SELECT 
			h.room_id,
			h.name,
			h.number,
			h.capacity,
			h.status,
			h.general_description,
			t.room_type_id,
			t.title,
			t.description,
			t.adult_capacity,
			t.children_capacity,
			t.beds_count,
			t.area,
			t.price
		FROM 
			room h
		INNER JOIN 
			room_type t ON h.room_type_id = t.room_type_id
		ORDER BY 
			h.room_id;`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying database: %w", err)
	}
	defer rows.Close()

	var habitaciones []domain.Habitacion
	for rows.Next() {
		var h domain.Habitacion
		err := rows.Scan(
			&h.ID,
			&h.Nombre,
			&h.Numero,
			&h.Capacidad,
			&h.Estado,
			&h.DescripcionGeneral,
			&h.TipoHabitacion.ID,
			&h.TipoHabitacion.Titulo,
			&h.TipoHabitacion.Descripcion,
			&h.TipoHabitacion.CapacidadAdultos,
			&h.TipoHabitacion.CapacidadNinhos,
			&h.TipoHabitacion.CantidadCamas,
			&h.TipoHabitacion.Area,
			&h.TipoHabitacion.Precio,
		)
		if err != nil {
			return nil, err
		}
		habitaciones = append(habitaciones, h)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Attach amenities for room types
	// Collect unique room_type IDs
	typeIDsMap := make(map[int]struct{})
	var typeIDs []int
	for _, h := range habitaciones {
		if _, ok := typeIDsMap[h.TipoHabitacion.ID]; !ok {
			typeIDsMap[h.TipoHabitacion.ID] = struct{}{}
			typeIDs = append(typeIDs, h.TipoHabitacion.ID)
		}
	}

	amenitiesMap, err := r.getAmenitiesForRoomTypeIDs(typeIDs)
	if err != nil {
		return nil, fmt.Errorf("error fetching amenities: %w", err)
	}

	for i := range habitaciones {
		if a, ok := amenitiesMap[habitaciones[i].TipoHabitacion.ID]; ok {
			habitaciones[i].TipoHabitacion.Amenities = a
		}
	}

	return habitaciones, nil
}

// GetDisponibilidadFechas implementa domain.HabitacionRepository
func (r *habitacionRepository) GetDisponibilidadFechas(desde, hasta time.Time) ([]domain.DisponibilidadFecha, error) {
	query := `
		WITH RECURSIVE fechas AS (
			SELECT date(cast($1 as timestamp)) as fecha
			UNION ALL
			SELECT fecha + interval '1 day'
			FROM fechas
			WHERE fecha < date(cast($2 as timestamp))
		),
		habitaciones_totales AS (
			SELECT COUNT(*) as total
			FROM room
			WHERE status = 'Disponible'
		),
		habitaciones_ocupadas AS (
			SELECT date(f.fecha) as fecha, 
				   COUNT(DISTINCT rh.room_id) as ocupadas
			FROM fechas f
			LEFT JOIN reservation_room rh ON 
				date(f.fecha) BETWEEN date(rh.check_in_date) AND date(rh.check_out_date)
			LEFT JOIN reservation r ON r.reservation_id = rh.reservation_id
				AND rh.status = 1
				AND r.status = 'Confirmada'
			GROUP BY date(f.fecha)
		)
		SELECT 
			f.fecha,
			CASE 
				WHEN (ht.total - COALESCE(ho.ocupadas, 0)) > 0 THEN true 
				ELSE false 
			END as disponible,
			(ht.total - COALESCE(ho.ocupadas, 0)) as habitaciones_disponibles
		FROM fechas f
		CROSS JOIN habitaciones_totales ht
		LEFT JOIN habitaciones_ocupadas ho ON date(f.fecha) = date(ho.fecha)
		ORDER BY f.fecha;`

	rows, err := r.db.Query(query, desde, hasta)
	if err != nil {
		return nil, fmt.Errorf("error querying disponibilidad: %w", err)
	}
	defer rows.Close()

	var disponibilidades []domain.DisponibilidadFecha
	for rows.Next() {
		var d domain.DisponibilidadFecha
		err := rows.Scan(&d.Fecha, &d.Disponible, &d.Habitaciones)
		if err != nil {
			return nil, fmt.Errorf("error scanning disponibilidad: %w", err)
		}
		disponibilidades = append(disponibilidades, d)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating disponibilidad rows: %w", err)
	}

	return disponibilidades, nil
}

// GetFechasBloqueadas implementa domain.HabitacionRepository
func (r *habitacionRepository) GetFechasBloqueadas(desde, hasta time.Time) (*domain.FechasBloqueadas, error) {
	query := `
		WITH RECURSIVE fechas AS (
			SELECT cast($1 as date) as fecha
			UNION ALL
			SELECT (fecha + interval '1 day')::date
			FROM fechas
			WHERE fecha < cast($2 as date)
		),
		habitaciones_totales AS (
			SELECT COUNT(*) as total
			FROM room h
			WHERE h.status = 'Disponible'
		),
		habitaciones_ocupadas AS (
			SELECT date(f.fecha) as fecha, 
				   COUNT(DISTINCT rh.room_id) as habitaciones_ocupadas
			FROM fechas f
			LEFT JOIN reservation_room rh ON 
				f.fecha BETWEEN cast(rh.check_in_date as date) AND cast(rh.check_out_date as date)
			LEFT JOIN reservation r ON r.reservation_id = rh.reservation_id
				AND rh.status = 1
				AND r.status = 'Confirmada'
			GROUP BY f.fecha
			HAVING COUNT(DISTINCT rh.room_id) >= (SELECT total FROM habitaciones_totales)
		)
		SELECT fecha::date
		FROM habitaciones_ocupadas
		ORDER BY fecha;`

	rows, err := r.db.Query(query, desde, hasta)
	if err != nil {
		return nil, fmt.Errorf("error querying fechas bloqueadas: %w", err)
	}
	defer rows.Close()

	fechasBloqueadas := &domain.FechasBloqueadas{
		FechasNoDisponibles: make([]time.Time, 0),
	}

	for rows.Next() {
		var fecha time.Time
		if err := rows.Scan(&fecha); err != nil {
			return nil, fmt.Errorf("error scanning fecha bloqueada: %w", err)
		}
		fechasBloqueadas.FechasNoDisponibles = append(fechasBloqueadas.FechasNoDisponibles, fecha)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fechas bloqueadas: %w", err)
	}

	return fechasBloqueadas, nil
}

// GetAvailableRooms implements domain.HabitacionRepository
func (r *habitacionRepository) GetAvailableRooms(fechaEntrada, fechaSalida time.Time) ([]domain.Habitacion, error) {
	query := `
		SELECT DISTINCT 
			h.room_id,
			h.name,
			h.number,
			h.capacity,
			h.status,
			h.general_description,
			t.room_type_id,
			t.title,
			t.description,
			t.adult_capacity,
			t.children_capacity,
			t.beds_count,
			t.area,
			t.price
		FROM 
			room h
		INNER JOIN 
			room_type t ON h.room_type_id = t.room_type_id
		WHERE 
			h.status = 'Disponible'
			AND NOT EXISTS (
				SELECT 1 FROM reservation_room rh
				JOIN reservation r ON r.reservation_id = rh.reservation_id
				WHERE rh.room_id = h.room_id
				AND rh.status = 1
				AND r.status = 'Confirmada'
				AND (
					(rh.check_in_date <= $1 AND rh.check_out_date >= $1)
					OR (rh.check_in_date <= $2 AND rh.check_out_date >= $2)
					OR (rh.check_in_date >= $1 AND rh.check_out_date <= $2)
				)
			)
		ORDER BY 
			h.room_id;`

	rows, err := r.db.Query(query, fechaEntrada, fechaSalida)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var habitaciones []domain.Habitacion
	for rows.Next() {
		var h domain.Habitacion
		err := rows.Scan(
			&h.ID,
			&h.Nombre,
			&h.Numero,
			&h.Capacidad,
			&h.Estado,
			&h.DescripcionGeneral,
			&h.TipoHabitacion.ID,
			&h.TipoHabitacion.Titulo,
			&h.TipoHabitacion.Descripcion,
			&h.TipoHabitacion.CapacidadAdultos,
			&h.TipoHabitacion.CapacidadNinhos,
			&h.TipoHabitacion.CantidadCamas,
			&h.TipoHabitacion.Area,
			&h.TipoHabitacion.Precio,
		)
		if err != nil {
			return nil, err
		}
		habitaciones = append(habitaciones, h)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Attach amenities for room types similar to GetAllRooms
	typeIDsMap := make(map[int]struct{})
	var typeIDs []int
	for _, h := range habitaciones {
		if _, ok := typeIDsMap[h.TipoHabitacion.ID]; !ok {
			typeIDsMap[h.TipoHabitacion.ID] = struct{}{}
			typeIDs = append(typeIDs, h.TipoHabitacion.ID)
		}
	}

	amenitiesMap, err := r.getAmenitiesForRoomTypeIDs(typeIDs)
	if err != nil {
		return nil, fmt.Errorf("error fetching amenities: %w", err)
	}

	for i := range habitaciones {
		if a, ok := amenitiesMap[habitaciones[i].TipoHabitacion.ID]; ok {
			habitaciones[i].TipoHabitacion.Amenities = a
		}
	}

	return habitaciones, nil
}

// GetRoomTypes implements domain.HabitacionRepository
func (r *habitacionRepository) GetRoomTypes() ([]domain.TipoHabitacion, error) {
	query := `
		SELECT 
			room_type_id,
			title,
			description,
			adult_capacity,
			children_capacity,
			beds_count,
			area,
			price
		FROM 
			room_type
		ORDER BY 
			room_type_id;`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying room types: %w", err)
	}
	defer rows.Close()

	var roomTypes []domain.TipoHabitacion
	for rows.Next() {
		var rt domain.TipoHabitacion
		err := rows.Scan(
			&rt.ID,
			&rt.Titulo,
			&rt.Descripcion,
			&rt.CapacidadAdultos,
			&rt.CapacidadNinhos,
			&rt.CantidadCamas,
			&rt.Area,
			&rt.Precio,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning room type: %w", err)
		}
		roomTypes = append(roomTypes, rt)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating room types rows: %w", err)
	}

	// Attach amenities to room types
	var typeIDs []int
	for _, rt := range roomTypes {
		typeIDs = append(typeIDs, rt.ID)
	}

	amenitiesMap, err := r.getAmenitiesForRoomTypeIDs(typeIDs)
	if err != nil {
		return nil, fmt.Errorf("error fetching amenities for room types: %w", err)
	}

	for i := range roomTypes {
		if a, ok := amenitiesMap[roomTypes[i].ID]; ok {
			roomTypes[i].Amenities = a
		}
	}

	return roomTypes, nil
}

// getAmenitiesForRoomTypeIDs fetches amenities for the provided room_type IDs
func (r *habitacionRepository) getAmenitiesForRoomTypeIDs(ids []int) (map[int][]domain.Amenity, error) {
	result := make(map[int][]domain.Amenity)
	if len(ids) == 0 {
		return result, nil
	}

	// Build placeholders and args for IN clause
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT rta.room_type_id, a.id, a.name, a.description
		FROM room_type_amenities rta
		JOIN amenities a ON rta.amenity_id = a.id
		WHERE rta.room_type_id IN (%s);`, strings.Join(placeholders, ","))

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying amenities: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var roomTypeID int
		var a domain.Amenity
		if err := rows.Scan(&roomTypeID, &a.ID, &a.Name, &a.Description); err != nil {
			return nil, fmt.Errorf("error scanning amenity row: %w", err)
		}
		result[roomTypeID] = append(result[roomTypeID], a)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating amenity rows: %w", err)
	}

	return result, nil
}
