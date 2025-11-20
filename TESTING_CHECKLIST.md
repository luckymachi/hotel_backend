# 🧪 Checklist de Pruebas - Sistema de Reservas con Chatbot

## 📋 Resumen de Cambios

Este checklist cubre las pruebas para verificar:
1. ✅ Fix de creación de reservas (campos faltantes en Person)
2. ✅ Migración de base de datos (tabla person con birth_date)
3. ✅ Endpoint de verificación de reservas
4. ✅ Logging mejorado para debugging
5. ✅ Corrección de métodos undefined (GetByID)

---

## 🚀 PASO 0: Actualizar el Código

**IMPORTANTE**: Antes de probar, asegúrate de tener el código actualizado.

```bash
# 1. Detener el servidor (Ctrl+C si está corriendo)

# 2. Actualizar código desde el repositorio
cd /home/user/hotel_backend
git pull origin claude/chatbot-room-booking-019QTAbu8LfntyUfKK8wtAKt

# 3. Verificar que estás en el branch correcto
git branch
# Debe mostrar: * claude/chatbot-room-booking-019QTAbu8LfntyUfKK8wtAKt

# 4. Ver últimos commits
git log --oneline -5
# Debes ver:
# - ea86ead fix: Agregar método GetByID a ClientRepository...
# - 4882169 fix: Corregir creación de reservas que fallaba silenciosamente
# - 4b76ca9 feat: Agregar endpoint de verificación de reservas...

# 5. Recompilar (opcional pero recomendado)
go build ./cmd/server

# 6. Ejecutar migración de base de datos
psql -U postgres -d hotel_db -f migrations/002_fix_person_table.sql

# Salida esperada:
# NOTICE:  Tabla persona renombrada a person...
# NOTICE:  Campo birth_date agregado a tabla person
# NOTICE:  ✅ Migración completada exitosamente...
```

---

## 📝 PASO 1: Verificar Base de Datos

### 1.1 Verificar que la tabla `person` existe con todos los campos

```sql
-- Conectar a la BD
psql -U postgres -d hotel_db

-- Verificar estructura de la tabla person
\d person

-- Debe mostrar:
-- person_id | integer (primary key)
-- name | varchar
-- first_surname | varchar
-- second_surname | varchar
-- document_number | varchar
-- gender | varchar
-- email | varchar (unique)
-- phone_1 | varchar
-- phone_2 | varchar
-- reference_city | varchar
-- reference_country | varchar
-- active | boolean
-- creation_date | timestamp
-- birth_date | timestamp  ← IMPORTANTE: Debe existir

-- Salir de psql
\q
```

**✅ Criterio de Éxito**: La tabla `person` debe tener 14 columnas incluyendo `birth_date`.

---

## 🖥️ PASO 2: Iniciar el Servidor

```bash
# Desde el directorio del proyecto
cd /home/user/hotel_backend

# Ejecutar servidor
go run cmd/server/main.go

# Salida esperada (sin errores):
# Server starting on :8080
# Connected to database
```

**⚠️ Si hay errores de compilación**:
- Error: `GetHabitacionByID undefined` → Hacer `git pull` y recompilar
- Error: `GetByID undefined` → Hacer `git pull` y recompilar
- Error de BD → Ejecutar migración del Paso 0

---

## 🤖 PASO 3: Probar Flujo Completo del Chatbot

### 3.1 Iniciar Conversación

**Postman Request**:
```
POST http://localhost:8080/api/chatbot/chat
Content-Type: application/json

{
  "message": "Hola, quiero hacer una reserva",
  "clienteId": 11
}
```

**Respuesta Esperada**:
```json
{
  "message": "¡Hola! Con gusto te ayudo a hacer una reserva...",
  "conversationId": "conv-uuid-123",
  "requiresHuman": false
}
```

**✅ Verificar**:
- [ ] Respuesta es coherente
- [ ] Se genera `conversationId`
- [ ] `requiresHuman` es `false`

---

### 3.2 Proporcionar Fechas

**Postman Request**:
```json
{
  "message": "Del 25 de diciembre al 30 de diciembre",
  "clienteId": 11,
  "conversationId": "conv-uuid-123"
}
```

**Respuesta Esperada**:
```json
{
  "message": "Perfecto, del 25 al 30 de diciembre (5 noches).\n\n[RESULTADO DE CHECK_AVAILABILITY]:\nHabitaciones disponibles...",
  "conversationId": "conv-uuid-123",
  "reservationInProgress": {
    "fechaEntrada": "2025-12-25",
    "fechaSalida": "2025-12-30",
    "step": "room_selection"
  }
}
```

**✅ Verificar**:
- [ ] Las fechas se detectaron correctamente
- [ ] Muestra habitaciones disponibles
- [ ] `reservationInProgress` tiene fechas

