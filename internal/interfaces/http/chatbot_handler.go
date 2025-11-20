package http

import (
	"log"

	"github.com/Maxito7/hotel_backend/internal/application"
	"github.com/Maxito7/hotel_backend/internal/domain"
	"github.com/gofiber/fiber/v2"
)

type ChatbotHandler struct {
	service *application.ChatbotService
}

func NewChatbotHandler(service *application.ChatbotService) *ChatbotHandler {
	return &ChatbotHandler{
		service: service,
	}
}

func (h *ChatbotHandler) Chat(c *fiber.Ctx) error {
	// Log del body para depuración
	log.Printf("Received body: %s", string(c.Body()))
	log.Printf("Content-Type: %s", c.Get("Content-Type"))

	var req domain.ChatRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request format",
			"details": err.Error(),
			"body":    string(c.Body()),
		})
	}

	log.Printf("Parsed request: %+v", req)

	if req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Message is required",
		})
	}

	// Usar ProcessMessageV2 que tiene detección automática de intenciones
	response, err := h.service.ProcessMessageV2(req)
	if err != nil {
		log.Printf("Error processing message: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(response)
}

func (h *ChatbotHandler) GetConversation(c *fiber.Ctx) error {
	conversationID := c.Params("id")
	if conversationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Conversation ID is required",
		})
	}

	conversation, err := h.service.GetConversationHistory(conversationID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if conversation == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Conversation not found",
		})
	}

	return c.JSON(conversation)
}

func (h *ChatbotHandler) GetClientConversations(c *fiber.Ctx) error {
	clienteID, err := c.ParamsInt("clienteId")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid cliente ID",
		})
	}

	conversations, err := h.service.GetClientConversations(clienteID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(conversations)
}
