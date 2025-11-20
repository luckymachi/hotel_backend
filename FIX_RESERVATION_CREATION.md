# Fix: Reservas del Chatbot No Se Crean en Base de Datos

## 🐛 Problema Identificado

El chatbot mostraba mensajes de éxito al crear reservas, pero las reservas NO se estaban guardando en la base de datos.

### Causa Raíz

El código tenía **3 problemas críticos**:

1. **Falta de campos requeridos en Person**:
   - El código intentaba insertar en la tabla `person` con campos `birth_date`, `active` y `creation_date`
   - Pero al crear el objeto `Person` en `chatbot_tools.go`, estos campos NO se estaban asignando
   - Resultado: valores vacíos (`time.Time{}` y `false`) que podían causar errores en la BD

2. **Discrepancia de nombres de tabla/columnas**:
   - El código usa nombres en inglés (`person`, `person_id`, `name`, etc.)
   - El schema en `tables.txt` usa nombres en español (`persona`, `personaid`, `nombre`, etc.)
   - Esto causaba que las queries fallaran porque la tabla o columnas no existían

3. **Falta de logging detallado**:
   - Los errores NO se estaban logueando adecuadamente
   - Era difícil identificar dónde fallaba el proceso

## ✅ Solución Implementada

### 1. Corrección de Campos en Person (chatbot_tools.go:243-261)

```go
// Antes (INCORRECTO):
person := &domain.Person{
    Name:             input.PersonalData.Nombre,
    FirstSurname:     input.PersonalData.PrimerApellido,
    // ... otros campos ...
    // ❌ Faltaban: BirthDate, Active, CreationDate
}

// Después (CORRECTO):
defaultBirthDate := time.Now().AddDate(-18, 0, 0) // 18 años atrás

person := &domain.Person{
    Name:             input.PersonalData.Nombre,
    FirstSurname:     input.PersonalData.PrimerApellido,
    // ... otros campos ...
    ReferenceCity:    getStringValue(input.PersonalData.CiudadReferencia),
    ReferenceCountry: getStringValue(input.PersonalData.PaisReferencia),
    Active:           true,                    // ✅ NUEVO
    CreationDate:     time.Now(),              // ✅ NUEVO
    BirthDate:        defaultBirthDate,        // ✅ NUEVO
}
```

### 2. Migración de Base de Datos (migrations/002_fix_person_table.sql)

Cree una migración que:
- ✅ Renombra tabla `persona` → `person` (si existe)
- ✅ Renombra todas las columnas de español a inglés
- ✅ Agrega campo `birth_date` si no existe
- ✅ Actualiza referencias en tabla `cliente`/`client`
- ✅ Crea índices para mejorar performance
- ✅ Verifica que todo se ejecutó correctamente

### 3. Logging Mejorado

Agregué logs detallados en múltiples puntos:

**chatbot_tools.go:283-297**:
```go
log.Printf("[CreateReservation] Creando reserva para cliente: %s %s (DNI: %s)",
    person.Name, person.FirstSurname, person.DocumentNumber)

if err := rt.reservaService.CreateReservaWithClient(person, reserva); err != nil {
    log.Printf("[CreateReservation] ERROR al crear reserva: %v", err)
    return "", fmt.Errorf("error al crear la reserva: %w", err)
}

// Verificar que la reserva se creó (ID > 0)
if reserva.ID == 0 {
    log.Printf("[CreateReservation] ERROR: Reserva creada pero ID es 0")
    return "", fmt.Errorf("error: la reserva no se creó correctamente en la base de datos")
}

log.Printf("[CreateReservation] Reserva creada exitosamente con ID: %d", reserva.ID)
```

**intent_detector.go:173-180**:
```go
log.Printf("[IntentDetector] Ejecutando create_reservation con datos: %s", string(args))
result, err := d.reservationTools.CreateReservation(string(args))

if err != nil {
    log.Printf("[IntentDetector] ERROR en create_reservation: %v", err)
} else {
    log.Printf("[IntentDetector] create_reservation exitoso: %s", result)
}
```

