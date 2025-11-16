package application

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Maxito7/hotel_backend/internal/domain"
	"github.com/Maxito7/hotel_backend/internal/openai"
	"github.com/Maxito7/hotel_backend/internal/tavily"
	"github.com/google/uuid"
)

type ChatbotService struct {
	repo           domain.ChatbotRepository
	openaiClient   *openai.Client
	habitacionRepo domain.HabitacionRepository
	tavilyClient   *tavily.Client
	searchService  *SearchService
	location       string
}

func NewChatbotService(
	repo domain.ChatbotRepository,
	openaiClient *openai.Client,
	habitacionRepo domain.HabitacionRepository,
	tavilyClient *tavily.Client,
	location string,
	searchService *SearchService,
) *ChatbotService {
	return &ChatbotService{
		repo:           repo,
		openaiClient:   openaiClient,
		habitacionRepo: habitacionRepo,
		tavilyClient:   tavilyClient,
		searchService:  searchService,
		location:       location,
	}
}

func (s *ChatbotService) ProcessMessage(req domain.ChatRequest) (*domain.ChatResponse, error) {
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
			ID:        uuid.New().String(),
			ClienteID: req.ClienteID,
			Messages:  []domain.ChatMessage{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// 2. Agregar mensaje del usuario
	conversation.Messages = append(conversation.Messages, domain.ChatMessage{
		Role:    "user",
		Content: req.Message,
	})

	var webContext string
	useWeb := false
	var tavilyResp *tavily.SearchResponse

	// Decidir si usar búsqueda web: prioridad al flag del request (UseWeb)
	if req.UseWeb != nil {
		useWeb = *req.UseWeb
	} else {
		useWeb = s.shouldSearchWeb(req.Message)
	}

	if useWeb {
		// Preferir SearchService si está disponible
		log.Printf("Chatbot: performing web search message (near %s), useWeb=%v", s.location, useWeb)
		// Construir query con ubicación para focalizar resultados locales
		query := req.Message
		if s.location != "" {
			query = fmt.Sprintf("%s near %s", req.Message, s.location)
		}

		if s.searchService != nil {
			input := SearchInput{Query: query, MaxResults: 3}
			if resp, err := s.searchService.SearchWeb(input); err == nil {
				tavilyResp = resp
				webContext = s.formatWebResults(resp)
			} else {
				log.Printf("searchService error: %v", err)
			}
		} else if s.tavilyClient != nil {
			if resp, err := s.tavilyClient.Search(tavily.SearchRequest{Query: query, MaxResults: 3}); err == nil {
				tavilyResp = resp
				webContext = s.formatWebResults(resp)
			} else {
				log.Printf("tavily search error: %v", err)
			}
		}
	}

	// 3. Obtener información real del hotel desde la BD
	hotelInfo, err := s.getHotelInfo(req)
	if err != nil {
		return nil, fmt.Errorf("error getting hotel info: %w", err)
	}

	// 4. Preparar contexto con información real (incluye resultados web si hay)
	systemPrompt := s.buildSystemPrompt(req.Context, hotelInfo+webContext)

	// 5. Construir mensajes para OpenAI/Groq
	messages := []openai.Message{
		{Role: "system", Content: systemPrompt},
	}

	// Agregar historial (últimos 10 mensajes para no exceder tokens)
	startIdx := 0
	if len(conversation.Messages) > 10 {
		startIdx = len(conversation.Messages) - 10
	}

	for _, msg := range conversation.Messages[startIdx:] {
		messages = append(messages, openai.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// 6. Llamar a OpenAI/Groq
	openaiReq := openai.ChatCompletionRequest{
		Model:       "llama-3.1-8b-instant", // Modelo de Groq
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   500,
	}

	openaiResp, err := s.openaiClient.CreateChatCompletion(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("error calling OpenAI: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	assistantMessage := openaiResp.Choices[0].Message.Content

	// 7. Agregar respuesta del asistente
	conversation.Messages = append(conversation.Messages, domain.ChatMessage{
		Role:    "assistant",
		Content: assistantMessage,
	})

	// 8. Guardar conversación
	conversation.UpdatedAt = time.Now()
	if req.ConversationID == nil || *req.ConversationID == "" {
		err = s.repo.SaveConversation(conversation)
	} else {
		err = s.repo.UpdateConversation(conversation)
	}

	if err != nil {
		return nil, fmt.Errorf("error saving conversation: %w", err)
	}

	// 9. Si el cliente está identificado, guardar en tabla mensaje
	if req.ClienteID != nil {
		_ = s.repo.SaveMessage(*req.ClienteID, req.Message)
	}

	// 10. Analizar si requiere intervención humana
	requiresHuman := s.detectHumanRequired(req.Message, assistantMessage)

	// 11. Generar acciones sugeridas
	suggestedActions := s.generateSuggestedActions(req.Message, assistantMessage)

	sources := []string{"hotel"}
	if useWeb {
		sources = append(sources, "web")
	}

	metadata := map[string]interface{}{
		"tokensUsed": openaiResp.Usage.TotalTokens,
		"sources":    sources,
	}
	// incluir resultados web crudos para uso del frontend (si existen)
	if tavilyResp != nil {
		metadata["webResults"] = tavilyResp
	}

	return &domain.ChatResponse{
		Message:          assistantMessage,
		ConversationID:   conversation.ID,
		SuggestedActions: suggestedActions,
		RequiresHuman:    requiresHuman,
		Metadata:         metadata,
	}, nil
}

func (s *ChatbotService) shouldSearchWeb(message string) bool {
	messageLower := strings.ToLower(message)

	webKeywords := []string{
		"clima", "weather", "temperatura",
		"restaurantes cerca", "donde comer", "dónde comer",
		"atracciones", "lugares para visitar", "que hacer", "qué hacer", "que visitar",
		"eventos", "festivales",
		"transporte", "como llegar", "cómo llegar", "taxi", "bus", "uber", "metropolitano",
		"aeropuerto", "vuelo", "flight", "terminal",
		"noticias", "actualidad",
	}

	for _, keyword := range webKeywords {
		if strings.Contains(messageLower, keyword) {
			return true
		}
	}

	return false
}

func (s *ChatbotService) searchWeb(query string) (string, error) {
	fullQuery := query
	if s.location != "" {
		fullQuery += " near " + s.location
	}
	req := tavily.SearchRequest{
		Query:      fullQuery,
		MaxResults: 3,
	}

	resp, err := s.tavilyClient.Search(req)
	if err != nil {
		return "", err
	}
	return s.formatWebResults(resp), nil
}

func (s *ChatbotService) formatWebResults(resp *tavily.SearchResponse) string {
	if resp == nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("\n===INFORMACIÓN DE LA WEB (BÚSQUEDA EXTERNA) ===\n")
	sb.WriteString(fmt.Sprintf("Consulta: %s\n\n", resp.Query))

	for i, r := range resp.Results {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Title))
		if r.Content != "" {
			content := r.Content
			if len(content) > 300 {
				content = content[:300] + "..."
			}
			sb.WriteString(fmt.Sprintf("   Contenido: %s\n", content))
		}
		if r.URL != "" {
			sb.WriteString(fmt.Sprintf("  Fuente: %s\n", r.URL))
		}
		sb.WriteString("\n")
		if i >= 2 {
			break
		}
	}
	sb.WriteString("==== FIN INFORMACIÓN WEB ====\n\n")
	return sb.String()
}

// Nueva función para obtener información real del hotel
func (s *ChatbotService) getHotelInfo(req domain.ChatRequest) (string, error) {
	var info strings.Builder

	// 1. Obtener todos los tipos de habitaciones disponibles
	habitaciones, err := s.habitacionRepo.GetAllRooms()
	if err != nil {
		return "", err
	}

	// Agrupar por tipo de habitación
	tiposMap := make(map[string]domain.TipoHabitacion)
	for _, hab := range habitaciones {
		if _, exists := tiposMap[hab.TipoHabitacion.Titulo]; !exists {
			tiposMap[hab.TipoHabitacion.Titulo] = hab.TipoHabitacion
		}
	}

	info.WriteString("\n=== INFORMACIÓN REAL DEL HOTEL ===\n\n")
	info.WriteString("TIPOS DE HABITACIONES DISPONIBLES:\n")

	for titulo, tipo := range tiposMap {
		info.WriteString(fmt.Sprintf("\n• %s:\n", titulo))
		info.WriteString(fmt.Sprintf("  - Precio: S/%.2f por noche\n", tipo.Precio))
		info.WriteString(fmt.Sprintf("  - Capacidad: %d adultos, %d niños\n",
			tipo.CapacidadAdultos, tipo.CapacidadNinhos))
		info.WriteString(fmt.Sprintf("  - Camas: %d\n", tipo.CantidadCamas))
		info.WriteString(fmt.Sprintf("  - Descripción: %s\n", tipo.Descripcion))
	}
	info.WriteString("\nDISPONIBILIDAD:\n")
	fechasInicio := time.Now()
	fechasFin := fechasInicio.AddDate(0, 0, 30) // Próximos 30 días

	disponibles, err := s.habitacionRepo.GetAvailableRooms(fechasInicio, fechasFin)
	if err == nil {
		info.WriteString(fmt.Sprintf("Habitaciones disponibles para el próximo mes (%s - %s): %d\n",
			fechasInicio.Format("2006-01-02"), fechasFin.Format("2006-01-02"), len(disponibles)))
	} else {
		// si falla, no romper; solo no incluimos la línea de disponibilidad
		fmt.Printf("Warning: no se pudo obtener disponibilidad próxima: %v\n", err)
	}

	// 3. Información general del hotel
	info.WriteString("\n=== INFORMACIÓN GENERAL ===\n")
	if s.location != "" {
		info.WriteString(fmt.Sprintf("• Ubicación: %s\n", s.location))
	} else {
		info.WriteString("• Ubicación: [Definir en config]\n")
	}
	info.WriteString("• Check-in: 14:00 hrs\n")
	info.WriteString("• Check-out: 12:00 hrs\n")
	info.WriteString("• WiFi: Gratuito en todas las áreas\n")
	info.WriteString("• Estacionamiento: Disponible\n")

	// 4. Políticas
	info.WriteString("\n=== POLÍTICAS ===\n")
	info.WriteString("• Cancelación gratuita hasta 48 horas antes\n")
	info.WriteString("• Mascotas: No permitidas\n")
	info.WriteString("• Métodos de pago: Efectivo, tarjeta, transferencia\n")

	// 2. Si hay contexto de fechas, obtener disponibilidad
	if req.Context != nil && req.Context.FechaEntrada != nil && req.Context.FechaSalida != nil {
		fechaEntrada, err := time.Parse("2006-01-02", *req.Context.FechaEntrada)
		if err == nil {
			fechaSalida, err := time.Parse("2006-01-02", *req.Context.FechaSalida)
			if err == nil {
				tiposDisponibles, err := s.habitacionRepo.GetAvailableRooms(fechaEntrada, fechaSalida)
				if err == nil {
					info.WriteString(fmt.Sprintf("\n\nDISPONIBILIDAD PARA %s - %s:\n",
						*req.Context.FechaEntrada, *req.Context.FechaSalida))

					if len(tiposDisponibles) == 0 {
						info.WriteString("❌ No hay habitaciones disponibles para estas fechas.\n")
					} else {
						for _, tipo := range tiposDisponibles {
							info.WriteString(fmt.Sprintf("✅ %s: Disponible (Precio: $%.2f, Capacidad: %d adultos + %d niños)\n",
								tipo.Titulo, tipo.Precio, tipo.CapacidadAdultos, tipo.CapacidadNinhos))
						}
					}
				}
			}
		}
	}

	info.WriteString("\n=== FIN DE INFORMACIÓN REAL ===\n")

	return info.String(), nil
}

func (s *ChatbotService) buildSystemPrompt(context *domain.ChatContext, hotelInfo string) string {
	basePrompt := `Eres un asistente virtual amable y profesional de un hotel en Lima, Perú. 

INSTRUCCIONES CRÍTICAS:
- Si te preguntan sobre información externa (clima, restaurantes, atracciones), 
  usarás la información proporcionada en "INFORMACIÓN DE LA WEB".
- Si no hay información web disponible, indica que no tienes esos datos en tiempo real.
- Para información del hotel, usa siempre los datos reales proporcionados.
- Sé amable, profesional y conciso.
- Responde en español.

Tu objetivo es ayudar a los huéspedes con:
- Información sobre habitaciones (SOLO las que aparecen en la información real)
- Proceso de reservas
- Políticas del hotel (check-in 14:00, check-out 12:00)
- Tarifas reales del sistema

POLÍTICAS DEL HOTEL:
- Check-in: 14:00 hrs
- Check-out: 12:00 hrs
- WiFi gratuito
- Desayuno buffet incluido (si aplica)
- Recepción 24 horas

IMPORTANTE:
- Siempre sé cortés y profesional
- Si no sabes algo, admítelo y ofrece transferir a un agente humano
- Proporciona información clara y concisa basada en los datos reales
- Si el usuario quiere hacer una reserva, guíalo paso a paso
- Responde en español a menos que el usuario escriba en otro idioma

`

	// Agregar información real del hotel
	basePrompt += hotelInfo

	if context != nil {
		basePrompt += "\n\nCONTEXTO DE LA CONVERSACIÓN:\n"
		if context.FechaEntrada != nil && context.FechaSalida != nil {
			basePrompt += fmt.Sprintf("- El usuario está consultando para: %s a %s\n",
				*context.FechaEntrada, *context.FechaSalida)
		}
		if context.CantidadAdultos != nil {
			basePrompt += fmt.Sprintf("- Cantidad de adultos: %d\n", *context.CantidadAdultos)
		}
		if context.CantidadNinhos != nil && *context.CantidadNinhos > 0 {
			basePrompt += fmt.Sprintf("- Cantidad de niños: %d\n", *context.CantidadNinhos)
		}
	}

	return basePrompt
}

func (s *ChatbotService) detectHumanRequired(userMsg, botMsg string) bool {
	// Palabras clave que indican necesidad de humano
	keywords := []string{
		"queja", "problema", "insatisfecho", "gerente", "supervisor",
		"hablar con alguien", "hablar con persona", "no entiendo",
		"emergencia", "urgente", "reclamo", "molesto",
	}

	userMsgLower := strings.ToLower(userMsg)
	for _, keyword := range keywords {
		if strings.Contains(userMsgLower, keyword) {
			return true
		}
	}

	// Si el bot menciona que no puede ayudar
	botMsgLower := strings.ToLower(botMsg)
	if strings.Contains(botMsgLower, "no puedo") ||
		strings.Contains(botMsgLower, "transferir") ||
		strings.Contains(botMsgLower, "agente humano") {
		return true
	}

	return false
}

func (s *ChatbotService) generateSuggestedActions(userMsg, botMsg string) []string {
	actions := []string{}

	userMsgLower := strings.ToLower(userMsg)
	botMsgLower := strings.ToLower(botMsg)

	// Sugerencias basadas en el contexto
	if strings.Contains(userMsgLower, "reserva") || strings.Contains(botMsgLower, "reserva") {
		actions = append(actions, "Ver habitaciones disponibles", "Consultar disponibilidad")
	}

	if strings.Contains(userMsgLower, "precio") || strings.Contains(userMsgLower, "tarifa") {
		actions = append(actions, "Ver todas las tarifas", "Consultar promociones")
	}

	if strings.Contains(userMsgLower, "servicios") {
		actions = append(actions, "Ver servicios del hotel", "Ver instalaciones")
	}

	if strings.Contains(userMsgLower, "disponib") {
		actions = append(actions, "Consultar fechas específicas", "Hacer una reserva")
	}

	// Siempre ofrecer hablar con humano si no hay otras acciones
	if len(actions) == 0 {
		actions = append(actions, "Ver habitaciones disponibles", "Hablar con un agente")
	}

	return actions
}

func (s *ChatbotService) GetConversationHistory(conversationID string) (*domain.ConversationHistory, error) {
	return s.repo.GetConversation(conversationID)
}

func (s *ChatbotService) GetClientConversations(clienteID int) ([]domain.ConversationHistory, error) {
	return s.repo.GetClientConversations(clienteID)
}
