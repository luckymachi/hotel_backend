# Chatbot V2: Sistema de Detección Automática de Intenciones

## 🎯 Problema Resuelto

### Problema Anterior
El chatbot dependía de que el modelo LLM (llama-3.1-8b-instant) usara el formato `[USE_TOOL: nombre]` para ejecutar herramientas. Esto causaba:
- ❌ El flujo se perdía, preguntando lo mismo repetidamente
- ❌ Las herramientas no se ejecutaban de manera consistente
- ❌ El modelo pequeño no seguía instrucciones complejas de tool calling
- ❌ Conversaciones circulares sin avanzar en el proceso de reserva

### Solución Implementada
**Detección Automática de Intenciones** - El sistema ahora detecta intenciones directamente del mensaje del usuario y ejecuta herramientas automáticamente, SIN depender del LLM.

## 🏗️ Arquitectura del Nuevo Sistema

```
Usuario escribe mensaje
        ↓
[IntentDetector] Analiza el mensaje
        ↓
Detecta automáticamente:
  - Fechas (check-in/check-out)
  - Cantidad de personas
  - Tipo de habitación seleccionado
  - Datos personales
  - Confirmación
        ↓
Ejecuta herramientas automáticamente:
  - check_availability (si hay fechas)
  - get_room_types (si pide opciones)
  - calculate_price (si selecciona habitación)
  - create_reservation (si confirma)
        ↓
Agrega resultados al contexto
        ↓
[LLM] Solo conversa usando los resultados
        ↓
Respuesta al usuario
```

## 📦 Componentes Nuevos

### 1. IntentDetector (`intent_detector.go`)

Detecta intenciones y extrae información del mensaje del usuario.

**Capacidades:**
- ✅ **Extracción de fechas**: Reconoce múltiples formatos
  - `2025-12-15` (YYYY-MM-DD)
  - `15/12/2025` (DD/MM/YYYY)
  - `15-12-2025` (DD-MM-YYYY)
  - `del 15 al 20 de diciembre` (texto natural)

- ✅ **Extracción de huéspedes**:
  - `2 adultos`, `3 adultos y 2 niños`
  - `5 personas` (asume adultos)

- ✅ **Selección de habitación**:
  - `habitación 2`, `tipo 3`, `opción 1`
  - `habitación doble`, `suite presidencial`

- ✅ **Datos personales**:
  - Detecta cuando el mensaje contiene email, teléfono y documento
  - Extrae automáticamente los datos

- ✅ **Confirmación**:
  - `sí`, `confirmo`, `ok`, `adelante`, `procede`

**Método principal:**
```go
func (d *IntentDetector) DetectAndProcess(
    message string,
    reservation *ReservationInProgress
) (*DetectedIntent, error)
```

### 2. ChatbotService V2 (`chatbot_service_v2.go`)

Nueva versión del servicio que usa detección automática.

**Método principal:**
```go
func (s *ChatbotService) ProcessMessageV2(req domain.ChatRequest) (*domain.ChatResponse, error)
```

**Flujo:**
1. Obtiene/crea conversación
2. **Detecta intenciones automáticamente**
3. **Ejecuta herramientas según lo detectado**
4. Actualiza estado de reserva
5. Agrega resultados de herramientas al contexto
6. Llama al LLM solo para conversar
7. Guarda conversación
8. Retorna respuesta

### 3. Prompt Simplificado

El nuevo prompt ya NO pide al LLM que decida qué herramientas usar:

```
Eres un asistente virtual...

IMPORTANTE:
- Toda la información que necesitas está en el contexto proporcionado
- NO inventes información que no esté en el contexto
- Si se te proporcionan RESULTADOS DE HERRAMIENTAS, úsalos para responder
- Sé conciso, amable y profesional

- Si ves resultados de CHECK_AVAILABILITY, informa al usuario
- Si ves resultados de CALCULATE_PRICE, menciona el precio
- Si ves resultados de CREATE_RESERVATION, confirma la reserva
- Guía al usuario según el paso actual
```

El LLM ahora solo:
- ✅ Lee los resultados de las herramientas
- ✅ Conversa amablemente con el usuario
- ✅ Guía al usuario en el flujo

## 🚀 Ejemplo de Flujo Completo

