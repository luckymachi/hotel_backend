package application

import (
	"fmt"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

type PersonService struct {
	personRepo domain.PersonRepository
}

// NewPersonService crea una nueva instancia del servicio de personas
func NewPersonService(personRepo domain.PersonRepository) *PersonService {
	return &PersonService{
		personRepo: personRepo,
	}
}

// GetPersonByDocumentNumber obtiene una persona por su número de documento
func (s *PersonService) GetPersonByDocumentNumber(documentNumber string) (*domain.Person, error) {
	if documentNumber == "" {
		return nil, fmt.Errorf("el número de documento es requerido")
	}

	person, err := s.personRepo.FindByDocumentNumber(documentNumber)
	if err != nil {
		return nil, fmt.Errorf("error al buscar persona: %w", err)
	}

	if person == nil {
		return nil, fmt.Errorf("persona con documento %s no encontrada", documentNumber)
	}

	return person, nil
}
