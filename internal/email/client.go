package email

import (
	"crypto/tls"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/wneessen/go-mail"
)

// Client representa el cliente de correo electrónico
type Client struct {
	host      string
	port      int
	user      string
	password  string
	fromName  string
	fromEmail string
}

// NewClient crea una nueva instancia del cliente de email
func NewClient(host, portStr, user, password, fromName, fromEmail string) (*Client, error) {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("puerto SMTP inválido: %w", err)
	}

	return &Client{
		host:      host,
		port:      port,
		user:      user,
		password:  password,
		fromName:  fromName,
		fromEmail: fromEmail,
	}, nil
}

// SendEmail envía un correo electrónico
func (c *Client) SendEmail(to, subject, htmlBody string) error {
	// Crear mensaje
	m := mail.NewMsg()

	// Configurar remitente
	if err := m.From(fmt.Sprintf("%s <%s>", c.fromName, c.fromEmail)); err != nil {
		return fmt.Errorf("error al configurar remitente: %w", err)
	}

	// Configurar destinatario
	if err := m.To(to); err != nil {
		return fmt.Errorf("error al configurar destinatario: %w", err)
	}

	// Configurar asunto
	m.Subject(subject)

	// Configurar cuerpo HTML
	m.SetBodyString(mail.TypeTextHTML, htmlBody)

	// Crear cliente SMTP
	log.Printf("SMTP: connecting to %s:%d as user=%s", c.host, c.port, c.user)

	client, err := mail.NewClient(c.host,
		mail.WithPort(c.port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(c.user),
		mail.WithPassword(c.password),
		mail.WithTLSPolicy(mail.TLSMandatory),
		mail.WithTLSConfig(&tls.Config{
			ServerName: c.host,
		}),
	)
	if err != nil {
		return fmt.Errorf("error al crear cliente SMTP (host=%s port=%d user=%s): %w", c.host, c.port, c.user, err)
	}

	// Enviar correo
	if err := client.DialAndSend(m); err != nil {
		// Añadir contexto útil al error sin exponer credenciales
		return fmt.Errorf("error al enviar correo (host=%s port=%d user=%s): %w", c.host, c.port, c.user, err)
	}

	return nil
}

// ReservaInfo contiene la información de la reserva para el email
type ReservaInfo struct {
	ID                int
	CodigoReserva     string
	ClienteEmail      string
	ClienteNombre     string
	CantidadAdultos   int
	CantidadNinhos    int
	FechaConfirmacion time.Time
	Subtotal          float64
	Descuento         float64
	Total             float64
	Habitaciones      []HabitacionInfo
}

// HabitacionInfo contiene la información de una habitación
type HabitacionInfo struct {
	Nombre       string
	Numero       string
	FechaEntrada time.Time
	FechaSalida  time.Time
	Precio       float64
	Noches       int
}

// SendReservaConfirmacion envía un correo de confirmación de reserva
func (c *Client) SendReservaConfirmacion(reserva ReservaInfo) error {
	subject := fmt.Sprintf("Confirmación de Reserva #%d - %s", reserva.ID, c.fromName)
	htmlBody := generarHTMLConfirmacion(reserva)

	return c.SendEmail(reserva.ClienteEmail, subject, htmlBody)
}

// generarHTMLConfirmacion genera el HTML del correo de confirmación
func generarHTMLConfirmacion(reserva ReservaInfo) string {
	// Generar lista de habitaciones
	habitacionesHTML := ""
	for _, hab := range reserva.Habitaciones {
		habitacionesHTML += fmt.Sprintf(`
			<tr>
				<td style="padding: 15px; border-bottom: 1px solid #e0e0e0;">
					<strong>%s</strong><br>
					Habitación N° %s
				</td>
				<td style="padding: 15px; border-bottom: 1px solid #e0e0e0; text-align: center;">
					%s
				</td>
				<td style="padding: 15px; border-bottom: 1px solid #e0e0e0; text-align: center;">
					%s
				</td>
				<td style="padding: 15px; border-bottom: 1px solid #e0e0e0; text-align: right;">
					%d noche(s)
				</td>
				<td style="padding: 15px; border-bottom: 1px solid #e0e0e0; text-align: right;">
					$%.2f
				</td>
			</tr>
		`,
			hab.Nombre,
			hab.Numero,
			hab.FechaEntrada.Format("02/01/2006"),
			hab.FechaSalida.Format("02/01/2006"),
			hab.Noches,
			hab.Precio*float64(hab.Noches),
		)
	}

	// Plantilla HTML completa
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="es">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Confirmación de Reserva</title>
</head>
<body style="margin: 0; padding: 0; font-family: Arial, sans-serif; background-color: #f4f4f4;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="background-color: #f4f4f4; padding: 20px;">
		<tr>
			<td align="center">
				<!-- Contenedor principal -->
				<table width="600" cellpadding="0" cellspacing="0" style="background-color: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
					<!-- Header -->
					<tr>
						<td style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 40px 20px; text-align: center;">
							<h1 style="color: #ffffff; margin: 0; font-size: 28px;">¡Reserva Confirmada!</h1>
							<p style="color: #ffffff; margin: 10px 0 0 0; font-size: 16px;">Gracias por reservar con nosotros</p>
						</td>
					</tr>

					<!-- Contenido -->
					<tr>
						<td style="padding: 40px 30px;">
							<!-- Información de la reserva -->
							<div style="background-color: #f8f9fa; border-left: 4px solid #667eea; padding: 20px; margin-bottom: 30px;">
								<h2 style="margin: 0 0 15px 0; color: #333; font-size: 20px;">Detalles de la Reserva</h2>
								<table width="100%%" cellpadding="0" cellspacing="0">
									<tr>
										<td style="padding: 8px 0;"><strong>ID de Reserva:</strong></td>
										<td style="padding: 8px 0; text-align: right;">#%d</td>
									</tr>
									<tr>
										<td style="padding: 8px 0;"><strong>Fecha de Confirmación:</strong></td>
										<td style="padding: 8px 0; text-align: right;">%s</td>
									</tr>
									<tr>
										<td style="padding: 8px 0;"><strong>Huéspedes:</strong></td>
										<td style="padding: 8px 0; text-align: right;">%d adulto(s)%s</td>
									</tr>
								</table>
							</div>

							<!-- Habitaciones -->
							<h3 style="color: #333; margin-bottom: 20px;">Habitaciones Reservadas</h3>
							<table width="100%%" cellpadding="0" cellspacing="0" style="border: 1px solid #e0e0e0; border-radius: 8px; overflow: hidden;">
								<thead>
									<tr style="background-color: #667eea; color: #ffffff;">
										<th style="padding: 15px; text-align: left;">Habitación</th>
										<th style="padding: 15px; text-align: center;">Check-in</th>
										<th style="padding: 15px; text-align: center;">Check-out</th>
										<th style="padding: 15px; text-align: right;">Noches</th>
										<th style="padding: 15px; text-align: right;">Precio</th>
									</tr>
								</thead>
								<tbody>
									%s
								</tbody>
							</table>

							<!-- Totales -->
							<div style="margin-top: 30px; padding: 20px; background-color: #f8f9fa; border-radius: 8px;">
								<table width="100%%" cellpadding="0" cellspacing="0">
									<tr>
										<td style="padding: 8px 0;"><strong>Subtotal:</strong></td>
										<td style="padding: 8px 0; text-align: right;">$%.2f</td>
									</tr>
									<tr>
										<td style="padding: 8px 0;"><strong>Descuento:</strong></td>
										<td style="padding: 8px 0; text-align: right; color: #28a745;">-$%.2f</td>
									</tr>
									<tr style="border-top: 2px solid #667eea;">
										<td style="padding: 15px 0 0 0;"><strong style="font-size: 18px;">Total:</strong></td>
										<td style="padding: 15px 0 0 0; text-align: right;"><strong style="font-size: 24px; color: #667eea;">$%.2f</strong></td>
									</tr>
								</table>
							</div>

							<!-- Información adicional -->
							<div style="margin-top: 30px; padding: 20px; background-color: #fff3cd; border-radius: 8px; border-left: 4px solid #ffc107;">
								<h4 style="margin: 0 0 10px 0; color: #856404;">Información Importante</h4>
								<ul style="margin: 0; padding-left: 20px; color: #856404;">
									<li>Presentar este correo al momento del check-in</li>
									<li>Check-in: 15:00 hrs | Check-out: 12:00 hrs</li>
									<li>Para cancelaciones, contactar con 48 horas de anticipación</li>
								</ul>
							</div>
						</td>
					</tr>

					<!-- Footer -->
					<tr>
						<td style="background-color: #f8f9fa; padding: 30px; text-align: center; border-top: 1px solid #e0e0e0;">
							<p style="margin: 0 0 10px 0; color: #666; font-size: 14px;">
								Si tiene alguna pregunta, no dude en contactarnos
							</p>
							<p style="margin: 0; color: #999; font-size: 12px;">
								Este es un correo automático, por favor no responder directamente
							</p>
						</td>
					</tr>
				</table>
			</td>
		</tr>
	</table>
</body>
</html>
	`,
		reserva.ID,
		reserva.FechaConfirmacion.Format("02/01/2006 15:04"),
		reserva.CantidadAdultos,
		func() string {
			if reserva.CantidadNinhos > 0 {
				return fmt.Sprintf(", %d niño(s)", reserva.CantidadNinhos)
			}
			return ""
		}(),
		habitacionesHTML,
		reserva.Subtotal,
		reserva.Descuento,
		reserva.Total,
	)

	return html
}
