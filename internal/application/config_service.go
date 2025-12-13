package application

import (
	"github.com/Maxito7/hotel_backend/internal/domain"
)

type ConfigService struct {
	repo domain.ConfigurationRepository
}

func NewConfigService(repo domain.ConfigurationRepository) *ConfigService {
	return &ConfigService{repo: repo}
}

func (s *ConfigService) GetConfig(key string) (*domain.HotelConfiguration, error) {
	return s.repo.GetByKey(key)
}

func (s *ConfigService) GetAllConfigs() ([]*domain.HotelConfiguration, error) {
	return s.repo.GetAll()
}

func (s *ConfigService) UpdateConfig(key string, value string) error {
	return s.repo.Update(key, value)
}
