package domain

import "time"

type GalleryImage struct {
	ID        int       `json:"id"`
	URL       string    `json:"url"`
	AltText   string    `json:"alt_text"`
	SortOrder int       `json:"sort_order"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type GalleryRepository interface {
	GetAll() ([]GalleryImage, error)
	Create(image *GalleryImage) error
	Delete(id int) error
	Update(image *GalleryImage) error
	UpdateOrder(id int, order int) error
}
