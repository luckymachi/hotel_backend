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
			ID:                    uuid.New().String(),
			ClienteID:             req.ClienteID,
			Messages:              []domain.ChatMessage{},
			CreatedAt:             time.Now(),
			UpdatedAt:             time.Now(),
			ReservationInProgress: nil,
		}
	}

	// 1.5 Detectar intención de reserva y actualizar estado
	conversation = s.updateReservationState(conversation, req.Message)

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

	// ✅ ADD THIS: Log what LLM actually generated
	log.Printf("LLM raw response: %s", assistantMessage)

	// 6.5 Detectar y ejecutar herramientas si es necesario
	finalMessage := assistantMessage
	toolExecuted := false
	var toolErr error

	// Intentar ejecutar herramientas (máximo 3 intentos para evitar loops)
	// maxToolIterations := 3

	// Ejecutar herramienta una sola vez
	finalMessage, toolExecuted, toolErr = s.detectAndExecuteTools(assistantMessage)

	if toolErr != nil {
		finalMessage = fmt.Sprintf("Lo siento, hubo un error: %s\n\nPor favor, intenta de nuevo.", toolErr.Error())
		log.Printf("Error ejecutando herramienta: %v", toolErr)
	}

	// Si se ejecutó una herramienta, usar el resultado directamente
	// NO hacer segunda llamada al LLM - el resultado de la herramienta ya está formateado
	if toolExecuted && toolErr == nil {
		log.Printf("Tool executed successfully, using result directly (no second LLM call)")
	}
	/*
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
	*/

	// 7. Agregar respuesta final del asistente
	conversation.Messages = append(conversation.Messages, domain.ChatMessage{
		Role:    "assistant",
		Content: finalMessage,
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

	metadata := map[string]interface{}{
		"tokensUsed": openaiResp.Usage.TotalTokens,
		"sources":    sources,
	}
	// incluir resultados web crudos para uso del frontend (si existen)
	if tavilyResp != nil {
		metadata["webResults"] = tavilyResp
	}

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

	// ✅ NEW: Clean the message before sending to frontend
	cleanedMessage := cleanResponseForFrontend(finalMessage)

	return &domain.ChatResponse{
		Message:               cleanedMessage,
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
- Recomendaciones personalizadas basadas en necesidades del huésped
- Políticas del hotel (check-in 14:00, check-out 12:00)
- Tarifas reales del sistema
IMPORTANTE SOBRE CONTEXTO:
- RECUERDA el tipo de habitación que el usuario eligió
- Si el usuario dice "Suite", el tipoHabitacionId es 6
- Si el usuario dice "Doble", el tipoHabitacionId es 5
- Si el usuario dice "Simple", el tipoHabitacionId es 4
- Si el usuario dice "Presidencial", el tipoHabitacionId es 9
- Si el usuario dice "Familiar", el tipoHabitacionId es 10
- NO inventes IDs que no existen
- Cuando uses check_availability o calculate_price, RECUERDA el ID que usaste
- Cuando uses generate_booking_link, USA EL MISMO ID que usaste en calculate_price
FLUJO DE RESERVAS CON RECOMENDACIONES INTELIGENTES:
Cuando un usuario quiera hacer una reserva, sigue estos pasos EXACTAMENTE:
1. Pregunta fechas de entrada y salida
2. Pregunta cantidad de adultos y niños
3. EJECUTA: [USE_TOOL: check_availability]
   {"fechaEntrada": "YYYY-MM-DD", "fechaSalida": "YYYY-MM-DD"}
   [END_TOOL]
4. EJECUTA: [USE_TOOL: get_room_types]
   {}
   [END_TOOL]
5. **ANALIZA la composición de huéspedes y HAZ UNA RECOMENDACIÓN PERSONALIZADA**
6. Muestra tu recomendación destacada + otras opciones disponibles
7. Usuario selecciona habitación (puede aceptar tu recomendación o elegir otra)
8. EJECUTA: [USE_TOOL: calculate_price]
   {"tipoHabitacionId": X, "fechaEntrada": "YYYY-MM-DD", "fechaSalida": "YYYY-MM-DD"}
   [END_TOOL]
9. Muestra el precio total
10. Pregunta si quiere continuar/confirmar
11. CUANDO USUARIO CONFIRMA (dice "sí", "confirmar", "continuar", etc.), EJECUTA INMEDIATAMENTE:
    [USE_TOOL: generate_booking_link]
    {"fechaEntrada": "YYYY-MM-DD", "fechaSalida": "YYYY-MM-DD", "tipoHabitacionId": X, "cantidadAdultos": X, "cantidadNinhos": X, "email": "opcional@email.com"}
    [END_TOOL]
LÓGICA DE RECOMENDACIONES (USA DESPUÉS DE check_availability Y get_room_types):
Analiza la composición de huéspedes y recomienda según estos criterios:
**1 adulto, 0 niños (VIAJERO SOLO):**
   → Recomienda: Simple (ID 4) o Doble (ID 5) según disponibilidad y presupuesto
   → Razón: "Como viajas solo, te recomiendo la [Simple/Doble] que ofrece el mejor valor para una persona"
   → Menciona: Precio económico, espacio adecuado, comodidad
**2 adultos, 0 niños (PAREJA):**
   → Recomienda: Doble (ID 5) como primera opción
   → Razón: "Para una pareja, la Doble es perfecta - cómoda y con excelente precio"
   → Menciona: Cama matrimonial, espacio romántico, buen valor
   → Opción premium: "Si prefieres más lujo, tenemos la Suite (ID 6) con amenidades extras"
**3+ adultos, 0 niños (GRUPO/AMIGOS):**
   → Recomienda: Familiar (ID 10) o Suite (ID 6) según capacidad
   → Razón: "Para tu grupo de [X] personas, recomiendo la [Familiar/Suite] que tiene capacidad amplia"
   → Menciona: Espacio, capacidad, comodidad para grupos, precio por persona
**1-2 adultos + 1 niño (FAMILIA PEQUEÑA):**
   → Recomienda: Doble (ID 5) o Familiar (ID 10) según presupuesto
   → Razón: "Para tu familia con 1 niño, la [Doble/Familiar] es ideal - cómoda y a buen precio"
   → Menciona: Espacio para el niño, precio familiar accesible
**2+ adultos + 2+ niños (FAMILIA GRANDE):**
   → Recomienda: Suite (ID 6) o Familiar (ID 10)
   → Razón: "Para tu familia con [X] niños, la [Suite/Familiar] es la mejor opción - amplia y perfecta para familias"
   → Menciona: Espacio amplio, capacidad para todos, comodidad familiar
**FACTORES ADICIONALES A CONSIDERAR:**
- **Señales de presupuesto**: Si mencionan "económica", "barata", "presupuesto ajustado" → recomienda la más económica que cumple necesidades
- **Señales de lujo**: Si mencionan "lujosa", "premium", "la mejor", "consentirme" → recomienda Presidencial (ID 9) o Suite (ID 6)
- **Capacidad**: SIEMPRE verifica que la habitación tenga capacidad suficiente
- **Valor**: Si no mencionan presupuesto, recomienda el mejor equilibrio calidad-precio
**FORMATO DE PRESENTACIÓN DE RECOMENDACIONES:**
Cuando presentes opciones, usa EXACTAMENTE este formato:
✨ **MI RECOMENDACIÓN PARA TI:**
[Nombre de habitación] - S/[precio]/noche
[Emoji relevante] [Razón principal de recomendación basada en su situación]
• Característica 1
• Característica 2
• Característica 3
• Total estimado: S/[precio_total] por [X] noches
📋 **OTRAS OPCIONES DISPONIBLES:**
• [Habitación 2] - S/[precio]/noche - [Breve descripción y capacidad]
• [Habitación 3] - S/[precio]/noche - [Breve descripción y capacidad]
¿Cuál te gustaría reservar?
**EMOJIS POR SITUACIÓN:**
- Viajero solo: 🎒 🌟
- Pareja: 💑 ❤️ 🌹
- Familia: 👨‍👩‍👧‍👦 🏡 👶
- Grupo/Amigos: 👥 🎉
- Lujo/Premium: ✨ 👑 💎
- Valor/Económico: 💰 🎯
**EJEMPLO COMPLETO DE RECOMENDACIÓN:**
Usuario: "Quiero reservar para 2 adultos y 1 niño del 20 al 27 de diciembre"
Tu respuesta:
[Ejecutas check_availability y get_room_types]
¡Perfecto! Déjame verificar disponibilidad...
[DESPUÉS DE RECIBIR RESULTADOS]
✨ **MI RECOMENDACIÓN PARA TI:**
Habitación Familiar - S/150/noche
👨‍👩‍👧‍👦 Ideal para tu familia de 3 - espaciosa y muy cómoda para familias con niños
• Capacidad: hasta 4 personas (perfecto para ustedes)
• Cama matrimonial + cama individual
• Baño amplio con regadera
• WiFi y TV incluidos
• Total estimado: S/1,050 por 7 noches
📋 **OTRAS OPCIONES DISPONIBLES:**
• Suite - S/297/noche - Más lujosa con jacuzzi (S/2,079 total)
• Doble - S/90/noche - Más económica pero capacidad limitada (S/630 total)
¿Cuál te gustaría reservar?
CRÍTICO - USAR HERRAMIENTAS:
- Para generar el enlace de reserva, DEBES usar EXACTAMENTE este formato:
  [USE_TOOL: generate_booking_link]
  {"fechaEntrada": "2025-12-27", "fechaSalida": "2026-01-04", "tipoHabitacionId": 6, "cantidadAdultos": 2, "cantidadNinhos": 0, "email": "ga@gmail.com"}
  [END_TOOL]
- NO inventes URLs como "http://www.hotel.com/reserva/..."
- NO generes enlaces manualmente
- SIEMPRE usa la herramienta generate_booking_link
- El resultado de la herramienta ya contiene el enlace correcto y el mensaje formateado
EJEMPLO COMPLETO de cuándo usar generate_booking_link:
Usuario: "Sí, confirmo" o "Quiero la Suite" (después de ver el precio)
Tu respuesta DEBE ser:
[USE_TOOL: generate_booking_link]
{"fechaEntrada": "2025-12-28", "fechaSalida": "2026-01-04", "tipoHabitacionId": 6, "cantidadAdultos": 2, "cantidadNinhos": 0, "email": "ga@gmail.com"}
[END_TOOL]
NO respondas con texto antes del tool call cuando el usuario confirma una reserva.
CRÍTICO - NO MOSTRAR AL USUARIO:
- NO muestres [USE_TOOL...], [RESULTADO...], [FIN RESULTADO]
- NO digas "Espero un momento mientras..." - solo ejecuta la herramienta
- NO muestres URLs falsas o de ejemplo
- NO reformules el resultado de las herramientas - son perfectos tal cual
- Los marcadores internos son SOLO para ti - el usuario ve respuestas limpias
IMPORTANTE SOBRE FECHAS:
- Si el usuario dice algo como "del 27 de diciembre al 4 de enero" asume que enero es del AÑO SIGUIENTE
- Ejemplo: 27 dic 2025 al 4 ene = 2025-12-27 a 2026-01-04
- Si las fechas cruzan el año nuevo, USA el año correcto para cada fecha
- Formato SIEMPRE: YYYY-MM-DD
IMPORTANTE SOBRE generate_booking_link:
- Cuando el usuario dice "quiero hacer la reserva" o "confirmar", USA generate_booking_link INMEDIATAMENTE
- NO preguntes más cosas si ya tienes: fechas, adultos, niños, tipo de habitación
- El correo es OPCIONAL - si no lo tienes, genera el link sin él
- El resultado de la herramienta YA está perfectamente formateado - NO lo reformules
- NO agregues texto adicional después del resultado de la herramienta
IMPORTANTE SOBRE RECOMENDACIONES:
- SIEMPRE haz una recomendación personalizada basada en la composición de huéspedes
- Muestra confianza en tu recomendación pero respeta si el usuario prefiere otra opción
- Si el usuario rechaza tu recomendación, pregunta cuál prefiere de las otras opciones
- Explica SIEMPRE por qué recomiendas esa habitación específica
- Menciona capacidad, precio y características clave
- Usa emojis para hacer las recomendaciones más visuales y amigables
POLÍTICAS DEL HOTEL:
- Check-in: 14:00 hrs
- Check-out: 12:00 hrs
- WiFi gratuito
- Desayuno buffet incluido
- Recepción 24 horas
- Estacionamiento disponible (consultar precio)
- Se aceptan mascotas pequeñas (consultar condiciones)
IMPORTANTE:
- Siempre sé cortés y profesional
- Si no sabes algo, admítelo y ofrece transferir a un agente humano
- Proporciona información clara y concisa basada en los datos reales
- Cuando uses una herramienta, NO expliques al usuario que la estás usando - solo hazlo
- Responde en español a menos que el usuario escriba en otro idioma
- NUNCA inventes información, usa siempre las herramientas o la información proporcionada
- Actúa como un concierge experto que conoce a fondo el hotel y puede aconsejar bien
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

	//	isReservationIntent := false
	isNewReservationIntent := false
	for _, keyword := range reservaKeywords {
		if strings.Contains(msgLower, keyword) {
			isNewReservationIntent = true
			break
		}
	}

	/*
		// Si hay intención de reserva y no hay una en progreso, iniciar una nueva
		if isReservationIntent && conversation.ReservationInProgress == nil {
			conversation.ReservationInProgress = &domain.ReservationInProgress{
				Step: "dates",
			}
			log.Printf("Nueva reserva iniciada en paso: dates")
		}
	*/
	// Solo iniciar NUEVA reserva si NO hay una en progreso
	if isNewReservationIntent && conversation.ReservationInProgress == nil {
		conversation.ReservationInProgress = &domain.ReservationInProgress{
			Step: "dates",
		}
		log.Printf("Nueva reserva iniciada en paso: dates")
	} else if conversation.ReservationInProgress != nil {
		log.Printf("Reserva existente, paso actual: %s", conversation.ReservationInProgress.Step)
	}

	// Si hay una reserva en progreso, intentar extraer información del mensaje
	if conversation.ReservationInProgress != nil {
		s.extractReservationData(conversation.ReservationInProgress, userMessage)
	}

	return conversation
}

// extractReservationData intenta extraer datos de reserva del mensaje del usuario
func (s *ChatbotService) extractReservationData(reservation *domain.ReservationInProgress, message string) {
	// Intentar extraer fechas (formato YYYY-MM-DD o DD/MM/YYYY)
	// Esto es básico, se podría mejorar con NLP más sofisticado

	// Intentar extraer cantidad de adultos
	if strings.Contains(strings.ToLower(message), "adulto") {
		// Buscar números en el mensaje
		var num int
		if _, err := fmt.Sscanf(message, "%d", &num); err == nil && num > 0 {
			reservation.CantidadAdultos = &num
			log.Printf("Cantidad de adultos extraída: %d", num)
		}
	}

	// Intentar extraer cantidad de niños
	if strings.Contains(strings.ToLower(message), "niño") || strings.Contains(strings.ToLower(message), "niños") {
		var num int
		if _, err := fmt.Sscanf(message, "%d", &num); err == nil {
			reservation.CantidadNinhos = &num
			log.Printf("Cantidad de niños extraída: %d", num)
		}
	}

	// Actualizar el paso basado en la información disponible
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

// cleanResponseForFrontend removes all tool markers and internal reasoning from the message
// Keeps them in logs for debugging, but sends clean output to users
func cleanResponseForFrontend(message string) string {
	lines := strings.Split(message, "\n")
	var cleaned []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip marker lines
		if strings.HasPrefix(trimmed, "[RESULTADO DE") ||
			strings.HasPrefix(trimmed, "[FIN RESULTADO") ||
			strings.HasPrefix(trimmed, "[USE_TOOL:") ||
			strings.HasPrefix(trimmed, "[END_TOOL") {
			continue
		}

		// ✅ NEW: Skip error detail lines (SQL, stack traces, etc.)
		if strings.Contains(trimmed, "sql:") ||
			strings.Contains(trimmed, "no rows in result set") ||
			strings.Contains(trimmed, "error ejecutando herramienta") {
			continue
		}

		cleaned = append(cleaned, line)
	}

	result := strings.TrimSpace(strings.Join(cleaned, "\n"))

	// ✅ NEW: Replace technical errors with friendly message
	if strings.Contains(result, "Lo siento, hubo un error:") {
		result = "Lo siento, hubo un problema al procesar tu reserva. Por favor, intenta nuevamente o contacta con un agente humano para asistencia."
	}

	return result
}
