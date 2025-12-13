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
	repo             domain.ChatbotRepository
	openaiClient     *openai.Client
	habitacionRepo   domain.HabitacionRepository
	tavilyClient     *tavily.Client
	searchService    *SearchService
	location         string
	reservationTools *ReservationTools
	// Nuevas utilidades
	dateParser   *DateParser
	validator    *Validator
	faqHandler   *FAQHandler
	webCache     *WebCache
	rateLimiter  *RateLimiter
}

func NewChatbotService(
	repo domain.ChatbotRepository,
	openaiClient *openai.Client,
	habitacionRepo domain.HabitacionRepository,
	tavilyClient *tavily.Client,
	location string,
	searchService *SearchService,
	reservaService *ReservaService,
	personRepo domain.PersonRepository,
	clientRepo domain.ClientRepository,
) *ChatbotService {
	// Crear las herramientas de reserva
	reservationTools := NewReservationTools(habitacionRepo, reservaService, personRepo, clientRepo)

	return &ChatbotService{
		repo:             repo,
		openaiClient:     openaiClient,
		habitacionRepo:   habitacionRepo,
		tavilyClient:     tavilyClient,
		searchService:    searchService,
		location:         location,
		reservationTools: reservationTools,
		// Inicializar nuevas utilidades
		dateParser:   &DateParser{},
		validator:    &Validator{},
		faqHandler:   NewFAQHandler(location),
		webCache:     NewWebCache(1 * time.Hour), // Caché de 1 hora
		rateLimiter:  NewRateLimiter(1*time.Minute, 20), // 20 mensajes por minuto
	}
}

