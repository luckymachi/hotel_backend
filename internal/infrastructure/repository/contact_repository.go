package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

type ContactRepository interface {
	Create(ctx context.Context, c domain.CreateContactRequest) (int64, error)
	List(ctx context.Context) ([]domain.Contact, error)
	UpdateEstado(ctx context.Context, id int64, estado domain.EstadoFormulario) error
}

type contactRepository struct {
	db *sql.DB
}

func NewContactRepository(db *sql.DB) ContactRepository {
	return &contactRepository{db: db}
}

func (r *contactRepository) Create(ctx context.Context, req domain.CreateContactRequest) (int64, error) {
	// Primero, verificar si ya existe un lead con este email
	var existingLeadID int64
	err := r.db.QueryRowContext(ctx, `SELECT lead_id FROM lead WHERE email=$1 LIMIT 1`, req.Email).Scan(&existingLeadID)
	if err == nil {
		// Si ya existe, devolver error y no insertar
		return 0, fmt.Errorf("El correo ya se encuentra registrado")
	} else if err != sql.ErrNoRows {
		// Error inesperado al consultar lead
		return 0, err
	}

	// No existe lead: insertar el contact_form
	insertQuery := `
	INSERT INTO contact_form (name, email, phone, message, status)
	VALUES ($1, $2, $3, $4, 'Nuevo')
	RETURNING form_id
`
	var id int64
	err = r.db.QueryRowContext(ctx, insertQuery,
		req.Nombre, req.Email, req.Telefono, req.Mensaje,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	// Después de insertar, intentar asociar lead_id por si se creó posteriormente
	var leadID int64
	err = r.db.QueryRowContext(ctx, `SELECT lead_id FROM lead WHERE email=$1 LIMIT 1`, req.Email).Scan(&leadID)
	if err == nil {
		_, updErr := r.db.ExecContext(ctx, `UPDATE contact_form SET lead_id=$1 WHERE form_id=$2`, leadID, id)
		if updErr != nil {
			return id, updErr
		}
	} else if err != sql.ErrNoRows {
		return id, err
	}

	return id, nil
}

func (r *contactRepository) List(ctx context.Context) ([]domain.Contact, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT form_id, name, email, phone, message, status, sent_date, response_date
		FROM contact_form ORDER BY sent_date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []domain.Contact
	for rows.Next() {
		var c domain.Contact
		if err := rows.Scan(
			&c.ID, &c.Nombre, &c.Email, &c.Telefono,
			&c.Mensaje, &c.Estado, &c.FechaEnvio, &c.FechaRespuesta,
		); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, nil
}

func (r *contactRepository) UpdateEstado(ctx context.Context, id int64, estado domain.EstadoFormulario) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE contact_form SET status=$1, response_date=NOW() WHERE form_id=$2`,
		estado, id)
	return err
}
