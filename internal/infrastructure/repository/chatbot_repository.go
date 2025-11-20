package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Maxito7/hotel_backend/internal/domain"
	"github.com/google/uuid"
)

type chatbotRepository struct {
	db *sql.DB
}

func NewChatbotRepository(db *sql.DB) domain.ChatbotRepository {
	return &chatbotRepository{db: db}
}

func (r *chatbotRepository) SaveConversation(conversation *domain.ConversationHistory) error {
	// Generamos ID si no existe
	if conversation.ID == "" {
		conversation.ID = uuid.New().String()
	}

	messagesJSON, err := json.Marshal(conversation.Messages)
	if err != nil {
		return fmt.Errorf("error marshaling messages: %w", err)
	}

	// Marshal reservation state
	var reservationStateJSON []byte
	if conversation.ReservationInProgress != nil {
		reservationStateJSON, err = json.Marshal(conversation.ReservationInProgress)
		if err != nil {
			return fmt.Errorf("error marshaling reservation state: %w", err)
		}
	}

	// Try to insert with reservation_state column (if migration was applied)
	query := `
		INSERT INTO conversation_history
		(id, client_id, messages, created_at, updated_at, reservation_state)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = r.db.Exec(query,
		conversation.ID,
		conversation.ClienteID,
		messagesJSON,
		conversation.CreatedAt,
		conversation.UpdatedAt,
		reservationStateJSON,
	)

	// If column doesn't exist, fallback to old schema
	if err != nil && (err.Error() == "column \"reservation_state\" of relation \"conversation_history\" does not exist" ||
		err.Error() == "pq: column \"reservation_state\" of relation \"conversation_history\" does not exist") {
		queryOld := `
			INSERT INTO conversation_history
			(id, client_id, messages, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5)
		`
		_, err = r.db.Exec(queryOld,
			conversation.ID,
			conversation.ClienteID,
			messagesJSON,
			conversation.CreatedAt,
			conversation.UpdatedAt,
		)
	}

	return err
}

func (r *chatbotRepository) GetConversation(conversationID string) (*domain.ConversationHistory, error) {
	// Try to query with reservation_state column first
	query := `
		SELECT id, client_id, messages, created_at, updated_at, reservation_state
		FROM conversation_history
		WHERE id = $1
	`

	var conversation domain.ConversationHistory
	var messagesJSON []byte
	var reservationStateJSON []byte
	var clienteID sql.NullInt64

	err := r.db.QueryRow(query, conversationID).Scan(
		&conversation.ID,
		&clienteID,
		&messagesJSON,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
		&reservationStateJSON,
	)

	// If column doesn't exist, fallback to old schema
	if err != nil && (err.Error() == "column \"reservation_state\" does not exist" ||
		err.Error() == "pq: column \"reservation_state\" does not exist") {
		queryOld := `
			SELECT id, client_id, messages, created_at, updated_at
			FROM conversation_history
			WHERE id = $1
		`
		err = r.db.QueryRow(queryOld, conversationID).Scan(
			&conversation.ID,
			&clienteID,
			&messagesJSON,
			&conversation.CreatedAt,
			&conversation.UpdatedAt,
		)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if clienteID.Valid {
		id := int(clienteID.Int64)
		conversation.ClienteID = &id
	}

	if err := json.Unmarshal(messagesJSON, &conversation.Messages); err != nil {
		return nil, fmt.Errorf("error unmarshaling messages: %w", err)
	}

	// Unmarshal reservation state if exists
	if len(reservationStateJSON) > 0 {
		var reservationState domain.ReservationInProgress
		if err := json.Unmarshal(reservationStateJSON, &reservationState); err != nil {
			return nil, fmt.Errorf("error unmarshaling reservation state: %w", err)
		}
		conversation.ReservationInProgress = &reservationState
	}

	return &conversation, nil
}

func (r *chatbotRepository) UpdateConversation(conversation *domain.ConversationHistory) error {
	messagesJSON, err := json.Marshal(conversation.Messages)
	if err != nil {
		return fmt.Errorf("error marshaling messages: %w", err)
	}

	// Marshal reservation state
	var reservationStateJSON []byte
	if conversation.ReservationInProgress != nil {
		reservationStateJSON, err = json.Marshal(conversation.ReservationInProgress)
		if err != nil {
			return fmt.Errorf("error marshaling reservation state: %w", err)
		}
	}

	// Try to update with reservation_state column
	query := `
		UPDATE conversation_history
		SET messages = $1, updated_at = $2, client_id = $3, reservation_state = $4
		WHERE id = $5
	`

	_, err = r.db.Exec(query,
		messagesJSON,
		time.Now(),
		conversation.ClienteID,
		reservationStateJSON,
		conversation.ID,
	)

	// If column doesn't exist, fallback to old schema
	if err != nil && (err.Error() == "column \"reservation_state\" of relation \"conversation_history\" does not exist" ||
		err.Error() == "pq: column \"reservation_state\" of relation \"conversation_history\" does not exist") {
		queryOld := `
			UPDATE conversation_history
			SET messages = $1, updated_at = $2, client_id = $3
			WHERE id = $4
		`
		_, err = r.db.Exec(queryOld,
			messagesJSON,
			time.Now(),
			conversation.ClienteID,
			conversation.ID,
		)
	}

	return err
}

func (r *chatbotRepository) SaveMessage(clienteID int, contenido string) error {
	query := `
		INSERT INTO message (content, client_id, registration_date) 
		VALUES ($1, $2, $3)
	`

	_, err := r.db.Exec(query, contenido, clienteID, time.Now())
	return err
}

func (r *chatbotRepository) GetClientConversations(clienteID int) ([]domain.ConversationHistory, error) {
	// Try to query with reservation_state column first
	query := `
		SELECT id, client_id, messages, created_at, updated_at, reservation_state
		FROM conversation_history
		WHERE client_id = $1
		ORDER BY updated_at DESC
	`

	rows, err := r.db.Query(query, clienteID)

	// If column doesn't exist, fallback to old schema
	if err != nil && (err.Error() == "column \"reservation_state\" does not exist" ||
		err.Error() == "pq: column \"reservation_state\" does not exist") {
		queryOld := `
			SELECT id, client_id, messages, created_at, updated_at
			FROM conversation_history
			WHERE client_id = $1
			ORDER BY updated_at DESC
		`
		rows, err = r.db.Query(queryOld, clienteID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []domain.ConversationHistory
	for rows.Next() {
		var conv domain.ConversationHistory
		var messagesJSON []byte
		var reservationStateJSON []byte
		var cID sql.NullInt64

		// Try scanning with reservation_state
		err := rows.Scan(
			&conv.ID,
			&cID,
			&messagesJSON,
			&conv.CreatedAt,
			&conv.UpdatedAt,
			&reservationStateJSON,
		)

		// If we get column count error, it means we're using old schema
		if err != nil {
			err = rows.Scan(
				&conv.ID,
				&cID,
				&messagesJSON,
				&conv.CreatedAt,
				&conv.UpdatedAt,
			)
			if err != nil {
				return nil, err
			}
		}

		if cID.Valid {
			id := int(cID.Int64)
			conv.ClienteID = &id
		}

		if err := json.Unmarshal(messagesJSON, &conv.Messages); err != nil {
			return nil, fmt.Errorf("error unmarshaling messages: %w", err)
		}

		// Unmarshal reservation state if exists
		if len(reservationStateJSON) > 0 {
			var reservationState domain.ReservationInProgress
			if err := json.Unmarshal(reservationStateJSON, &reservationState); err != nil {
				return nil, fmt.Errorf("error unmarshaling reservation state: %w", err)
			}
			conv.ReservationInProgress = &reservationState
		}

		conversations = append(conversations, conv)
	}

	return conversations, nil
}
