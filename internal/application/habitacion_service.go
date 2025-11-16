package application

import (
	"fmt"
	"strings"
	"time"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

type HabitacionService struct {
	repo domain.HabitacionRepository
}

func NewHabitacionService(repo domain.HabitacionRepository) *HabitacionService {
	return &HabitacionService{
		repo: repo,
	}
}

func (s *HabitacionService) GetAllRooms() ([]domain.Habitacion, error) {
	return s.repo.GetAllRooms()
}

func (s *HabitacionService) GetAvailableRooms(fechaEntrada, fechaSalida time.Time) ([]domain.TipoHabitacion, error) {
	return s.repo.GetAvailableRooms(fechaEntrada, fechaSalida)
}

func (s *HabitacionService) GetFechasBloqueadas(desde, hasta time.Time) (*domain.FechasBloqueadas, error) {
	return s.repo.GetFechasBloqueadas(desde, hasta)
}

func (s *HabitacionService) GetRoomTypes() ([]domain.TipoHabitacion, error) {
	return s.repo.GetRoomTypes()
}

// Room type management
func (s *HabitacionService) CreateRoomType(rt domain.TipoHabitacion, amenityIDs []int, images []domain.RoomImage) (int, error) {
	return s.repo.CreateRoomType(rt, amenityIDs, images)
}

func (s *HabitacionService) UpdateRoomType(id int, rt domain.TipoHabitacion, amenityIDs []int, images []domain.RoomImage) error {
	return s.repo.UpdateRoomType(id, rt, amenityIDs, images)
}

func (s *HabitacionService) DeleteRoomType(id int) error {
	return s.repo.DeleteRoomType(id)
}

func (s *HabitacionService) GetRoomTypeByID(id int) (domain.TipoHabitacion, error) {
	return s.repo.GetRoomTypeByID(id)
}

// Room CRUD
func (s *HabitacionService) CreateRoom(h domain.Habitacion) (int, error) {
	return s.repo.CreateRoom(h)
}

func (s *HabitacionService) UpdateRoom(id int, h domain.Habitacion) error {
	return s.repo.UpdateRoom(id, h)
}

func (s *HabitacionService) DeleteRoom(id int) error {
	return s.repo.DeleteRoom(id)
}

func (s *HabitacionService) GetRoomByID(id int) (domain.Habitacion, error) {
	return s.repo.GetRoomByID(id)
}

func (s *HabitacionService) SetAmenitiesForRoomType(roomTypeID int, amenityIDs []int) error {
	return s.repo.SetAmenitiesForRoomType(roomTypeID, amenityIDs)
}

func (s *HabitacionService) SetImagesForRoomType(roomTypeID int, images []domain.RoomImage) error {
	// Basic validations:
	// - url must be present
	// - sortOrder must be >= 0
	// - at most one image marked as primary
	primaryCount := 0
	for i, img := range images {
		if strings.TrimSpace(img.URL) == "" {
			return fmt.Errorf("validation: image at index %d missing url", i)
		}
		if img.SortOrder < 0 {
			return fmt.Errorf("validation: image at index %d has invalid sortOrder", i)
		}
		if img.IsPrimary {
			primaryCount++
		}
		// normalize alt text
		images[i].AltText = strings.TrimSpace(img.AltText)
	}
	if primaryCount > 1 {
		return fmt.Errorf("validation: more than one primary image provided")
	}

	return s.repo.SetImagesForRoomType(roomTypeID, images)
}

// ListAmenities returns all amenities
func (s *HabitacionService) ListAmenities() ([]domain.Amenity, error) {
	return s.repo.ListAmenities()
}
