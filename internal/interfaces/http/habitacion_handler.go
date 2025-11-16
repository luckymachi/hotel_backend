package http

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Maxito7/hotel_backend/internal/application"
	"github.com/Maxito7/hotel_backend/internal/domain"
	"github.com/gofiber/fiber/v2"
)

// Zona horaria de Perú (UTC-5)
var peruLocation *time.Location

func init() {
	var err error
	peruLocation, err = time.LoadLocation("America/Lima")
	if err != nil {
		// Fallback a UTC-5 si no se puede cargar la zona horaria
		peruLocation = time.FixedZone("PET", -5*60*60)
	}
}

// parseDatePeru parsea una fecha en formato YYYY-MM-DD y la retorna en zona horaria de Perú
func parseDatePeru(dateStr string) (time.Time, error) {
	// Parsear en UTC primero y luego convertir a zona horaria de Perú
	utcTime, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, err
	}
	// Crear la fecha en zona horaria de Perú a las 00:00:00
	return time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), 0, 0, 0, 0, peruLocation), nil
}

// getTodayPeru retorna la fecha de hoy a las 00:00:00 en zona horaria de Perú
func getTodayPeru() time.Time {
	now := time.Now().In(peruLocation)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, peruLocation)
}

type HabitacionHandler struct {
	service *application.HabitacionService
}

func NewHabitacionHandler(service *application.HabitacionService) *HabitacionHandler {
	return &HabitacionHandler{
		service: service,
	}
}

type availableRoomsRequest struct {
	FechaEntrada string `json:"fechaEntrada"`
	FechaSalida  string `json:"fechaSalida"`
}

func (h *HabitacionHandler) GetAllRooms(c *fiber.Ctx) error {
	// Get all rooms
	habitaciones, err := h.service.GetAllRooms()
	if err != nil {
		log.Printf("Error getting rooms: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Error al obtener las habitaciones: %v", err),
		})
	}

	return c.JSON(habitaciones)
}

func (h *HabitacionHandler) GetRoomTypes(c *fiber.Ctx) error {
	// Get all room types
	roomTypes, err := h.service.GetRoomTypes()
	if err != nil {
		log.Printf("Error getting room types: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Error al obtener los tipos de habitación: %v", err),
		})
	}

	return c.JSON(roomTypes)
}

// --- Admin endpoints for room types ---
type roomTypeRequest struct {
	Titulo           string             `json:"titulo"`
	Descripcion      string             `json:"descripcion"`
	CapacidadAdultos int                `json:"capacidadAdultos"`
	CapacidadNinhos  int                `json:"capacidadNinhos"`
	CantidadCamas    int                `json:"cantidadCamas"`
	Area             float64            `json:"area"`
	Precio           float64            `json:"precio"`
	AmenityIDs       []int              `json:"amenity_ids,omitempty"`
	Images           []domain.RoomImage `json:"images,omitempty"`
}

type roomRequest struct {
	Nombre             string `json:"nombre"`
	Numero             string `json:"numero"`
	Capacidad          int    `json:"capacidad"`
	Estado             string `json:"estado"`
	DescripcionGeneral string `json:"descripcionGeneral"`
	RoomTypeID         int    `json:"roomTypeId"`
}

