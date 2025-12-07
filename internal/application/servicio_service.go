package application

import "github.com/Maxito7/hotel_backend/internal/domain"

type ServicioService struct {
	repo domain.ServicioRepository
}

func NewServicioService(repo domain.ServicioRepository) *ServicioService {
	return &ServicioService{
		repo: repo,
	}
}
func (s *ServicioService) CreateService(servicio *domain.Servicio) error {
	return s.repo.CreateService(servicio)
}

func (s *ServicioService) UpdateService(servicio *domain.Servicio) error {
	return s.repo.UpdateService(servicio)
}

func (s *ServicioService) DeleteService(id int) error {
	return s.repo.DeleteService(id)
}

func (s *ServicioService) GetAllServices() ([]domain.Servicio, error) {
	// Ya retorna todos los campos, incluyendo icon_key y status
	return s.repo.GetAllServices()
}
