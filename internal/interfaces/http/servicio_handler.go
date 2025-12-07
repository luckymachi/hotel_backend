package http

import (
	"github.com/Maxito7/hotel_backend/internal/application"
	"github.com/Maxito7/hotel_backend/internal/domain"
	"github.com/gofiber/fiber/v2"
)

// ServicioRequest representa el payload para crear/editar servicios
type ServicioRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	IconKey     string  `json:"icon_key"`
	Status      int     `json:"status"`
}
type ServicioHandler struct {
	service *application.ServicioService
}

func NewServicioHandler(service *application.ServicioService) *ServicioHandler {
	return &ServicioHandler{
		service: service,
	}
}

// Crear un nuevo servicio
func (h *ServicioHandler) CreateService(c *fiber.Ctx) error {
	var req ServicioRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Datos inválidos"})
	}
	servicio := &domain.Servicio{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		IconKey:     req.IconKey,
		Status:      req.Status,
	}
	if servicio.Status == 0 {
		servicio.Status = 1 // Por defecto disponible
	}
	if err := h.service.CreateService(servicio); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "No se pudo crear el servicio"})
	}
	return c.Status(fiber.StatusCreated).JSON(servicio)
}

// Editar un servicio existente
func (h *ServicioHandler) UpdateService(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID inválido"})
	}
	var req ServicioRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Datos inválidos"})
	}
	servicio := &domain.Servicio{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		IconKey:     req.IconKey,
		Status:      req.Status,
	}
	if err := h.service.UpdateService(servicio); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "No se pudo actualizar el servicio"})
	}
	return c.JSON(servicio)
}

// Eliminación lógica de un servicio
func (h *ServicioHandler) DeleteService(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID inválido"})
	}
	if err := h.service.DeleteService(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "No se pudo eliminar el servicio"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *ServicioHandler) GetAllServices(c *fiber.Ctx) error {
	servicios, err := h.service.GetAllServices()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener los servicios",
		})
	}

	// Mapear explícitamente para asegurar los campos icon_key y status
	type ServicioResponse struct {
		ID          int     `json:"id"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		IconKey     string  `json:"icon_key"`
		Status      int     `json:"status"`
	}
	var resp []ServicioResponse
	for _, s := range servicios {
		resp = append(resp, ServicioResponse{
			ID:          s.ID,
			Name:        s.Name,
			Description: s.Description,
			Price:       s.Price,
			IconKey:     s.IconKey,
			Status:      s.Status,
		})
	}
	return c.JSON(resp)
}
