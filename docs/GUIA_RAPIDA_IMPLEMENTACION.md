# Guía Rápida de Implementación - Sistema de Encuestas con Tokens

## ✅ YA IMPLEMENTADO (Backend Completo)

### 1. Migraciones de Base de Datos
```bash
# Ejecutar en PostgreSQL:
psql -U usuario -d hotel_db -f migrations/create_survey_tokens_table.sql
psql -U usuario -d hotel_db -f migrations/create_satisfaction_survey_table.sql
```

### 2. Código Backend
- ✅ Domain models: `survey_token.go`, `satisfaction_survey.go`
- ✅ Repositories: `survey_token_repository.go`, `satisfaction_survey_repository.go`
- ✅ Service: `satisfaction_survey_service.go` (con métodos de tokens)
- ✅ Handler: `satisfaction_survey_handler.go` (con endpoint de validación)
- ✅ Routes: Registradas en `cmd/server/main.go`

### 3. Endpoints Disponibles
```
GET  /api/encuestas/validar/:token      - Validar token
POST /api/encuestas                      - Crear encuesta con token
GET  /api/encuestas/reserva/:id         - Obtener encuesta por reserva
GET  /api/encuestas/cliente/:id         - Obtener encuestas de cliente
GET  /api/encuestas/all?limit&offset    - Listar todas (paginado)
GET  /api/encuestas/promedios           - Obtener promedios
```

---

## 🚧 PENDIENTE DE IMPLEMENTAR

### Backend: Integración con Checkout

**Ubicación:** `internal/application/reserva_service.go` (o donde manejes checkout)

**Código a agregar:**

```go
import (
    "fmt"
    "log"
)

// Dentro del método que maneja el checkout:
func (s *ReservaService) Checkout(reservaID int) error {
    // ... código de checkout existente ...
    
    // ============= AGREGAR ESTO AL FINAL =============
    
    // 1. Obtener datos de la reserva
    reserva, err := s.reservaRepo.GetByID(reservaID)
    if err != nil {
        log.Printf("Error al obtener reserva para encuesta: %v", err)
        return nil // No fallar el checkout
    }
    
    // 2. Generar token de encuesta
    token, err := s.surveyService.CreateTokenForReservation(
        reserva.ReservaID,
        reserva.ClienteID,
    )
    if err != nil {
        log.Printf("Error al crear token de encuesta: %v", err)
        return nil // No fallar el checkout
    }
    
    // 3. Construir link de encuesta
    surveyLink := fmt.Sprintf(
        "https://tu-hotel.com/encuesta?token=%s",
        token.Token,
    )
    
    // 4. Enviar email
    emailBody := fmt.Sprintf(`
        <!DOCTYPE html>
        <html>
        <head>
            <meta charset="UTF-8">
        </head>
        <body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
            <div style="background-color: #f8f9fa; padding: 20px; border-radius: 10px;">
                <h2 style="color: #333;">¡Gracias por hospedarte con nosotros!</h2>
                
                <p style="color: #555; font-size: 16px;">
                    Esperamos que hayas disfrutado tu estadía en nuestro hotel.
                </p>
                
                <p style="color: #555; font-size: 16px;">
                    Nos encantaría conocer tu opinión para seguir mejorando 
                    nuestros servicios. Por favor, completa nuestra breve 
                    encuesta de satisfacción:
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
                    <strong>Equipo Hotel Paradise</strong>
                </p>
            </div>
        </body>
        </html>
    `, surveyLink)
    
    err = s.emailClient.SendEmail(
        reserva.ClienteEmail,
        "Encuesta de Satisfacción - Tu Opinión es Importante",
        emailBody,
    )
    
    if err != nil {
        log.Printf("Error al enviar email de encuesta: %v", err)
        // No fallar el checkout por esto
    } else {
        log.Printf("Email de encuesta enviado a: %s", reserva.ClienteEmail)
    }
    
    // ============= FIN DEL CÓDIGO A AGREGAR =============
    
    return nil
}
```

