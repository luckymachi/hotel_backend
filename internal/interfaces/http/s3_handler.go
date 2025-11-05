package http

import (
	"fmt"
	"log"

	services "github.com/Maxito7/hotel_backend/internal/service"
	"github.com/gofiber/fiber/v2"
)

type S3Handler struct {
	service *services.S3Service
}

func NewS3Handler(service *services.S3Service) *S3Handler {
	return &S3Handler{
		service: service,
	}
}

func (h *S3Handler) HandleUploadFile(c *fiber.Ctx) error {
	// Parse the file from the request
	fileHeader, err := c.FormFile("file")
	if err != nil {
		log.Printf("Failed to retrieve file %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Error al obtener el archivo: %v", err),
		})
	}
	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		log.Printf("Failed to open file %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Error al abrir el archivo: %v", err),
		})
	}
	defer file.Close()

	// Upload file to S3
	url, err := services.UploadFile(h.service, file, fileHeader, true)
	if err != nil {
		log.Printf("Failed to upload file %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Error al subir el archivo: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"url": url,
	})
}
