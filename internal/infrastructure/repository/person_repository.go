package repository

import (
	"database/sql"
	"fmt"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

type personRepository struct {
	db *sql.DB
}

// NewPersonRepository crea una nueva instancia del repositorio de personas
func NewPersonRepository(db *sql.DB) domain.PersonRepository {
	return &personRepository{db: db}
}

// FindByDocumentNumber busca una persona por su número de documento
func (r *personRepository) FindByDocumentNumber(documentNumber string) (*domain.Person, error) {
	query := `
		SELECT 
			person_id,
			name,
			first_surname,
			second_surname,
			document_number,
			gender,
			email,
			phone_1,
			phone_2,
			reference_city,
			reference_country,
			active,
			creation_date,
			birth_date
		FROM person
		WHERE document_number = $1
	`

	person := &domain.Person{}
	var phone2 sql.NullString
	var secondSurname sql.NullString

	err := r.db.QueryRow(query, documentNumber).Scan(
		&person.PersonID,
		&person.Name,
		&person.FirstSurname,
		&secondSurname,
		&person.DocumentNumber,
		&person.Gender,
		&person.Email,
		&person.Phone1,
		&phone2,
		&person.ReferenceCity,
		&person.ReferenceCountry,
		&person.Active,
		&person.CreationDate,
		&person.BirthDate,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No existe, devolver nil sin error
	}

	if err != nil {
		return nil, fmt.Errorf("error al buscar persona: %w", err)
	}

	// Convertir sql.NullString a *string
	if secondSurname.Valid {
		person.SecondSurname = &secondSurname.String
	}
	if phone2.Valid {
		person.Phone2 = &phone2.String
	}

	return person, nil
}

// Create crea una nueva persona
func (r *personRepository) Create(person *domain.Person) error {
	query := `
		INSERT INTO person (
			name,
			first_surname,
			second_surname,
			document_number,
			gender,
			email,
			phone_1,
			phone_2,
			reference_city,
			reference_country,
			active,
			creation_date,
			birth_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING person_id
	`

	// Convertir *string a sql.NullString
	var secondSurname sql.NullString
	if person.SecondSurname != nil {
		secondSurname = sql.NullString{String: *person.SecondSurname, Valid: true}
	}

	var phone2 sql.NullString
	if person.Phone2 != nil {
		phone2 = sql.NullString{String: *person.Phone2, Valid: true}
	}

	err := r.db.QueryRow(
		query,
		person.Name,
		person.FirstSurname,
		secondSurname,
		person.DocumentNumber,
		person.Gender,
		person.Email,
		person.Phone1,
		phone2,
		person.ReferenceCity,
		person.ReferenceCountry,
		person.Active,
		person.CreationDate,
		person.BirthDate,
	).Scan(&person.PersonID)

	if err != nil {
		return fmt.Errorf("error al crear persona: %w", err)
	}

	return nil
}

// GetByID obtiene una persona por su ID
func (r *personRepository) GetByID(id int) (*domain.Person, error) {
	query := `
		SELECT 
			person_id,
			name,
			first_surname,
			second_surname,
			document_number,
			gender,
			email,
			phone_1,
			phone_2,
			reference_city,
			reference_country,
			active,
			creation_date,
			birth_date
		FROM person
		WHERE person_id = $1
	`

	person := &domain.Person{}
	var phone2 sql.NullString
	var secondSurname sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&person.PersonID,
		&person.Name,
		&person.FirstSurname,
		&secondSurname,
		&person.DocumentNumber,
		&person.Gender,
		&person.Email,
		&person.Phone1,
		&phone2,
		&person.ReferenceCity,
		&person.ReferenceCountry,
		&person.Active,
		&person.CreationDate,
		&person.BirthDate,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("persona con ID %d no encontrada", id)
	}

	if err != nil {
		return nil, fmt.Errorf("error al obtener persona: %w", err)
	}

	// Convertir sql.NullString a *string
	if secondSurname.Valid {
		person.SecondSurname = &secondSurname.String
	}
	if phone2.Valid {
		person.Phone2 = &phone2.String
	}

	return person, nil
}

// Update actualiza los datos de una persona existente
func (r *personRepository) Update(person *domain.Person) error {
	query := `
		UPDATE person
		SET 
			name = $1,
			first_surname = $2,
			second_surname = $3,
			gender = $4,
			email = $5,
			phone_1 = $6,
			phone_2 = $7,
			reference_city = $8,
			reference_country = $9,
			birth_date = $10
		WHERE person_id = $11
	`

	// Convertir *string a sql.NullString
	var secondSurname sql.NullString
	if person.SecondSurname != nil {
		secondSurname = sql.NullString{String: *person.SecondSurname, Valid: true}
	}

	var phone2 sql.NullString
	if person.Phone2 != nil {
		phone2 = sql.NullString{String: *person.Phone2, Valid: true}
	}

	result, err := r.db.Exec(
		query,
		person.Name,
		person.FirstSurname,
		secondSurname,
		person.Gender,
		person.Email,
		person.Phone1,
		phone2,
		person.ReferenceCity,
		person.ReferenceCountry,
		person.BirthDate,
		person.PersonID,
	)

	if err != nil {
		return fmt.Errorf("error al actualizar persona: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error al verificar actualización: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("persona con ID %d no encontrada", person.PersonID)
	}

	return nil
}