**🔍 Revisar Logs del Servidor**:
```bash
# Buscar logs de detección de intención
grep "IntentDetector" server.log

# Debe mostrar:
# [IntentDetector] Ejecutando check_availability con datos: {"fechaEntrada":"2025-12-25"...}
```

---

### 3.3 Seleccionar Habitación

**Postman Request**:
```json
{
  "message": "La suite presidencial",
  "clienteId": 11,
  "conversationId": "conv-uuid-123"
}
```

**Respuesta Esperada**:
```json
{
  "message": "Excelente elección. La Suite Presidencial...\n\n¿Cuántas personas se hospedarán?",
  "conversationId": "conv-uuid-123",
  "reservationInProgress": {
    "fechaEntrada": "2025-12-25",
    "fechaSalida": "2025-12-30",
    "tipoHabitacionId": 1,
    "step": "guest_count"
  }
}
```

**✅ Verificar**:
- [ ] Se seleccionó el tipo de habitación
- [ ] `tipoHabitacionId` está en `reservationInProgress`
- [ ] Pregunta por cantidad de personas

---

### 3.4 Proporcionar Cantidad de Personas

**Postman Request**:
```json
{
  "message": "2 adultos",
  "clienteId": 11,
  "conversationId": "conv-uuid-123"
}
```

**Respuesta Esperada**:
```json
{
  "message": "Perfecto, 2 adultos...\n\n[RESULTADO DE CALCULATE_PRICE]:\nCálculo de Precio:\n...\nTotal: S/1000.00",
  "conversationId": "conv-uuid-123",
  "reservationInProgress": {
    "fechaEntrada": "2025-12-25",
    "fechaSalida": "2025-12-30",
    "cantidadAdultos": 2,
    "cantidadNinhos": 0,
    "tipoHabitacionId": 1,
    "precioCalculado": 1000.00,
    "step": "personal_data"
  }
}
```

**✅ Verificar**:
- [ ] Se detectó cantidad de adultos
- [ ] Se calculó el precio automáticamente
- [ ] `precioCalculado` está presente
- [ ] Pregunta por datos personales

---

### 3.5 Proporcionar Datos Personales

**Postman Request**:
```json
{
  "message": "Mi nombre es Juan Pérez García, DNI 87654321, correo juan.perez@email.com, teléfono 987654321",
  "clienteId": 11,
  "conversationId": "conv-uuid-123"
}
```

**Respuesta Esperada**:
```json
{
  "message": "Gracias, Juan Pérez. He registrado tus datos:\n- Nombre: Juan Pérez García\n- DNI: 87654321\n- Email: juan.perez@email.com\n- Teléfono: 987654321\n\n¿Confirmas la reserva?",
  "conversationId": "conv-uuid-123",
  "reservationInProgress": {
    "fechaEntrada": "2025-12-25",
    "fechaSalida": "2025-12-30",
    "cantidadAdultos": 2,
    "cantidadNinhos": 0,
    "tipoHabitacionId": 1,
    "precioCalculado": 1000.00,
    "personalData": {
      "nombre": "Juan",
      "primerApellido": "Pérez",
      "segundoApellido": "García",
      "numeroDocumento": "87654321",
      "correo": "juan.perez@email.com",
      "telefono1": "987654321"
    },
    "step": "confirmation"
  }
}
```

**✅ Verificar**:
- [ ] Se detectaron todos los datos personales
- [ ] `personalData` está completo
- [ ] Pregunta por confirmación

---

### 3.6 Confirmar Reserva ⭐ **PASO CRÍTICO**

**Postman Request**:
```json
{
  "message": "Sí, confirmo",
  "clienteId": 11,
  "conversationId": "conv-uuid-123"
}
```

**Respuesta Esperada**:
```json
{
  "message": "✅ Reserva creada exitosamente!\n\nNúmero de Reserva: #15\nCliente: Juan Pérez\nEmail: juan.perez@email.com\nTipo de Habitación: Suite Presidencial\nCheck-in: 2025-12-25\nCheck-out: 2025-12-30\nNoches: 5\nAdultos: 2\nNiños: 0\nTotal: S/1000.00\nEstado: Pendiente\n\nSe ha enviado un email de confirmación a juan.perez@email.com",
  "conversationId": "conv-uuid-123",
  "reservationCreated": 15,
  "reservationInProgress": null
}
```

**✅ Verificar**:
- [ ] Muestra "✅ Reserva creada exitosamente!"
- [ ] Tiene `reservationCreated` con ID de reserva
- [ ] `reservationInProgress` es `null` (se limpió)
- [ ] Muestra "Número de Reserva: #X"