func (h *HabitacionHandler) CreateRoomType(c *fiber.Ctx) error {
	var req roomTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	rt := domain.TipoHabitacion{
		Titulo:           req.Titulo,
		Descripcion:      req.Descripcion,
		CapacidadAdultos: req.CapacidadAdultos,
		CapacidadNinhos:  req.CapacidadNinhos,
		CantidadCamas:    req.CantidadCamas,
		Area:             req.Area,
		Precio:           req.Precio,
	}

	newID, err := h.service.CreateRoomType(rt, req.AmenityIDs, req.Images)
	if err != nil {
		log.Printf("CreateRoomType error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "error creating room type"})
	}

	created, err := h.service.GetRoomTypeByID(newID)
	if err != nil {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": newID})
	}
	return c.Status(fiber.StatusCreated).JSON(created)
}

func (h *HabitacionHandler) UpdateRoomType(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req roomTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	rt := domain.TipoHabitacion{
		Titulo:           req.Titulo,
		Descripcion:      req.Descripcion,
		CapacidadAdultos: req.CapacidadAdultos,
		CapacidadNinhos:  req.CapacidadNinhos,
		CantidadCamas:    req.CantidadCamas,
		Area:             req.Area,
		Precio:           req.Precio,
	}

	if err := h.service.UpdateRoomType(id, rt, req.AmenityIDs, req.Images); err != nil {
		log.Printf("UpdateRoomType error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "error updating room type"})
	}

	updated, err := h.service.GetRoomTypeByID(id)
	if err != nil {
		return c.SendStatus(fiber.StatusNoContent)
	}
	return c.JSON(updated)
}

func (h *HabitacionHandler) DeleteRoomType(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.service.DeleteRoomType(id); err != nil {
		log.Printf("DeleteRoomType error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "error deleting room type"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *HabitacionHandler) GetRoomTypeByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	rt, err := h.service.GetRoomTypeByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "room type not found"})
	}
	return c.JSON(rt)
}

func (h *HabitacionHandler) CreateRoom(c *fiber.Ctx) error {
	var req roomRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	room := domain.Habitacion{
		Nombre:             req.Nombre,
		Numero:             req.Numero,
		Capacidad:          req.Capacidad,
		Estado:             req.Estado,
		DescripcionGeneral: req.DescripcionGeneral,
		TipoHabitacion:     domain.TipoHabitacion{ID: req.RoomTypeID},
	}

	newID, err := h.service.CreateRoom(room)
	if err != nil {
		log.Printf("CreateRoom error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "error creating room"})
	}

	created, err := h.service.GetRoomByID(newID)
	if err != nil {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": newID})
	}
	return c.Status(fiber.StatusCreated).JSON(created)
}

func (h *HabitacionHandler) UpdateRoom(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var req roomRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	room := domain.Habitacion{
		Nombre:             req.Nombre,
		Numero:             req.Numero,
		Capacidad:          req.Capacidad,
		Estado:             req.Estado,
		DescripcionGeneral: req.DescripcionGeneral,
		TipoHabitacion:     domain.TipoHabitacion{ID: req.RoomTypeID},
	}

	if err := h.service.UpdateRoom(id, room); err != nil {
		log.Printf("UpdateRoom error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "error updating room"})
	}

	updated, err := h.service.GetRoomByID(id)
	if err != nil {
		return c.SendStatus(fiber.StatusNoContent)
	}
	return c.JSON(updated)
}

func (h *HabitacionHandler) DeleteRoom(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.service.DeleteRoom(id); err != nil {
		log.Printf("DeleteRoom error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "error deleting room"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *HabitacionHandler) GetRoomByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	room, err := h.service.GetRoomByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "room not found"})
	}
	return c.JSON(room)
}

func (h *HabitacionHandler) SetRoomTypeAmenities(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var payload struct {
		AmenityIDs []int `json:"amenity_ids"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if err := h.service.SetAmenitiesForRoomType(id, payload.AmenityIDs); err != nil {
		log.Printf("SetRoomTypeAmenities error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "error setting amenities"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// UpdateRoomTypeImages reemplaza las imágenes de un tipo de habitación (solo imágenes)
func (h *HabitacionHandler) UpdateRoomTypeImages(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var payload struct {
		Images []domain.RoomImage `json:"images"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if err := h.service.SetImagesForRoomType(id, payload.Images); err != nil {
		log.Printf("UpdateRoomTypeImages error: %v", err)
		// Distinguish validation errors
		if strings.HasPrefix(err.Error(), "validation:") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": strings.TrimPrefix(err.Error(), "validation: ")})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "error updating images"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *HabitacionHandler) GetFechasBloqueadas(c *fiber.Ctx) error {
	// Obtener parámetros de consulta
	desdeStr := c.Query("desde")
	hastaStr := c.Query("hasta", "") // Si no se proporciona, usaremos 3 meses por defecto

	// Parsear fecha desde
	desde, err := parseDatePeru(desdeStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid desde format. Use YYYY-MM-DD",
		})
	}

	// Si no se proporciona fecha hasta, usar 3 meses después de desde
	var hasta time.Time
	if hastaStr == "" {
		hasta = desde.AddDate(0, 3, 0) // 3 meses después
	} else {
		hasta, err = parseDatePeru(hastaStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid hasta format. Use YYYY-MM-DD",
			})
		}
	}

	// Validar que desde sea menor o igual que hasta
	if desde.After(hasta) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "desde must be before or equal to hasta",
		})
	}

	// Validar que desde no sea anterior a hoy
	hoy := getTodayPeru()
	if desde.Before(hoy) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "desde cannot be before today",
		})
	}

	// Obtener fechas bloqueadas
	fechasBloqueadas, err := h.service.GetFechasBloqueadas(desde, hasta)
	if err != nil {
		log.Printf("Error en GetFechasBloqueadas: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Error al obtener las fechas bloqueadas: %v", err),
		})
	}

	return c.JSON(fechasBloqueadas)
}