**Asegúrate de:**
1. Tener acceso a `surveyService` en tu `ReservaService`
2. Si no lo tienes, agregarlo al constructor:

```go
type ReservaService struct {
    // ... campos existentes ...
    surveyService *application.SatisfactionSurveyService // AGREGAR
}

func NewReservaService(
    // ... parámetros existentes ...
    surveyService *application.SatisfactionSurveyService, // AGREGAR
) *ReservaService {
    return &ReservaService{
        // ... asignaciones existentes ...
        surveyService: surveyService, // AGREGAR
    }
}
```

3. Actualizar `main.go` para pasar el servicio:

```go
// Asegúrate de que surveyService esté antes de reservaService
surveyService := application.NewSatisfactionSurveyService(surveyRepo, reservaRepo, tokenRepo)

// Luego pásalo al constructor de ReservaService
reservaService := application.NewReservaService(
    reservaRepo,
    reservaHabitacionRepo,
    habitacionRepo,
    personRepo,
    clientRepo,
    paymentRepo,
    emailClient,
    surveyService, // AGREGAR este parámetro
)
```

---

### Frontend: Crear Página de Encuesta

**Archivo:** `src/pages/SurveyPage.tsx` (o equivalente en tu estructura)

**Pasos:**

1. **Crear la ruta en tu router:**
```tsx
// En tu archivo de rutas (App.tsx, routes.tsx, etc.)
import SurveyPage from './pages/SurveyPage';

<Route path="/encuesta" element={<SurveyPage />} />
```

2. **Crear el archivo del componente:**
```bash
# En tu carpeta de páginas
touch src/pages/SurveyPage.tsx
```

3. **Copiar el código del componente:**
   - Ver `docs/ENCUESTAS_TOKEN_FLOW.md` sección "Componente de Página de Encuesta"
   - Código completo con validación de token, formulario y estados

4. **Crear el API client:**
```bash
touch src/api/survey.ts
```
   - Ver `docs/ENCUESTAS_TOKEN_FLOW.md` sección "API Client"

5. **Agregar estilos (opcional):**
```css
/* src/styles/survey.css */
.survey-container {
  max-width: 600px;
  margin: 40px auto;
  padding: 20px;
}

.survey-form {
  background: white;
  padding: 30px;
  border-radius: 10px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

.question-group {
  margin-bottom: 30px;
}

.rating-buttons {
  display: flex;
  gap: 10px;
  margin-top: 10px;
}

.rating-buttons button {
  flex: 1;
  padding: 12px;
  border: 2px solid #ddd;
  background: white;
  border-radius: 5px;
  cursor: pointer;
  font-size: 16px;
  transition: all 0.3s;
}

.rating-buttons button.active {
  background: #4CAF50;
  color: white;
  border-color: #4CAF50;
}

.submit-button {
  width: 100%;
  padding: 15px;
  background: #4CAF50;
  color: white;
  border: none;
  border-radius: 5px;
  font-size: 18px;
  cursor: pointer;
  transition: background 0.3s;
}

.submit-button:hover:not(:disabled) {
  background: #45a049;
}

.submit-button:disabled {
  background: #ccc;
  cursor: not-allowed;
}

.error-alert {
  padding: 15px;
  background: #ffebee;
  color: #c62828;
  border-radius: 5px;
  margin-bottom: 20px;
}
```

---

## 🧪 Testing Rápido

### 1. Crear Token Manual (para testing)

```sql
-- Ejecutar en PostgreSQL para crear un token de prueba
INSERT INTO survey_tokens (token, reservation_id, client_id, expires_at, used)
VALUES (
    'test123456789abcdef0123456789abcdef0123456789abcdef0123456789ab',
    1, -- ID de una reserva válida
    1, -- ID de un cliente válido
    NOW() + INTERVAL '30 days',
    false
);
```

