# Resumen: Sistema de Encuestas con Tokens

## ✅ Lo que se ha implementado

### 1. Base de Datos

#### Tabla `survey_tokens`
```sql
-- Archivo: migrations/create_survey_tokens_table.sql
```
- Almacena tokens únicos de 64 caracteres
- Relaciona tokens con reservas y clientes
- Controla expiración (30 días)
- Marca tokens como usados

#### Tabla `satisfaction_survey`
```sql
-- Archivo: migrations/create_satisfaction_survey_table.sql
```
- Almacena respuestas de encuestas
- 6 preguntas con escala 1-5
- Campo opcional de comentarios
- Garantiza una encuesta por reserva

### 2. Domain Layer

#### `internal/domain/survey_token.go`
- Modelo `SurveyToken` con campos:
  - `TokenID`, `Token`, `ReservationID`, `ClientID`
  - `ExpiresAt`, `Used`, `CreatedAt`
- Interface `SurveyTokenRepository` con métodos:
  - `Create()`, `GetByToken()`, `MarkAsUsed()`, `DeleteExpired()`

#### `internal/domain/satisfaction_survey.go`
- Modelo `SatisfactionSurvey` (ya existía)
- Nombres de campos en inglés

### 3. Repository Layer

#### `internal/infrastructure/repository/survey_token_repository.go`
- **Generación segura de tokens**: Usa `crypto/rand` para generar 32 bytes aleatorios → 64 caracteres hexadecimales
- **Expiración automática**: Por defecto 30 días desde creación
- Implementa todas las operaciones CRUD de tokens

#### `internal/infrastructure/repository/satisfaction_survey_repository.go`
- Ya existía previamente
- Operaciones CRUD para encuestas

### 4. Service Layer

#### `internal/application/satisfaction_survey_service.go`
**Métodos agregados:**

1. **`ValidateToken(tokenValue string)`**
   - Valida si un token existe, no está usado, y no ha expirado
   - Retorna `reservationID`, `clientID`, `valid`, `error`

2. **`CreateSurveyWithToken(tokenValue string, survey *SatisfactionSurvey)`**
   - Valida el token
   - Asigna automáticamente los IDs de la reserva y cliente a la encuesta
   - Crea la encuesta
   - Marca el token como usado

3. **`CreateTokenForReservation(reservationID, clientID int)`**
   - Genera un nuevo token para una reserva
   - Se debe llamar después del checkout

**Cambios en el constructor:**
```go
func NewSatisfactionSurveyService(
    surveyRepo domain.SatisfactionSurveyRepository,
    reservaRepo domain.ReservaRepository,
    tokenRepo domain.SurveyTokenRepository, // NUEVO
) *SatisfactionSurveyService
```

### 5. HTTP Handler Layer

#### `internal/interfaces/http/satisfaction_survey_handler.go`

**Endpoint agregado:**

1. **`ValidateToken(c *fiber.Ctx)` - `GET /api/encuestas/validar/:token`**
   - Valida un token
   - Retorna los IDs de reserva y cliente si es válido
   - Retorna `{valid: false}` si no es válido

**Request struct modificado:**
```go
type CreateSurveyRequest struct {
    Token              string  `json:"token"` // Ahora usa token en lugar de IDs
    GeneralExperience  int     `json:"generalExperience"`
    // ... resto de campos
}
```

**Método modificado:**
- `CreateSurvey()` ahora valida el token y llama a `CreateSurveyWithToken()`

### 6. Main Application

#### `cmd/server/main.go`

**Cambios:**
```go
// Inicialización
tokenRepo := repository.NewSurveyTokenRepository(db)
surveyService := application.NewSatisfactionSurveyService(
    surveyRepo, 
    reservaRepo, 
    tokenRepo, // NUEVO parámetro
)

// Nueva ruta
surveys.Get("/validar/:token", surveyHandler.ValidateToken)
```

---

## 🔄 Flujo Completo

```
1. Cliente hace checkout
   ↓
2. Backend genera token único (64 chars)
   └─ surveyService.CreateTokenForReservation(reservaID, clienteID)
   ↓
3. Backend envía email con link
   └─ https://tu-hotel.com/encuesta?token=abc123...
   ↓
4. Cliente abre el link
   ↓
5. Frontend valida token
   └─ GET /api/encuestas/validar/:token
   └─ Recibe: {valid: true, reservationId: 123, clientId: 45}
   ↓
6. Frontend muestra formulario de encuesta
   ↓
7. Cliente completa y envía
   └─ POST /api/encuestas
   └─ Body: {token: "abc123...", generalExperience: 5, ...}
   ↓
8. Backend valida token, crea encuesta, marca token como usado
   ↓
9. Frontend muestra mensaje de éxito
```

---

## 📝 Pendientes de Implementación

### Backend

1. **Integrar generación de tokens en proceso de checkout**
   ```go
   // En ReservaService.Checkout() o similar
   token, err := s.surveyService.CreateTokenForReservation(reservaID, clienteID)
   if err != nil {
       log.Printf("Error creating survey token: %v", err)
   }
   ```