### Mensaje 1: Inicio de Reserva
```
Usuario: "Hola, quiero reservar una habitación del 15 al 20 de diciembre para 2 personas"

[IntentDetector detecta]
- fechaEntrada: "2025-12-15"
- fechaSalida: "2025-12-20"
- cantidadAdultos: 2
- cantidadNinhos: 0

[Ejecuta automáticamente]
- check_availability("2025-12-15", "2025-12-20")

[Resultado de la herramienta]
"Habitaciones disponibles para 2025-12-15 - 2025-12-20:
 ✅ Suite Presidencial (ID: 1) - S/500 por noche
 ✅ Habitación Doble (ID: 2) - S/200 por noche"

[LLM recibe contexto con resultado]
Sistema: "...
[RESULTADO DE CHECK_AVAILABILITY]:
Habitaciones disponibles para 2025-12-15 - 2025-12-20:
 ✅ Suite Presidencial (ID: 1) - S/500 por noche
 ✅ Habitación Doble (ID: 2) - S/200 por noche
..."

[LLM responde]
Chatbot: "¡Perfecto! Tenemos disponibilidad para esas fechas.
Te muestro las opciones:
- Suite Presidencial: S/500 por noche
- Habitación Doble: S/200 por noche
¿Cuál prefieres?"
```

### Mensaje 2: Selección de Habitación
```
Usuario: "La habitación doble"

[IntentDetector detecta]
- tipoHabitacionId: 2

[Ejecuta automáticamente]
- calculate_price(tipoHabitacionId: 2, fechaEntrada: "2025-12-15", fechaSalida: "2025-12-20")

[Resultado]
"Cálculo de Precio:
Habitación: Habitación Doble
Precio por noche: S/200
Número de noches: 5
Total: S/1000"

[LLM responde]
Chatbot: "Excelente elección. El total sería:
- 5 noches x S/200 = S/1000

Para confirmar tu reserva, necesito los siguientes datos:
- Nombre completo
- Número de documento
- Email
- Teléfono"
```

### Mensaje 3: Datos Personales
```
Usuario: "Juan Pérez García, DNI 12345678, juan@email.com, 987654321"

[IntentDetector detecta]
- personalData:
  - nombre: "Juan"
  - primerApellido: "Pérez"
  - segundoApellido: "García"
  - numeroDocumento: "12345678"
  - correo: "juan@email.com"
  - telefono1: "987654321"
  - genero: "M"

[LLM responde]
Chatbot: "Perfecto, Juan. He recibido tus datos:
- Nombre: Juan Pérez García
- DNI: 12345678
- Email: juan@email.com
- Teléfono: 987654321

¿Confirmas la reserva?"
```

### Mensaje 4: Confirmación
```
Usuario: "Sí, confirmo"

[IntentDetector detecta]
- isConfirmation: true

[Ejecuta automáticamente]
- create_reservation(todos los datos recopilados)

[Resultado]
"✅ Reserva creada exitosamente!
Número de Reserva: #123
..."

[LLM responde]
Chatbot: "¡Reserva confirmada! ✅

Tu número de reserva es: #123
Te hemos enviado un email de confirmación a juan@email.com

Detalles:
- Check-in: 15/12/2025
- Check-out: 20/12/2025
- Habitación: Doble
- Total: S/1000

¡Gracias por tu reserva!"
```

## 🔍 Métodos de Detección

### Detección de Fechas

```go
// Patrones soportados
datePatterns := []string{
    `\d{4}-\d{2}-\d{2}`,      // 2025-12-15
    `\d{2}/\d{2}/\d{4}`,      // 15/12/2025
    `\d{2}-\d{2}-\d{4}`,      // 15-12-2025
}

// Texto natural
"del 15 al 20 de diciembre" → "2025-12-15" a "2025-12-20"
```

### Detección de Huéspedes

```go
// Regex patterns
`(\d+)\s*adult[oa]s?`     // "2 adultos"
`(\d+)\s*niñ[oa]s?`       // "3 niños"
`(\d+)\s*personas?`       // "5 personas" → 5 adultos, 0 niños
```

### Detección de Selección de Habitación

```go
// Patrones numéricos
`tipo\s+(\d+)`            // "tipo 2"
`habitaci[oó]n\s+(\d+)`   // "habitación 3"
`opci[oó]n\s+(\d+)`       // "opción 1"

// Palabras clave
"doble" → Tipo 2
"suite" o "presidencial" → Tipo 1
```

### Detección de Datos Personales

```go
hasEmail := strings.Contains(message, "@")
hasPhone := regexp.MustCompile(`\d{9,10}`).MatchString(message)
hasDocument := regexp.MustCompile(`(?i)dni|documento.*\d{8}`).MatchString(message)

// Si tiene al menos 2 de 3, se considera que son datos personales
```

## 📊 Ventajas del Nuevo Sistema

