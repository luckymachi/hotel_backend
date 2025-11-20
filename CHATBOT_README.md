# Chatbot de Reservas de Hotel

## Descripción

Este chatbot es un asistente virtual inteligente que puede:
- ✅ Responder preguntas sobre el hotel y sus servicios
- ✅ Buscar información en tiempo real en la web (clima, restaurantes, atracciones)
- ✅ **Crear reservas completas de manera conversacional**
- ✅ Verificar disponibilidad de habitaciones
- ✅ Calcular precios
- ✅ Gestionar datos de clientes

## Arquitectura

### Componentes Principales

1. **Domain (internal/domain/chatbot.go)**
   - `ChatMessage`: Mensajes de la conversación
   - `ChatRequest`: Solicitud del usuario
   - `ChatResponse`: Respuesta del chatbot
   - `ReservationInProgress`: Estado de una reserva en progreso
   - `PersonalDataInput`: Datos personales del cliente

2. **Repository (internal/infrastructure/repository/chatbot_repository.go)**
   - Persistencia de conversaciones en PostgreSQL
   - Almacenamiento del estado de reservas en progreso
   - Historial de mensajes en formato JSONB

3. **Service (internal/application/chatbot_service.go)**
   - Lógica principal del chatbot
   - Integración con Groq/OpenAI para LLM
   - Integración con Tavily para búsquedas web
   - Detección y ejecución de herramientas
   - Gestión de estado de reservas

4. **Tools (internal/application/chatbot_tools.go)**
   - `get_room_types`: Obtiene todos los tipos de habitaciones
   - `check_availability`: Verifica disponibilidad para fechas específicas
   - `calculate_price`: Calcula el precio total de una reserva
   - `create_reservation`: Crea una nueva reserva

5. **Handler (internal/interfaces/http/chatbot_handler.go)**
   - Endpoints HTTP para el chatbot
   - `/api/chatbot/chat`: Enviar mensaje al chatbot
   - `/api/chatbot/conversation/:id`: Obtener historial de conversación
   - `/api/chatbot/client/:clienteId/conversations`: Obtener conversaciones de un cliente

## Flujo de Reserva

El chatbot guía al usuario a través de un flujo conversacional para crear una reserva:

### Paso 1: Fechas
```
Usuario: Quiero reservar una habitación
Chatbot: ¡Con gusto! ¿Para qué fechas necesitas la reserva?
Usuario: Del 15 al 20 de diciembre
```

### Paso 2: Huéspedes
```
Chatbot: Perfecto. ¿Cuántas personas se hospedarán?
Usuario: 2 adultos
```

### Paso 3: Verificar Disponibilidad
El chatbot usa la herramienta `check_availability` automáticamente:
```
[USE_TOOL: check_availability]
{"fechaEntrada": "2025-12-15", "fechaSalida": "2025-12-20"}
[END_TOOL]
```

### Paso 4: Selección de Habitación
```
Chatbot: Tenemos las siguientes habitaciones disponibles:
  • Suite Presidencial - S/500 por noche
  • Habitación Doble - S/200 por noche
  ¿Cuál prefieres?
Usuario: La habitación doble
```

### Paso 5: Calcular Precio
El chatbot usa `calculate_price`:
```
[USE_TOOL: calculate_price]
{"tipoHabitacionId": 2, "fechaEntrada": "2025-12-15", "fechaSalida": "2025-12-20"}
[END_TOOL]

Resultado: 5 noches x S/200 = S/1000
```

### Paso 6: Datos Personales
```
Chatbot: El total sería S/1000. Para confirmar, necesito tus datos:
  - Nombre completo
  - Número de documento
  - Email
  - Teléfono
```

### Paso 7: Confirmación
El chatbot usa `create_reservation`:
```
[USE_TOOL: create_reservation]
{
  "fechaEntrada": "2025-12-15",
  "fechaSalida": "2025-12-20",
  "cantidadAdultos": 2,
  "cantidadNinhos": 0,
  "tipoHabitacionId": 2,
  "personalData": {
    "nombre": "Juan",
    "primerApellido": "Pérez",
    "numeroDocumento": "12345678",
    "correo": "juan@example.com",
    "telefono1": "987654321",
    "genero": "M"
  }
}
[END_TOOL]

Chatbot: ✅ ¡Reserva creada exitosamente! #123
```

