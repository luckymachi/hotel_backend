package repository

import (
	"database/sql"
	"fmt"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

type GalleryRepository struct {
	db *sql.DB
}

func NewGalleryRepository(db *sql.DB) *GalleryRepository {
	return &GalleryRepository{db: db}
}

func (r *GalleryRepository) GetAll() ([]domain.GalleryImage, error) {
	rows, err := r.db.Query(`
		SELECT id, url, alt_text, sort_order, is_active, created_at 
		FROM gallery_images 
		ORDER BY sort_order ASC, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []domain.GalleryImage
	for rows.Next() {
		var img domain.GalleryImage
		if err := rows.Scan(&img.ID, &img.URL, &img.AltText, &img.SortOrder, &img.IsActive, &img.CreatedAt); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, nil
}

func (r *GalleryRepository) Create(image *domain.GalleryImage) error {
	query := `
		INSERT INTO gallery_images (url, alt_text, sort_order, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	return r.db.QueryRow(query, image.URL, image.AltText, image.SortOrder, image.IsActive).Scan(&image.ID, &image.CreatedAt)
}

func (r *GalleryRepository) Delete(id int) error {
	result, err := r.db.Exec("DELETE FROM gallery_images WHERE id = $1", id)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("image not found")
	}
	return nil
}

func (r *GalleryRepository) UpdateOrder(id int, order int) error {
	_, err := r.db.Exec("UPDATE gallery_images SET sort_order = $1 WHERE id = $2", order, id)
	return err
}

func (r *GalleryRepository) Update(image *domain.GalleryImage) error {
	_, err := r.db.Exec("UPDATE gallery_images SET alt_text = $1, sort_order = $2 WHERE id = $3", image.AltText, image.SortOrder, image.ID)
	return err
}
