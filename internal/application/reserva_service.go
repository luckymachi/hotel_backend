package application

import (
	"fmt"
	"time"

	"github.com/Maxito7/hotel_backend/internal/domain"
	"github.com/Maxito7/hotel_backend/internal/email"
)

// ReservationVerification contiene información completa para verificar una reserva
type ReservationVerification struct {
	Reservation      *domain.Reserva   `json:"reservation"`
	Client           *domain.Client    `json:"client"`
	Person           *domain.Person    `json:"person"`
	Rooms            []RoomDetails     `json:"rooms"`
	Payments         []domain.Payment  `json:"payments,omitempty"`
	ConversationID   *string           `json:"conversationId,omitempty"`
	VerificationTime time.Time         `json:"verificationTime"`
}

// RoomDetails contiene detalles de habitación para verificación
type RoomDetails struct {
	RoomID       int       `json:"roomId"`
	RoomNumber   string    `json:"roomNumber"`
	RoomName     string    `json:"roomName"`
	RoomType     string    `json:"roomType"`
	CheckInDate  time.Time `json:"checkInDate"`
	CheckOutDate time.Time `json:"checkOutDate"`
	Price        float64   `json:"price"`
	Nights       int       `json:"nights"`
	TotalPrice   float64   `json:"totalPrice"`
}

type ReservaService struct {
	reservaRepo           domain.ReservaRepository
	reservaHabitacionRepo domain.ReservaHabitacionRepository
	habitacionRepo        domain.HabitacionRepository
	personRepo            domain.PersonRepository
	clientRepo            domain.ClientRepository
	paymentRepo           domain.PaymentRepository
	reservationGuestRepo  domain.ReservationGuestRepository
	emailClient           *email.Client
	surveyService         *SatisfactionSurveyService
}

// NewReservaService crea una nueva instancia del servicio de reservas
func NewReservaService(
	reservaRepo domain.ReservaRepository,
	reservaHabitacionRepo domain.ReservaHabitacionRepository,
	habitacionRepo domain.HabitacionRepository,
	personRepo domain.PersonRepository,
	clientRepo domain.ClientRepository,
	paymentRepo domain.PaymentRepository,
	reservationGuestRepo domain.ReservationGuestRepository,
	emailClient *email.Client,
	surveyService *SatisfactionSurveyService,
) *ReservaService {
	return &ReservaService{
		reservaRepo:           reservaRepo,
		reservaHabitacionRepo: reservaHabitacionRepo,
		habitacionRepo:        habitacionRepo,
		personRepo:            personRepo,
		clientRepo:            clientRepo,
		paymentRepo:           paymentRepo,
		reservationGuestRepo:  reservationGuestRepo,
		emailClient:           emailClient,
		surveyService:         surveyService,
	}
}

