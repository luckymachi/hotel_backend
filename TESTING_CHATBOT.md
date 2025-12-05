# 🧪 Guía de Prueba del Chatbot Mejorado

Esta guía te muestra cómo probar todas las funcionalidades del chatbot usando `curl` o herramientas como Postman/Insomnia.

## 📋 Prerequisitos

1. El servidor debe estar corriendo: `go run cmd/server/main.go`
2. Base de datos PostgreSQL configurada
3. Variables de entorno configuradas (GROQ_API_KEY, TAVILY_API_KEY)

**URL base**: `http://localhost:8080` (o tu puerto configurado)

---

## 🔥 Endpoints Disponibles

### 1. POST /api/chatbot/chat
Enviar un mensaje al chatbot

### 2. GET /api/chatbot/conversation/:id
Obtener historial de una conversación

### 3. GET /api/chatbot/client/:clienteId/conversations
Obtener todas las conversaciones de un cliente

---

## 📝 Ejemplos de Prueba

### **Prueba 1: Pregunta Frecuente (FAQ) - Respuesta Instantánea**

```bash
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "¿Cuál es el horario de check-in?"
  }'
```

**Resultado esperado:**
- Respuesta instantánea (~5ms)
- `"source": "faq"` en metadata
- Sin llamada al LLM

---

### **Prueba 2: Parser de Fechas en Lenguaje Natural**

```bash
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Quiero reservar para el próximo fin de semana"
  }'
```

**Resultado esperado:**
- Chatbot entiende "próximo fin de semana"
- Extrae fechas automáticamente
- Continúa el flujo de reserva

---

### **Prueba 3: Flujo Completo de Reserva**

#### Paso 1: Iniciar reserva
```bash
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Quiero reservar una habitación del 15 al 20 de diciembre para 2 adultos"
  }' | jq
```

**Guarda el `conversationId` del response**

#### Paso 2: Seleccionar habitación
```bash
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Quiero la habitación doble",
    "conversationId": "TU_CONVERSATION_ID_AQUI"
  }' | jq
```

#### Paso 3: Proporcionar datos personales
```bash
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Mi nombre es Juan Pérez, documento 12345678, email juan@example.com, teléfono 987654321",
    "conversationId": "TU_CONVERSATION_ID_AQUI"
  }' | jq
```

---

### **Prueba 4: Validación de Datos (Error Esperado)**

```bash
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Mi email es invalido y mi teléfono es 123"
  }' | jq
```

**Resultado esperado:**
- Mensaje de error con lista de problemas de validación
- Formato claro y amigable

---

### **Prueba 5: Caché de Búsqueda Web**

#### Primera consulta (cache MISS)
```bash
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "¿Cómo está el clima en Lima?"
  }' | jq
```

**Tiempo esperado:** ~2-3 segundos

#### Segunda consulta (cache HIT)
```bash
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "¿Cómo está el clima en Lima?"
  }' | jq
```

**Tiempo esperado:** ~500ms
**Metadata:** `"webCacheHit": true`

---

### **Prueba 6: Rate Limiting**

Enviar muchos mensajes rápidamente:

```bash
# Bash script para enviar 25 mensajes
for i in {1..25}; do
  echo "Mensaje $i"
  curl -X POST http://localhost:8080/api/chatbot/chat \
    -H "Content-Type: application/json" \
    -d "{
      \"message\": \"test $i\"
    }" | jq -r '.message'
  echo ""
done
```

**Resultado esperado:**
- Primeros 20 mensajes: OK ✅
- Mensajes 21+: Error de rate limit ⚠️

---

### **Prueba 7: Cancelación de Reserva**

#### Iniciar reserva
```bash
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Quiero hacer una reserva"
  }' | jq '.conversationId'
```

#### Cancelar reserva
```bash
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Cancelar todo",
    "conversationId": "TU_CONVERSATION_ID"
  }' | jq
```

**Resultado esperado:**
- Confirmación de cancelación
- `reservationInProgress: null`
- Sugerencias para empezar de nuevo

---

### **Prueba 8: Obtener Historial de Conversación**

```bash
curl -X GET "http://localhost:8080/api/chatbot/conversation/TU_CONVERSATION_ID" | jq
```

**Resultado esperado:**
- Array completo de mensajes
- Estado de reserva si existe
- Timestamps de creación y actualización

---

### **Prueba 9: Obtener Conversaciones de un Cliente**

```bash
curl -X GET "http://localhost:8080/api/chatbot/client/123/conversations" | jq
```

**Resultado esperado:**
- Array de todas las conversaciones del cliente
- Ordenadas por fecha de actualización

---

## 🎯 Pruebas de Características Específicas

### **A. Probar FAQs (Sin LLM)**

Estas preguntas se responden instantáneamente sin llamar al LLM:

```bash
# Check-in
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "¿A qué hora es el check-in?"}' | jq -r '.message'

# Check-out
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "¿Cuándo debo hacer check-out?"}' | jq -r '.message'

# WiFi
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "¿Tienen WiFi gratis?"}' | jq -r '.message'

# Mascotas
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "¿Puedo traer a mi perro?"}' | jq -r '.message'

# Métodos de pago
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "¿Qué métodos de pago aceptan?"}' | jq -r '.message'
```

