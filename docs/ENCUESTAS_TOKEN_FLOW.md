# API de Encuestas de Satisfacción - Flujo con Tokens

Esta documentación describe el sistema de encuestas de satisfacción que utiliza tokens seguros enviados por email.

## Base URL
```
http://localhost:3001/api/encuestas
```

---

## 📋 Flujo de Trabajo Completo

### 1. Checkout de Reserva
Cuando un cliente completa su estadía:
- El sistema genera automáticamente un **token único de 64 caracteres**
- Token válido por **30 días**
- Se asocia con `reservationId` y `clientId`

### 2. Envío de Email
Se envía un email al cliente con:
```
https://tu-hotel.com/encuesta?token=abc123def456...xyz789
```

### 3. Cliente Accede al Link
El frontend:
1. Extrae el token del URL (`?token=...`)
2. Llama a `GET /api/encuestas/validar/:token`
3. Obtiene `reservationId` y `clientId` si el token es válido

### 4. Cliente Completa la Encuesta
- Sin necesidad de login
- Califica 6 aspectos (1-5)
- Opcionalmente agrega comentarios

### 5. Envío de Encuesta
- Frontend envía las respuestas con el token
- Backend valida el token automáticamente
- Token se marca como **usado** (no reutilizable)

---

## 🔒 Seguridad

✅ **Tokens únicos**: Generados con `crypto/rand` (32 bytes → 64 chars hex)  
✅ **Un solo uso**: Token se marca como usado después de crear la encuesta  
✅ **Expiración**: 30 días desde su creación  
✅ **Sin autenticación**: Cliente no necesita cuenta  
✅ **Validación backend**: Toda verificación en el servidor  

---

## 📡 Endpoints

### 1. Validar Token

**Endpoint:** `GET /api/encuestas/validar/:token`

**Descripción:** Valida un token y retorna los IDs asociados. **Llamar PRIMERO** al cargar la página de encuesta.

**Parámetros URL:**
- `token`: Token de 64 caracteres recibido por email

**Response Success (200 OK):**
```json
{
  "valid": true,
  "reservationId": 123,
  "clientId": 45
}
```

**Response Error (400 Bad Request):**
```json
{
  "valid": false
}
```

**Estados de error:**
- Token no existe
- Token ya fue usado
- Token expiró (más de 30 días)

---

### 2. Crear Encuesta con Token

**Endpoint:** `POST /api/encuestas`

**Descripción:** Crea una encuesta usando un token válido. El token se valida y marca como usado automáticamente.

**Request Body:**
```json
{
  "token": "abc123def456...xyz789",
  "generalExperience": 5,
  "cleanliness": 5,
  "staffAttention": 4,
  "comfort": 5,
  "recommendation": 5,
  "additionalServices": 4,
  "comments": "Excelente estadía, muy recomendable"
}
```

**Campos:**
| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `token` | string | ✅ Sí | Token de 64 caracteres |
| `generalExperience` | number | ✅ Sí | Experiencia general (1-5) |
| `cleanliness` | number | ✅ Sí | Limpieza (1-5) |
| `staffAttention` | number | ✅ Sí | Atención del personal (1-5) |
| `comfort` | number | ✅ Sí | Confort (1-5) |
| `recommendation` | number | ✅ Sí | Recomendarías el hotel (1-5) |
| `additionalServices` | number | ✅ Sí | Servicios adicionales (1-5) |
| `comments` | string | ❌ No | Comentarios adicionales |

**Response Success (201 Created):**
```json
{
  "message": "Encuesta creada exitosamente",
  "data": {
    "surveyId": 1,
    "reservationId": 123,
    "clientId": 45,
    "generalExperience": 5,
    "cleanliness": 5,
    "staffAttention": 4,
    "comfort": 5,
    "recommendation": 5,
    "additionalServices": 4,
    "comments": "Excelente estadía, muy recomendable",
    "responseDate": "2024-01-15T10:30:00Z"
  }
}
```

