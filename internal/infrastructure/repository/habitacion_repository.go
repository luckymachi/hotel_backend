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

	// Attach images for room types
	imagesMap, err := r.getImagesForRoomTypeIDs(typeIDs)
	if err != nil {
		return nil, fmt.Errorf("error fetching images for room types: %w", err)
	}
	for i := range roomTypes {
		if imgs, ok := imagesMap[roomTypes[i].ID]; ok {
			roomTypes[i].Images = imgs
		}
	}

	return roomTypes, nil
}

// GetRoomTypeByID devuelve un tipo de habitación con amenities e imágenes
func (r *habitacionRepository) GetRoomTypeByID(id int) (domain.TipoHabitacion, error) {
	query := `
		SELECT room_type_id, title, description, adult_capacity, children_capacity, beds_count, area, price
		FROM room_type
		WHERE room_type_id = $1;`

	var rt domain.TipoHabitacion
	err := r.db.QueryRow(query, id).Scan(
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
		if err == sql.ErrNoRows {
			return rt, fmt.Errorf("room type not found: %w", err)
		}
		return rt, fmt.Errorf("error querying room type: %w", err)
	}

	// amenities
	am, err := r.getAmenitiesForRoomTypeIDs([]int{rt.ID})
	if err != nil {
		return rt, err
	}
	if a, ok := am[rt.ID]; ok {
		rt.Amenities = a
	}

	// images
	imgsMap, err := r.getImagesForRoomTypeIDs([]int{rt.ID})
	if err != nil {
		return rt, err
	}
	if imgs, ok := imgsMap[rt.ID]; ok {
		rt.Images = imgs
	}

	return rt, nil
}

// CreateRoom crea una habitación nueva y devuelve su id
func (r *habitacionRepository) CreateRoom(h domain.Habitacion) (int, error) {
	var newID int
	err := r.db.QueryRow(
		`INSERT INTO room (name, number, capacity, status, general_description, room_type_id) VALUES ($1,$2,$3,$4,$5,$6) RETURNING room_id`,
		h.Nombre, h.Numero, h.Capacidad, h.Estado, h.DescripcionGeneral, h.TipoHabitacion.ID,
	).Scan(&newID)
	if err != nil {
		return 0, fmt.Errorf("error inserting room: %w", err)
	}
	return newID, nil
}

// UpdateRoom actualiza una habitación existente
func (r *habitacionRepository) UpdateRoom(id int, h domain.Habitacion) error {
	_, err := r.db.Exec(`UPDATE room SET name=$1, number=$2, capacity=$3, status=$4, general_description=$5, room_type_id=$6 WHERE room_id=$7`,
		h.Nombre, h.Numero, h.Capacidad, h.Estado, h.DescripcionGeneral, h.TipoHabitacion.ID, id)
	if err != nil {
		return fmt.Errorf("error updating room: %w", err)
	}
	return nil
}

// DeleteRoom elimina una habitación por id
func (r *habitacionRepository) DeleteRoom(id int) error {
	_, err := r.db.Exec(`DELETE FROM room WHERE room_id = $1`, id)
	if err != nil {
		return fmt.Errorf("error deleting room: %w", err)
	}
	return nil
}

// GetRoomByID retorna una habitación con su tipo y relaciones del tipo
func (r *habitacionRepository) GetRoomByID(id int) (domain.Habitacion, error) {
	query := `
		SELECT room_id, name, number, capacity, status, general_description, room_type_id
		FROM room
		WHERE room_id = $1;`

	var h domain.Habitacion
	var roomTypeID int
	err := r.db.QueryRow(query, id).Scan(&h.ID, &h.Nombre, &h.Numero, &h.Capacidad, &h.Estado, &h.DescripcionGeneral, &roomTypeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return h, fmt.Errorf("room not found: %w", err)
		}
		return h, fmt.Errorf("error querying room: %w", err)
	}

	// Get room type details
	rt, err := r.GetRoomTypeByID(roomTypeID)
	if err != nil {
		return h, fmt.Errorf("error fetching room type for room: %w", err)
	}
	h.TipoHabitacion = rt

	return h, nil
}

