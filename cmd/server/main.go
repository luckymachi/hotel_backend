package main

import (
	"database/sql"
	"log"

	"github.com/Maxito7/hotel_backend/internal/application"
	"github.com/Maxito7/hotel_backend/internal/config"
	"github.com/Maxito7/hotel_backend/internal/email"
	"github.com/Maxito7/hotel_backend/internal/infrastructure/repository"
	handlers "github.com/Maxito7/hotel_backend/internal/interfaces/http"
	"github.com/Maxito7/hotel_backend/internal/openai"
	"github.com/Maxito7/hotel_backend/internal/scheduler"
	services "github.com/Maxito7/hotel_backend/internal/service"
	"github.com/Maxito7/hotel_backend/internal/tavily"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	db, err := sql.Open("postgres", cfg.GetDBConnString())
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
		ExposeHeaders:    "Content-Length",
		MaxAge:           86400,
	}))

	// Habitaciones
	habitacionRepo := repository.NewHabitacionRepository(db)
	habitacionService := application.NewHabitacionService(habitacionRepo)
	habitacionHandler := handlers.NewHabitacionHandler(habitacionService)

	// Search
	tavilyClient := tavily.NewClient(cfg.TavilyAPIKey)
	searchService := application.NewSearchService(tavilyClient)
	searchHandler := handlers.NewSearchHandler(searchService)

	// Servicios
	servicioRepo := repository.NewServicioRepository(db)
	servicioService := application.NewServicioService(servicioRepo)
	servicioHandler := handlers.NewServicioHandler(servicioService)

	// Chatbot - NUEVO
	openaiClient := openai.NewClient(cfg.OpenAIAPIKey)
	chatbotRepo := repository.NewChatbotRepository(db)
	chatbotService := application.NewChatbotService(chatbotRepo, openaiClient, habitacionRepo, tavilyClient, cfg.HotelLocation, searchService)
	chatbotHandler := handlers.NewChatbotHandler(chatbotService)

	// Email Client
	emailClient, err := email.NewClient(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUser,
		cfg.SMTPPassword,
		cfg.SMTPFromName,
		cfg.SMTPFromEmail,
	)
	if err != nil {
		log.Printf("Warning: Email client initialization failed: %v", err)
		emailClient = nil // Continuar sin email
	}

	// Contacto (después del email client)
	contactRepo := repository.NewContactRepository(db)
	contactService := application.NewContactService(contactRepo, emailClient)
	contactHandler := handlers.NewContactHandler(contactService)

	// Personas y Clientes (necesarios para reservas y encuestas)
	personRepo := repository.NewPersonRepository(db)
	clientRepo := repository.NewClientRepository(db)
	personService := application.NewPersonService(personRepo)
	personHandler := handlers.NewPersonHandler(personService)

	// Reservas (repositorios)
	paymentRepo := repository.NewPaymentRepository(db)
	reservaRepo := repository.NewReservaRepository(db)
	reservaHabitacionRepo := repository.NewReservaHabitacionRepository(db)
	reservationGuestRepo := repository.NewReservationGuestRepository(db)

	// Encuestas de satisfacción (crear ANTES de ReservaService)
	surveyRepo := repository.NewSatisfactionSurveyRepository(db)
	tokenRepo := repository.NewSurveyTokenRepository(db)
	surveyService := application.NewSatisfactionSurveyService(surveyRepo, reservaRepo, tokenRepo)
	surveyHandler := handlers.NewSatisfactionSurveyHandler(surveyService)

	// Reservas (servicio - ahora puede usar surveyService)
	reservaService := application.NewReservaService(reservaRepo, reservaHabitacionRepo, habitacionRepo, personRepo, clientRepo, paymentRepo, reservationGuestRepo, emailClient, surveyService)
	reservaHandler := handlers.NewReservaHandler(reservaService)

	// Scheduler para actualizar reservas completadas automáticamente
	reservationScheduler := scheduler.NewReservationScheduler(reservaRepo)
	reservationScheduler.Start()

	// S3
	S3Service, err := services.NewS3Service()
	S3Handler := handlers.NewS3Handler(S3Service)

	api := app.Group("/api")

	// Rutas existentes
	habitaciones := api.Group("/habitaciones")
	habitaciones.Get("/", habitacionHandler.GetAllRooms)
	habitaciones.Post("/", habitacionHandler.CreateRoom)
	habitaciones.Get("/tipos", habitacionHandler.GetRoomTypes)
	habitaciones.Get("/disponibles", habitacionHandler.GetAvailableRooms)
	habitaciones.Get("/fechas-bloqueadas", habitacionHandler.GetFechasBloqueadas)

	// Public amenities list
	api.Get("/amenities", habitacionHandler.ListAmenities)

	// Admin routes for room types
	habitaciones.Post("/tipos", habitacionHandler.CreateRoomType)
	habitaciones.Get("/tipos/:id", habitacionHandler.GetRoomTypeByID)
	habitaciones.Put("/tipos/:id", habitacionHandler.UpdateRoomType)
	habitaciones.Put("/tipos/:id/images", habitacionHandler.UpdateRoomTypeImages)
	habitaciones.Delete("/tipos/:id", habitacionHandler.DeleteRoomType)
	habitaciones.Put("/tipos/:id/amenities", habitacionHandler.SetRoomTypeAmenities)

	// Room item routes
	habitaciones.Get("/:id", habitacionHandler.GetRoomByID)
	habitaciones.Put("/:id", habitacionHandler.UpdateRoom)
	habitaciones.Delete("/:id", habitacionHandler.DeleteRoom)

	// duplicate earlier listing route (kept)
	habitaciones.Get("/tipos", habitacionHandler.GetRoomTypes)

	api.Post("/search", searchHandler.Search)

	contacto := api.Group("/contact")
	contacto.Post("/", contactHandler.Create)
	contacto.Get("/", contactHandler.List)
	contacto.Patch("/:id/estado", contactHandler.UpdateEstado)

	// Rutas de servicios
	servicios := api.Group("/servicios")
	servicios.Get("/all", servicioHandler.GetAllServices)

	// Rutas del chatbot - NUEVO
	chatbot := api.Group("/chatbot")
	chatbot.Post("/chat", chatbotHandler.Chat)
	chatbot.Get("/conversation/:id", chatbotHandler.GetConversation)
	chatbot.Get("/client/:clienteId/conversations", chatbotHandler.GetClientConversations)

	// Rutas de reservas
	reservas := api.Group("/reservas")
	reservas.Post("/", reservaHandler.CreateReserva)
	reservas.Get("/:id", reservaHandler.GetReservaByID)
	reservas.Get("/cliente/:clienteId", reservaHandler.GetReservasCliente)
	reservas.Patch("/:id/estado", reservaHandler.UpdateReservaEstado)
	reservas.Post("/:id/cancelar", reservaHandler.CancelarReserva)
	reservas.Post("/:id/confirmar", reservaHandler.ConfirmarReserva)
	reservas.Post("/:id/confirmar-pago", reservaHandler.ConfirmarPago) // NUEVO: Confirma pago y envía email
	reservas.Post("/verificar-disponibilidad", reservaHandler.VerificarDisponibilidad)
	reservas.Get("/rango", reservaHandler.GetReservasEnRango)

	// Rutas de personas
	personas := api.Group("/personas")
	personas.Get("/buscar", personHandler.GetPersonByDocumentNumber)

	// Rutas de encuestas de satisfacción
	surveys := api.Group("/encuestas")
	surveys.Post("/", surveyHandler.CreateSurvey)                                // Crear encuesta con token
	surveys.Get("/validar/:token", surveyHandler.ValidateToken)                  // Validar token
	surveys.Get("/reserva/:reservationId", surveyHandler.GetSurveyByReservation) // Obtener por reserva
	surveys.Get("/cliente/:clientId", surveyHandler.GetSurveysByClient)          // Obtener por cliente
	surveys.Get("/all", surveyHandler.GetAllSurveys)                             // Obtener todas (con paginación)
	surveys.Get("/promedios", surveyHandler.GetAverageScores)                    // Obtener promedios
	surveys.Get("/mejores", surveyHandler.GetTopRatedSurveys)                    // Obtener mejores para landing page

	// Rutas de S3
	s3 := api.Group("/upload")
	s3.Post("/imagenes", S3Handler.HandleUploadFile)

	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := app.Listen(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