**🔍 Revisar Logs del Servidor** (MUY IMPORTANTE):
```bash
# Buscar logs de creación de reserva
grep "CreateReservation" server.log | tail -20

# Debe mostrar:
# [CreateReservation] Creando reserva para cliente: Juan Pérez (DNI: 87654321)
# [CreateReservation] Reserva creada exitosamente con ID: 15
# [IntentDetector] create_reservation exitoso: ✅ Reserva creada exitosamente!
```

**❌ Si los logs muestran ERROR**:
```bash
# Ver errores
grep "ERROR" server.log | tail -10

# Errores comunes:
# - "column birth_date does not exist" → Ejecutar migración
# - "GetHabitacionByID undefined" → git pull y recompilar
# - "error al crear persona" → Revisar campos de Person
```

---

## ✅ PASO 4: Verificar en Base de Datos

### 4.1 Verificar Reserva con Endpoint de Verificación

**Postman Request** (usa el ID del paso 3.6):
```
GET http://localhost:8080/api/reservas/15/verify
```

**Respuesta Esperada**:
```json
{
  "success": true,
  "message": "Reserva verificada exitosamente",
  "data": {
    "reservation": {
      "id": 15,
      "cantidadAdultos": 2,
      "cantidadNinhos": 0,
      "estado": "Pendiente",
      "clienteId": 11,
      "subtotal": 1000.00,
      "descuento": 0,
      "fechaConfirmacion": "2025-11-20T...",
      "habitaciones": [...]
    },
    "client": {
      "clientId": 11,
      "personId": 25
    },
    "person": {
      "personId": 25,
      "name": "Juan",
      "first_surname": "Pérez",
      "second_surname": "García",
      "document_number": "87654321",
      "gender": "M",
      "email": "juan.perez@email.com",
      "phone_1": "987654321",
      "birth_date": "2007-11-20T...",
      "active": true
    },
    "rooms": [
      {
        "roomId": 5,
        "roomNumber": "201",
        "roomName": "Suite 201",
        "roomType": "Suite Presidencial",
        "checkInDate": "2025-12-25T00:00:00Z",
        "checkOutDate": "2025-12-30T00:00:00Z",
        "price": 200.00,
        "nights": 5,
        "totalPrice": 1000.00
      }
    ],
    "payments": [],
    "verificationTime": "2025-11-20T..."
  }
}
```

**✅ Verificar**:
- [ ] `success` es `true`
- [ ] `reservation` existe con ID correcto
- [ ] `client` tiene `clientId` y `personId`
- [ ] `person` tiene todos los datos (✨ especialmente `birth_date` y `active: true`)
- [ ] `rooms` tiene al menos 1 habitación
- [ ] `roomType` muestra nombre legible (ej: "Suite Presidencial")
- [ ] `nights` y `totalPrice` están calculados correctamente

---

### 4.2 Verificar con SQL Directo

```sql
-- Conectar a BD
psql -U postgres -d hotel_db

-- 1. Ver última reserva creada
SELECT * FROM reservation ORDER BY reservation_id DESC LIMIT 1;

-- Verificar:
-- ✅ reservation_id = 15 (o el ID que obtuviste)
-- ✅ adults_count = 2
-- ✅ status = 'Pendiente'
-- ✅ subtotal = 1000.00

-- 2. Ver persona creada
SELECT * FROM person ORDER BY person_id DESC LIMIT 1;

-- Verificar:
-- ✅ name = 'Juan'
-- ✅ first_surname = 'Pérez'
-- ✅ email = 'juan.perez@email.com'
-- ✅ birth_date NO ES NULL ← IMPORTANTE
-- ✅ active = true ← IMPORTANTE

-- 3. Ver cliente
SELECT * FROM client WHERE client_id = 11;

-- Verificar:
-- ✅ person_id apunta a la persona creada

-- 4. Ver habitación asignada
SELECT * FROM reservation_room WHERE reservation_id = 15;

-- Verificar:
-- ✅ room_id existe
-- ✅ check_in_date = '2025-12-25'
-- ✅ check_out_date = '2025-12-30'
-- ✅ price = 200.00

-- 5. Verificación completa (JOIN)
SELECT
    r.reservation_id,
    r.status,
    r.subtotal,
    p.name,
    p.email,
    p.birth_date,
    p.active,
    rr.check_in_date,
    rr.check_out_date
FROM reservation r
JOIN client c ON r.client_id = c.client_id
JOIN person p ON c.person_id = p.person_id
JOIN reservation_room rr ON r.reservation_id = rr.reservation_id
WHERE r.reservation_id = 15;

-- Debe mostrar todos los datos relacionados
```

**✅ Criterios de Éxito**:
- [ ] Reserva existe en `reservation`
- [ ] Persona existe en `person` con `birth_date` y `active = true`
- [ ] Cliente existe en `client`
- [ ] Habitación asignada existe en `reservation_room`

---

## 🧪 PASO 5: Probar Casos Especiales