// CreateReserva crea una nueva reserva validando disponibilidad
func (s *ReservaService) CreateReserva(reserva *domain.Reserva) error {
	// Validar que la reserva tenga habitaciones
	if len(reserva.Habitaciones) == 0 {
		return fmt.Errorf("la reserva debe tener al menos una habitación")
	}

	// Validar fechas y disponibilidad de cada habitación
	for i, hab := range reserva.Habitaciones {
		// Validar que fecha de salida sea posterior a fecha de entrada
		if !hab.FechaSalida.After(hab.FechaEntrada) {
			return fmt.Errorf("la fecha de salida debe ser posterior a la fecha de entrada para la habitación %d", hab.HabitacionID)
		}

		// Verificar disponibilidad
		disponible, err := s.reservaHabitacionRepo.VerificarDisponibilidad(
			hab.HabitacionID,
			hab.FechaEntrada,
			hab.FechaSalida,
		)
		if err != nil {
			return fmt.Errorf("error al verificar disponibilidad: %w", err)
		}

		if !disponible {
			return fmt.Errorf("la habitación %d no está disponible para las fechas seleccionadas", hab.HabitacionID)
		}

		// Validar que se haya proporcionado un precio
		if hab.Precio <= 0 {
			return fmt.Errorf("el precio de la habitación %d debe ser mayor a 0", hab.HabitacionID)
		}

		reserva.Habitaciones[i].Precio = hab.Precio
	}

	// Calcular subtotal solo si no fue proporcionado
	if reserva.Subtotal <= 0 {
		subtotal := 0.0
		for _, hab := range reserva.Habitaciones {
			// Calcular días de estancia
			dias := hab.FechaSalida.Sub(hab.FechaEntrada).Hours() / 24
			if dias < 1 {
				dias = 1
			}
			subtotal += hab.Precio * dias
		}
		reserva.Subtotal = subtotal
	}

	// Si no se especificó descuento, establecerlo en 0
	if reserva.Descuento < 0 {
		reserva.Descuento = 0
	}

	// Validar que el descuento no sea mayor al subtotal
	if reserva.Descuento > reserva.Subtotal {
		return fmt.Errorf("el descuento no puede ser mayor al subtotal")
	}

	// Establecer fecha de confirmación si no se proporcionó
	if reserva.FechaConfirmacion.IsZero() {
		reserva.FechaConfirmacion = time.Now()
	}

	// Establecer estado inicial si no se especificó
	if reserva.Estado == "" {
		reserva.Estado = domain.ReservaPendiente
	}

	// Crear la reserva
	if err := s.reservaRepo.CreateReserva(reserva); err != nil {
		return fmt.Errorf("error al crear reserva: %w", err)
	}

	return nil
}

// FindAvailableRoomByType busca una habitación disponible de un tipo específico para las fechas dadas
func (s *ReservaService) FindAvailableRoomByType(roomTypeID int, fechaEntrada, fechaSalida time.Time) (int, error) {
	return s.habitacionRepo.FindAvailableRoomByType(roomTypeID, fechaEntrada, fechaSalida)
}

// CreateReservaWithClient crea una reserva buscando/creando primero el cliente
func (s *ReservaService) CreateReservaWithClient(person *domain.Person, reserva *domain.Reserva) error {
	// 1. Buscar persona por document_number
	existingPerson, err := s.personRepo.FindByDocumentNumber(person.DocumentNumber)
	if err != nil {
		return fmt.Errorf("error al buscar persona: %w", err)
	}

	var personID int

	// 2. Si no existe, crear la persona
	if existingPerson == nil {
		if err := s.personRepo.Create(person); err != nil {
			return fmt.Errorf("error al crear persona: %w", err)
		}
		personID = person.PersonID
	} else {
		personID = existingPerson.PersonID
	}

	// 3. Buscar el client_id usando person_id
	clientID, err := s.clientRepo.GetClientIDByPersonID(personID)
	if err != nil {
		// Si no existe el cliente, crearlo
		clientID, err = s.clientRepo.Create(personID, domain.CaptureChannelWebpage, domain.CaptureStatusCliente, reserva.CantidadNinhos)
		if err != nil {
			return fmt.Errorf("error al crear cliente: %w", err)
		}
	}

	// 4. Asignar el client_id a la reserva
	reserva.ClienteID = clientID

	// 5. Crear la reserva con el resto de la lógica existente
	return s.CreateReserva(reserva)
}

