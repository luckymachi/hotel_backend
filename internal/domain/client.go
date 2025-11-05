package domain

// Client representa un cliente del hotel
type Client struct {
	ClientID int `json:"clientId"`
	PersonID int `json:"personId"`
}

// ClientRepository define las operaciones con clientes
type ClientRepository interface {
	// GetClientIDByPersonID obtiene el client_id dado un person_id
	GetClientIDByPersonID(personID int) (int, error)
	// Create crea un nuevo cliente
	Create(personID int) (int, error)
	// GetPersonEmailByClientID obtiene el email de la persona asociada a un cliente
	GetPersonEmailByClientID(clientID int) (string, error)
}
