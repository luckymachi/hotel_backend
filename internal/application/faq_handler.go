package application

import (
	"strings"
)

// FAQHandler maneja respuestas rápidas a preguntas frecuentes
type FAQHandler struct {
	location string
}

// NewFAQHandler crea un nuevo manejador de FAQ
func NewFAQHandler(location string) *FAQHandler {
	return &FAQHandler{
		location: location,
	}
}

// FAQEntry representa una entrada de preguntas frecuentes
type FAQEntry struct {
	Keywords []string
	Answer   string
}

// GetQuickResponse intenta encontrar una respuesta rápida para el mensaje
func (fh *FAQHandler) GetQuickResponse(message string) (string, bool) {
	messageLower := strings.ToLower(message)

	// Definir FAQs
	faqs := []FAQEntry{
		{
			Keywords: []string{"horario check-in", "hora de entrada", "hora check in", "check in", "cuando puedo entrar"},
			Answer: "✅ El horario de check-in es a partir de las **14:00 hrs** (2:00 PM).\n\n" +
				"Si llegas antes, con gusto podemos guardar tu equipaje mientras preparamos tu habitación.\n\n" +
				"¿Te gustaría hacer una reserva?",
		},
		{
			Keywords: []string{"horario check-out", "hora de salida", "hora check out", "check out", "cuando debo salir"},
			Answer: "✅ El horario de check-out es hasta las **12:00 hrs** (12:00 PM).\n\n" +
				"Si necesitas un late check-out, consulta disponibilidad con recepción.\n\n" +
				"¿Puedo ayudarte con algo más?",
		},
		{
			Keywords: []string{"wifi", "wi-fi", "internet", "conexión", "conexion a internet"},
			Answer: "✅ ¡Por supuesto! Contamos con **WiFi gratuito** en todas las áreas del hotel.\n\n" +
				"La contraseña se proporciona al momento del check-in.\n\n" +
				"¿Necesitas información sobre algo más?",
		},
		{
			Keywords: []string{"estacionamiento", "parking", "donde aparcar", "dónde aparcar", "parqueo"},
			Answer: "✅ Sí, contamos con **estacionamiento disponible** para nuestros huéspedes.\n\n" +
				"El servicio está incluido en tu reserva.\n\n" +
				"¿Te gustaría hacer una reserva?",
		},
		{
			Keywords: []string{"desayuno", "breakfast", "incluye desayuno", "comida"},
			Answer: "✅ Ofrecemos un delicioso **desayuno buffet** para nuestros huéspedes.\n\n" +
				"Horario: 7:00 AM - 10:00 AM\n" +
				"El desayuno está incluido en todas nuestras tarifas.\n\n" +
				"¿Quieres ver nuestras habitaciones disponibles?",
		},
		{
			Keywords: []string{"mascotas", "perro", "gato", "mascota", "animales", "pet"},
			Answer: "❌ Lo sentimos, actualmente **no permitimos mascotas** en el hotel.\n\n" +
				"Esta política nos ayuda a mantener la comodidad de todos nuestros huéspedes.\n\n" +
				"¿Puedo ayudarte con alguna otra consulta?",
		},
		{
			Keywords: []string{"ubicación", "ubicacion", "dirección", "direccion", "donde están", "dónde están", "cómo llegar", "como llegar"},
			Answer: "📍 Nuestra ubicación es: **" + fh.location + "**\n\n" +
				"Estamos en una zona céntrica con fácil acceso a las principales atracciones.\n\n" +
				"¿Necesitas información sobre transporte o te gustaría hacer una reserva?",
		},
		{
			Keywords: []string{"cancelación", "cancelacion", "política de cancelación", "politica de cancelacion", "puedo cancelar"},
			Answer: "✅ Política de Cancelación:\n\n" +
				"• **Cancelación gratuita** hasta 48 horas antes del check-in\n" +
				"• Cancelaciones posteriores tienen un cargo del 50% del total\n" +
				"• No-show (no presentarse): cargo del 100%\n\n" +
				"¿Te gustaría hacer una reserva?",
		},
		{
			Keywords: []string{"métodos de pago", "metodos de pago", "formas de pago", "cómo pagar", "como pagar", "tarjeta", "efectivo"},
			Answer: "💳 Aceptamos los siguientes métodos de pago:\n\n" +
				"• Efectivo (soles y dólares)\n" +
				"• Tarjetas de crédito (Visa, MasterCard, American Express)\n" +
				"• Tarjetas de débito\n" +
				"• Transferencia bancaria\n\n" +
				"¿Quieres proceder con una reserva?",
		},
		{
			Keywords: []string{"recepción 24 horas", "recepcion 24 horas", "atencion 24", "atención 24"},
			Answer: "✅ Sí, contamos con **recepción 24 horas** para atenderte en cualquier momento.\n\n" +
				"Nuestro personal está siempre disponible para ayudarte con lo que necesites.\n\n" +
				"¿Hay algo más en lo que pueda ayudarte?",
		},
		{
			Keywords: []string{"servicios", "qué servicios", "que servicios", "amenidades", "facilidades"},
			Answer: "✨ **Servicios del Hotel:**\n\n" +
				"• WiFi gratuito en todas las áreas\n" +
				"• Desayuno buffet incluido\n" +
				"• Recepción 24 horas\n" +
				"• Estacionamiento\n" +
				"• Servicio de habitaciones\n" +
				"• Limpieza diaria\n\n" +
				"¿Te gustaría conocer nuestras habitaciones disponibles?",
		},
		{
			Keywords: []string{"hola", "buenos días", "buenos dias", "buenas tardes", "buenas noches", "hi", "hello"},
			Answer: "¡Hola! 👋 Bienvenido a nuestro hotel. Soy tu asistente virtual.\n\n" +
				"Estoy aquí para ayudarte con:\n" +
				"• Información sobre habitaciones\n" +
				"• Hacer reservas\n" +
				"• Consultar disponibilidad\n" +
				"• Responder tus preguntas\n\n" +
				"¿En qué puedo ayudarte hoy?",
		},
		{
			Keywords: []string{"gracias", "muchas gracias", "perfecto", "excelente", "ok", "vale"},
			Answer: "¡De nada! 😊 Es un placer ayudarte.\n\n" +
				"Si necesitas algo más, no dudes en escribirme.\n\n" +
				"¿Hay algo más en lo que pueda asistirte?",
		},
		{
			Keywords: []string{"precio", "tarifa", "costo", "cuánto cuesta", "cuanto cuesta", "precios"},
			Answer: "💰 Nuestras tarifas varían según el tipo de habitación y la temporada.\n\n" +
				"Para darte un precio exacto, necesito saber:\n" +
				"• Fechas de entrada y salida\n" +
				"• Cantidad de personas\n\n" +
				"¿Me proporcionas esta información para consultar la disponibilidad y precio?",
		},
	}

	// Buscar coincidencias
	for _, faq := range faqs {
		for _, keyword := range faq.Keywords {
			if strings.Contains(messageLower, keyword) {
				return faq.Answer, true
			}
		}
	}

	return "", false
}

// ShouldUseFAQ determina si el mensaje es simple y puede responderse con FAQ
func (fh *FAQHandler) ShouldUseFAQ(message string) bool {
	// Si el mensaje es muy largo, probablemente no sea una FAQ simple
	if len(message) > 100 {
		return false
	}

	// Si contiene fechas específicas o números complejos, probablemente necesita procesamiento LLM
	if strings.Contains(message, "2024") || strings.Contains(message, "2025") || strings.Contains(message, "2026") {
		// Probablemente está especificando fechas, necesita LLM
		return false
	}

	return true
}