// CreateReservaWithClientAndPayment crea una reserva con cliente, huéspedes adicionales y pago
func (s *ReservaService) CreateReservaWithClientAndPayment(
	person *domain.Person,
	reserva *domain.Reserva,
	huespedes []domain.Person,
	payment *domain.Payment,
) error {
	// 1. Buscar persona por document_number
	existingPerson, err := s.personRepo.FindByDocumentNumber(person.DocumentNumber)
	if err != nil {
		return fmt.Errorf("error al buscar persona: %w", err)
	}

	var personID int

	// 2. Si no existe, crear la persona con los datos del JSON
	if existingPerson == nil {
		if err := s.personRepo.Create(person); err != nil {
			return fmt.Errorf("error al crear persona: %w", err)
		}
		personID = person.PersonID
	} else {
		// Si existe, actualizar sus datos con la información del JSON
		// Esto asegura que el email y otros datos estén actualizados
		existingPerson.Name = person.Name
		existingPerson.FirstSurname = person.FirstSurname
		existingPerson.SecondSurname = person.SecondSurname
		existingPerson.Gender = person.Gender
		existingPerson.Email = person.Email // ← IMPORTANTE: Actualizar el email del JSON
		existingPerson.Phone1 = person.Phone1
		existingPerson.Phone2 = person.Phone2
		existingPerson.ReferenceCity = person.ReferenceCity
		existingPerson.ReferenceCountry = person.ReferenceCountry
		existingPerson.BirthDate = person.BirthDate

		if err := s.personRepo.Update(existingPerson); err != nil {
			return fmt.Errorf("error al actualizar persona: %w", err)
		}
		personID = existingPerson.PersonID
	}

	// 3. Buscar el client_id usando person_id
	clientID, err := s.clientRepo.GetClientIDByPersonID(personID)
	if err != nil {
		// Si no existe el cliente, crearlo
		clientID, err = s.clientRepo.Create(personID, domain.CaptureChannelWebpage, domain.CaptureStatusCliente, reserva.CantidadNinhos)
		if err != nil {
			return fmt.Errorf("error al crear cliente: %w", err)
		}
	}

	// 4. Asignar el client_id a la reserva
	reserva.ClienteID = clientID

	// 5. Crear la reserva
	if err := s.CreateReserva(reserva); err != nil {
		return err
	}

	// 6. Crear los huéspedes adicionales (si existen)
	if len(huespedes) > 0 {
		var personIDs []int

		for i := range huespedes {
			// Buscar si el huésped ya existe por document_number
			existingGuest, err := s.personRepo.FindByDocumentNumber(huespedes[i].DocumentNumber)
			if err != nil {
				return fmt.Errorf("error al buscar huésped %d: %w", i+1, err)
			}

			var guestPersonID int

			if existingGuest == nil {
				// Si no existe, crear la persona
				if err := s.personRepo.Create(&huespedes[i]); err != nil {
					return fmt.Errorf("error al crear huésped %d: %w", i+1, err)
				}
				guestPersonID = huespedes[i].PersonID
			} else {
				// Si existe, actualizar sus datos con la información del JSON
				existingGuest.Name = huespedes[i].Name
				existingGuest.FirstSurname = huespedes[i].FirstSurname
				existingGuest.SecondSurname = huespedes[i].SecondSurname
				existingGuest.Gender = huespedes[i].Gender
				existingGuest.Email = huespedes[i].Email   // ✅ Actualizar email
				existingGuest.Phone1 = huespedes[i].Phone1 // ✅ Actualizar teléfono
				existingGuest.BirthDate = huespedes[i].BirthDate

				if err := s.personRepo.Update(existingGuest); err != nil {
					return fmt.Errorf("error al actualizar huésped %d: %w", i+1, err)
				}
				guestPersonID = existingGuest.PersonID
			}

			personIDs = append(personIDs, guestPersonID)
		}

		// Crear las relaciones en reservation_guest
		if err := s.reservationGuestRepo.CreateMultiple(reserva.ID, personIDs); err != nil {
			return fmt.Errorf("reserva creada pero error al registrar huéspedes: %w", err)
		}
	}

	// 7. Si se proporcionó pago, crearlo
	if payment != nil {
		payment.ReservationID = reserva.ID
		if err := s.paymentRepo.Create(payment); err != nil {
			// Si falla el pago, la reserva ya fue creada
			// Puedes decidir si quieres rollback o solo registrar el error
			return fmt.Errorf("reserva creada pero error al registrar pago: %w", err)
		}
	}

	// 8. Generar token de encuesta y enviar email (solo si surveyService está disponible)
	if s.surveyService != nil {
		s.generarYEnviarEncuesta(reserva.ID, clientID, person.Email)
	}

	return nil
}