### 2. Probar Validación de Token

```bash
# Windows PowerShell
$token = "test123456789abcdef0123456789abcdef0123456789abcdef0123456789ab"
Invoke-RestMethod -Uri "http://localhost:3001/api/encuestas/validar/$token" -Method GET
```

**Respuesta esperada:**
```json
{
  "valid": true,
  "reservationId": 1,
  "clientId": 1
}
```

### 3. Probar Crear Encuesta

```powershell
$body = @{
    token = "test123456789abcdef0123456789abcdef0123456789abcdef0123456789ab"
    generalExperience = 5
    cleanliness = 5
    staffAttention = 4
    comfort = 5
    recommendation = 5
    additionalServices = 4
    comments = "Excelente servicio!"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:3001/api/encuestas" -Method POST -Body $body -ContentType "application/json"
```

### 4. Probar en Frontend

```
# Abrir en navegador:
http://localhost:3000/encuesta?token=test123456789abcdef0123456789abcdef0123456789abcdef0123456789ab
```

**Debería:**
1. Validar el token automáticamente
2. Mostrar el formulario de encuesta
3. Permitir completar y enviar
4. Mostrar mensaje de éxito

---

## 📋 Checklist Final

### Backend
- [x] Migraciones ejecutadas en DB
- [x] Código de tokens implementado
- [x] Endpoints funcionando
- [ ] Integración con checkout implementada
- [ ] Email de encuesta configurado
- [ ] Probado con token manual

### Frontend
- [ ] Página de encuesta creada
- [ ] Ruta `/encuesta` configurada
- [ ] API client implementado
- [ ] Estilos agregados
- [ ] Probado end-to-end con token real

### Testing
- [ ] Token se genera en checkout
- [ ] Email se envía correctamente
- [ ] Link en email funciona
- [ ] Formulario valida token
- [ ] Encuesta se guarda correctamente
- [ ] Token se marca como usado
- [ ] No se puede reutilizar token

---

## 🆘 Solución de Problemas

### El token no se valida
- Verificar que tenga exactamente 64 caracteres
- Confirmar que existe en la tabla `survey_tokens`
- Verificar que `used = false` y `expires_at > NOW()`

### El email no se envía
- Verificar configuración SMTP en `.env`:
  ```
  SMTP_HOST=smtp.gmail.com
  SMTP_PORT=587
  SMTP_USER=tu_email@gmail.com
  SMTP_PASSWORD=tu_app_password
  ```
- Revisar logs del servidor para errores
- Probar envío de email manualmente

### Frontend no muestra el formulario
- Abrir DevTools → Console para ver errores
- Verificar que la URL tenga el parámetro `?token=...`
- Confirmar que el endpoint de validación responda correctamente
- Revisar CORS si hay errores de conexión

### Error al crear encuesta
- Confirmar que todas las puntuaciones estén entre 1-5
- Verificar que el token sea válido y no esté usado
- Revisar que la reserva exista en la base de datos

---

## 📚 Documentación Completa

Para más detalles, consultar:
- `docs/ENCUESTAS_TOKEN_FLOW.md` - Guía completa con código
- `docs/RESUMEN_IMPLEMENTACION_TOKENS.md` - Resumen de implementación
- `docs/DIAGRAMA_FLUJO_TOKENS.md` - Diagramas visuales del flujo
- `docs/ENCUESTAS_API.md` - Documentación original de la API

---

## 🎉 ¡Listo!

Una vez completados todos los pasos pendientes:
1. El sistema generará tokens automáticamente en checkout
2. Los clientes recibirán emails con links únicos
3. Podrán completar la encuesta sin necesidad de login
4. Las respuestas se guardarán vinculadas a su reserva
5. Los tokens se marcarán como usados automáticamente

**Tiempo estimado de implementación pendiente:** 2-3 horas
- Backend checkout: 30 min
- Frontend página: 1-2 horas
- Testing: 30 min
