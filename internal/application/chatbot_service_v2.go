package application

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Maxito7/hotel_backend/internal/domain"
	"github.com/Maxito7/hotel_backend/internal/openai"
	"github.com/google/uuid"
)

// ProcessMessageV2 es una versión mejorada que usa detección automática de intenciones
func (s *ChatbotService) ProcessMessageV2(req domain.ChatRequest) (*domain.ChatResponse, error) {
	log.Printf("[ChatbotV2] Processing message from client %v: %s", req.ClienteID, req.Message)

	// 1. Obtener o crear historial de conversación
	var conversation *domain.ConversationHistory
	var err error

	if req.ConversationID != nil && *req.ConversationID != "" {
		conversation, err = s.repo.GetConversation(*req.ConversationID)
		if err != nil {
			return nil, fmt.Errorf("error getting conversation: %w", err)
		}
	}

	if conversation == nil {
		conversation = &domain.ConversationHistory{
			ID:                    uuid.New().String(),
			ClienteID:             req.ClienteID,
			Messages:              []domain.ChatMessage{},
			CreatedAt:             time.Now(),
			UpdatedAt:             time.Now(),
			ReservationInProgress: nil,
		}
	}

	// 2. Detectar intención automáticamente y ejecutar herramientas
	detected, err := s.intentDetector.DetectAndProcess(req.Message, conversation.ReservationInProgress)
	if err != nil {
		log.Printf("[ChatbotV2] Error detecting intent: %v", err)
		// Continuar de todos modos
	}

	log.Printf("[ChatbotV2] Detected intent: %s, tools executed: %d", detected.Intent, len(detected.ToolResults))

	// 3. Actualizar estado de reserva con información detectada
	if conversation.ReservationInProgress == nil && detected.Intent != "" {
		conversation.ReservationInProgress = &domain.ReservationInProgress{
			Step: "dates",
		}
	}

	if conversation.ReservationInProgress != nil {
		s.updateReservationWithDetected(conversation.ReservationInProgress, detected)
	}

	// 4. Agregar mensaje del usuario
	conversation.Messages = append(conversation.Messages, domain.ChatMessage{
		Role:    "user",
		Content: req.Message,
	})

	// 5. Preparar contexto con resultados de herramientas
	var toolResultsContext string
	for _, result := range detected.ToolResults {
		if result.Error != nil {
			toolResultsContext += fmt.Sprintf("\n[ERROR EN %s]: %s\n", strings.ToUpper(result.ToolName), result.Error.Error())
		} else {
			toolResultsContext += fmt.Sprintf("\n[RESULTADO DE %s]:\n%s\n", strings.ToUpper(result.ToolName), result.Result)
		}
	}

	// 6. Obtener información del hotel
	hotelInfo, err := s.getHotelInfo(req)
	if err != nil {
		return nil, fmt.Errorf("error getting hotel info: %w", err)
	}

	// 7. Preparar prompt del sistema simplificado (sin tool descriptions)
	reservationContext := s.buildReservationContext(conversation.ReservationInProgress)
	systemPrompt := s.buildSimplifiedSystemPrompt(hotelInfo + reservationContext + toolResultsContext)

	// 8. Construir mensajes para el LLM
	messages := []openai.Message{
		{Role: "system", Content: systemPrompt},
	}

	// Agregar historial reciente (últimos 6 mensajes)
	startIdx := 0
	if len(conversation.Messages) > 6 {
		startIdx = len(conversation.Messages) - 6
	}

	for _, msg := range conversation.Messages[startIdx:] {
		messages = append(messages, openai.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// 9. Llamar al LLM
	openaiReq := openai.ChatCompletionRequest{
		Model:       "llama-3.1-8b-instant",
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   500,
	}

	openaiResp, err := s.openaiClient.CreateChatCompletion(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("error calling LLM: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	assistantMessage := openaiResp.Choices[0].Message.Content

	// 10. Agregar respuesta del asistente
	conversation.Messages = append(conversation.Messages, domain.ChatMessage{
		Role:    "assistant",
		Content: assistantMessage,
	})

	// 11. Guardar conversación
	conversation.UpdatedAt = time.Now()
	if req.ConversationID == nil || *req.ConversationID == "" {
		err = s.repo.SaveConversation(conversation)
	} else {
		err = s.repo.UpdateConversation(conversation)
	}

	if err != nil {
		return nil, fmt.Errorf("error saving conversation: %w", err)
	}

	// 12. Guardar mensaje si el cliente está identificado
	if req.ClienteID != nil {
		_ = s.repo.SaveMessage(*req.ClienteID, req.Message)
	}

	// 13. Analizar si requiere intervención humana
	requiresHuman := s.detectHumanRequired(req.Message, assistantMessage)

	// 14. Generar acciones sugeridas
	suggestedActions := s.generateSuggestedActionsV2(conversation.ReservationInProgress, detected)

	// 15. Preparar metadata
	sources := []string{"hotel"}
	if len(detected.ToolResults) > 0 {
		sources = append(sources, "automated_tools")
	}

	metadata := map[string]interface{}{
		"tokensUsed":     openaiResp.Usage.TotalTokens,
		"sources":        sources,
		"toolsExecuted":  len(detected.ToolResults),
		"intentDetected": detected.Intent,
	}

	// 16. Verificar si se creó una reserva
	var reservaCreada *int
	for _, result := range detected.ToolResults {
		if result.ToolName == "create_reservation" && result.Error == nil {
			// Extraer ID de reserva del resultado
			if strings.Contains(result.Result, "Número de Reserva: #") {
				var id int
				if _, err := fmt.Sscanf(result.Result, "✅ Reserva creada exitosamente!\n\nNúmero de Reserva: #%d", &id); err == nil {
					reservaCreada = &id
					// Limpiar estado de reserva
					conversation.ReservationInProgress = nil
					// Actualizar conversación con estado limpio
					_ = s.repo.UpdateConversation(conversation)
				}
			}
		}
	}

	log.Printf("[ChatbotV2] Response generated, reservation created: %v", reservaCreada != nil)

	return &domain.ChatResponse{
		Message:              assistantMessage,
		ConversationID:       conversation.ID,
		SuggestedActions:     suggestedActions,
		RequiresHuman:        requiresHuman,
		Metadata:             metadata,
		ReservationInProgress: conversation.ReservationInProgress,
		ReservationCreated:   reservaCreada,
	}, nil
}

// updateReservationWithDetected actualiza el estado de reserva con información detectada
func (s *ChatbotService) updateReservationWithDetected(reservation *domain.ReservationInProgress, detected *DetectedIntent) {
	if detected.FechaEntrada != nil {
		reservation.FechaEntrada = detected.FechaEntrada
	}
	if detected.FechaSalida != nil {
		reservation.FechaSalida = detected.FechaSalida
	}
	if detected.CantidadAdultos != nil {
		reservation.CantidadAdultos = detected.CantidadAdultos
	}
	if detected.CantidadNinhos != nil {
		reservation.CantidadNinhos = detected.CantidadNinhos
	}
	if detected.TipoHabitacionID != nil {
		reservation.TipoHabitacionID = detected.TipoHabitacionID
	}
	if detected.PersonalData != nil {
		reservation.PersonalData = detected.PersonalData
	}

	// Actualizar el paso según la información disponible
	if reservation.FechaEntrada != nil && reservation.FechaSalida != nil && reservation.Step == "dates" {
		reservation.Step = "guests"
	}
	if reservation.CantidadAdultos != nil && reservation.Step == "guests" {
		reservation.Step = "room_type"
	}
	if reservation.TipoHabitacionID != nil && reservation.Step == "room_type" {
		reservation.Step = "personal_data"
	}
	if reservation.PersonalData != nil && reservation.Step == "personal_data" {
		reservation.Step = "confirmation"
	}
}

// buildSimplifiedSystemPrompt crea un prompt más simple que no depende de tool calling
func (s *ChatbotService) buildSimplifiedSystemPrompt(contextInfo string) string {
	basePrompt := `Eres un asistente virtual amable y profesional de un hotel en Lima, Perú.

IMPORTANTE:
- Toda la información que necesitas está en el contexto proporcionado
- NO inventes información que no esté en el contexto
- Si se te proporcionan RESULTADOS DE HERRAMIENTAS, úsalos para responder
- Sé conciso, amable y profesional
- Ayuda al usuario a completar su reserva paso a paso

FLUJO DE RESERVA:
1. Fechas de entrada y salida
2. Cantidad de huéspedes (adultos y niños)
3. Tipo de habitación (se verificará disponibilidad automáticamente)
4. Datos personales (nombre, email, teléfono, documento)
5. Confirmación

INSTRUCCIONES:
- Si ves resultados de CHECK_AVAILABILITY, informa al usuario sobre la disponibilidad
- Si ves resultados de CALCULATE_PRICE, menciona el precio total
- Si ves resultados de CREATE_RESERVATION, confirma que la reserva fue creada
- Guía al usuario según el paso actual de la reserva
- No pidas información que ya tienes en el contexto
- Sé directo y claro

`

	return basePrompt + contextInfo
}

// generateSuggestedActionsV2 genera acciones sugeridas basadas en el estado actual
func (s *ChatbotService) generateSuggestedActionsV2(reservation *domain.ReservationInProgress, detected *DetectedIntent) []string {
	actions := []string{}

	if reservation == nil {
		return []string{"Ver habitaciones disponibles", "Iniciar una reserva"}
	}

	switch reservation.Step {
	case "dates":
		actions = []string{"Ver habitaciones disponibles", "Consultar disponibilidad"}
	case "guests":
		actions = []string{"Ver tipos de habitaciones", "Consultar precios"}
	case "room_type":
		actions = []string{"Ver detalles de habitaciones", "Calcular precio"}
	case "personal_data":
		actions = []string{"Completar reserva"}
	case "confirmation":
		actions = []string{"Confirmar reserva", "Modificar datos"}
	default:
		actions = []string{"Continuar con la reserva"}
	}

	return actions
}