## Sistema de Herramientas (Tool Calling)

El chatbot puede ejecutar acciones mediante un sistema de herramientas:

### Formato de Herramientas
```
[USE_TOOL: nombre_herramienta]
{argumentos en JSON}
[END_TOOL]
```

### Proceso
1. El LLM decide usar una herramienta basándose en el contexto
2. El servicio detecta el patrón `[USE_TOOL: ...]`
3. Ejecuta la herramienta con los argumentos proporcionados
4. Agrega el resultado a la conversación
5. Hace una segunda llamada al LLM con el resultado
6. El LLM responde al usuario con la información obtenida

## Migración de Base de Datos

Para agregar soporte de estado de reserva, ejecutar:

```sql
-- Archivo: migrations/001_add_reservation_state.sql
ALTER TABLE conversation_history
ADD COLUMN IF NOT EXISTS reservation_state JSONB;

CREATE INDEX IF NOT EXISTS idx_conversation_reservation
ON conversation_history ((reservation_state IS NOT NULL))
WHERE reservation_state IS NOT NULL;
```

## Configuración

Variables de entorno necesarias:
- `OPENAI_API_KEY` o `GROQ_API_KEY`: API key para Groq/OpenAI
- `TAVILY_API_KEY`: API key para Tavily (búsquedas web)
- `HOTEL_LOCATION`: Ubicación del hotel (ej: "Lima, Perú")
- `DATABASE_URL`: Conexión a PostgreSQL

## Uso del API

### Enviar Mensaje
```bash
POST /api/chatbot/chat
Content-Type: application/json

{
  "message": "Quiero reservar una habitación",
  "conversationId": "uuid-opcional",
  "clienteId": 123,
  "useWeb": false
}
```

### Respuesta
```json
{
  "message": "¡Con gusto! ¿Para qué fechas necesitas la reserva?",
  "conversationId": "uuid-12345",
  "suggestedActions": [
    "Ver habitaciones disponibles",
    "Consultar disponibilidad"
  ],
  "requiresHuman": false,
  "metadata": {
    "tokensUsed": 150,
    "sources": ["hotel", "tools"]
  },
  "reservationInProgress": {
    "step": "dates",
    "fechaEntrada": null,
    "fechaSalida": null,
    "cantidadAdultos": null
  },
  "reservationCreated": null
}
```

## Características Avanzadas

### Búsqueda Web Automática
El chatbot detecta automáticamente cuando necesita información externa:
- Clima
- Restaurantes cercanos
- Atracciones turísticas
- Eventos locales
- Transporte

### Gestión de Contexto
- Mantiene el historial de la conversación
- Persiste el estado de la reserva en progreso
- Recupera automáticamente el contexto al continuar la conversación

### Detección de Intervención Humana
El chatbot detecta cuando necesita transferir a un agente humano:
- Quejas o problemas
- Solicitudes complejas que no puede resolver
- Emergencias

### Compatibilidad
El código es compatible con:
- PostgreSQL sin la columna `reservation_state` (modo legacy)
- PostgreSQL con la columna `reservation_state` (modo completo)

## Testing

```bash
# Ejemplo de conversación de prueba
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hola, quiero reservar una habitación para 2 personas del 15 al 20 de diciembre"
  }'
```

## Roadmap

- [ ] Soporte para múltiples idiomas
- [ ] Procesamiento de lenguaje natural mejorado para extracción de fechas
- [ ] Integración con sistemas de pago
- [ ] Chatbot de voz
- [ ] Análisis de sentimiento
- [ ] Recomendaciones personalizadas basadas en historial

## Contribuciones

Para agregar nuevas herramientas al chatbot:

1. Agregar la herramienta en `internal/application/chatbot_tools.go`
2. Implementar la función `Execute` con la lógica
3. Agregar la descripción de la herramienta en `GetAvailableTools()`
4. El chatbot automáticamente podrá usar la nueva herramienta
