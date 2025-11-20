# Sistema de Almacenamiento de Conversaciones del Chatbot

## 📋 Tabla de Contenidos
1. [Resumen](#resumen)
2. [Estructura de Base de Datos](#estructura-de-base-de-datos)
3. [Cómo se Guardan las Conversaciones](#cómo-se-guardan-las-conversaciones)
4. [Cómo Mostrar Conversaciones al Cliente](#cómo-mostrar-conversaciones-al-cliente)
5. [Ejemplos de Uso](#ejemplos-de-uso)
6. [Migración SQL](#migración-sql)

---

## Resumen

El sistema de chatbot **automáticamente guarda** todas las conversaciones en la tabla `conversation_history`. Cada conversación incluye:
- ✅ Historial completo de mensajes (usuario y asistente)
- ✅ Estado de reserva en progreso
- ✅ Vinculación con el cliente
- ✅ Timestamps de creación y actualización

---

## Estructura de Base de Datos

### Tabla `conversation_history`

```sql
create table conversation_history
(
    id                varchar(100)  PRIMARY KEY,  -- UUID único de la conversación
    client_id         integer       REFERENCES client,  -- ID del cliente
    messages          jsonb         NOT NULL,     -- Historial de mensajes
    created_at        timestamp     DEFAULT NOW(),
    updated_at        timestamp     DEFAULT NOW(),
    reservation_state jsonb                       -- Estado de reserva (después de migración)
);

-- Índices para búsquedas eficientes
CREATE INDEX idx_conversation_client ON conversation_history (client_id);
CREATE INDEX idx_conversation_updated ON conversation_history (updated_at DESC);
CREATE INDEX idx_conversation_reservation ON conversation_history ((reservation_state IS NOT NULL))
WHERE reservation_state IS NOT NULL;
```

### Estructura del Campo `messages` (JSONB)

```json
[
  {
    "role": "user",
    "content": "Hola, quiero reservar una habitación"
  },
  {
    "role": "assistant",
    "content": "¡Con gusto! ¿Para qué fechas necesitas la reserva?"
  },
  {
    "role": "user",
    "content": "Del 15 al 20 de diciembre"
  },
  {
    "role": "assistant",
    "content": "Perfecto. Déjame verificar la disponibilidad..."
  }
]
```

### Estructura del Campo `reservation_state` (JSONB)

```json
{
  "step": "personal_data",
  "fechaEntrada": "2025-12-15",
  "fechaSalida": "2025-12-20",
  "cantidadAdultos": 2,
  "cantidadNinhos": 0,
  "tipoHabitacionId": 3,
  "precioCalculado": 1000.00,
  "personalData": null
}
```

---

## Cómo se Guardan las Conversaciones

### Flujo Automático

1. **Primera Interacción:**
   ```go
   // El usuario envía un mensaje SIN conversationId
   POST /api/chatbot/chat
   {
     "message": "Hola",
     "clienteId": 123  // Opcional pero recomendado
   }

   // El sistema automáticamente:
   // - Genera un UUID nuevo
   // - Crea un nuevo registro en conversation_history
   // - Retorna el conversationId en la respuesta
   ```

2. **Continuación de Conversación:**
   ```go
   // El usuario envía el conversationId para continuar
   POST /api/chatbot/chat
   {
     "message": "Quiero reservar",
     "conversationId": "uuid-12345",  // UUID de la conversación anterior
     "clienteId": 123
   }

   // El sistema:
   // - Recupera el historial de conversation_history
   // - Agrega el nuevo mensaje
   // - Actualiza el registro (UPDATE)
   ```

3. **Guardado Automático:**
   - ✅ Cada mensaje del usuario se guarda inmediatamente
   - ✅ Cada respuesta del asistente se guarda
   - ✅ El estado de reserva se actualiza en tiempo real
   - ✅ El campo `updated_at` se actualiza automáticamente

---

## Cómo Mostrar Conversaciones al Cliente

### Opción 1: Obtener UNA Conversación Específica

**Endpoint:**
```http
GET /api/chatbot/conversation/:id
```

**Ejemplo:**
```bash
curl http://localhost:8080/api/chatbot/conversation/uuid-12345
```

**Respuesta:**
```json
{
  "id": "uuid-12345",
  "clienteId": 123,
  "messages": [
    {"role": "user", "content": "Hola"},
    {"role": "assistant", "content": "¡Hola! ¿En qué puedo ayudarte?"}
  ],
  "createdAt": "2025-11-20T10:00:00Z",
  "updatedAt": "2025-11-20T10:05:00Z",
  "reservationInProgress": {
    "step": "dates",
    "fechaEntrada": "2025-12-15",
    "fechaSalida": "2025-12-20"
  }
}
```

### Opción 2: Obtener TODAS las Conversaciones de un Cliente

**Endpoint:**
```http
GET /api/chatbot/client/:clienteId/conversations
```

**Ejemplo:**
```bash
curl http://localhost:8080/api/chatbot/client/123/conversations
```

**Respuesta:**
```json
[
  {
    "id": "uuid-12345",
    "clienteId": 123,
    "messages": [...],
    "createdAt": "2025-11-20T10:00:00Z",
    "updatedAt": "2025-11-20T10:05:00Z",
    "reservationInProgress": null
  },
  {
    "id": "uuid-67890",
    "clienteId": 123,
    "messages": [...],
    "createdAt": "2025-11-19T14:00:00Z",
    "updatedAt": "2025-11-19T14:30:00Z",
    "reservationInProgress": {
      "step": "confirmation",
      ...
    }
  }
]
```

**Notas:**
- Las conversaciones se ordenan por `updated_at DESC` (más recientes primero)
- Incluye el estado de reserva si hay una en progreso
- Solo muestra conversaciones del cliente autenticado

---

## Ejemplos de Uso

### Ejemplo 1: Primera Conversación (Frontend React)

```javascript
// 1. Usuario inicia conversación
const startConversation = async (message) => {
  const response = await fetch('/api/chatbot/chat', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      message: message,
      clienteId: currentUser.id  // ID del cliente logueado
    })
  });

  const data = await response.json();

  // Guardar el conversationId para continuar después
  setConversationId(data.conversationId);

  return data;
};
```

### Ejemplo 2: Continuar Conversación

```javascript
// 2. Usuario continúa la conversación
const sendMessage = async (message) => {
  const response = await fetch('/api/chatbot/chat', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      message: message,
      conversationId: conversationId,  // UUID de la conversación actual
      clienteId: currentUser.id
    })
  });

  const data = await response.json();

  // Verificar si se creó una reserva
  if (data.reservationCreated) {
    showReservationConfirmation(data.reservationCreated);
  }

  return data;
};
```

### Ejemplo 3: Mostrar Historial de Conversaciones

```javascript
// 3. Obtener todas las conversaciones del cliente
const getConversationHistory = async () => {
  const response = await fetch(`/api/chatbot/client/${currentUser.id}/conversations`);
  const conversations = await response.json();

  return conversations;
};

// 4. Renderizar lista de conversaciones
const ConversationList = () => {
  const [conversations, setConversations] = useState([]);

  useEffect(() => {
    getConversationHistory().then(setConversations);
  }, []);

  return (
    <div className="conversation-list">
      {conversations.map(conv => (
        <ConversationCard
          key={conv.id}
          conversation={conv}
          onClick={() => loadConversation(conv.id)}
        />
      ))}
    </div>
  );
};
```

### Ejemplo 4: Reanudar Conversación Anterior

```javascript
// 5. Cargar una conversación anterior para continuarla
const loadConversation = async (conversationId) => {
  const response = await fetch(`/api/chatbot/conversation/${conversationId}`);
  const conversation = await response.json();

  // Mostrar todo el historial
  setMessages(conversation.messages);
  setConversationId(conversation.id);

  // Verificar si hay una reserva en progreso
  if (conversation.reservationInProgress) {
    showReservationProgress(conversation.reservationInProgress);
  }

  return conversation;
};
```

---

## Componente de Vista Completa (React)

```jsx
import React, { useState, useEffect } from 'react';

const ChatbotInterface = ({ clientId }) => {
  const [conversationId, setConversationId] = useState(null);
  const [messages, setMessages] = useState([]);
  const [inputMessage, setInputMessage] = useState('');
  const [conversations, setConversations] = useState([]);
  const [showHistory, setShowHistory] = useState(false);

  // Cargar historial de conversaciones al montar
  useEffect(() => {
    loadConversationHistory();
  }, [clientId]);

  const loadConversationHistory = async () => {
    const res = await fetch(`/api/chatbot/client/${clientId}/conversations`);
    const data = await res.json();
    setConversations(data);
  };

  const sendMessage = async (message) => {
    // Agregar mensaje del usuario a la UI inmediatamente
    setMessages(prev => [...prev, { role: 'user', content: message }]);

    // Enviar al backend
    const res = await fetch('/api/chatbot/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        message,
        conversationId,
        clienteId
      })
    });

    const data = await res.json();

    // Guardar conversationId si es nueva
    if (!conversationId) {
      setConversationId(data.conversationId);
    }

    // Agregar respuesta del asistente
    setMessages(prev => [...prev, {
      role: 'assistant',
      content: data.message
    }]);

    // Actualizar historial
    loadConversationHistory();
  };

  const loadPreviousConversation = async (convId) => {
    const res = await fetch(`/api/chatbot/conversation/${convId}`);
    const conv = await res.json();

    setConversationId(conv.id);
    setMessages(conv.messages);
    setShowHistory(false);
  };

  const startNewConversation = () => {
    setConversationId(null);
    setMessages([]);
  };

  return (
    <div className="chatbot-container">
      {/* Barra superior con historial */}
      <div className="chatbot-header">
        <button onClick={() => setShowHistory(!showHistory)}>
          📜 Ver Historial ({conversations.length})
        </button>
        <button onClick={startNewConversation}>
          ➕ Nueva Conversación
        </button>
      </div>

      {/* Lista de conversaciones anteriores */}
      {showHistory && (
        <div className="conversation-history">
          <h3>Conversaciones Anteriores</h3>
          {conversations.map(conv => (
            <div
              key={conv.id}
              className="conversation-item"
              onClick={() => loadPreviousConversation(conv.id)}
            >
              <p className="date">
                {new Date(conv.updatedAt).toLocaleString()}
              </p>
              <p className="preview">
                {conv.messages[conv.messages.length - 1]?.content}
              </p>
              {conv.reservationInProgress && (
                <span className="badge">Reserva en progreso</span>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Chat actual */}
      <div className="chat-messages">
        {messages.map((msg, idx) => (
          <div key={idx} className={`message ${msg.role}`}>
            <p>{msg.content}</p>
          </div>
        ))}
      </div>

      {/* Input */}
      <div className="chat-input">
        <input
          value={inputMessage}
          onChange={(e) => setInputMessage(e.target.value)}
          onKeyPress={(e) => {
            if (e.key === 'Enter') {
              sendMessage(inputMessage);
              setInputMessage('');
            }
          }}
          placeholder="Escribe tu mensaje..."
        />
        <button onClick={() => {
          sendMessage(inputMessage);
          setInputMessage('');
        }}>
          Enviar
        </button>
      </div>
    </div>
  );
};

export default ChatbotInterface;
```

---

## Migración SQL

Para habilitar el guardado del estado de reserva, ejecutar:

```bash
# Aplicar migración
psql -d hotel_db -f migrations/001_add_reservation_state.sql
```

**Contenido de la migración:**
```sql
ALTER TABLE conversation_history
ADD COLUMN IF NOT EXISTS reservation_state JSONB;

CREATE INDEX IF NOT EXISTS idx_conversation_reservation
ON conversation_history ((reservation_state IS NOT NULL))
WHERE reservation_state IS NOT NULL;

COMMENT ON COLUMN conversation_history.reservation_state IS 'Stores the state of a reservation in progress as JSON';
```

---

## Consultas SQL Útiles

### Ver todas las conversaciones de un cliente
```sql
SELECT id, created_at, updated_at,
       jsonb_array_length(messages) as message_count,
       reservation_state IS NOT NULL as has_reservation
FROM conversation_history
WHERE client_id = 123
ORDER BY updated_at DESC;
```

### Ver conversaciones con reservas en progreso
```sql
SELECT id, client_id, created_at,
       reservation_state->>'step' as reservation_step
FROM conversation_history
WHERE reservation_state IS NOT NULL
ORDER BY updated_at DESC;
```

### Ver último mensaje de cada conversación
```sql
SELECT id,
       client_id,
       messages->-1->>'content' as last_message,
       updated_at
FROM conversation_history
ORDER BY updated_at DESC
LIMIT 10;
```

### Estadísticas de uso del chatbot
```sql
SELECT
    COUNT(*) as total_conversations,
    COUNT(DISTINCT client_id) as unique_clients,
    AVG(jsonb_array_length(messages)) as avg_messages_per_conversation,
    COUNT(*) FILTER (WHERE reservation_state IS NOT NULL) as conversations_with_reservations
FROM conversation_history;
```

---

## Ventajas del Sistema

✅ **Persistencia Completa:** Nada se pierde, todo se guarda
✅ **Continuidad:** El cliente puede retomar donde dejó
✅ **Historial:** Ver todas las conversaciones pasadas
✅ **Estado de Reserva:** Saber en qué paso se quedó
✅ **Performance:** Índices optimizados para búsquedas rápidas
✅ **Escalable:** JSONB permite almacenar datos flexibles

---

## Roadmap Futuro

- [ ] Exportar conversaciones a PDF
- [ ] Búsqueda de texto completo en conversaciones
- [ ] Analytics de conversaciones (temas más comunes)
- [ ] Etiquetar conversaciones (resuelta, pendiente, escalada)
- [ ] Archivar conversaciones antiguas
- [ ] Integración con sistema de tickets

---

## Soporte

Para preguntas o issues:
- Ver: `CHATBOT_README.md` para documentación del chatbot
- GitHub Issues: https://github.com/luckymachi/hotel_backend/issues