**Response Error (400 Bad Request):**
```json
{
  "error": "el token ya fue utilizado"
}
```

---

### 3. Obtener Encuesta por Reserva

**Endpoint:** `GET /api/encuestas/reserva/:reservationId`

**Response (200 OK):**
```json
{
  "data": {
    "surveyId": 1,
    "reservationId": 123,
    "clientId": 45,
    "generalExperience": 5,
    "cleanliness": 5,
    "staffAttention": 4,
    "comfort": 5,
    "recommendation": 5,
    "additionalServices": 4,
    "comments": "Excelente estadía",
    "responseDate": "2024-01-15T10:30:00Z"
  }
}
```

---

### 4. Obtener Encuestas por Cliente

**Endpoint:** `GET /api/encuestas/cliente/:clientId`

**Response (200 OK):**
```json
{
  "data": [
    {
      "surveyId": 1,
      "reservationId": 123,
      "clientId": 45,
      "generalExperience": 5,
      "cleanliness": 5,
      "staffAttention": 4,
      "comfort": 5,
      "recommendation": 5,
      "additionalServices": 4,
      "comments": "Primera estadía",
      "responseDate": "2024-01-15T10:30:00Z"
    }
  ]
}
```

---

### 5. Obtener Todas las Encuestas (Paginación)

**Endpoint:** `GET /api/encuestas/all?limit=50&offset=0`

**Query Params:**
- `limit` (default: 50): Máximo de encuestas a retornar
- `offset` (default: 0): Número de encuestas a saltar

**Response (200 OK):**
```json
{
  "data": [
    { /* encuesta 1 */ },
    { /* encuesta 2 */ }
  ]
}
```

---

### 6. Obtener Promedios de Calificaciones

**Endpoint:** `GET /api/encuestas/promedios`

**Response (200 OK):**
```json
{
  "data": {
    "generalExperience": 4.5,
    "cleanliness": 4.8,
    "staffAttention": 4.6,
    "comfort": 4.7,
    "recommendation": 4.5,
    "additionalServices": 4.3,
    "totalSurveys": 10
  }
}
```

---

## 💻 Integración Frontend (React + TypeScript)

### Tipos TypeScript

```typescript
// types.ts
interface CreateSurveyDTO {
  token: string;
  generalExperience: number;
  cleanliness: number;
  staffAttention: number;
  comfort: number;
  recommendation: number;
  additionalServices: number;
  comments?: string;
}

interface SatisfactionSurvey {
  surveyId: number;
  reservationId: number;
  clientId: number;
  generalExperience: number;
  cleanliness: number;
  staffAttention: number;
  comfort: number;
  recommendation: number;
  additionalServices: number;
  comments?: string;
  responseDate: string;
}

interface TokenValidationResponse {
  valid: boolean;
  reservationId?: number;
  clientId?: number;
}
```

---

### API Client

```typescript
// api/survey.ts
const API_BASE_URL = 'http://localhost:3001/api';

export const surveyAPI = {
  // Validar token
  validateToken: async (token: string): Promise<TokenValidationResponse> => {
    const response = await fetch(
      `${API_BASE_URL}/encuestas/validar/${token}`
    );
    
    if (!response.ok) {
      return { valid: false };
    }
    
    return await response.json();
  },

  // Crear encuesta con token
  createSurvey: async (data: CreateSurveyDTO): Promise<SatisfactionSurvey> => {
    const response = await fetch(`${API_BASE_URL}/encuestas`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Error al crear la encuesta');
    }
    
    const result = await response.json();
    return result.data;
  },

  // Obtener promedios
  getAverageScores: async (): Promise<Record<string, number>> => {
    const response = await fetch(`${API_BASE_URL}/encuestas/promedios`);
    
    if (!response.ok) {
      throw new Error('Error al obtener los promedios');
    }
    
    const result = await response.json();
    return result.data;
  },
};
```

