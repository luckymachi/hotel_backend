package repository

import (
	"database/sql"
	"fmt"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

type configRepository struct {
	db *sql.DB
}

func NewConfigRepository(db *sql.DB) domain.ConfigurationRepository {
	return &configRepository{db: db}
}

func (r *configRepository) GetByKey(key string) (*domain.HotelConfiguration, error) {
	query := `SELECT id, config_key, config_value, description, updated_at 
			  FROM hotel_configuration 
			  WHERE config_key = $1`
	
	var config domain.HotelConfiguration
	err := r.db.QueryRow(query, key).Scan(
		&config.ID, 
		&config.ConfigKey, 
		&config.ConfigValue, 
		&config.Description, 
		&config.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("configuration key not found: %s", key)
		}
		return nil, err
	}

	return &config, nil
}

func (r *configRepository) Update(key string, value string) error {
	query := `UPDATE hotel_configuration 
			  SET config_value = $1, updated_at = NOW() 
			  WHERE config_key = $2`
	
	result, err := r.db.Exec(query, value, key)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no configuration found with key: %s", key)
	}

	return nil
}

func (r *configRepository) GetAll() ([]*domain.HotelConfiguration, error) {
	query := `SELECT id, config_key, config_value, description, updated_at 
	          FROM hotel_configuration 
	          ORDER BY id ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*domain.HotelConfiguration
	for rows.Next() {
		var c domain.HotelConfiguration
		if err := rows.Scan(&c.ID, &c.ConfigKey, &c.ConfigValue, &c.Description, &c.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, &c)
	}

	return configs, nil
}