func (h *HabitacionHandler) GetAvailableRooms(c *fiber.Ctx) error {
	// Parse query parameters
	fechaEntradaStr := c.Query("fechaEntrada")
	fechaSalidaStr := c.Query("fechaSalida")
	capacidadAdultosStr := c.Query("capacidadAdultos")
	capacidadNinhosStr := c.Query("capacidadNinhos")

	if fechaEntradaStr == "" || fechaSalidaStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "fechaEntrada and fechaSalida are required",
		})
	}

	if capacidadAdultosStr == "" || capacidadNinhosStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "capacidadAdultos and capacidadNinhos are required",
		})
	}

	// Parse dates
	fechaEntrada, err := parseDatePeru(fechaEntradaStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid fechaEntrada format. Use YYYY-MM-DD",
		})
	}

	fechaSalida, err := parseDatePeru(fechaSalidaStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid fechaSalida format. Use YYYY-MM-DD",
		})
	}

	// Parse capacity parameters
	capacidadAdultos, err := strconv.Atoi(capacidadAdultosStr)
	if err != nil || capacidadAdultos < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid capacidadAdultos. Must be a non-negative integer",
		})
	}

	capacidadNinhos, err := strconv.Atoi(capacidadNinhosStr)
	if err != nil || capacidadNinhos < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid capacidadNinhos. Must be a non-negative integer",
		})
	}

	// Validate dates
	if fechaEntrada.After(fechaSalida) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "fechaEntrada must be before fechaSalida",
		})
	}

	// Validar que fechaEntrada no sea anterior a hoy
	hoy := getTodayPeru()
	if fechaEntrada.Before(hoy) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "fechaEntrada cannot be before today",
		})
	}

	// Get available rooms
	habitaciones, err := h.service.GetAvailableRooms(fechaEntrada, fechaSalida)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener las habitaciones disponibles",
		})
	}

	// Filter rooms by capacity
	habitacionesFiltradas := make([]domain.Habitacion, 0)
	for _, habitacion := range habitaciones {
		if habitacion.TipoHabitacion.CapacidadAdultos >= capacidadAdultos &&
			habitacion.TipoHabitacion.CapacidadNinhos >= capacidadNinhos {
			habitacionesFiltradas = append(habitacionesFiltradas, habitacion)
		}
	}

	return c.JSON(habitacionesFiltradas)
}

// ListAmenities returns all amenities (public)
func (h *HabitacionHandler) ListAmenities(c *fiber.Ctx) error {
	amenities, err := h.service.ListAmenities()
	if err != nil {
		log.Printf("Error listing amenities: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "error fetching amenities"})
	}
	return c.JSON(amenities)
}