---

### Componente de Página de Encuesta

```tsx
// pages/SurveyPage.tsx
import React, { useState, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { surveyAPI } from '../api/survey';
import type { CreateSurveyDTO } from '../types';

const SurveyPage: React.FC = () => {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token');
  
  // Estados
  const [validating, setValidating] = useState(true);
  const [tokenValid, setTokenValid] = useState(false);
  const [loading, setLoading] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  const [formData, setFormData] = useState({
    generalExperience: 5,
    cleanliness: 5,
    staffAttention: 5,
    comfort: 5,
    recommendation: 5,
    additionalServices: 5,
    comments: '',
  });

  // Validar token al montar el componente
  useEffect(() => {
    const validate = async () => {
      if (!token) {
        setTokenValid(false);
        setValidating(false);
        return;
      }

      try {
        const result = await surveyAPI.validateToken(token);
        setTokenValid(result.valid);
      } catch {
        setTokenValid(false);
      } finally {
        setValidating(false);
      }
    };

    validate();
  }, [token]);

  // Manejar envío del formulario
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!token) return;

    setLoading(true);
    setError(null);

    try {
      await surveyAPI.createSurvey({
        token,
        ...formData,
        comments: formData.comments || undefined,
      });
      
      setSubmitted(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error al enviar');
    } finally {
      setLoading(false);
    }
  };

  const updateRating = (field: keyof typeof formData, value: number) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  // Estado: Validando token
  if (validating) {
    return (
      <div className="survey-container">
        <h2>Validando encuesta...</h2>
      </div>
    );
  }

  // Estado: Token inválido
  if (!tokenValid) {
    return (
      <div className="survey-container">
        <h2>⚠️ Token Inválido</h2>
        <p>
          El link de la encuesta no es válido, ya fue usado, o ha expirado.
        </p>
        <p>
          Por favor, verifica el link en tu email o contacta con el hotel.
        </p>
      </div>
    );
  }

  // Estado: Encuesta enviada
  if (submitted) {
    return (
      <div className="survey-container">
        <h2>✅ ¡Gracias por tu Feedback!</h2>
        <p>Tu encuesta ha sido enviada exitosamente.</p>
        <p>
          Valoramos mucho tu opinión y la usaremos para mejorar 
          nuestros servicios.
        </p>
      </div>
    );
  }

  // Estado: Formulario de encuesta
  return (
    <div className="survey-container">
      <form onSubmit={handleSubmit} className="survey-form">
        <h2>Encuesta de Satisfacción</h2>
        <p className="subtitle">
          Por favor, califica tu experiencia del 1 al 5
        </p>
        <p className="scale-info">
          1 = Muy insatisfecho | 5 = Muy satisfecho
        </p>
        
        {error && (
          <div className="error-alert">{error}</div>
        )}
        
        {/* Experiencia General */}
        <div className="question-group">
          <label>¿Cómo fue tu experiencia general?</label>
          <div className="rating-buttons">
            {[1, 2, 3, 4, 5].map(num => (
              <button
                key={num}
                type="button"
                className={formData.generalExperience === num ? 'active' : ''}
                onClick={() => updateRating('generalExperience', num)}
              >
                {num}
              </button>
            ))}
          </div>
        </div>

        {/* Limpieza */}
        <div className="question-group">
          <label>¿Cómo calificas la limpieza?</label>
          <div className="rating-buttons">
            {[1, 2, 3, 4, 5].map(num => (
              <button
                key={num}
                type="button"
                className={formData.cleanliness === num ? 'active' : ''}
                onClick={() => updateRating('cleanliness', num)}
              >
                {num}
              </button>
            ))}
          </div>
        </div>

        {/* Atención del Personal */}
        <div className="question-group">
          <label>¿Cómo fue la atención del personal?</label>
          <div className="rating-buttons">
            {[1, 2, 3, 4, 5].map(num => (
              <button
                key={num}
                type="button"
                className={formData.staffAttention === num ? 'active' : ''}
                onClick={() => updateRating('staffAttention', num)}
              >
                {num}
              </button>
            ))}
          </div>
        </div>

        {/* Confort */}
        <div className="question-group">
          <label>¿Cómo calificas el confort de las instalaciones?</label>
          <div className="rating-buttons">
            {[1, 2, 3, 4, 5].map(num => (
              <button
                key={num}
                type="button"
                className={formData.comfort === num ? 'active' : ''}
                onClick={() => updateRating('comfort', num)}
              >
                {num}
              </button>
            ))}
          </div>
        </div>

        {/* Recomendación */}
        <div className="question-group">
          <label>¿Recomendarías nuestro hotel?</label>
          <div className="rating-buttons">
            {[1, 2, 3, 4, 5].map(num => (
              <button
                key={num}
                type="button"
                className={formData.recommendation === num ? 'active' : ''}
                onClick={() => updateRating('recommendation', num)}
              >
                {num}
              </button>
            ))}
          </div>
        </div>

        {/* Servicios Adicionales */}
        <div className="question-group">
          <label>¿Cómo calificas los servicios adicionales?</label>
          <div className="rating-buttons">
            {[1, 2, 3, 4, 5].map(num => (
              <button
                key={num}
                type="button"
                className={formData.additionalServices === num ? 'active' : ''}
                onClick={() => updateRating('additionalServices', num)}
              >
                {num}
              </button>
            ))}
          </div>
        </div>

        {/* Comentarios */}
        <div className="question-group">
          <label>Comentarios adicionales (opcional):</label>
          <textarea
            value={formData.comments}
            onChange={(e) => setFormData(prev => ({ 
              ...prev, 
              comments: e.target.value 
            }))}
            rows={4}
            placeholder="Cuéntanos más sobre tu experiencia..."
          />
        </div>

        <button 
          type="submit" 
          disabled={loading}
          className="submit-button"
        >
          {loading ? 'Enviando...' : 'Enviar Encuesta'}
        </button>
      </form>
    </div>
  );
};

export default SurveyPage;
```