### 4. Función Helper para *string

Agregué una función helper para convertir `*string` a `string` (chatbot_tools.go:317-323):

```go
func getStringValue(s *string) string {
    if s == nil {
        return ""
    }
    return *s
}
```

## 🚀 Cómo Aplicar la Solución

### Paso 1: Actualizar el Código

```bash
git pull origin claude/chatbot-room-booking-019QTAbu8LfntyUfKK8wtAKt
```

### Paso 2: Ejecutar la Migración

```bash
psql -U postgres -d hotel_db -f migrations/002_fix_person_table.sql
```

**Salida esperada**:
```
NOTICE:  Tabla persona renombrada a person y columnas traducidas al inglés
NOTICE:  Campo birth_date agregado a tabla person
NOTICE:  Constraint UNIQUE agregado a email
NOTICE:  Columna personaid renombrada a person_id en tabla cliente
NOTICE:  ✅ Migración completada exitosamente. Tabla person tiene todas las columnas necesarias.
```

### Paso 3: Reiniciar el Servidor

```bash
# Detener el servidor actual
# Compilar y ejecutar de nuevo
go run cmd/server/main.go
```

### Paso 4: Probar el Flujo Completo

#### Con Postman:

1. **Iniciar conversación**:
```json
POST http://localhost:8080/api/chatbot/chat

{
  "message": "Hola, quiero hacer una reserva",
  "clienteId": 11
}
```

2. **Proporcionar fechas**:
```json
{
  "message": "Del 15 al 20 de diciembre",
  "clienteId": 11,
  "conversationId": "<ID_ANTERIOR>"
}
```

3. **Seleccionar habitación**:
```json
{
  "message": "La suite presidencial",
  "clienteId": 11,
  "conversationId": "<ID_ANTERIOR>"
}
```

4. **Proporcionar cantidad de personas**:
```json
{
  "message": "2 adultos",
  "clienteId": 11,
  "conversationId": "<ID_ANTERIOR>"
}
```

5. **Proporcionar datos personales**:
```json
{
  "message": "Mi nombre es Juan Pérez, DNI 12345678, correo juan@email.com, teléfono 987654321",
  "clienteId": 11,
  "conversationId": "<ID_ANTERIOR>"
}
```

6. **Confirmar**:
```json
{
  "message": "Sí, confirmo",
  "clienteId": 11,
  "conversationId": "<ID_ANTERIOR>"
}
```

#### Verificar en Base de Datos:

```sql
-- Ver última reserva creada
SELECT * FROM reservation ORDER BY reservation_id DESC LIMIT 1;

-- Ver datos del cliente
SELECT p.*, c.*
FROM person p
JOIN client c ON p.person_id = c.person_id
WHERE c.client_id = 11;

-- Verificación completa con el nuevo endpoint
```

#### O usar el Endpoint de Verificación:

```bash
# Si la respuesta del chatbot dice "Número de Reserva: #123"
curl http://localhost:8080/api/reservas/123/verify
```

## 📊 Logs a Revisar

Después de ejecutar el flujo, revisa los logs del servidor:

```bash
# Buscar logs de creación de reserva
grep "CreateReservation" server.log

# Buscar errores
grep "ERROR" server.log

# Ver flujo completo del intent detector
grep "IntentDetector" server.log
```

### Logs Exitosos (Esperados):
```
[IntentDetector] Ejecutando create_reservation con datos: {"fechaEntrada":"2025-12-15",...}
[CreateReservation] Creando reserva para cliente: Juan Pérez (DNI: 12345678)
[CreateReservation] Reserva creada exitosamente con ID: 123
[IntentDetector] create_reservation exitoso: ✅ Reserva creada exitosamente!...
```

### Logs con Error (Problemas):
```
[IntentDetector] ERROR en create_reservation: error al crear persona: pq: column "birth_date" does not exist
[CreateReservation] ERROR al crear reserva: error al crear persona: ...
```

