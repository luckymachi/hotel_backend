# Chatbot API Specifications

This document provides detailed specifications for integrating with the hotel chatbot backend API. It covers all endpoints, data models, the reservation flow, and implementation details.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [API Endpoints](#api-endpoints)
3. [Data Models](#data-models)
4. [Reservation Flow](#reservation-flow)
5. [Tool System](#tool-system)
6. [Database Tables](#database-tables)
7. [LLM Integration](#llm-integration)
8. [Web Search Integration](#web-search-integration)
9. [Example Conversations](#example-conversations)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              FRONTEND                                        │
│                         (React/Next.js)                                      │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          API LAYER (Fiber)                                   │
│  POST /api/chatbot/chat                                                      │
│  GET  /api/chatbot/conversation/:id                                          │
│  GET  /api/chatbot/client/:clienteId/conversations                           │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         SERVICE LAYER                                        │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐           │
│  │  ChatbotService  │  │ ReservationTools │  │  SearchService   │           │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘           │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
          ┌─────────────────────────┼─────────────────────────┐
          ▼                         ▼                         ▼
┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐
│   Groq/LLaMA     │    │    PostgreSQL    │    │   Tavily API     │
│   (LLM API)      │    │   (Database)     │    │   (Web Search)   │
└──────────────────┘    └──────────────────┘    └──────────────────┘
```

### Key Components

| Component | File | Description |
|-----------|------|-------------|
| Handler | `internal/interfaces/http/chatbot_handler.go` | HTTP request handling |
| Service | `internal/application/chatbot_service.go` | Main business logic |
| Tools | `internal/application/chatbot_tools.go` | Reservation tool implementations |
| Domain | `internal/domain/chatbot.go` | Data models and interfaces |
| Repository | `internal/infrastructure/repository/chatbot_repository.go` | Database operations |
| LLM Client | `internal/openai/client.go` | Groq API client |

---

## API Endpoints

### 1. Send Message to Chatbot

**Endpoint:** `POST /api/chatbot/chat`

**Description:** Send a message to the chatbot and receive a response. This is the main endpoint for conversational interactions.

#### Request Body

```typescript
interface ChatRequest {
  message: string;                    // Required: User's message
  conversationId?: string;            // Optional: UUID of existing conversation
  clienteId?: number;                 // Optional: Client ID for association
  context?: ChatContext;              // Optional: Additional context
  useWeb?: boolean;                   // Optional: Force/disable web search (null = auto)
}

interface ChatContext {
  fechaEntrada?: string;              // "YYYY-MM-DD" format
  fechaSalida?: string;               // "YYYY-MM-DD" format
  cantidadAdultos?: number;
  cantidadNinhos?: number;
}
```

#### Response Body

```typescript
interface ChatResponse {
  message: string;                           // Chatbot's response text
  conversationId: string;                    // UUID for conversation continuity
  suggestedActions?: string[];               // UI suggestions for next actions
  requiresHuman: boolean;                    // Flag if human intervention needed
  metadata?: {
    tokensUsed: number;                      // LLM tokens consumed
    sources: string[];                       // ["hotel", "web", "tools"]
    webResults?: TavilySearchResponse;       // Raw web search results (if any)
  };
  reservationInProgress?: ReservationInProgress;  // Current reservation state
  reservationCreated?: number;               // Reservation ID if just created
}
```

#### cURL Examples

**Start a new conversation:**
```bash
curl -X POST "http://localhost:8080/api/chatbot/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hola, quisiera hacer una reserva",
    "clienteId": 11
  }'
```

**Continue an existing conversation:**
```bash
curl -X POST "http://localhost:8080/api/chatbot/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Del 20 de diciembre al 27 porfa",
    "clienteId": 11,
    "conversationId": "fd3dbc17-c989-4ba1-a06e-f01541ac1d08"
  }'
```

**With explicit context:**
```bash
curl -X POST "http://localhost:8080/api/chatbot/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Quiero ver habitaciones disponibles",
    "clienteId": 11,
    "context": {
      "fechaEntrada": "2025-12-20",
      "fechaSalida": "2025-12-27",
      "cantidadAdultos": 2,
      "cantidadNinhos": 2
    }
  }'
```

**Force web search:**
```bash
curl -X POST "http://localhost:8080/api/chatbot/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "¿Cómo está el clima en Lima?",
    "clienteId": 11,
    "conversationId": "fd3dbc17-c989-4ba1-a06e-f01541ac1d08",
    "useWeb": true
  }'
```

---

### 2. Get Conversation History

**Endpoint:** `GET /api/chatbot/conversation/:id`

**Description:** Retrieve the full history of a specific conversation.

#### Response Body

```typescript
interface ConversationHistory {
  id: string;                                // UUID
  clienteId?: number;                        // Associated client
  messages: ChatMessage[];                   // Message history
  createdAt: string;                         // ISO timestamp
  updatedAt: string;                         // ISO timestamp
  reservationInProgress?: ReservationInProgress;
}

interface ChatMessage {
  role: "user" | "assistant" | "system";
  content: string;
}
```

#### cURL Example

```bash
curl -X GET "http://localhost:8080/api/chatbot/conversation/fd3dbc17-c989-4ba1-a06e-f01541ac1d08"
```

---

### 3. Get Client Conversations

**Endpoint:** `GET /api/chatbot/client/:clienteId/conversations`

**Description:** Retrieve all conversations for a specific client, ordered by most recent.

#### Response Body

```typescript
ConversationHistory[]  // Array of conversations
```

#### cURL Example

```bash
curl -X GET "http://localhost:8080/api/chatbot/client/11/conversations"
```

---

## Data Models

### ReservationInProgress

Tracks the current state of a reservation being built through the conversation.

```typescript
interface ReservationInProgress {
  step: "dates" | "guests" | "room_type" | "personal_data" | "confirmation" | "completed";
  fechaEntrada?: string;           // "YYYY-MM-DD"
  fechaSalida?: string;            // "YYYY-MM-DD"
  cantidadAdultos?: number;
  cantidadNinhos?: number;
  tipoHabitacionId?: number;       // Room type ID
  personalData?: PersonalDataInput;
  precioCalculado?: number;        // Calculated total price
}
```

### PersonalDataInput

Guest personal data required to complete a reservation.

```typescript
interface PersonalDataInput {
  nombre: string;                  // First name
  primerApellido: string;          // First surname
  segundoApellido?: string;        // Second surname (optional)
  numeroDocumento: string;         // ID document number
  genero: string;                  // Gender
  correo: string;                  // Email
  telefono1: string;               // Primary phone
  telefono2?: string;              // Secondary phone (optional)
  ciudadReferencia?: string;       // Reference city (optional)
  paisReferencia?: string;         // Reference country (optional)
}
```

### Room Type (from database)

```typescript
interface TipoHabitacion {
  id: number;                      // room_type_id
  titulo: string;                  // title (e.g., "Suite", "Deluxe")
  descripcion: string;             // description
  capacidadAdultos: number;        // adult_capacity
  capacidadNinhos: number;         // children_capacity
  cantidadCamas: number;           // beds_count
  precio: number;                  // price per night in Soles
}
```

---

## Reservation Flow

The chatbot follows a structured flow to complete reservations:

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   DATES     │────▶│   GUESTS    │────▶│  ROOM_TYPE  │────▶│PERSONAL_DATA│────▶│CONFIRMATION │
│             │     │             │     │             │     │             │     │             │
│ fechaEntrada│     │ adultos     │     │ tipoHab.ID  │     │ nombre      │     │ crear       │
│ fechaSalida │     │ niños       │     │ precio calc │     │ apellidos   │     │ reserva     │
│             │     │             │     │             │     │ documento   │     │             │
│             │     │             │     │             │     │ email       │     │             │
│             │     │             │     │             │     │ telefono    │     │             │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
```

### Step Descriptions

| Step | Description | Required Data | Tools Used |
|------|-------------|---------------|------------|
| `dates` | Collect check-in and check-out dates | `fechaEntrada`, `fechaSalida` | `check_availability` |
| `guests` | Collect number of adults and children | `cantidadAdultos`, `cantidadNinhos` | - |
| `room_type` | Select room type | `tipoHabitacionId` | `get_room_types`, `calculate_price` |
| `personal_data` | Collect guest information | All `PersonalDataInput` fields | - |
| `confirmation` | Confirm and create reservation | - | `create_reservation` |
| `completed` | Reservation created successfully | - | - |

### Frontend Integration Tips

1. **Track `reservationInProgress` in responses** to show progress indicators
2. **Use `suggestedActions` array** to display quick action buttons
3. **Check `requiresHuman` flag** to show "Talk to agent" option
4. **Monitor `reservationCreated`** to detect successful reservation creation
5. **Pass `context`** when you have pre-selected dates/guests from UI

---

## Tool System

The chatbot uses a text-based tool calling system with specific format markers.

### Tool Call Format (In LLM Response)

```
[USE_TOOL: tool_name]
{"param1": "value1", "param2": "value2"}
[END_TOOL]
```

### Available Tools

#### 1. get_room_types

**Description:** Lists all available room types with prices and characteristics.

**Arguments:** None (empty JSON `{}`)

**Returns:**
```
Tipos de Habitaciones Disponibles:

• Suite (ID: 6)
  Precio: S/297.00 por noche
  Capacidad: 4 adultos, 2 niños
  Camas: 2
  Descripción: Habitación amplia con sala de estar...

• Deluxe (ID: 5)
  Precio: S/180.00 por noche
  ...
```

#### 2. check_availability

**Description:** Checks room availability for specific dates.

**Arguments:**
```json
{
  "fechaEntrada": "2025-12-20",
  "fechaSalida": "2025-12-27"
}
```

**Returns:**
```
Habitaciones disponibles para 2025-12-20 - 2025-12-27:

✅ Suite (ID: 6)
   Precio: S/297.00 por noche
   Capacidad: 4 adultos, 2 niños

✅ Deluxe (ID: 5)
   Precio: S/180.00 por noche
   ...
```

#### 3. calculate_price

**Description:** Calculates total price for a reservation.

**Arguments:**
```json
{
  "tipoHabitacionId": 6,
  "fechaEntrada": "2025-12-20",
  "fechaSalida": "2025-12-27"
}
```

**Returns:**
```
Cálculo de Precio:

Habitación: Suite
Precio por noche: S/297.00
Número de noches: 7
Total: S/2079.00
```

#### 4. create_reservation

**Description:** Creates a new reservation with all client data.

**Arguments:**
```json
{
  "fechaEntrada": "2025-12-20",
  "fechaSalida": "2025-12-27",
  "tipoHabitacionId": 6,
  "cantidadAdultos": 2,
  "cantidadNinhos": 2,
  "personalData": {
    "nombre": "Kevin",
    "primerApellido": "Levano",
    "segundoApellido": "Guehi",
    "numeroDocumento": "77777777",
    "genero": "M",
    "correo": "kevin@kevin.com",
    "telefono1": "+51999999999"
  }
}
```

**Returns (on success):**
```
✅ Reserva creada exitosamente!

Número de Reserva: #123
Cliente: Kevin Levano
Email: kevin@kevin.com
Tipo de Habitación: Suite
Check-in: 2025-12-20
Check-out: 2025-12-27
Noches: 7
Adultos: 2
Niños: 2
Total: S/2079.00
Estado: pending

Se ha enviado un email de confirmación a kevin@kevin.com
```

### Tool Execution Flow

```
User Message
     │
     ▼
┌─────────────────────────────────┐
│   LLM generates response with   │
│   [USE_TOOL: ...] markers       │
└─────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────┐
│   detectAndExecuteTools()       │
│   parses and executes tool      │
└─────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────┐
│   Tool result appended to       │
│   conversation as:              │
│   [RESULTADO DE TOOL_NAME]:     │
│   ...result...                  │
│   [FIN RESULTADO]               │
└─────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────┐
│   Second LLM call to format     │
│   final human-readable response │
└─────────────────────────────────┘
     │
     ▼
Final Response to User
```

---

## Database Tables

### conversation_history

Stores all chatbot conversations.

```sql
CREATE TABLE conversation_history (
    id              VARCHAR(100) PRIMARY KEY,  -- UUID
    client_id       INTEGER REFERENCES client(client_id),
    messages        JSONB NOT NULL,            -- Array of {role, content}
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    reservation_state JSONB                    -- ReservationInProgress object
);

CREATE INDEX idx_conversation_client ON conversation_history(client_id);
CREATE INDEX idx_conversation_updated ON conversation_history(updated_at DESC);
```

#### messages JSONB Structure

```json
[
  {"role": "user", "content": "Hola, quisiera hacer una reserva"},
  {"role": "assistant", "content": "¡Bienvenido! ¿Podrías decirme las fechas..."},
  {"role": "user", "content": "Del 20 al 27 de diciembre"},
  {"role": "assistant", "content": "Excelente. ¿Cuántos adultos y niños?"}
]
```

#### reservation_state JSONB Structure

```json
{
  "step": "room_type",
  "fechaEntrada": "2025-12-20",
  "fechaSalida": "2025-12-27",
  "cantidadAdultos": 2,
  "cantidadNinhos": 2,
  "tipoHabitacionId": null,
  "precioCalculado": null,
  "personalData": null
}
```

### message

Logs individual client messages (separate from conversation history).

```sql
CREATE TABLE message (
    message_id        SERIAL PRIMARY KEY,
    content           VARCHAR(100) NOT NULL,
    registration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    client_id         INTEGER REFERENCES client(client_id)
);
```

### Related Tables

| Table | Relationship |
|-------|-------------|
| `client` | Links conversations to clients via `client_id` |
| `person` | Stores guest personal data for reservations |
| `reservation` | Created by `create_reservation` tool |
| `reservation_room` | Links reservation to specific room |
| `room_type` | Room categories and pricing |
| `room` | Individual room instances |

---

## LLM Integration

### Provider: Groq

The chatbot uses Groq's API (OpenAI-compatible) with the LLaMA model.

**Configuration:**
- Base URL: `https://api.groq.com/openai/v1`
- Model: `llama-3.1-8b-instant`
- Temperature: `0.7`
- Max Tokens: `500`

### System Prompt Structure

The system prompt is dynamically built with:

1. **Base instructions** - Role, behavior guidelines, language
2. **Hotel information** - Real room types, prices, availability from DB
3. **Web search results** - If web search was triggered
4. **Tool descriptions** - Available tools and usage format
5. **Reservation context** - Current reservation state if in progress
6. **User context** - Dates/guests if provided in request

### Message History

Only the last **10 messages** are sent to the LLM to manage token usage.

---

## Web Search Integration

### Provider: Tavily API

Automatic web search is triggered for queries containing:

```go
webKeywords := []string{
    "clima", "weather", "temperatura",
    "restaurantes cerca", "donde comer", "dónde comer",
    "atracciones", "lugares para visitar", "que hacer", "qué hacer",
    "eventos", "festivales",
    "transporte", "como llegar", "cómo llegar", "taxi", "bus", "uber",
    "aeropuerto", "vuelo", "flight", "terminal",
    "noticias", "actualidad",
}
```

### Override Behavior

- `useWeb: null` (default) - Auto-detect based on keywords
- `useWeb: true` - Force web search
- `useWeb: false` - Disable web search

### Response Format

Web results are included in `metadata.webResults`:

```typescript
interface TavilySearchResponse {
  query: string;
  results: Array<{
    title: string;
    url: string;
    content: string;
    score: number;
  }>;
}
```

---

## Example Conversations

### Complete Reservation Flow

```
USER: Hola, quisiera hacer una reserva
BOT:  ¡Bienvenido! Estoy encantado de ayudarte con tu reserva.
      ¿Podrías decirme las fechas de entrada y salida?

USER: Del 20 de diciembre al 27 porfa
BOT:  Excelente elección. Ahora, necesito saber cuántas personas
      estarán viajando. ¿Cuántos adultos y niños?

USER: Somos 2 adultos y dos niños
BOT:  ¿Qué tipo de habitación prefiere?
      [Uses get_room_types tool internally]

USER: Una suite porfa
BOT:  La Suite es una excelente opción para su familia.
      El precio total sería S/2,079.00 por 7 noches.
      ¿Está de acuerdo con el precio?
      [Uses calculate_price tool internally]

USER: Si estoy de acuerdo
BOT:  ¡Excelente! Necesito sus datos personales para crear la reserva:
      - Nombre y apellidos
      - Documento de identificación
      - Correo electrónico
      - Teléfono

USER: Nombre: Kevin Levano Guehi, DNI: 77777777,
      Email: kevin@kevin.com, Tel: +51 999999999
BOT:  ✅ Reserva creada exitosamente!
      Número de Reserva: #123
      ...
      [Uses create_reservation tool internally]
```

### With Web Search

```
USER: ¿Cómo está el clima en Lima esta semana?
BOT:  [Searches web automatically]
      Según la información más reciente, el clima en Lima esta semana...
      Fuente: weather.com

metadata.sources: ["hotel", "web"]
metadata.webResults: { ... tavily results ... }
```

### Human Escalation

```
USER: Tengo una queja sobre mi última estadía
BOT:  Lamento mucho escuchar eso. Voy a transferir su caso a un
      agente humano para que pueda atenderle personalmente...

requiresHuman: true
```

---

## Error Handling

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `Invalid request format` | Malformed JSON in request body | Check JSON syntax |
| `Message is required` | Empty `message` field | Provide message text |
| `Conversation not found` | Invalid `conversationId` | Start new conversation |
| `debe haber al menos 1 adulto` | Tool validation failed | Ensure at least 1 adult |
| `fechas de entrada y salida son requeridas` | Missing dates in tool | Collect dates first |

### Error Response Format

```json
{
  "error": "Error message here",
  "details": "Additional details (if available)"
}
```

---

## Environment Variables

Required configuration for the chatbot service:

```env
# LLM API (Groq)
OPENAI_API_KEY=gsk_...

# Web Search (Tavily)
TAVILY_API_KEY=tvly-...

# Hotel Context
HOTEL_LOCATION=Lima, Perú

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=lapantera
DB_PASSWORD=...
DB_NAME=hotel

# Email (for reservation confirmations)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=...
SMTP_PASSWORD=...
SMTP_FROM_NAME=Hotel
SMTP_FROM_EMAIL=reservas@hotel.com
```

---

## Notes for Frontend Implementation

1. **Always store `conversationId`** from responses and pass it in subsequent requests to maintain conversation continuity.

2. **Handle `reservationInProgress`** to show a progress indicator during the booking flow.

3. **Use `suggestedActions`** to display quick-reply buttons for common next actions.

4. **Check `requiresHuman`** to show a "Talk to human agent" option when needed.

5. **Parse `reservationCreated`** to detect when a reservation was successfully created and show appropriate success UI.

6. **The chatbot can fail to create reservations** if the LLM doesn't correctly parse user data. Consider implementing a fallback form in the frontend for collecting structured personal data.

7. **Current limitation:** The `create_reservation` tool execution sometimes fails because the LLM doesn't properly extract all required fields from natural language. A frontend form as fallback is recommended.