---

## 🔧 Integración Backend (Generar Tokens)

### Agregar a tu ReservaService

```go
// En internal/application/reserva_service.go

// Después del checkout, generar token y enviar email
func (s *ReservaService) Checkout(reservaID int) error {
    // ... lógica de checkout existente ...
    
    // Obtener datos de la reserva
    reserva, err := s.reservaRepo.GetByID(reservaID)
    if err != nil {
        return fmt.Errorf("error al obtener reserva: %w", err)
    }
    
    // Generar token de encuesta
    token, err := s.surveyService.CreateTokenForReservation(
        reserva.ReservaID, 
        reserva.ClienteID,
    )
    if err != nil {
        // Log pero no fallar el checkout
        log.Printf("Error creating survey token: %v", err)
    } else {
        // Enviar email con link de encuesta
        surveyLink := fmt.Sprintf(
            "https://tu-hotel.com/encuesta?token=%s", 
            token.Token,
        )
        
        emailBody := fmt.Sprintf(`
            <html>
            <body>
                <h2>¡Gracias por hospedarte con nosotros!</h2>
                <p>Esperamos que hayas disfrutado tu estadía.</p>
                <p>
                    Nos encantaría conocer tu opinión. 
                    Por favor, completa nuestra breve encuesta de satisfacción:
                </p>
                <p>
                    <a href="%s" style="
                        background-color: #4CAF50;
                        color: white;
                        padding: 14px 20px;
                        text-decoration: none;
                        border-radius: 4px;
                    ">
                        Completar Encuesta
                    </a>
                </p>
                <p><small>Este link es válido por 30 días.</small></p>
            </body>
            </html>
        `, surveyLink)
        
        err = s.emailClient.SendEmail(
            reserva.ClienteEmail,
            "Encuesta de Satisfacción - Tu Opinión es Importante",
            emailBody,
        )
        
        if err != nil {
            log.Printf("Error sending survey email: %v", err)
        }
    }
    
    return nil
}
```