## 🔍 Troubleshooting

### Error: "column birth_date does not exist"

**Causa**: La migración no se ejecutó correctamente.

**Solución**:
```bash
# Verificar que la tabla person existe
psql -U postgres -d hotel_db -c "\d person"

# Re-ejecutar migración
psql -U postgres -d hotel_db -f migrations/002_fix_person_table.sql
```

### Error: "table person does not exist"

**Causa**: La migración no pudo encontrar ni tabla `person` ni `persona`.

**Solución**:
```bash
# Verificar qué tablas existen
psql -U postgres -d hotel_db -c "\dt"

# Si existe "persona", ejecutar migración
# Si no existe ninguna, crear manualmente
```

### El chatbot sigue mostrando éxito pero no crea la reserva

**Causa**: El código antiguo aún está en ejecución.

**Solución**:
1. Detener el servidor completamente
2. Hacer `git pull` para obtener los cambios
3. Recompilar: `go build cmd/server/main.go`
4. Ejecutar de nuevo

### Campos NULL en person

**Causa**: reference_city o reference_country son NULL pero se esperan vacíos.

**Solución**:
La función `getStringValue()` ya maneja esto, pero verificar que se esté usando:
```go
ReferenceCity:    getStringValue(input.PersonalData.CiudadReferencia),
ReferenceCountry: getStringValue(input.PersonalData.PaisReferencia),
```

## 📝 Archivos Modificados

| Archivo | Cambios |
|---------|---------|
| `internal/application/chatbot_tools.go` | ✅ Agregados campos BirthDate, Active, CreationDate<br>✅ Agregada función `getStringValue`<br>✅ Agregado logging detallado<br>✅ Agregada validación de reserva.ID > 0 |
| `internal/application/intent_detector.go` | ✅ Agregado logging de create_reservation |
| `migrations/002_fix_person_table.sql` | ✅ Nueva migración para renombrar y agregar campos |
| `FIX_RESERVATION_CREATION.md` | ✅ Nueva documentación del problema y solución |

## ✅ Checklist de Verificación

Después de aplicar la solución, verificar:

- [ ] La migración se ejecutó sin errores
- [ ] La tabla `person` existe con todas las columnas
- [ ] El servidor se reinició con el código actualizado
- [ ] Los logs muestran `[CreateReservation]` messages
- [ ] Crear una reserva de prueba con el chatbot
- [ ] Verificar la reserva con: `GET /api/reservas/{id}/verify`
- [ ] Confirmar que la reserva existe en BD
- [ ] Confirmar que la persona existe en BD
- [ ] Confirmar que el cliente existe en BD

## 🎯 Resultado Esperado

Después de aplicar la solución:

1. **Chatbot**:
   - ✅ Muestra "✅ Reserva creada exitosamente! Número de Reserva: #123"
   - ✅ Los logs muestran el ID de la reserva creada

2. **Base de Datos**:
   - ✅ Existe un registro en `reservation` con el ID correcto
   - ✅ Existe un registro en `person` con birth_date, active=true
   - ✅ Existe un registro en `client` vinculado a la persona
   - ✅ Existe un registro en `reservation_room` con las fechas

3. **API de Verificación**:
   - ✅ `GET /api/reservas/123/verify` retorna todos los datos completos
   - ✅ Muestra información de reserva, cliente, persona y habitaciones

## 📚 Referencias

- [VERIFICATION_API.md](./VERIFICATION_API.md) - Cómo verificar reservas con la API
- [DATABASE_VERIFICATION_GUIDE.md](./DATABASE_VERIFICATION_GUIDE.md) - Consultas SQL para verificar
- [CHATBOT_V2_INTENT_DETECTION.md](./CHATBOT_V2_INTENT_DETECTION.md) - Cómo funciona el chatbot

---

Si el problema persiste después de aplicar esta solución, revisar los logs del servidor y compartir los mensajes de error específicos.
