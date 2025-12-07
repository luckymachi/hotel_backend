package repository

import (
	"database/sql"
	"fmt"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

type servicioRepository struct {
	db *sql.DB
}

// CreateService inserta un nuevo servicio en la base de datos
func (r *servicioRepository) CreateService(servicio *domain.Servicio) error {
	query := `INSERT INTO service (name, description, price, icon_key, status) VALUES ($1, $2, $3, $4, $5) RETURNING service_id`
	return r.db.QueryRow(query, servicio.Name, servicio.Description, servicio.Price, servicio.IconKey, servicio.Status).Scan(&servicio.ID)
}

// UpdateService actualiza un servicio existente
func (r *servicioRepository) UpdateService(servicio *domain.Servicio) error {
	query := `UPDATE service SET name=$1, description=$2, price=$3, icon_key=$4, status=$5 WHERE service_id=$6`
	_, err := r.db.Exec(query, servicio.Name, servicio.Description, servicio.Price, servicio.IconKey, servicio.Status, servicio.ID)
	return err
}

// DeleteService realiza una eliminación lógica (status=0)
func (r *servicioRepository) DeleteService(id int) error {
	query := `UPDATE service SET status=0 WHERE service_id=$1`
	_, err := r.db.Exec(query, id)
	return err
}

// NewServicioRepository crea una nueva instancia de servicioRepository
func NewServicioRepository(db *sql.DB) domain.ServicioRepository {
	return &servicioRepository{
		db: db,
	}
}

// GetAllServices implementa domain.ServicioRepository
func (r *servicioRepository) GetAllServices() ([]domain.Servicio, error) {
	query := `
		SELECT 
			service_id,
			name,
			description,
			price
            ,icon_key
            ,status
		FROM 
			service
		ORDER BY 
			service_id;`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying services: %w", err)
	}
	defer rows.Close()

	var servicios []domain.Servicio
	for rows.Next() {
		var s domain.Servicio
		err := rows.Scan(
			&s.ID,
			&s.Name,
			&s.Description,
			&s.Price,
			&s.IconKey,
			&s.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning service: %w", err)
		}
		servicios = append(servicios, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating services: %w", err)
	}

	return servicios, nil
}