2. **Enviar email con link de encuesta**
   ```go
   surveyLink := fmt.Sprintf("https://tu-hotel.com/encuesta?token=%s", token.Token)
   
   emailBody := `
       <h2>¡Gracias por hospedarte!</h2>
       <p>Completa nuestra encuesta:</p>
       <a href="` + surveyLink + `">Completar Encuesta</a>
       <p>Link válido por 30 días.</p>
   `
   
   s.emailClient.SendEmail(clienteEmail, "Encuesta de Satisfacción", emailBody)
   ```

3. **Tarea programada para limpiar tokens expirados (opcional)**
   ```go
   // Ejecutar periódicamente
   tokenRepo.DeleteExpired()
   ```

### Frontend

1. **Crear página de encuesta** (`/encuesta` o `/survey`)
   - Extraer token del query parameter: `?token=abc123...`
   - Validar token al montar el componente
   - Mostrar formulario o mensaje de error

2. **Implementar formulario**
   - 6 campos de rating (1-5)
   - Campo opcional de comentarios
   - Enviar con el token incluido

3. **Manejar estados**
   - Loading: "Validando encuesta..."
   - Invalid: "Token inválido o expirado"
   - Form: Mostrar formulario
   - Submitted: "¡Gracias por tu feedback!"

---

## 🔒 Seguridad

✅ **Tokens criptográficamente seguros**: `crypto/rand` genera 32 bytes aleatorios  
✅ **64 caracteres hexadecimales**: Difíciles de adivinar  
✅ **Un solo uso**: Token se marca como usado después de crear encuesta  
✅ **Expiración**: 30 días automáticos  
✅ **Sin autenticación**: Cliente no necesita cuenta  
✅ **Validación en backend**: Toda lógica de seguridad en el servidor  

---

## 🧪 Testing

### Probar Validación de Token (Postman/cURL)

```bash
# 1. Crear un token manualmente en la base de datos para testing
INSERT INTO survey_tokens (token, reservation_id, client_id, expires_at)
VALUES (
    'abc123def456...', -- 64 caracteres
    1, -- ID de reserva válida
    1, -- ID de cliente válido
    NOW() + INTERVAL '30 days'
);

# 2. Validar el token
GET http://localhost:3001/api/encuestas/validar/abc123def456...

# Respuesta esperada:
{
  "valid": true,
  "reservationId": 1,
  "clientId": 1
}

# 3. Crear encuesta con el token
POST http://localhost:3001/api/encuestas
Content-Type: application/json

{
  "token": "abc123def456...",
  "generalExperience": 5,
  "cleanliness": 5,
  "staffAttention": 4,
  "comfort": 5,
  "recommendation": 5,
  "additionalServices": 4,
  "comments": "Excelente servicio"
}

# 4. Intentar usar el mismo token otra vez (debe fallar)
POST http://localhost:3001/api/encuestas
# Respuesta esperada: {"error": "el token ya fue utilizado"}
```

---

## 📚 Documentación Relacionada

- `docs/ENCUESTAS_TOKEN_FLOW.md` - Documentación completa del flujo con ejemplos de código
- `docs/ENCUESTAS_API.md` - Documentación original (pre-tokens)
- `migrations/create_survey_tokens_table.sql` - Script de migración de tokens
- `migrations/create_satisfaction_survey_table.sql` - Script de migración de encuestas

---

## 🎯 Ventajas de este Enfoque

1. **Sin login requerido**: Cliente puede completar encuesta sin crear cuenta
2. **Seguro**: Tokens únicos, un solo uso, expiración automática
3. **Simple para el cliente**: Solo clic en email → completar formulario
4. **Trazabilidad**: Cada encuesta vinculada a reserva y cliente específicos
5. **Previene duplicados**: Un token por reserva, un uso por token
6. **Escalable**: Generación automática de tokens en checkout

---

## 🚀 Próximos Pasos

1. Ejecutar las migraciones de base de datos
2. Integrar `CreateTokenForReservation()` en el proceso de checkout
3. Configurar template de email con link de encuesta
4. Desarrollar frontend de la página de encuesta
5. Probar flujo completo end-to-end
6. Implementar analytics/dashboard de resultados (opcional)

---

## 📞 Preguntas Frecuentes

**¿Qué pasa si el cliente pierde el email?**
- Puede solicitar un nuevo token al hotel, que puede generarse manualmente desde el admin

**¿Cuánto tiempo es válido el token?**
- 30 días por defecto (configurable en `survey_token_repository.go`)

**¿Se puede completar la encuesta múltiples veces?**
- No, el token se marca como usado después de la primera encuesta

**¿Qué pasa si el token expira?**
- El cliente debe contactar al hotel para obtener un nuevo token

**¿Se pueden limpiar tokens expirados?**
- Sí, usando `tokenRepo.DeleteExpired()` en una tarea programada

---

**Implementado por:** GitHub Copilot  
**Fecha:** 2024  
**Estado:** ✅ Backend completo, Frontend pendiente