// CreateRoomType crea un tipo de habitación y sus relaciones (amenities, images) en una transacción
func (r *habitacionRepository) CreateRoomType(rt domain.TipoHabitacion, amenityIDs []int, images []domain.RoomImage) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("error starting tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var newID int
	err = tx.QueryRow(
		`INSERT INTO room_type (title, description, adult_capacity, children_capacity, beds_count, area, price) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING room_type_id`,
		rt.Titulo, rt.Descripcion, rt.CapacidadAdultos, rt.CapacidadNinhos, rt.CantidadCamas, rt.Area, rt.Precio,
	).Scan(&newID)
	if err != nil {
		return 0, fmt.Errorf("error inserting room_type: %w", err)
	}

	// amenities
	if len(amenityIDs) > 0 {
		// build bulk insert
		var vals []string
		var args []interface{}
		for i, aid := range amenityIDs {
			vals = append(vals, fmt.Sprintf("($%d,$%d)", i*2+1, i*2+2))
			args = append(args, newID, aid)
		}
		q := fmt.Sprintf("INSERT INTO room_type_amenities (room_type_id, amenity_id) VALUES %s", strings.Join(vals, ","))
		if _, err = tx.Exec(q, args...); err != nil {
			return 0, fmt.Errorf("error inserting room_type_amenities: %w", err)
		}
	}

	// images
	if len(images) > 0 {
		var vals []string
		var args []interface{}
		for i, img := range images {
			// columns: room_type_id, url, alt_text, is_primary, sort_order, is_active
			vals = append(vals, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d)", i*6+1, i*6+2, i*6+3, i*6+4, i*6+5, i*6+6))
			args = append(args, newID, img.URL, img.AltText, img.IsPrimary, img.SortOrder, img.IsActive)
		}
		q := fmt.Sprintf("INSERT INTO room_type_images (room_type_id, url, alt_text, is_primary, sort_order, is_active) VALUES %s", strings.Join(vals, ","))
		if _, err = tx.Exec(q, args...); err != nil {
			return 0, fmt.Errorf("error inserting room_type_images: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("error committing tx: %w", err)
	}

	return newID, nil
}

// UpdateRoomType actualiza los campos y reemplaza amenities e imágenes dentro de una transacción
func (r *habitacionRepository) UpdateRoomType(id int, rt domain.TipoHabitacion, amenityIDs []int, images []domain.RoomImage) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.Exec(`UPDATE room_type SET title=$1, description=$2, adult_capacity=$3, children_capacity=$4, beds_count=$5, area=$6, price=$7 WHERE room_type_id=$8`,
		rt.Titulo, rt.Descripcion, rt.CapacidadAdultos, rt.CapacidadNinhos, rt.CantidadCamas, rt.Area, rt.Precio, id)
	if err != nil {
		return fmt.Errorf("error updating room_type: %w", err)
	}

	// replace amenities
	if _, err = tx.Exec(`DELETE FROM room_type_amenities WHERE room_type_id = $1`, id); err != nil {
		return fmt.Errorf("error deleting old amenities: %w", err)
	}
	if len(amenityIDs) > 0 {
		var vals []string
		var args []interface{}
		for i, aid := range amenityIDs {
			vals = append(vals, fmt.Sprintf("($%d,$%d)", i*2+1, i*2+2))
			args = append(args, id, aid)
		}
		q := fmt.Sprintf("INSERT INTO room_type_amenities (room_type_id, amenity_id) VALUES %s", strings.Join(vals, ","))
		if _, err = tx.Exec(q, args...); err != nil {
			return fmt.Errorf("error inserting new amenities: %w", err)
		}
	}

	// replace images
	if _, err = tx.Exec(`DELETE FROM room_type_images WHERE room_type_id = $1`, id); err != nil {
		return fmt.Errorf("error deleting old images: %w", err)
	}
	if len(images) > 0 {
		var vals []string
		var args []interface{}
		for i, img := range images {
			vals = append(vals, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d)", i*6+1, i*6+2, i*6+3, i*6+4, i*6+5, i*6+6))
			args = append(args, id, img.URL, img.AltText, img.IsPrimary, img.SortOrder, img.IsActive)
		}
		q := fmt.Sprintf("INSERT INTO room_type_images (room_type_id, url, alt_text, is_primary, sort_order, is_active) VALUES %s", strings.Join(vals, ","))
		if _, err = tx.Exec(q, args...); err != nil {
			return fmt.Errorf("error inserting new images: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error committing tx: %w", err)
	}

	return nil
}

// DeleteRoomType elimina un tipo de habitación y sus relaciones
func (r *habitacionRepository) DeleteRoomType(id int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// eliminar relaciones primero
	if _, err = tx.Exec(`DELETE FROM room_type_amenities WHERE room_type_id = $1`, id); err != nil {
		return fmt.Errorf("error deleting amenities relations: %w", err)
	}
	if _, err = tx.Exec(`DELETE FROM room_type_images WHERE room_type_id = $1`, id); err != nil {
		return fmt.Errorf("error deleting images relations: %w", err)
	}
	if _, err = tx.Exec(`DELETE FROM room_type WHERE room_type_id = $1`, id); err != nil {
		return fmt.Errorf("error deleting room_type: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error committing tx: %w", err)
	}
	return nil
}

// SetAmenitiesForRoomType reemplaza amenities de un tipo (operación separada, sin tx externo)
func (r *habitacionRepository) SetAmenitiesForRoomType(roomTypeID int, amenityIDs []int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`DELETE FROM room_type_amenities WHERE room_type_id = $1`, roomTypeID); err != nil {
		return fmt.Errorf("error deleting old amenities: %w", err)
	}
	if len(amenityIDs) > 0 {
		var vals []string
		var args []interface{}
		for i, aid := range amenityIDs {
			vals = append(vals, fmt.Sprintf("($%d,$%d)", i*2+1, i*2+2))
			args = append(args, roomTypeID, aid)
		}
		q := fmt.Sprintf("INSERT INTO room_type_amenities (room_type_id, amenity_id) VALUES %s", strings.Join(vals, ","))
		if _, err = tx.Exec(q, args...); err != nil {
			return fmt.Errorf("error inserting amenities: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error committing tx: %w", err)
	}
	return nil
}

// SetImagesForRoomType reemplaza las imágenes de un tipo de habitación
func (r *habitacionRepository) SetImagesForRoomType(roomTypeID int, images []domain.RoomImage) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`DELETE FROM room_type_images WHERE room_type_id = $1`, roomTypeID); err != nil {
		return fmt.Errorf("error deleting old images: %w", err)
	}
	if len(images) > 0 {
		var vals []string
		var args []interface{}
		for i, img := range images {
			vals = append(vals, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d)", i*6+1, i*6+2, i*6+3, i*6+4, i*6+5, i*6+6))
			args = append(args, roomTypeID, img.URL, img.AltText, img.IsPrimary, img.SortOrder, img.IsActive)
		}
		q := fmt.Sprintf("INSERT INTO room_type_images (room_type_id, url, alt_text, is_primary, sort_order, is_active) VALUES %s", strings.Join(vals, ","))
		if _, err = tx.Exec(q, args...); err != nil {
			return fmt.Errorf("error inserting images: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error committing tx: %w", err)
	}
	return nil
}

// getImagesForRoomTypeIDs devuelve un mapa room_type_id -> []RoomImage
func (r *habitacionRepository) getImagesForRoomTypeIDs(ids []int) (map[int][]domain.RoomImage, error) {
	result := make(map[int][]domain.RoomImage)
	if len(ids) == 0 {
		return result, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query := fmt.Sprintf(`SELECT room_type_id, id, url, alt_text, is_primary, sort_order, is_active FROM room_type_images WHERE room_type_id IN (%s) ORDER BY sort_order;`, strings.Join(placeholders, ","))
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying images: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rtID int
		var img domain.RoomImage
		if err := rows.Scan(&rtID, &img.ID, &img.URL, &img.AltText, &img.IsPrimary, &img.SortOrder, &img.IsActive); err != nil {
			return nil, fmt.Errorf("error scanning image row: %w", err)
		}
		result[rtID] = append(result[rtID], img)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating image rows: %w", err)
	}
	return result, nil
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

// ListAmenities returns all amenities available in the system
func (r *habitacionRepository) ListAmenities() ([]domain.Amenity, error) {
	query := `SELECT id, name, description FROM amenities ORDER BY name;`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying amenities: %w", err)
	}
	defer rows.Close()

	var amenities []domain.Amenity
	for rows.Next() {
		var a domain.Amenity
		if err := rows.Scan(&a.ID, &a.Name, &a.Description); err != nil {
			return nil, fmt.Errorf("error scanning amenity row: %w", err)
		}
		amenities = append(amenities, a)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating amenity rows: %w", err)
	}
	return amenities, nil
}
