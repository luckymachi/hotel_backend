package domain

import "time"

type HotelConfiguration struct {
	ID          int       `json:"id"`
	ConfigKey   string    `json:"config_key"`
	ConfigValue string    `json:"config_value"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ConfigurationRepository interface {
	GetByKey(key string) (*HotelConfiguration, error)
	Update(key string, value string) error
	GetAll() ([]*HotelConfiguration, error)
}