// GetReservaByID obtiene una reserva por su ID
func (s *ReservaService) GetReservaByID(id int) (*domain.Reserva, error) {
	return s.reservaRepo.GetReservaByID(id)
}

// GetReservasCliente obtiene todas las reservas de un cliente
func (s *ReservaService) GetReservasCliente(clienteID int) ([]domain.Reserva, error) {
	return s.reservaRepo.GetReservasCliente(clienteID)
}

// UpdateReservaEstado actualiza el estado de una reserva
func (s *ReservaService) UpdateReservaEstado(id int, estado domain.EstadoReserva) error {
	// Validar que el estado sea válido
	validEstados := map[domain.EstadoReserva]bool{
		domain.ReservaPendiente:  true,
		domain.ReservaConfirmada: true,
		domain.ReservaCancelada:  true,
		domain.ReservaCompletada: true,
	}

	if !validEstados[estado] {
		return fmt.Errorf("estado de reserva inválido: %s", estado)
	}

	// Obtener la reserva actual
	reserva, err := s.reservaRepo.GetReservaByID(id)
	if err != nil {
		return fmt.Errorf("error al obtener reserva: %w", err)
	}

	// Si se está cancelando, actualizar el estado de las habitaciones
	if estado == domain.ReservaCancelada {
		for _, hab := range reserva.Habitaciones {
			if err := s.reservaHabitacionRepo.UpdateReservaHabitacionEstado(
				id,
				hab.HabitacionID,
				0, // Estado cancelado
			); err != nil {
				return fmt.Errorf("error al cancelar habitación: %w", err)
			}
		}
	}

	return s.reservaRepo.UpdateReservaEstado(id, estado)
}

// CancelarReserva cancela una reserva completa
func (s *ReservaService) CancelarReserva(id int) error {
	return s.UpdateReservaEstado(id, domain.ReservaCancelada)
}

// ConfirmarReserva confirma una reserva pendiente y envía email de confirmación
func (s *ReservaService) ConfirmarReserva(id int) error {
	return s.confirmarReservaInternal(id, true) // true = enviar email
}

// ConfirmarReservaSinEmail confirma una reserva sin enviar email
func (s *ReservaService) ConfirmarReservaSinEmail(id int) error {
	return s.confirmarReservaInternal(id, false) // false = no enviar email
}

// confirmarReservaInternal es el método interno que maneja la confirmación
func (s *ReservaService) confirmarReservaInternal(id int, enviarEmail bool) error {
	// Actualizar estado
	if err := s.UpdateReservaEstado(id, domain.ReservaConfirmada); err != nil {
		return err
	}

	// Solo enviar email si se solicita
	if !enviarEmail {
		return nil
	}

	// Obtener datos completos de la reserva para el email
	reserva, err := s.GetReservaByID(id)
	if err != nil {
		// Log error pero no fallar si el email falla
		fmt.Printf("Error al obtener reserva para email: %v\n", err)
		return nil // La confirmación ya se hizo, solo falló el email
	}

	// Enviar email de confirmación
	if s.emailClient != nil {
		if err := s.enviarEmailConfirmacion(reserva); err != nil {
			// Log error pero no fallar
			fmt.Printf("Error al enviar email de confirmación: %v\n", err)
		}
	}

	return nil
}