---

### **B. Probar Parser de Fechas**

```bash
# Mañana
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Quiero reservar para mañana"}' | jq

# Próximo fin de semana
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Reserva para el próximo fin de semana"}' | jq

# Del 15 al 20
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Del 15 al 20 de diciembre"}' | jq

# En 3 días
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "En 3 días para 2 personas"}' | jq
```

---

## 📊 Verificar Metadata

Todas las respuestas incluyen metadata útil:

```json
{
  "metadata": {
    "tokensUsed": 245,          // Tokens consumidos del LLM
    "sources": ["hotel", "faq"], // Fuentes de información
    "responseTime": 1234,        // Tiempo en milisegundos
    "llmModel": "llama-3.1-8b-instant",
    "messageCount": 5,           // Número de mensajes en la conversación
    "webCacheHit": true,         // Si se usó caché web
    "rateLimitRemaining": 15     // Mensajes restantes
  }
}
```

---

## 🐛 Pruebas de Errores

### Error de validación
```bash
curl -X POST http://localhost:8080/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "email: abc, teléfono: 1"
  }' | jq
```

### Conversación inexistente
```bash
curl -X GET "http://localhost:8080/api/chatbot/conversation/inexistente" | jq
```

---

## 🔍 Logs del Servidor

Al ejecutar las pruebas, el servidor mostrará logs detallados:

```
✅ Fechas extraídas: 2025-12-15 a 2025-12-20
✅ Cantidad de adultos extraída: 2
📍 Paso actualizado a: guests
✅ LLM response received (took 1.8s, tokens: 245)
Web search cache HIT for: clima near Lima, Perú
✅ Total request processed in 2.3s (conversation: abc-123)
```

---

## 🎬 Script de Prueba Completo

Guarda esto como `test_chatbot.sh`:

```bash
#!/bin/bash

BASE_URL="http://localhost:8080"

echo "🧪 Iniciando pruebas del chatbot..."
echo ""

# Test 1: FAQ
echo "1️⃣ Probando FAQ (respuesta rápida)..."
curl -s -X POST $BASE_URL/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "¿Cuál es el horario de check-in?"}' | jq -r '.message'
echo ""

# Test 2: Parser de fechas
echo "2️⃣ Probando parser de fechas..."
RESP=$(curl -s -X POST $BASE_URL/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Quiero reservar para mañana, 2 personas"}')
CONV_ID=$(echo $RESP | jq -r '.conversationId')
echo "Conversation ID: $CONV_ID"
echo $RESP | jq -r '.message'
echo ""

# Test 3: Cancelación
echo "3️⃣ Probando cancelación..."
curl -s -X POST $BASE_URL/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d "{\"message\": \"Cancelar todo\", \"conversationId\": \"$CONV_ID\"}" | jq -r '.message'
echo ""

# Test 4: Búsqueda web (cache)
echo "4️⃣ Probando búsqueda web (primera vez)..."
START=$(date +%s%N)
curl -s -X POST $BASE_URL/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "¿Cómo está el clima?"}' > /dev/null
END=$(date +%s%N)
ELAPSED1=$((($END - $START) / 1000000))
echo "Tiempo: ${ELAPSED1}ms"

echo "5️⃣ Probando búsqueda web (segunda vez, con caché)..."
START=$(date +%s%N)
RESP2=$(curl -s -X POST $BASE_URL/api/chatbot/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "¿Cómo está el clima?"}')
END=$(date +%s%N)
ELAPSED2=$((($END - $START) / 1000000))
echo "Tiempo: ${ELAPSED2}ms"
echo "Cache hit: $(echo $RESP2 | jq -r '.metadata.webCacheHit // false')"
echo ""

echo "✅ Pruebas completadas!"
```

Ejecutar:
```bash
chmod +x test_chatbot.sh
./test_chatbot.sh
```

---

## 📚 Documentación Adicional

Para más detalles sobre cada mejora, consulta:
- `CHATBOT_IMPROVEMENTS.md` - Documentación completa de todas las mejoras
- `CHATBOT_README.md` - README original del chatbot

---

## ⚡ Tips de Prueba

1. **Usa `jq`** para formatear JSON: `| jq`
2. **Guarda conversationId** para seguir probando el flujo
3. **Revisa los logs del servidor** para ver detalles internos
4. **Prueba casos extremos**: emails inválidos, rate limiting, etc.
5. **Compara tiempos** entre FAQs y preguntas complejas

---

## 🎓 Conclusión

Con estas pruebas puedes verificar:
- ✅ Respuestas instantáneas con FAQ
- ✅ Parser de fechas naturales
- ✅ Validaciones robustas
- ✅ Caché funcionando
- ✅ Rate limiting activo
- ✅ Cancelación de reservas
- ✅ Flujo completo de reserva

¡El chatbot está listo para usar! 🚀