| Aspecto | Antes (V1) | Ahora (V2) |
|---------|------------|------------|
| **Detección** | Depende del LLM | Automática con regex |
| **Consistencia** | ❌ Errática | ✅ 100% consistente |
| **Herramientas** | LLM decide (a veces) | Siempre ejecutadas |
| **Flujo** | Se pierde fácilmente | Avanza linealmente |
| **Debugging** | Difícil | Logs detallados |
| **Performance** | Variable | Predecible |
| **Costo** | Mayor (más tokens) | Menor |

## 🔧 Configuración

No hay configuración adicional. El sistema funciona automáticamente.

### Activación

El `chatbot_handler.go` ahora usa:
```go
response, err := h.service.ProcessMessageV2(req)
```

En lugar de:
```go
response, err := h.service.ProcessMessage(req)  // Versión antigua
```

## 📝 Logging

El nuevo sistema incluye logging detallado:

```
[IntentDetector] Processing message: del 15 al 20 de diciembre para 2 adultos
[IntentDetector] Detected check-in date: 2025-12-15
[IntentDetector] Detected check-out date: 2025-12-20
[IntentDetector] Detected adults: 2
[IntentDetector] Checking availability for 2025-12-15 to 2025-12-20
[IntentDetector] Detected intent: check_availability, executed 1 tools
[ChatbotV2] Processing message from client 123: ...
[ChatbotV2] Detected intent: check_availability, tools executed: 1
[ChatbotV2] Response generated, reservation created: false
```

## 🧪 Testing con Postman

### Ejemplo 1: Reserva Completa en un Mensaje

```json
POST /api/chatbot/chat

{
  "message": "Quiero reservar del 15 al 20 de diciembre para 2 adultos",
  "clienteId": 123
}
```

**Respuesta:**
- Detecta fechas y adultos automáticamente
- Ejecuta `check_availability`
- Muestra habitaciones disponibles
- Guarda estado con fechas y adultos

### Ejemplo 2: Seleccionar Habitación

```json
POST /api/chatbot/chat

{
  "message": "La habitación doble por favor",
  "conversationId": "uuid-de-conversacion-anterior",
  "clienteId": 123
}
```

**Respuesta:**
- Detecta selección de tipo 2
- Ejecuta `calculate_price` automáticamente
- Muestra el precio total
- Pide datos personales

### Ejemplo 3: Completar Reserva

```json
POST /api/chatbot/chat

{
  "message": "Juan Pérez, DNI 12345678, juan@email.com, 987654321. Sí, confirmo",
  "conversationId": "uuid-de-conversacion-anterior",
  "clienteId": 123
}
```

**Respuesta:**
- Detecta datos personales
- Detecta confirmación
- Ejecuta `create_reservation`
- Retorna `reservationCreated: 123`

## 🐛 Resolución de Problemas

### Problema: No detecta fechas

**Solución:** Usar formato compatible:
- ✅ `2025-12-15 al 2025-12-20`
- ✅ `del 15 al 20 de diciembre`
- ✅ `15/12/2025 - 20/12/2025`

### Problema: No detecta cantidad de personas

**Solución:** Ser explícito:
- ✅ `2 adultos`
- ✅ `3 adultos y 2 niños`
- ✅ `5 personas` (se asume adultos)

### Problema: No detecta selección de habitación

**Solución:** Usar palabras clave:
- ✅ `la habitación 2`
- ✅ `tipo 1`
- ✅ `la doble`
- ✅ `la suite`

## 🔮 Mejoras Futuras

- [ ] Usar NLP más sofisticado (spaCy, BERT) para extracción de entidades
- [ ] Detectar más variaciones de fechas en lenguaje natural
- [ ] Soporte para rangos de fechas flexibles ("próximo fin de semana")
- [ ] Detección de preferencias ("habitación con vista al mar")
- [ ] Manejo de modificaciones de reserva
- [ ] Detección de preguntas vs. afirmaciones

## 📚 Referencias

- `intent_detector.go` - Lógica de detección
- `chatbot_service_v2.go` - Servicio mejorado
- `chatbot_handler.go` - Handler HTTP

## ✅ Resultado

**El chatbot ahora avanza linealmente en el proceso de reserva sin perderse ni hacer preguntas circulares.**

Cada mensaje del usuario:
1. Se analiza automáticamente
2. Se detectan intenciones
3. Se ejecutan herramientas necesarias
4. Se actualiza el estado
5. El LLM conversa con los resultados

**¡El flujo funciona de manera consistente y predecible!**
