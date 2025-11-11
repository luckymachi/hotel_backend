# API de Encuestas de Satisfacción

## Resumen
Sistema completo para gestionar encuestas de satisfacción de clientes del hotel.

## Base de Datos

### Ejecutar la migración
```sql
-- Ejecuta el archivo: migrations/create_satisfaction_survey_table.sql
```

La tabla incluye:
- 6 preguntas con puntuación 1-5
- Comentarios opcionales
- Relación con reserva y cliente
- Constraint único: una encuesta por reserva

## Endpoints de la API

### 1. Crear Encuesta
**POST** `/api/encuestas`

Guarda las respuestas de una encuesta de satisfacción.

**Request Body:**
```json
{
  "reservationId": 123,
  "clientId": 45,
  "experienciaGeneral": 5,
  "limpieza": 5,
  "atencionEquipo": 5,
  "comodidad": 5,
  "recomendacion": 5,
  "serviciosAdicionales": 4,
  "comentarios": "Excelente servicio, muy recomendado"
}
```

**Validaciones:**
- Todas las puntuaciones deben estar entre 1 y 5
- `reservationId` y `clientId` son requeridos
- La reserva debe existir
- El cliente debe coincidir con la reserva
- Solo se permite una encuesta por reserva

**Response 201 (Created):**
```json
{
  "message": "Encuesta creada exitosamente",
  "data": {
    "surveyId": 1,
    "reservationId": 123,
    "clientId": 45,
    "experienciaGeneral": 5,
    "limpieza": 5,
    "atencionEquipo": 5,
    "comodidad": 5,
    "recomendacion": 5,
    "serviciosAdicionales": 4,
    "comentarios": "Excelente servicio, muy recomendado",
    "fechaRespuesta": "2025-11-10T15:30:00Z",
    "createdAt": "2025-11-10T15:30:00Z"
  }
}
```

**Response 400 (Error):**
```json
{
  "error": "la puntuación de 'limpieza' debe estar entre 1 y 5, recibido: 6"
}
```

---

### 2. Obtener Encuesta por Reserva
**GET** `/api/encuestas/reserva/:reservationId`

Obtiene la encuesta de una reserva específica.

**Response 200:**
```json
{
  "data": {
    "surveyId": 1,
    "reservationId": 123,
    "clientId": 45,
    "experienciaGeneral": 5,
    "limpieza": 5,
    "atencionEquipo": 5,
    "comodidad": 5,
    "recomendacion": 5,
    "serviciosAdicionales": 4,
    "comentarios": "Excelente servicio",
    "fechaRespuesta": "2025-11-10T15:30:00Z",
    "createdAt": "2025-11-10T15:30:00Z"
  }
}
```

---

### 3. Obtener Encuestas por Cliente
**GET** `/api/encuestas/cliente/:clientId`

Obtiene todas las encuestas de un cliente (ordenadas por fecha descendente).

**Response 200:**
```json
{
  "data": [
    {
      "surveyId": 2,
      "reservationId": 124,
      "clientId": 45,
      "experienciaGeneral": 4,
      "limpieza": 5,
      "atencionEquipo": 5,
      "comodidad": 4,
      "recomendacion": 5,
      "serviciosAdicionales": 4,
      "fechaRespuesta": "2025-11-09T10:00:00Z",
      "createdAt": "2025-11-09T10:00:00Z"
    },
    {
      "surveyId": 1,
      "reservationId": 123,
      "clientId": 45,
      "experienciaGeneral": 5,
      "limpieza": 5,
      "atencionEquipo": 5,
      "comodidad": 5,
      "recomendacion": 5,
      "serviciosAdicionales": 4,
      "fechaRespuesta": "2025-11-05T15:30:00Z",
      "createdAt": "2025-11-05T15:30:00Z"
    }
  ]
}
```

---

### 4. Obtener Todas las Encuestas (con paginación)
**GET** `/api/encuestas/all?limit=50&offset=0`

Obtiene todas las encuestas con paginación.

**Query Parameters:**
- `limit` (opcional): Número de resultados (default: 50)
- `offset` (opcional): Desplazamiento (default: 0)

**Response 200:**
```json
{
  "data": [
    {
      "surveyId": 3,
      "reservationId": 125,
      "clientId": 46,
      ...
    },
    {
      "surveyId": 2,
      "reservationId": 124,
      "clientId": 45,
      ...
    }
  ]
}
```

---

### 5. Obtener Promedios
**GET** `/api/encuestas/promedios`

Obtiene los promedios de todas las encuestas y el promedio general.

**Response 200:**
```json
{
  "data": {
    "experienciaGeneral": 4.8,
    "limpieza": 4.5,
    "atencionEquipo": 4.7,
    "comodidad": 4.6,
    "recomendacion": 5.0,
    "serviciosAdicionales": 4.4,
    "totalEncuestas": 150,
    "promedioGeneral": 4.67
  }
}
```

---

## Flujo de Uso

### Para el Frontend (Formulario de Encuesta)

1. **El cliente completa la encuesta** en el frontend
2. **El frontend envía** POST a `/api/encuestas` con:
   - `reservationId`: ID de la reserva completada
   - `clientId`: ID del cliente
   - Las 6 puntuaciones (1-5)
   - Comentarios (opcional)

3. **El backend valida** y guarda la encuesta
4. **Respuesta exitosa** con los datos guardados

### Ejemplo de Integración en React/Next.js

```typescript
const submitSurvey = async (surveyData) => {
  try {
    const response = await fetch('http://localhost:8080/api/encuestas', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        reservationId: 123,
        clientId: 45,
        experienciaGeneral: surveyData.experienciaGeneral,
        limpieza: surveyData.limpieza,
        atencionEquipo: surveyData.atencionEquipo,
        comodidad: surveyData.comodidad,
        recomendacion: surveyData.recomendacion,
        serviciosAdicionales: surveyData.serviciosAdicionales,
        comentarios: surveyData.comentarios || null
      })
    });

    if (response.ok) {
      const result = await response.json();
      console.log('Encuesta enviada:', result);
      // Mostrar mensaje de éxito
    } else {
      const error = await response.json();
      console.error('Error:', error.error);
      // Mostrar mensaje de error
    }
  } catch (error) {
    console.error('Error de red:', error);
  }
};
```

---

## Próximos Pasos (Para Implementar)

1. **Envío de email con link de encuesta**
   - Agregar método en `ReservaService` para enviar email post-checkout
   - Generar token único para la encuesta
   - Incluir link: `https://tuhotel.com/encuesta?token=xyz`

2. **Validación de token**
   - Crear tabla `survey_tokens` con token, reservation_id, expiration
   - Endpoint para validar token antes de mostrar encuesta

3. **Dashboard de estadísticas**
   - Gráficos de evolución de promedios en el tiempo
   - Filtros por fecha, tipo de habitación, etc.

---

## Notas Técnicas

- ✅ Validación de puntuaciones (1-5)
- ✅ Una encuesta por reserva (constraint único)
- ✅ Comentarios opcionales (nullable)
- ✅ Cálculo automático de promedios
- ✅ Paginación en listados
- ✅ Índices para rendimiento óptimo