// enviarEmailConfirmacion envía el email de confirmación de la reserva
func (s *ReservaService) enviarEmailConfirmacion(reserva *domain.Reserva) error {
	// Obtener el email de la persona asociada al cliente
	email, err := s.clientRepo.GetPersonEmailByClientID(reserva.ClienteID)
	if err != nil {
		return fmt.Errorf("error al obtener email del cliente: %w", err)
	}

	// Construir el contenido del email
	subject := fmt.Sprintf("Confirmación de Reserva #%d - Hotel Inca", reserva.ID)

	// Crear el cuerpo del email en HTML
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
				.content { padding: 20px; background-color: #f9f9f9; }
				.footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
				.details { background-color: white; padding: 15px; margin: 10px 0; border-radius: 5px; }
				.total { font-size: 18px; font-weight: bold; color: #4CAF50; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Hotel Inca</h1>
					<h2>Confirmación de Reserva</h2>
				</div>
				<div class="content">
					<p>Estimado/a cliente,</p>
					<p>Su reserva ha sido confirmada exitosamente. A continuación los detalles:</p>
					
					<div class="details">
						<h3>Detalles de la Reserva</h3>
						<p><strong>Número de Reserva:</strong> #%d</p>
						<p><strong>Fecha de Confirmación:</strong> %s</p>
						<p><strong>Cantidad de Adultos:</strong> %d</p>
						<p><strong>Cantidad de Niños:</strong> %d</p>
						<p><strong>Estado:</strong> %s</p>
					</div>
					
					<div class="details">
						<h3>Información de Pago</h3>
						<p><strong>Subtotal:</strong> S/. %.2f</p>
						<p><strong>Descuento:</strong> S/. %.2f</p>
						<p class="total">Total: S/. %.2f</p>
					</div>
					
					<p>Gracias por confiar en Hotel Inca. Esperamos verle pronto.</p>
				</div>
				<div class="footer">
					<p>Hotel Inca - Reservas</p>
					<p>Este es un correo automático, por favor no responder.</p>
				</div>
			</div>
		</body>
		</html>
	`,
		reserva.ID,
		reserva.FechaConfirmacion.Format("02/01/2006 15:04"),
		reserva.CantidadAdultos,
		reserva.CantidadNinhos,
		reserva.Estado,
		reserva.Subtotal,
		reserva.Descuento,
		reserva.Subtotal-reserva.Descuento,
	)

	// Enviar el email
	if err := s.emailClient.SendEmail(email, subject, htmlBody); err != nil {
		return fmt.Errorf("error al enviar email: %w", err)
	}

	return nil
}

// CompletarReserva marca una reserva como completada
func (s *ReservaService) CompletarReserva(id int) error {
	return s.UpdateReservaEstado(id, domain.ReservaCompletada)
}

// VerificarDisponibilidad verifica si una habitación está disponible
func (s *ReservaService) VerificarDisponibilidad(habitacionID int, fechaEntrada, fechaSalida time.Time) (bool, error) {
	if !fechaSalida.After(fechaEntrada) {
		return false, fmt.Errorf("la fecha de salida debe ser posterior a la fecha de entrada")
	}

	return s.reservaHabitacionRepo.VerificarDisponibilidad(habitacionID, fechaEntrada, fechaSalida)
}

// GetReservasEnRango obtiene todas las reservas en un rango de fechas
func (s *ReservaService) GetReservasEnRango(fechaInicio, fechaFin time.Time) ([]domain.ReservaHabitacion, error) {
	if !fechaFin.After(fechaInicio) {
		return nil, fmt.Errorf("la fecha fin debe ser posterior a la fecha inicio")
	}

	return s.reservaHabitacionRepo.GetReservasEnRango(fechaInicio, fechaFin)
}

// generarYEnviarEncuesta genera un token de encuesta y envía el email al cliente
func (s *ReservaService) generarYEnviarEncuesta(reservaID, clienteID int, email string) {
	// Generar token de encuesta
	token, err := s.surveyService.CreateTokenForReservation(reservaID, clienteID)
	if err != nil {
		// Log el error pero no fallar la creación de reserva
		fmt.Printf("Error al crear token de encuesta: %v\n", err)
		return
	}

	// Construir link de encuesta (ajusta la URL según tu dominio)
	surveyLink := fmt.Sprintf("http://localhost:3000/encuesta?token=%s", token.Token)

	// Construir el email
	emailBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
		</head>
		<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<div style="background-color: #f8f9fa; padding: 20px; border-radius: 10px;">
				<h2 style="color: #333;">¡Gracias por tu reserva!</h2>
				
				<p style="color: #555; font-size: 16px;">
					Tu reserva ha sido confirmada exitosamente.
				</p>
				
				<p style="color: #555; font-size: 16px;">
					Cuando completes tu estadía, nos encantaría conocer tu opinión. 
					Por favor, completa nuestra breve encuesta de satisfacción:
				</p>
				
				<div style="text-align: center; margin: 30px 0;">
					<a href="%s" style="
						background-color: #4CAF50;
						color: white;
						padding: 15px 30px;
						text-decoration: none;
						border-radius: 5px;
						font-size: 18px;
						display: inline-block;
					">
						Completar Encuesta
					</a>
				</div>
				
				<p style="color: #999; font-size: 14px;">
					Este link es válido por 30 días.
				</p>
				
				<p style="color: #555; font-size: 16px;">
					Si tienes alguna pregunta o necesitas asistencia, 
					no dudes en contactarnos.
				</p>
				
				<p style="color: #333; font-size: 16px;">
					Saludos,<br>
					<strong>Equipo del Hotel</strong>
				</p>
			</div>
		</body>
		</html>
	`, surveyLink)

	// Enviar el email
	if s.emailClient != nil {
		err = s.emailClient.SendEmail(
			email,
			"Encuesta de Satisfacción - Tu Opinión es Importante",
			emailBody,
		)

		if err != nil {
			// Log el error pero no fallar la creación de reserva
			fmt.Printf("Error al enviar email de encuesta: %v\n", err)
		} else {
			fmt.Printf("Email de encuesta enviado a: %s\n", email)
		}
	}
}

// VerifyReservation obtiene información completa de una reserva para verificación
func (s *ReservaService) VerifyReservation(reservationID int) (*ReservationVerification, error) {
	// 1. Obtener la reserva
	reserva, err := s.reservaRepo.GetReservaByID(reservationID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener reserva: %w", err)
	}

	// 2. Obtener información del cliente
	client, err := s.clientRepo.GetByID(reserva.ClienteID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener cliente: %w", err)
	}

	// 3. Obtener información de la persona
	person, err := s.personRepo.GetByID(client.PersonID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener persona: %w", err)
	}

	// 4. Obtener detalles de las habitaciones
	rooms := make([]RoomDetails, 0)
	for _, resRoom := range reserva.Habitaciones {
		// Obtener la habitación
		room, err := s.habitacionRepo.GetHabitacionByID(resRoom.HabitacionID)
		if err != nil {
			// Log error pero continuar
			fmt.Printf("Error al obtener habitación %d: %v\n", resRoom.HabitacionID, err)
			continue
		}

		// Calcular noches
		nights := int(resRoom.FechaSalida.Sub(resRoom.FechaEntrada).Hours() / 24)
		if nights < 1 {
			nights = 1
		}

		roomDetail := RoomDetails{
			RoomID:       resRoom.HabitacionID,
			RoomNumber:   room.Numero,
			RoomName:     room.Nombre,
			RoomType:     room.TipoHabitacionID.String(), // Convertir a string, puede requerir lookup
			CheckInDate:  resRoom.FechaEntrada,
			CheckOutDate: resRoom.FechaSalida,
			Price:        resRoom.Precio,
			Nights:       nights,
			TotalPrice:   resRoom.Precio * float64(nights),
		}
		rooms = append(rooms, roomDetail)
	}

	// 5. Obtener pagos relacionados
	payments, err := s.paymentRepo.GetByReservationID(reservationID)
	if err != nil {
		// Los pagos son opcionales, solo log el error
		fmt.Printf("Error al obtener pagos para reserva %d: %v\n", reservationID, err)
		payments = []domain.Payment{}
	}

	// 6. Construir respuesta de verificación
	verification := &ReservationVerification{
		Reservation:      reserva,
		Client:           client,
		Person:           person,
		Rooms:            rooms,
		Payments:         payments,
		ConversationID:   nil, // Podría agregarse buscando en conversation_history
		VerificationTime: time.Now(),
	}

	return verification, nil
}
