package application

import "github.com/Maxito7/hotel_backend/internal/domain"

type GalleryService struct {
	repo domain.GalleryRepository
}

func NewGalleryService(repo domain.GalleryRepository) *GalleryService {
	return &GalleryService{repo: repo}
}

func (s *GalleryService) GetAllImages() ([]domain.GalleryImage, error) {
	return s.repo.GetAll()
}

func (s *GalleryService) AddImage(url string, altText string, sortOrder int) (*domain.GalleryImage, error) {
	img := &domain.GalleryImage{
		URL:       url,
		AltText:   altText,
		SortOrder: sortOrder,
		IsActive:  true,
	}
	err := s.repo.Create(img)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (s *GalleryService) DeleteImage(id int) error {
	return s.repo.Delete(id)
}

func (s *GalleryService) UpdateImage(id int, altText string, sortOrder int) error {
	img := &domain.GalleryImage{
		ID:        id,
		AltText:   altText,
		SortOrder: sortOrder,
	}
	return s.repo.Update(img)
}
