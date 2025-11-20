# Verificación de Reservas en Base de Datos

## 📋 Tabla de Contenidos
1. [API de Verificación](#api-de-verificación) ⭐ NUEVO
2. [Consultas SQL Rápidas](#consultas-sql-rápidas)
3. [Verificación Completa de una Reserva](#verificación-completa-de-una-reserva)
4. [Verificar el Flujo Completo del Chatbot](#verificar-el-flujo-completo-del-chatbot)
5. [Troubleshooting Común](#troubleshooting-común)
6. [Script de Verificación Automática](#script-de-verificación-automática)

---

## 🚀 API de Verificación

### ⭐ Método Recomendado: Usar el Endpoint de Verificación

La forma más sencilla de verificar una reserva es usando el endpoint de la API:

```bash
# Verificar reserva con ID 123
curl http://localhost:8080/api/reservas/123/verify
```

**Ventajas**:
- ✅ Retorna toda la información en una sola llamada
- ✅ Incluye datos de reserva, cliente, persona, habitaciones y pagos
- ✅ Fácil de usar desde Postman, scripts o aplicaciones
- ✅ Calcula automáticamente noches y precios totales
- ✅ Formato JSON listo para usar

**Ver documentación completa**: [VERIFICATION_API.md](./VERIFICATION_API.md)

---

## 🔍 Consultas SQL Rápidas

### 1. Ver Última Reserva Creada

```sql
-- Ver la reserva más reciente
SELECT
    r.reservation_id,
    r.adults_count,
    r.children_count,
    r.status,
    r.client_id,
    r.subtotal,
    r.confirmation_date,
    r.created_at
FROM reservation r
ORDER BY r.reservation_id DESC
LIMIT 1;
```

### 2. Ver Todas las Reservas Recientes (Últimas 10)

```sql
SELECT
    r.reservation_id,
    r.status,
    r.client_id,
    r.adults_count,
    r.children_count,
    r.subtotal,
    r.confirmation_date
FROM reservation r
ORDER BY r.reservation_id DESC
LIMIT 10;
```

### 3. Contar Reservas por Estado

```sql
SELECT
    status,
    COUNT(*) as total
FROM reservation
GROUP BY status
ORDER BY total DESC;
```

### 4. Verificar Reservas Creadas Hoy

```sql
SELECT
    r.reservation_id,
    r.client_id,
    r.status,
    r.subtotal,
    r.confirmation_date
FROM reservation r
WHERE DATE(r.confirmation_date) = CURRENT_DATE
ORDER BY r.confirmation_date DESC;
```

---

## 🔬 Verificación Completa de una Reserva

### Consulta Maestra: Ver TODO de una Reserva

```sql
-- Reemplaza 123 con el ID de tu reserva
WITH reservation_id_param AS (
    SELECT 123 AS rid
)
SELECT
    '=== RESERVA ===' AS section,
    r.reservation_id,
    r.status,
    r.adults_count,
    r.children_count,
    r.subtotal,
    r.discount,
    r.confirmation_date,
    r.client_id
FROM reservation r, reservation_id_param p
WHERE r.reservation_id = p.rid

UNION ALL

SELECT
    '=== CLIENTE ===' AS section,
    c.client_id::text,
    c.capture_channel::text,
    NULL,
    NULL,
    NULL,
    NULL,
    c.registration_date,
    c.person_id
FROM reservation r
JOIN client c ON r.client_id::int = c.client_id
JOIN reservation_id_param p ON r.reservation_id = p.rid

UNION ALL

SELECT
    '=== PERSONA ===' AS section,
    p.person_id::text,
    p.name || ' ' || p.first_surname || COALESCE(' ' || p.second_surname, ''),
    p.document_number,
    NULL,
    NULL,
    NULL,
    NULL,
    p.email
FROM reservation r
JOIN client c ON r.client_id::int = c.client_id
JOIN person p ON c.person_id = p.person_id
JOIN reservation_id_param param ON r.reservation_id = param.rid

UNION ALL

SELECT
    '=== HABITACIONES ===' AS section,
    rr.room_id::text,
    rt.title,
    NULL,
    NULL,
    rr.price::int,
    NULL,
    rr.check_in_date,
    rr.check_out_date::text
FROM reservation r
JOIN reservation_room rr ON r.reservation_id = rr.reservation_id
JOIN room rm ON rr.room_id = rm.room_id
JOIN room_type rt ON rm.room_type_id = rt.room_type_id
JOIN reservation_id_param p ON r.reservation_id = p.rid;
```

### Versión Simplificada: Reserva con Detalles

```sql
-- Reemplaza 123 con el ID de tu reserva
SELECT
    r.reservation_id,
    r.status AS reservation_status,
    r.adults_count,
    r.children_count,
    r.subtotal,
    r.confirmation_date,

    -- Datos del cliente
    c.client_id,
    p.name || ' ' || p.first_surname AS client_name,
    p.email,
    p.phone_1,
    p.document_number,

    -- Datos de la habitación
    rt.title AS room_type,
    rr.price AS room_price,
    rr.check_in_date,
    rr.check_out_date,

    -- Calcular noches
    EXTRACT(DAY FROM (rr.check_out_date - rr.check_in_date)) AS nights

FROM reservation r
LEFT JOIN client c ON r.client_id::int = c.client_id
LEFT JOIN person p ON c.person_id = p.person_id
LEFT JOIN reservation_room rr ON r.reservation_id = rr.reservation_id
LEFT JOIN room rm ON rr.room_id = rm.room_id
LEFT JOIN room_type rt ON rm.room_type_id = rt.room_type_id
WHERE r.reservation_id = 123;  -- <-- Cambiar aquí
```

---

## 💬 Verificar el Flujo Completo del Chatbot

### 1. Ver Conversaciones que Crearon Reservas

```sql
-- Encuentra conversaciones del chatbot que resultaron en reservas
SELECT
    ch.id AS conversation_id,
    ch.client_id,
    ch.created_at AS conversation_started,
    ch.updated_at AS conversation_updated,

    -- Extraer último mensaje
    ch.messages->-1->>'content' AS last_message,

    -- Estado de reserva (si está en progreso)
    ch.reservation_state->>'step' AS reservation_step,

    -- Buscar si hay reserva creada para este cliente
    (SELECT COUNT(*)
     FROM reservation r
     WHERE r.client_id::int = ch.client_id
     AND r.confirmation_date >= ch.created_at) AS reservations_created

FROM conversation_history ch
WHERE ch.client_id IS NOT NULL
ORDER BY ch.updated_at DESC
LIMIT 10;
```

### 2. Ver Mensajes de una Conversación Específica

```sql
-- Reemplaza 'uuid-aqui' con el ID de la conversación
SELECT
    id,
    client_id,
    created_at,
    updated_at,
    jsonb_pretty(messages) AS conversation_messages,
    jsonb_pretty(reservation_state) AS reservation_state
FROM conversation_history
WHERE id = 'uuid-aqui';
```

### 3. Relacionar Conversación con Reserva

```sql
-- Ver qué reservas se crearon después de una conversación
SELECT
    ch.id AS conversation_id,
    ch.client_id,
    ch.created_at AS conversation_started,

    r.reservation_id,
    r.confirmation_date,
    r.status,
    r.subtotal,

    -- Tiempo entre conversación y reserva
    r.confirmation_date - ch.created_at AS time_to_booking

FROM conversation_history ch
JOIN reservation r ON r.client_id::int = ch.client_id
WHERE r.confirmation_date >= ch.created_at
ORDER BY ch.updated_at DESC
LIMIT 20;
```

---

## 🔍 Verificación Paso a Paso

### Paso 1: Verificar que la Persona se Creó

```sql
SELECT
    person_id,
    name,
    first_surname,
    second_surname,
    document_number,
    email,
    phone_1,
    creation_date
FROM person
WHERE email = 'juan@email.com'  -- <-- Cambiar por el email usado
   OR document_number = '12345678'  -- <-- Cambiar por el DNI usado
ORDER BY creation_date DESC;
```

### Paso 2: Verificar que el Cliente se Creó/Vinculó

```sql
SELECT
    c.client_id,
    c.person_id,
    c.capture_channel,
    c.capture_status,
    c.registration_date,

    -- Datos de la persona
    p.name || ' ' || p.first_surname AS full_name,
    p.email

FROM client c
JOIN person p ON c.person_id = p.person_id
WHERE p.email = 'juan@email.com'  -- <-- Cambiar
   OR p.document_number = '12345678'  -- <-- Cambiar
ORDER BY c.registration_date DESC;
```

### Paso 3: Verificar que la Reserva se Creó

```sql
SELECT
    r.reservation_id,
    r.client_id,
    r.status,
    r.adults_count,
    r.children_count,
    r.subtotal,
    r.discount,
    r.confirmation_date
FROM reservation r
WHERE r.client_id = 'XXX'  -- <-- Cambiar por el client_id del paso 2
ORDER BY r.confirmation_date DESC;
```

### Paso 4: Verificar Habitaciones Asignadas

```sql
SELECT
    rr.reservation_id,
    rr.room_id,
    rr.check_in_date,
    rr.check_out_date,
    rr.price,
    rr.status,

    -- Detalles de la habitación
    rm.name AS room_name,
    rm.number AS room_number,
    rt.title AS room_type

FROM reservation_room rr
JOIN room rm ON rr.room_id = rm.room_id
JOIN room_type rt ON rm.room_type_id = rt.room_type_id
WHERE rr.reservation_id = 123;  -- <-- Cambiar por reservation_id del paso 3
```

### Paso 5: Verificar Pago (si aplica)

```sql
SELECT
    payment_id,
    reservation_id,
    amount,
    date,
    payment_method,
    status
FROM payment
WHERE reservation_id = 123;  -- <-- Cambiar
```

---

## 🐛 Troubleshooting Común

### Problema: No se creó la reserva

**Verificar logs del servidor:**
```bash
# Ver logs en tiempo real
tail -f /var/log/hotel_backend.log | grep "create_reservation"

# O si usas Docker
docker logs -f hotel_backend | grep "create_reservation"
```

**Consulta para ver errores recientes:**
```sql
-- Ver conversaciones que tienen reservation_state pero no crearon reserva
SELECT
    ch.id,
    ch.client_id,
    ch.reservation_state->>'step' AS step,
    ch.updated_at,

    -- Verificar si hay reserva
    (SELECT COUNT(*)
     FROM reservation r
     WHERE r.client_id::int = ch.client_id
     AND r.confirmation_date >= ch.created_at) AS has_reservation

FROM conversation_history ch
WHERE ch.reservation_state IS NOT NULL
  AND ch.reservation_state->>'step' = 'confirmation'
ORDER BY ch.updated_at DESC
LIMIT 10;
```

### Problema: Reserva creada pero sin habitación

```sql
-- Encontrar reservas sin habitación asignada
SELECT
    r.reservation_id,
    r.client_id,
    r.status,
    r.confirmation_date,
    COUNT(rr.room_id) AS rooms_assigned
FROM reservation r
LEFT JOIN reservation_room rr ON r.reservation_id = rr.reservation_id
GROUP BY r.reservation_id, r.client_id, r.status, r.confirmation_date
HAVING COUNT(rr.room_id) = 0
ORDER BY r.confirmation_date DESC;
```

### Problema: Cliente duplicado

```sql
-- Buscar personas duplicadas por documento
SELECT
    document_number,
    COUNT(*) AS count,
    string_agg(person_id::text, ', ') AS person_ids,
    string_agg(email, ', ') AS emails
FROM person
GROUP BY document_number
HAVING COUNT(*) > 1;

-- Buscar clientes duplicados por persona
SELECT
    person_id,
    COUNT(*) AS count,
    string_agg(client_id::text, ', ') AS client_ids
FROM client
GROUP BY person_id
HAVING COUNT(*) > 1;
```

---

## 📊 Dashboard de Verificación

### Consulta Maestra: Estado General del Sistema

```sql
SELECT
    'Total Reservas' AS metric,
    COUNT(*)::text AS value,
    NULL AS details
FROM reservation

UNION ALL

SELECT
    'Reservas Hoy',
    COUNT(*)::text,
    NULL
FROM reservation
WHERE DATE(confirmation_date) = CURRENT_DATE

UNION ALL

SELECT
    'Reservas Pendientes',
    COUNT(*)::text,
    NULL
FROM reservation
WHERE status = 'Pendiente'

UNION ALL

SELECT
    'Reservas Confirmadas',
    COUNT(*)::text,
    NULL
FROM reservation
WHERE status = 'Confirmada'

UNION ALL

SELECT
    'Conversaciones Total',
    COUNT(*)::text,
    NULL
FROM conversation_history

UNION ALL

SELECT
    'Conversaciones con Reserva en Progreso',
    COUNT(*)::text,
    NULL
FROM conversation_history
WHERE reservation_state IS NOT NULL

UNION ALL

SELECT
    'Clientes Total',
    COUNT(*)::text,
    NULL
FROM client

UNION ALL

SELECT
    'Habitaciones Disponibles',
    COUNT(*)::text,
    NULL
FROM room
WHERE status = 'Disponible';
```

---

## 🛠️ Script de Verificación Completa

```sql
-- SCRIPT MASTER DE VERIFICACIÓN
-- Ejecuta esto después de crear una reserva con el chatbot

-- 1. ÚLTIMA RESERVA CREADA
SELECT '====== ÚLTIMA RESERVA ======' AS title;
SELECT
    r.reservation_id,
    r.status,
    r.client_id,
    r.adults_count || ' adultos, ' || r.children_count || ' niños' AS guests,
    'S/ ' || r.subtotal AS total,
    r.confirmation_date
FROM reservation r
ORDER BY r.reservation_id DESC
LIMIT 1;

-- 2. DETALLES DEL CLIENTE
SELECT '====== DATOS DEL CLIENTE ======' AS title;
SELECT
    p.name || ' ' || p.first_surname || COALESCE(' ' || p.second_surname, '') AS full_name,
    p.document_number AS dni,
    p.email,
    p.phone_1 AS phone,
    c.client_id,
    c.capture_channel AS source
FROM reservation r
JOIN client c ON r.client_id::int = c.client_id
JOIN person p ON c.person_id = p.person_id
ORDER BY r.reservation_id DESC
LIMIT 1;

-- 3. HABITACIÓN RESERVADA
SELECT '====== HABITACIÓN RESERVADA ======' AS title;
SELECT
    rt.title AS room_type,
    rr.check_in_date,
    rr.check_out_date,
    EXTRACT(DAY FROM (rr.check_out_date - rr.check_in_date)) AS nights,
    'S/ ' || rr.price || ' x noche' AS price,
    'S/ ' || (rr.price * EXTRACT(DAY FROM (rr.check_out_date - rr.check_in_date))) AS total
FROM reservation r
JOIN reservation_room rr ON r.reservation_id = rr.reservation_id
JOIN room rm ON rr.room_id = rm.room_id
JOIN room_type rt ON rm.room_type_id = rt.room_type_id
ORDER BY r.reservation_id DESC
LIMIT 1;

-- 4. CONVERSACIÓN QUE GENERÓ LA RESERVA
SELECT '====== CONVERSACIÓN ======' AS title;
SELECT
    ch.id AS conversation_id,
    jsonb_array_length(ch.messages) AS total_messages,
    ch.messages->-2->>'content' AS last_user_message,
    ch.messages->-1->>'content' AS last_bot_message
FROM conversation_history ch
JOIN reservation r ON r.client_id::int = ch.client_id
WHERE r.confirmation_date >= ch.created_at
ORDER BY r.reservation_id DESC
LIMIT 1;
```

---

## 📱 Verificación Rápida con psql

```bash
# Conectar a la BD
psql -d hotel_db -U postgres

# Ver última reserva rápido
\x
SELECT * FROM reservation ORDER BY reservation_id DESC LIMIT 1;

# Ver con detalles
SELECT
    r.*,
    p.name,
    p.email,
    rt.title AS room_type
FROM reservation r
JOIN client c ON r.client_id::int = c.client_id
JOIN person p ON c.person_id = p.person_id
LEFT JOIN reservation_room rr ON r.reservation_id = rr.reservation_id
LEFT JOIN room rm ON rr.room_id = rm.room_id
LEFT JOIN room_type rt ON rm.room_type_id = rt.room_type_id
ORDER BY r.reservation_id DESC
LIMIT 1;
```

---

## 🎯 Checklist de Verificación

Después de hacer una prueba con Postman, verifica:

- [ ] **Persona creada** → `SELECT * FROM person WHERE email = '...'`
- [ ] **Cliente creado/vinculado** → `SELECT * FROM client WHERE person_id = ...`
- [ ] **Reserva creada** → `SELECT * FROM reservation WHERE client_id = ...`
- [ ] **Habitación asignada** → `SELECT * FROM reservation_room WHERE reservation_id = ...`
- [ ] **Fechas correctas** → Verificar check_in_date y check_out_date
- [ ] **Precio correcto** → Verificar subtotal y precio por noche
- [ ] **Estado correcto** → Debería estar en "Pendiente" o "Confirmada"
- [ ] **Conversación guardada** → `SELECT * FROM conversation_history WHERE client_id = ...`

---

## 💡 Tips

1. **Usa `\x` en psql** para formato expandido (más legible)
2. **Guarda el client_id** de tus pruebas para rastrear fácilmente
3. **Revisa los logs** en tiempo real mientras pruebas
4. **Usa transacciones** para pruebas que quieres revertir:
   ```sql
   BEGIN;
   -- hacer pruebas
   ROLLBACK;  -- revertir todo
   ```

---

¿Necesitas ayuda con alguna consulta específica o quieres que cree un endpoint de API para verificación automática?