### 5.1 Crear Reserva con Cliente Existente

```json
POST /api/chatbot/chat
{
  "message": "Quiero otra reserva del 1 al 5 de enero",
  "clienteId": 11
}
```

**✅ Verificar**:
- [ ] Usa el mismo `person_id` si el DNI ya existe
- [ ] Actualiza el email si cambió

---

### 5.2 Probar Sin Fechas Disponibles

```json
{
  "message": "Del 25 al 30 de diciembre"
}
```

**✅ Verificar**:
- [ ] Muestra "No hay habitaciones disponibles" si no hay
- [ ] No intenta crear reserva

---

### 5.3 Probar con Datos Incompletos

```json
{
  "message": "Sí, confirmo"
}
```

(Sin haber proporcionado datos personales)

**✅ Verificar**:
- [ ] No crea reserva
- [ ] Pide los datos faltantes

---

## 📊 PASO 6: Verificar Logs

```bash
# 1. Logs de intent detection
grep "IntentDetector" server.log | tail -20

# Debe mostrar:
# - Detected intent: check_availability
# - Detected intent: calculate_price
# - Detected intent: create_reservation

# 2. Logs de creación de reserva
grep "CreateReservation" server.log | tail -10

# Debe mostrar:
# - [CreateReservation] Creando reserva para cliente: ...
# - [CreateReservation] Reserva creada exitosamente con ID: X

# 3. Logs de errores (NO debe haber errores)
grep "ERROR" server.log | tail -10

# Si hay errores, investigar y corregir
```

---

## 🎯 PASO 7: Checklist Final

### Funcionalidad del Chatbot

- [ ] El chatbot responde coherentemente
- [ ] Detecta fechas automáticamente
- [ ] Verifica disponibilidad automáticamente
- [ ] Calcula precios automáticamente
- [ ] Detecta datos personales (nombre, DNI, email, teléfono)
- [ ] Crea reservas exitosamente
- [ ] Limpia el estado después de crear reserva
- [ ] Muestra ID de reserva creada

### Base de Datos

- [ ] Tabla `person` existe con todos los campos
- [ ] Campo `birth_date` existe y NO es NULL
- [ ] Campo `active` es `true` para nuevas personas
- [ ] Reservas se crean en tabla `reservation`
- [ ] Habitaciones se asignan en `reservation_room`
- [ ] Clientes se vinculan correctamente

### API

- [ ] `POST /api/chatbot/chat` funciona
- [ ] `GET /api/reservas/:id/verify` funciona
- [ ] `GET /api/reservas/:id/verify` retorna datos completos
- [ ] Endpoint muestra `roomType` legible (no números)

### Logging

- [ ] Logs muestran `[CreateReservation]` antes y después de crear
- [ ] Logs muestran `[IntentDetector]` con intenciones detectadas
- [ ] Logs muestran ID de reserva creada
- [ ] No hay errores de `undefined` en logs

---

## ❌ Troubleshooting

### Error: "GetHabitacionByID undefined"

**Causa**: Código antiguo en ejecución.

**Solución**:
```bash
git pull origin claude/chatbot-room-booking-019QTAbu8LfntyUfKK8wtAKt
go build ./cmd/server
# Detener servidor viejo (Ctrl+C)
./server  # O: go run cmd/server/main.go
```

---

### Error: "column birth_date does not exist"

**Causa**: Migración no ejecutada.

**Solución**:
```bash
psql -U postgres -d hotel_db -f migrations/002_fix_person_table.sql
```

---

### Error: Reserva muestra éxito pero no está en BD

**Causa**: Campos faltantes en Person o error silencioso.

**Solución**:
```bash
# Ver logs
grep "ERROR" server.log | tail -20

# Debe estar corregido en el commit 4882169
git log --oneline -1
# Debe mostrar: 4882169 fix: Corregir creación de reservas...
```

---

### Error: "GetByID undefined" para Client

**Causa**: Código antiguo.

**Solución**:
```bash
git pull
# Debe incluir commit ea86ead
```

---

## 📞 Soporte

Si alguna prueba falla:

1. **Revisar logs**: `grep "ERROR" server.log`
2. **Verificar commits**: `git log --oneline -5`
3. **Verificar migración**: `psql -U postgres -d hotel_db -c "\d person"`
4. **Verificar código actualizado**: `git status`

---

## ✅ Resultado Esperado Final

Después de completar todas las pruebas:

- ✅ Chatbot crea reservas exitosamente
- ✅ Reservas aparecen en base de datos
- ✅ Endpoint de verificación retorna datos completos
- ✅ Logs muestran todo el flujo sin errores
- ✅ No hay errores de `undefined` methods
- ✅ Campos `birth_date` y `active` están presentes

**¡El sistema está funcionando correctamente!** 🎉
