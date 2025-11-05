package http

import (
	"time"

	"github.com/Maxito7/hotel_backend/internal/application"
	"github.com/Maxito7/hotel_backend/internal/domain"
	"github.com/gofiber/fiber/v2"
)

type PersonHandler struct {
	service *application.PersonService
}

// PersonResponse representa la respuesta de persona con el género traducido
type PersonResponse struct {
	PersonID         int       `json:"personId"`
	Name             string    `json:"name"`
	FirstSurname     string    `json:"firstSurname"`
	SecondSurname    *string   `json:"secondSurname,omitempty"`
	DocumentNumber   string    `json:"documentNumber"`
	Gender           string    `json:"gender"` // "Masculino", "Femenino", "Otro"
	Email            string    `json:"email"`
	Phone1           string    `json:"phone1"`
	Phone2           *string   `json:"phone2,omitempty"`
	ReferenceCity    string    `json:"referenceCity"`
	ReferenceCountry string    `json:"referenceCountry"`
	Active           bool      `json:"active"`
	CreationDate     time.Time `json:"creationDate"`
	BirthDate        time.Time `json:"birthDate"`
}

// NewPersonHandler crea una nueva instancia del handler de personas
func NewPersonHandler(service *application.PersonService) *PersonHandler {
	return &PersonHandler{
		service: service,
	}
}

// toPersonResponse convierte domain.Person a PersonResponse
func toPersonResponse(person *domain.Person) PersonResponse {
	return PersonResponse{
		PersonID:         person.PersonID,
		Name:             person.Name,
		FirstSurname:     person.FirstSurname,
		SecondSurname:    person.SecondSurname,
		DocumentNumber:   person.DocumentNumber,
		Gender:           convertGenderToFrontend(person.Gender),
		Email:            person.Email,
		Phone1:           person.Phone1,
		Phone2:           person.Phone2,
		ReferenceCity:    person.ReferenceCity,
		ReferenceCountry: person.ReferenceCountry,
		Active:           person.Active,
		CreationDate:     person.CreationDate,
		BirthDate:        person.BirthDate,
	}
}

// GetPersonByDocumentNumber obtiene una persona por su número de documento
func (h *PersonHandler) GetPersonByDocumentNumber(c *fiber.Ctx) error {
	documentNumber := c.Query("documentNumber")

	if documentNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El parámetro documentNumber es requerido",
		})
	}

	person, err := h.service.GetPersonByDocumentNumber(documentNumber)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convertir a PersonResponse para transformar el género
	response := toPersonResponse(person)

	return c.JSON(fiber.Map{
		"data": response,
	})
}
