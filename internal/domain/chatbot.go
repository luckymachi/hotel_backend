package domain

import "time"

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Message        string       `json:"message"`
	ConversationID *string      `json:"conversationId,omitempty"`
	ClienteID      *int         `json:"clienteId,omitempty"`
	Context        *ChatContext `json:"context,omitempty"`
	// UseWeb: nil = auto (service decides), true = force web search, false = disable web search
	UseWeb *bool `json:"useWeb,omitempty"`
}

type ChatContext struct {
	FechaEntrada    *string `json:"fechaEntrada,omitempty"`
	FechaSalida     *string `json:"fechaSalida,omitempty"`
	CantidadAdultos *int    `json:"cantidadAdultos,omitempty"`
	CantidadNinhos  *int    `json:"cantidadNinhos,omitempty"`
}

// ReservationInProgress representa una reserva en progreso durante la conversación
type ReservationInProgress struct {
	FechaEntrada     *string            `json:"fechaEntrada,omitempty"`
	FechaSalida      *string            `json:"fechaSalida,omitempty"`
	CantidadAdultos  *int               `json:"cantidadAdultos,omitempty"`
	CantidadNinhos   *int               `json:"cantidadNinhos,omitempty"`
	TipoHabitacionID *int               `json:"tipoHabitacionId,omitempty"`
	PersonalData     *PersonalDataInput `json:"personalData,omitempty"`
	PrecioCalculado  *float64           `json:"precioCalculado,omitempty"`
	Step             string             `json:"step"` // dates, guests, room_type, personal_data, confirmation, completed
}

// PersonalDataInput representa los datos personales para crear una reserva
type PersonalDataInput struct {
	Nombre           string  `json:"nombre"`
	PrimerApellido   string  `json:"primerApellido"`
	SegundoApellido  *string `json:"segundoApellido,omitempty"`
	NumeroDocumento  string  `json:"numeroDocumento"`
	Genero           string  `json:"genero"`
	Correo           string  `json:"correo"`
	Telefono1        string  `json:"telefono1"`
	Telefono2        *string `json:"telefono2,omitempty"`
	CiudadReferencia *string `json:"ciudadReferencia,omitempty"`
	PaisReferencia   *string `json:"paisReferencia,omitempty"`
}

type ChatResponse struct {
	Message              string                 `json:"message"`
	ConversationID       string                 `json:"conversationId"`
	SuggestedActions     []string               `json:"suggestedActions,omitempty"`
	RequiresHuman        bool                   `json:"requiresHuman"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	ReservationInProgress *ReservationInProgress `json:"reservationInProgress,omitempty"`
	ReservationCreated   *int                   `json:"reservationCreated,omitempty"` // ID de reserva si fue creada
}

type ConversationHistory struct {
	ID                    string                 `json:"id"`
	ClienteID             *int                   `json:"clienteId,omitempty"`
	Messages              []ChatMessage          `json:"messages"`
	CreatedAt             time.Time              `json:"createdAt"`
	UpdatedAt             time.Time              `json:"updatedAt"`
	ReservationInProgress *ReservationInProgress `json:"reservationInProgress,omitempty"`
}

type ChatbotRepository interface {
	SaveConversation(conversation *ConversationHistory) error
	GetConversation(conversationID string) (*ConversationHistory, error)
	UpdateConversation(conversation *ConversationHistory) error
	SaveMessage(clienteID int, contenido string) error
	GetClientConversations(clienteID int) ([]ConversationHistory, error)
}
