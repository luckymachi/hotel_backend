package http

import (
	"github.com/Maxito7/hotel_backend/internal/application"
	"github.com/gofiber/fiber/v2"
)

type ConfigHandler struct {
	service *application.ConfigService
}

func NewConfigHandler(service *application.ConfigService) *ConfigHandler {
	return &ConfigHandler{service: service}
}

// GetConfig godoc
// @Summary Get configuration value by key
// @Description Get a specific configuration setting
// @Tags configuration
// @Accept json
// @Produce json
// @Param key path string true "Configuration Key"
// @Success 200 {object} domain.HotelConfiguration
// @Failure 404 {object} map[string]interface{}
func (h *ConfigHandler) GetConfig(c *fiber.Ctx) error {
	key := c.Params("key")
	config, err := h.service.GetConfig(key)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(config)
}

// GetAllConfigs godoc
// @Summary Get all configuration values
// @Description Get all configuration settings
// @Tags configuration
// @Produce json
// @Success 200 {array} domain.HotelConfiguration
func (h *ConfigHandler) GetAllConfigs(c *fiber.Ctx) error {
	configs, err := h.service.GetAllConfigs()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	// Return as a map for easier frontend consumption: { "key": "value", ... } 
	// Or just array. Let's return array, frontend can map it.
	return c.JSON(configs)
}

// UpdateConfigRequest structure
type UpdateConfigRequest struct {
	Value string `json:"value"`
}

// UpdateConfig godoc
// @Summary Update configuration value
// @Description Update a specific configuration setting
// @Tags configuration
// @Accept json
// @Produce json
// @Param key path string true "Configuration Key"
// @Param request body UpdateConfigRequest true "New Value"
// @Success 200 {object} map[string]interface{}
func (h *ConfigHandler) UpdateConfig(c *fiber.Ctx) error {
	key := c.Params("key")
	var req UpdateConfigRequest
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.service.UpdateConfig(key, req.Value); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Configuration updated successfully",
		"key": key,
	})
}