func (s *ChatbotService) ProcessMessage(req domain.ChatRequest) (*domain.ChatResponse, error) {
	startTime := time.Now()

	// 0. Rate limiting
	identifier := "anonymous"
	if req.ConversationID != nil && *req.ConversationID != "" {
		identifier = *req.ConversationID
	} else if req.ClienteID != nil {
		identifier = fmt.Sprintf("client_%d", *req.ClienteID)
	}

	allowed, err := s.rateLimiter.Allow(identifier)
	if !allowed {
		log.Printf("Rate limit exceeded for %s: %v", identifier, err)
		return &domain.ChatResponse{
			Message:          "⚠️ Has enviado muchos mensajes en poco tiempo. " + err.Error(),
			ConversationID:   "",
			RequiresHuman:    false,
			SuggestedActions: []string{"Espera un momento", "Intenta más tarde"},
		}, nil
	}

	// 1. Verificar si es una pregunta frecuente simple
	if s.faqHandler.ShouldUseFAQ(req.Message) {
		if quickResponse, found := s.faqHandler.GetQuickResponse(req.Message); found {
			log.Printf("FAQ quick response for: %s (took %v)", req.Message, time.Since(startTime))

			// Crear o recuperar conversación para guardar el mensaje
			var conversation *domain.ConversationHistory
			if req.ConversationID != nil && *req.ConversationID != "" {
				conversation, _ = s.repo.GetConversation(*req.ConversationID)
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

			// Guardar mensaje del usuario y respuesta
			conversation.Messages = append(conversation.Messages,
				domain.ChatMessage{Role: "user", Content: req.Message},
				domain.ChatMessage{Role: "assistant", Content: quickResponse},
			)
			conversation.UpdatedAt = time.Now()

			if req.ConversationID == nil || *req.ConversationID == "" {
				_ = s.repo.SaveConversation(conversation)
			} else {
				_ = s.repo.UpdateConversation(conversation)
			}

			return &domain.ChatResponse{
				Message:          quickResponse,
				ConversationID:   conversation.ID,
				RequiresHuman:    false,
				SuggestedActions: s.generateSuggestedActions(req.Message, quickResponse),
				Metadata: map[string]interface{}{
					"source":       "faq",
					"responseTime": time.Since(startTime).Milliseconds(),
				},
			}, nil
		}
	}

	// 2. Obtener o crear historial de conversación
	var conversation *domain.ConversationHistory

	if req.ConversationID != nil && *req.ConversationID != "" {
		conversation, err = s.repo.GetConversation(*req.ConversationID)
		if err != nil {
			log.Printf("Error getting conversation %s: %v", *req.ConversationID, err)
			return nil, fmt.Errorf("❌ No se pudo recuperar la conversación. Por favor, intenta de nuevo")
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
		log.Printf("New conversation created: %s", conversation.ID)
	}

	// 2.5 Detectar si el usuario quiere cancelar
	if s.detectCancelIntent(req.Message) && conversation.ReservationInProgress != nil {
		conversation.ReservationInProgress = nil
		conversation.Messages = append(conversation.Messages,
			domain.ChatMessage{Role: "user", Content: req.Message},
			domain.ChatMessage{Role: "assistant", Content: "✅ He cancelado la reserva en progreso. ¿En qué más puedo ayudarte?"},
		)
		_ = s.repo.UpdateConversation(conversation)

		return &domain.ChatResponse{
			Message:              "✅ He cancelado la reserva en progreso. ¿En qué más puedo ayudarte?",
			ConversationID:       conversation.ID,
			RequiresHuman:        false,
			ReservationInProgress: nil,
			SuggestedActions:     []string{"Ver habitaciones disponibles", "Hacer una nueva reserva"},
			Metadata: map[string]interface{}{
				"source":       "cancel",
				"responseTime": time.Since(startTime).Milliseconds(),
			},
		}, nil
	}

	// 3. Detectar intención de reserva y actualizar estado
	conversation = s.updateReservationState(conversation, req.Message)

	// 4. Agregar mensaje del usuario
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
		// Construir query con ubicación para focalizar resultados locales
		query := req.Message
		if s.location != "" {
			query = fmt.Sprintf("%s near %s", req.Message, s.location)
		}

		// Intentar obtener del caché primero
		if cachedResp, found := s.webCache.Get(query); found {
			log.Printf("Web search cache HIT for: %s", query)
			tavilyResp = cachedResp
			webContext = s.formatWebResults(cachedResp)
		} else {
			log.Printf("Web search cache MISS, performing search for: %s", query)

			// Realizar búsqueda web
			if s.searchService != nil {
				input := SearchInput{Query: query, MaxResults: 3}
				if resp, err := s.searchService.SearchWeb(input); err == nil {
					tavilyResp = resp
					webContext = s.formatWebResults(resp)
					// Guardar en caché
					s.webCache.Set(query, resp)
				} else {
					log.Printf("searchService error: %v", err)
				}
			} else if s.tavilyClient != nil {
				if resp, err := s.tavilyClient.Search(tavily.SearchRequest{Query: query, MaxResults: 3}); err == nil {
					tavilyResp = resp
					webContext = s.formatWebResults(resp)
					// Guardar en caché
					s.webCache.Set(query, resp)
				} else {
					log.Printf("tavily search error: %v", err)
				}
			}
		}
	}

	// 5. Obtener información real del hotel desde la BD
	hotelInfo, err := s.getHotelInfo(req)
	if err != nil {
		log.Printf("Error getting hotel info: %v", err)
		return nil, fmt.Errorf("❌ Error al obtener información del hotel. Por favor, intenta de nuevo")
	}

	// 4. Preparar contexto con información real (incluye resultados web y herramientas)
	toolsInfo := s.reservationTools.GetToolDescriptions()
	reservationContext := s.buildReservationContext(conversation.ReservationInProgress)
	systemPrompt := s.buildSystemPrompt(req.Context, hotelInfo+webContext+toolsInfo+reservationContext)

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
	llmStartTime := time.Now()
	openaiReq := openai.ChatCompletionRequest{
		Model:       "llama-3.1-8b-instant", // Modelo de Groq
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   500,
	}

	log.Printf("Calling LLM with %d messages, model: %s", len(messages), openaiReq.Model)

	openaiResp, err := s.openaiClient.CreateChatCompletion(openaiReq)
	if err != nil {
		log.Printf("❌ LLM error: %v (took %v)", err, time.Since(llmStartTime))
		return nil, fmt.Errorf("❌ Error al procesar tu mensaje. El servicio está temporalmente no disponible. Por favor, intenta de nuevo en unos momentos")
	}

	if len(openaiResp.Choices) == 0 {
		log.Printf("❌ LLM returned no choices (took %v)", time.Since(llmStartTime))
		return nil, fmt.Errorf("❌ No se pudo generar una respuesta. Por favor, intenta reformular tu pregunta")
	}

	log.Printf("✅ LLM response received (took %v, tokens: %d)", time.Since(llmStartTime), openaiResp.Usage.TotalTokens)

	assistantMessage := openaiResp.Choices[0].Message.Content

	// 6.5 Detectar y ejecutar herramientas si es necesario
	finalMessage := assistantMessage
	toolExecuted := false
	var toolErr error

	// Intentar ejecutar herramientas (máximo 3 intentos para evitar loops)
	maxToolIterations := 3
	for i := 0; i < maxToolIterations; i++ {
		var executed bool
		finalMessage, executed, toolErr = s.detectAndExecuteTools(finalMessage)

		if toolErr != nil {
			// Si hay error en la herramienta, agregar el error al mensaje
			finalMessage = fmt.Sprintf("%s\n\n[ERROR]: %s", finalMessage, toolErr.Error())
			log.Printf("Error ejecutando herramienta: %v", toolErr)
			break
		}

		if !executed {
			break
		}

		toolExecuted = true

		// Si se ejecutó una herramienta, hacer otra llamada al LLM con el resultado
		conversation.Messages = append(conversation.Messages, domain.ChatMessage{
			Role:    "assistant",
			Content: finalMessage,
		})

		// Construir mensajes para segunda llamada
		messages = []openai.Message{
			{Role: "system", Content: systemPrompt},
		}

		// Agregar últimos mensajes
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

		// Segunda llamada al LLM
		openaiReq = openai.ChatCompletionRequest{
			Model:       "llama-3.1-8b-instant",
			Messages:    messages,
			Temperature: 0.7,
			MaxTokens:   500,
		}

		openaiResp, err = s.openaiClient.CreateChatCompletion(openaiReq)
		if err != nil {
			return nil, fmt.Errorf("error calling OpenAI (second call): %w", err)
		}

		if len(openaiResp.Choices) == 0 {
			return nil, fmt.Errorf("no response from OpenAI (second call)")
		}

		finalMessage = openaiResp.Choices[0].Message.Content
	}

	// 7. Agregar respuesta final del asistente
	conversation.Messages = append(conversation.Messages, domain.ChatMessage{
		Role:    "assistant",
		Content: finalMessage,
	})

	// 8. Guardar conversación
	conversation.UpdatedAt = time.Now()
	if req.ConversationID == nil || *req.ConversationID == "" {
		err = s.repo.SaveConversation(conversation)
		if err != nil {
			log.Printf("❌ Error saving new conversation: %v", err)
			// No fallar completamente, solo logear
		} else {
			log.Printf("✅ New conversation saved: %s", conversation.ID)
		}
	} else {
		err = s.repo.UpdateConversation(conversation)
		if err != nil {
			log.Printf("❌ Error updating conversation %s: %v", conversation.ID, err)
			// No fallar completamente, solo logear
		}
	}

	// 9. Si el cliente está identificado, guardar en tabla mensaje
	if req.ClienteID != nil {
		_ = s.repo.SaveMessage(*req.ClienteID, req.Message)
	}

	// 10. Analizar si requiere intervención humana
	requiresHuman := s.detectHumanRequired(req.Message, finalMessage)

	// 11. Generar acciones sugeridas
	suggestedActions := s.generateSuggestedActions(req.Message, finalMessage)

	sources := []string{"hotel"}
	if useWeb {
		sources = append(sources, "web")
	}
	if toolExecuted {
		sources = append(sources, "tools")
	}

	// Metadata mejorado con más información útil
	metadata := map[string]interface{}{
		"tokensUsed":   openaiResp.Usage.TotalTokens,
		"sources":      sources,
		"responseTime": time.Since(startTime).Milliseconds(),
		"llmModel":     openaiReq.Model,
		"messageCount": len(conversation.Messages),
	}

	// incluir resultados web crudos para uso del frontend (si existen)
	if tavilyResp != nil {
		metadata["webResults"] = tavilyResp
		metadata["webCacheHit"] = false
		if cachedResp, found := s.webCache.Get(fmt.Sprintf("%s near %s", req.Message, s.location)); found && cachedResp != nil {
			metadata["webCacheHit"] = true
		}
	}

	// Información de rate limiting
	if req.ConversationID != nil {
		remaining := s.rateLimiter.GetRemaining(identifier)
		metadata["rateLimitRemaining"] = remaining
	}

	log.Printf("✅ Total request processed in %v (conversation: %s)", time.Since(startTime), conversation.ID)

	// 12. Verificar si se creó una reserva
	var reservaCreada *int
	if strings.Contains(finalMessage, "Reserva creada exitosamente") {
		// Intentar extraer el ID de la reserva del mensaje
		// El formato es "Número de Reserva: #ID"
		if strings.Contains(finalMessage, "Número de Reserva: #") {
			var id int
			if _, err := fmt.Sscanf(finalMessage, "Número de Reserva: #%d", &id); err == nil {
				reservaCreada = &id
				// Limpiar el estado de reserva en progreso
				conversation.ReservationInProgress = nil
			}
		}
	}

	return &domain.ChatResponse{
		Message:               finalMessage,
		ConversationID:        conversation.ID,
		SuggestedActions:      suggestedActions,
		RequiresHuman:         requiresHuman,
		Metadata:              metadata,
		ReservationInProgress: conversation.ReservationInProgress,
		ReservationCreated:    reservaCreada,
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
- Puedes ejecutar acciones usando las HERRAMIENTAS DISPONIBLES cuando sea necesario.

Tu objetivo es ayudar a los huéspedes con:
- Información sobre habitaciones (SOLO las que aparecen en la información real)
- Proceso de reservas COMPLETO (puedes crear reservas usando las herramientas)
- Políticas del hotel (check-in 14:00, check-out 12:00)
- Tarifas reales del sistema

FLUJO DE RESERVAS:
Cuando un usuario quiera hacer una reserva, sigue estos pasos:
1. Pregunta fechas de entrada y salida
2. Pregunta cantidad de adultos y niños
3. USA LA HERRAMIENTA 'check_availability' para verificar disponibilidad
4. Muestra las opciones disponibles usando 'get_room_types' si es necesario
5. Pregunta qué tipo de habitación prefiere
6. USA LA HERRAMIENTA 'calculate_price' para calcular el precio total
7. (OPCIONAL) Pregunta el correo electrónico del cliente para seguimiento
8. USA LA HERRAMIENTA 'generate_booking_link' con la información recopilada
9. Proporciona el enlace al usuario explicando que completará sus datos en el formulario

IMPORTANTE:
- NO intentes recopilar nombres completos o datos personales detallados
- El correo es OPCIONAL, solo para seguimiento CRM
- El usuario completará TODOS sus datos en el formulario web
- El formulario está diseñado para manejar múltiples huéspedes correctamente
- Solo recopila: fechas, cantidad de adultos, cantidad de niños, tipo de habitación, y opcionalmente email

POLÍTICAS DEL HOTEL:
- Check-in: 14:00 hrs
- Check-out: 12:00 hrs
- WiFi gratuito
- Desayuno buffet incluido
- Recepción 24 horas

IMPORTANTE:
- Siempre sé cortés y profesional
- Si no sabes algo, admítelo y ofrece transferir a un agente humano
- Proporciona información clara y concisa basada en los datos reales
- Cuando uses una herramienta, explica al usuario qué estás haciendo
- Responde en español a menos que el usuario escriba en otro idioma
- NUNCA inventes información, usa siempre las herramientas o la información proporcionada

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

// generateContextualSuggestedActions genera acciones sugeridas basadas en el estado de la reserva
func (s *ChatbotService) generateContextualSuggestedActions(reservation *domain.ReservationInProgress, userMsg, botMsg string) []string {
	// Si hay una reserva en progreso, sugerir según el paso actual
	if reservation != nil {
		switch reservation.Step {
		case "dates":
			return []string{"Consultar disponibilidad", "Ver habitaciones", "Cancelar reserva"}
		case "guests":
			return []string{"Continuar con reserva", "Cambiar fechas", "Cancelar reserva"}
		case "room_type":
			return []string{"Ver detalles de habitaciones", "Cambiar fechas", "Cancelar reserva"}
		case "personal_data":
			return []string{"Confirmar datos", "Modificar reserva", "Cancelar reserva"}
		case "confirmation":
			return []string{"Confirmar reserva", "Modificar datos", "Cancelar reserva"}
		}
	}

	// Si no hay reserva en progreso, usar las sugerencias generales
	return s.generateSuggestedActions(userMsg, botMsg)
}

func (s *ChatbotService) GetConversationHistory(conversationID string) (*domain.ConversationHistory, error) {
	return s.repo.GetConversation(conversationID)
}

func (s *ChatbotService) GetClientConversations(clienteID int) ([]domain.ConversationHistory, error) {
	return s.repo.GetClientConversations(clienteID)
}

// detectAndExecuteTools detecta si el mensaje del asistente contiene llamadas a herramientas y las ejecuta
func (s *ChatbotService) detectAndExecuteTools(message string) (string, bool, error) {
	// Buscar el patrón [USE_TOOL: nombre_herramienta]
	toolStartIdx := strings.Index(message, "[USE_TOOL:")
	if toolStartIdx == -1 {
		return message, false, nil
	}

	toolEndIdx := strings.Index(message, "[END_TOOL]")
	if toolEndIdx == -1 {
		return message, false, fmt.Errorf("tool call mal formateada: falta [END_TOOL]")
	}

	// Extraer el contenido del tool call
	toolCall := message[toolStartIdx : toolEndIdx+len("[END_TOOL]")]

	// Extraer el nombre de la herramienta
	toolNameStart := toolStartIdx + len("[USE_TOOL:")
	toolNameEnd := strings.Index(message[toolNameStart:], "]")
	if toolNameEnd == -1 {
		return message, false, fmt.Errorf("tool call mal formateada: falta ] después del nombre")
	}

	toolName := strings.TrimSpace(message[toolNameStart : toolNameStart+toolNameEnd])

	// Extraer los argumentos (el JSON entre el nombre y [END_TOOL])
	argsStart := toolNameStart + toolNameEnd + 1
	argsEnd := toolEndIdx
	args := strings.TrimSpace(message[argsStart:argsEnd])

	log.Printf("Executing tool: %s with args: %s", toolName, args)

	// Ejecutar la herramienta
	result, err := s.reservationTools.ExecuteTool(toolName, args)
	if err != nil {
		return message, true, fmt.Errorf("error ejecutando herramienta %s: %w", toolName, err)
	}

	log.Printf("Tool result: %s", result)

	// Reemplazar el tool call con el resultado
	messageWithResult := strings.Replace(message, toolCall, "", 1)
	messageWithResult += fmt.Sprintf("\n\n[RESULTADO DE %s]:\n%s\n[FIN RESULTADO]\n", strings.ToUpper(toolName), result)

	return messageWithResult, true, nil
}

// updateReservationState actualiza el estado de la reserva en progreso basado en el mensaje del usuario
func (s *ChatbotService) updateReservationState(conversation *domain.ConversationHistory, userMessage string) *domain.ConversationHistory {
	msgLower := strings.ToLower(userMessage)

	// Detectar intención de iniciar una reserva
	reservaKeywords := []string{
		"reservar", "reserva", "reservación",
		"habitación", "habitacion", "cuarto",
		"quiero hospedarme", "necesito una habitación",
		"book", "booking",
	}

	isReservationIntent := false
	for _, keyword := range reservaKeywords {
		if strings.Contains(msgLower, keyword) {
			isReservationIntent = true
			break
		}
	}

	// Si hay intención de reserva y no hay una en progreso, iniciar una nueva
	if isReservationIntent && conversation.ReservationInProgress == nil {
		conversation.ReservationInProgress = &domain.ReservationInProgress{
			Step: "dates",
		}
		log.Printf("Nueva reserva iniciada en paso: dates")
	}

	// Si hay una reserva en progreso, intentar extraer información del mensaje
	if conversation.ReservationInProgress != nil {
		s.extractReservationData(conversation.ReservationInProgress, userMessage)
	}

	return conversation
}

// extractReservationData intenta extraer datos de reserva del mensaje del usuario
func (s *ChatbotService) extractReservationData(reservation *domain.ReservationInProgress, message string) {
	now := time.Now().Truncate(24 * time.Hour)

	// Intentar extraer rango de fechas con el DateParser mejorado
	if reservation.Step == "dates" || (reservation.FechaEntrada == nil || reservation.FechaSalida == nil) {
		if startDate, endDate, err := s.dateParser.ExtractDateRange(message); err == nil {
			startDateStr := startDate.Format("2006-01-02")
			endDateStr := endDate.Format("2006-01-02")
			reservation.FechaEntrada = &startDateStr
			reservation.FechaSalida = &endDateStr
			log.Printf("✅ Fechas extraídas: %s a %s", startDateStr, endDateStr)
		} else {
			// Intentar extraer una sola fecha
			if singleDate, err := s.dateParser.ParseNaturalDate(message, now); err == nil {
				dateStr := singleDate.Format("2006-01-02")
				if reservation.FechaEntrada == nil {
					reservation.FechaEntrada = &dateStr
					log.Printf("✅ Fecha de entrada extraída: %s", dateStr)
				} else if reservation.FechaSalida == nil {
					reservation.FechaSalida = &dateStr
					log.Printf("✅ Fecha de salida extraída: %s", dateStr)
				}
			}
		}
	}

	// Intentar extraer cantidad de adultos y niños con números mejorados
	numbers := s.dateParser.ExtractNumbers(message)
	msgLower := strings.ToLower(message)

	if strings.Contains(msgLower, "adulto") && len(numbers) > 0 {
		reservation.CantidadAdultos = &numbers[0]
		log.Printf("✅ Cantidad de adultos extraída: %d", numbers[0])
	}

	if (strings.Contains(msgLower, "niño") || strings.Contains(msgLower, "niños")) && len(numbers) > 1 {
		reservation.CantidadNinhos = &numbers[1]
		log.Printf("✅ Cantidad de niños extraída: %d", numbers[1])
	} else if strings.Contains(msgLower, "sin niños") || strings.Contains(msgLower, "sin niño") {
		zero := 0
		reservation.CantidadNinhos = &zero
		log.Printf("✅ Cantidad de niños extraída: 0")
	}

	// Intentar extraer simplemente "X personas" o "X adultos"
	if len(numbers) > 0 {
		if strings.Contains(msgLower, "persona") && reservation.CantidadAdultos == nil {
			reservation.CantidadAdultos = &numbers[0]
			log.Printf("✅ Cantidad de adultos extraída de 'personas': %d", numbers[0])
		}
	}

	// Actualizar el paso basado en la información disponible
	if reservation.FechaEntrada != nil && reservation.FechaSalida != nil && reservation.Step == "dates" {
		reservation.Step = "guests"
		log.Printf("📍 Paso actualizado a: guests")
	}
	if reservation.CantidadAdultos != nil && reservation.Step == "guests" {
		reservation.Step = "room_type"
		log.Printf("📍 Paso actualizado a: room_type")
	}
	if reservation.TipoHabitacionID != nil && reservation.Step == "room_type" {
		reservation.Step = "personal_data"
		log.Printf("📍 Paso actualizado a: personal_data")
	}
	if reservation.PersonalData != nil && reservation.Step == "personal_data" {
		reservation.Step = "confirmation"
		log.Printf("📍 Paso actualizado a: confirmation")
	}
}

// buildReservationContext construye el contexto de una reserva en progreso
func (s *ChatbotService) buildReservationContext(reservation *domain.ReservationInProgress) string {
	if reservation == nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("\n=== RESERVA EN PROGRESO ===\n")
	sb.WriteString(fmt.Sprintf("Paso actual: %s\n", reservation.Step))

	if reservation.FechaEntrada != nil {
		sb.WriteString(fmt.Sprintf("Fecha de entrada: %s\n", *reservation.FechaEntrada))
	}
	if reservation.FechaSalida != nil {
		sb.WriteString(fmt.Sprintf("Fecha de salida: %s\n", *reservation.FechaSalida))
	}
	if reservation.CantidadAdultos != nil {
		sb.WriteString(fmt.Sprintf("Cantidad de adultos: %d\n", *reservation.CantidadAdultos))
	}
	if reservation.CantidadNinhos != nil {
		sb.WriteString(fmt.Sprintf("Cantidad de niños: %d\n", *reservation.CantidadNinhos))
	}
	if reservation.TipoHabitacionID != nil {
		sb.WriteString(fmt.Sprintf("Tipo de habitación seleccionado: ID %d\n", *reservation.TipoHabitacionID))
	}
	if reservation.PrecioCalculado != nil {
		sb.WriteString(fmt.Sprintf("Precio calculado: S/%.2f\n", *reservation.PrecioCalculado))
	}
	if reservation.PersonalData != nil {
		sb.WriteString("Datos personales proporcionados\n")
	}

	sb.WriteString("\nRecuerda continuar el proceso de reserva según el paso actual.\n")
	sb.WriteString("=== FIN RESERVA EN PROGRESO ===\n\n")

	return sb.String()
}

// detectCancelIntent detecta si el usuario quiere cancelar la reserva en progreso
func (s *ChatbotService) detectCancelIntent(message string) bool {
	msgLower := strings.ToLower(message)

	cancelKeywords := []string{
		"cancelar", "cancela", "cancelar reserva",
		"empezar de nuevo", "empezar otra vez",
		"borrar", "eliminar", "deshacer",
		"no quiero", "ya no quiero",
		"mejor no", "olvídalo", "olvidalo",
		"reiniciar", "restart", "reset",
	}

	for _, keyword := range cancelKeywords {
		if strings.Contains(msgLower, keyword) {
			return true
		}
	}

	return false
}
