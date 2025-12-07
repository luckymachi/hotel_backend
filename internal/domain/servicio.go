package domain

// Servicio representa un servicio del hotel
type Servicio struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	IconKey     string  `json:"icon_key"`
	Status      int     `json:"status"`
}

// ServicioRepository define la interfaz para operaciones de datos de servicios
type ServicioRepository interface {
	// GetAllServices retorna todos los servicios disponibles
	GetAllServices() ([]Servicio, error)
	// CreateService crea un nuevo servicio
	CreateService(servicio *Servicio) error
	// UpdateService actualiza un servicio existente
	UpdateService(servicio *Servicio) error
	// DeleteService realiza una eliminación lógica (status=0)
	DeleteService(id int) error
}