---

## 📊 Base de Datos

### Tabla: survey_tokens

```sql
CREATE TABLE IF NOT EXISTS survey_tokens (
    token_id SERIAL PRIMARY KEY,
    token VARCHAR(64) UNIQUE NOT NULL,
    reservation_id INTEGER NOT NULL REFERENCES reservas(reserva_id),
    client_id INTEGER NOT NULL REFERENCES clientes(cliente_id),
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_reservation_token UNIQUE (reservation_id)
);

CREATE INDEX idx_survey_tokens_token ON survey_tokens(token);
CREATE INDEX idx_survey_tokens_reservation ON survey_tokens(reservation_id);
CREATE INDEX idx_survey_tokens_expires ON survey_tokens(expires_at);
```

### Tabla: satisfaction_survey

```sql
CREATE TABLE IF NOT EXISTS satisfaction_survey (
    survey_id SERIAL PRIMARY KEY,
    reservation_id INTEGER NOT NULL REFERENCES reservas(reserva_id),
    client_id INTEGER NOT NULL REFERENCES clientes(cliente_id),
    general_experience INTEGER NOT NULL CHECK (general_experience BETWEEN 1 AND 5),
    cleanliness INTEGER NOT NULL CHECK (cleanliness BETWEEN 1 AND 5),
    staff_attention INTEGER NOT NULL CHECK (staff_attention BETWEEN 1 AND 5),
    comfort INTEGER NOT NULL CHECK (comfort BETWEEN 1 AND 5),
    recommendation INTEGER NOT NULL CHECK (recommendation BETWEEN 1 AND 5),
    additional_services INTEGER NOT NULL CHECK (additional_services BETWEEN 1 AND 5),
    comments TEXT,
    response_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_reservation_survey UNIQUE (reservation_id)
);

CREATE INDEX idx_satisfaction_survey_client ON satisfaction_survey(client_id);
CREATE INDEX idx_satisfaction_survey_reservation ON satisfaction_survey(reservation_id);
CREATE INDEX idx_satisfaction_survey_date ON satisfaction_survey(response_date);
```

---

## ✅ Checklist de Implementación

### Backend
- [x] Crear migración de tablas `survey_tokens` y `satisfaction_survey`
- [x] Implementar repositorio de tokens con generación segura
- [x] Implementar repositorio de encuestas
- [x] Crear servicio de encuestas con validación de tokens
- [x] Crear handlers HTTP para endpoints
- [x] Registrar rutas en `main.go`
- [ ] Integrar generación de tokens en proceso de checkout
- [ ] Configurar envío de emails con links de encuesta

### Frontend
- [ ] Crear página de encuesta (`/encuesta`)
- [ ] Implementar validación de token al cargar
- [ ] Crear formulario con 6 preguntas (rating 1-5)
- [ ] Agregar campo de comentarios opcional
- [ ] Manejar estados: validando, inválido, formulario, enviado
- [ ] Agregar estilos responsive
- [ ] Probar flujo completo end-to-end

---

## 🐛 Troubleshooting

### Token Inválido
- Verificar que el token tenga 64 caracteres
- Confirmar que no esté expirado (< 30 días)
- Validar que no haya sido usado previamente

### Error al Crear Encuesta
- Verificar que el token sea válido
- Confirmar que las puntuaciones estén entre 1-5
- Asegurar que no exista encuesta previa para esa reserva

### Email no Recibido
- Verificar configuración SMTP
- Revisar logs del servidor
- Confirmar email del cliente

---

## 📞 Soporte

Para más información o problemas, contacta al equipo de desarrollo.
