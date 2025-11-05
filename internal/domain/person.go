package domain

import "time"

// Person representa una persona en el sistema
type Person struct {
	PersonID         int       `json:"personId"`
	Name             string    `json:"name"`
	FirstSurname     string    `json:"firstSurname"`
	SecondSurname    *string   `json:"secondSurname,omitempty"` // Puntero para permitir NULL
	DocumentNumber   string    `json:"documentNumber"`
	Gender           string    `json:"gender"`
	Email            string    `json:"email"`
	Phone1           string    `json:"phone1"`
	Phone2           *string   `json:"phone2,omitempty"` // Puntero para permitir NULL
	ReferenceCity    string    `json:"referenceCity"`
	ReferenceCountry string    `json:"referenceCountry"`
	Active           bool      `json:"active"`
	CreationDate     time.Time `json:"creationDate"`
	BirthDate        time.Time `json:"birthDate"`
}

// PersonRepository define las operaciones con personas
type PersonRepository interface {
	// FindByDocumentNumber busca una persona por su número de documento
	FindByDocumentNumber(documentNumber string) (*Person, error)
	// Create crea una nueva persona
	Create(person *Person) error
	// GetByID obtiene una persona por su ID
	GetByID(id int) (*Person, error)
	// Update actualiza los datos de una persona existente
	Update(person *Person) error
}
