# API de Verificación de Reservas

## 📋 Descripción

Este documento describe cómo usar el endpoint de verificación de reservas para asegurar que las reservas se crearon correctamente en la base de datos.

---

## 🔍 Endpoint de Verificación

### **GET** `/api/reservas/:id/verify`

Obtiene información completa y detallada de una reserva para verificación.

#### **Parámetros**

| Parámetro | Tipo | Ubicación | Descripción |
|-----------|------|-----------|-------------|
| `id` | integer | path | ID de la reserva a verificar |

#### **Respuesta Exitosa** (200 OK)

```json
{
  "success": true,
  "message": "Reserva verificada exitosamente",
  "data": {
    "reservation": {
      "id": 123,
      "cantidadAdultos": 2,
      "cantidadNinhos": 0,
      "estado": "Pendiente",
      "clienteId": 45,
      "subtotal": 1000.00,
      "descuento": 0,
      "fechaConfirmacion": "2025-11-20T10:30:00Z",
      "habitaciones": [
        {
          "habitacionId": 5,
          "precio": 200.00,
          "fechaEntrada": "2025-12-15T00:00:00Z",
          "fechaSalida": "2025-12-20T00:00:00Z",
          "estado": 1
        }
      ],
      "servicios": []
    },
    "client": {
      "client_id": 45,
      "person_id": 78,
      "capture_channel": "chatbot",
      "capture_status": "cliente",
      "registration_date": "2025-11-20T10:29:00Z"
    },
    "person": {
      "person_id": 78,
      "name": "Juan",
      "first_surname": "Pérez",
      "second_surname": "García",
      "document_number": "12345678",
      "gender": "M",
      "email": "juan.perez@email.com",
      "phone_1": "987654321",
      "phone_2": null,
      "reference_city": "Lima",
      "reference_country": "Perú",
      "birth_date": "1990-05-15T00:00:00Z",
      "creation_date": "2025-11-20T10:29:00Z",
      "active": true
    },
    "rooms": [
      {
        "roomId": 5,
        "roomNumber": "201",
        "roomName": "Habitación Deluxe 201",
        "roomType": "Suite Presidencial",
        "checkInDate": "2025-12-15T00:00:00Z",
        "checkOutDate": "2025-12-20T00:00:00Z",
        "price": 200.00,
        "nights": 5,
        "totalPrice": 1000.00
      }
    ],
    "payments": [
      {
        "payment_id": 10,
        "reservation_id": 123,
        "amount": 1000.00,
        "date": "2025-11-20T10:30:00Z",
        "payment_method": "card",
        "status": "completed"
      }
    ],
    "conversationId": null,
    "verificationTime": "2025-11-20T11:00:00Z"
  }
}
```

#### **Respuesta de Error** (404 Not Found)

```json
{
  "error": "error al obtener reserva: reservation not found"
}
```

#### **Respuesta de Error** (400 Bad Request)

```json
{
  "error": "ID de reserva inválido"
}
```

---

## 📝 Ejemplos de Uso

### Con cURL

```bash
# Verificar reserva con ID 123
curl http://localhost:8080/api/reservas/123/verify
```

### Con Postman

1. **Method**: GET
2. **URL**: `http://localhost:8080/api/reservas/123/verify`
3. **Headers**: (ninguno requerido)
4. **Click**: Send

### Con JavaScript/Fetch

```javascript
async function verifyReservation(reservationId) {
  const response = await fetch(`http://localhost:8080/api/reservas/${reservationId}/verify`);

  if (!response.ok) {
    throw new Error('Error al verificar reserva');
  }

  const data = await response.json();
  return data.data; // Retorna los datos de verificación
}

// Uso
verifyReservation(123)
  .then(verification => {
    console.log('Reserva verificada:', verification);
    console.log('Cliente:', verification.person.name);
    console.log('Total:', verification.reservation.subtotal);
    console.log('Habitaciones:', verification.rooms.length);
  })
  .catch(error => console.error(error));
```

### Con Python (requests)

```python
import requests

def verify_reservation(reservation_id):
    url = f"http://localhost:8080/api/reservas/{reservation_id}/verify"
    response = requests.get(url)

    if response.status_code == 200:
        data = response.json()
        return data['data']
    else:
        raise Exception(f"Error: {response.json().get('error')}")

# Uso
try:
    verification = verify_reservation(123)
    print(f"Reserva ID: {verification['reservation']['id']}")
    print(f"Cliente: {verification['person']['name']} {verification['person']['first_surname']}")
    print(f"Email: {verification['person']['email']}")
    print(f"Total: S/ {verification['reservation']['subtotal']}")
    print(f"Habitaciones reservadas: {len(verification['rooms'])}")

    for room in verification['rooms']:
        print(f"  - {room['roomType']}: {room['nights']} noches = S/ {room['totalPrice']}")

except Exception as e:
    print(f"Error al verificar: {e}")
```

---

## 🔍 Qué Verificar

Cuando uses este endpoint después de crear una reserva con el chatbot, verifica:

### ✅ Checklist de Verificación

- [ ] **Reserva creada**: `reservation.id` existe y es mayor que 0
- [ ] **Estado correcto**: `reservation.estado` es "Pendiente" o "Confirmada"
- [ ] **Huéspedes correctos**: `reservation.cantidadAdultos` y `cantidadNinhos` coinciden
- [ ] **Cliente vinculado**: `client` no es null
- [ ] **Persona creada**: `person` tiene los datos correctos (nombre, email, documento)
- [ ] **Email correcto**: `person.email` coincide con el proporcionado
- [ ] **Habitación asignada**: `rooms` tiene al menos 1 elemento
- [ ] **Fechas correctas**: `checkInDate` y `checkOutDate` son las esperadas
- [ ] **Precio correcto**: `totalPrice` coincide con `reservation.subtotal`
- [ ] **Noches calculadas**: `nights` = días entre check-in y check-out
- [ ] **Pago registrado**: `payments` tiene registros (si se procesó pago)

### 🐛 Problemas Comunes

#### 1. Reserva sin habitación

```json
{
  "rooms": []
}
```

**Causa**: No se asignó habitación o no había disponibilidad.

**Solución**: Verificar disponibilidad con consultas SQL del DATABASE_VERIFICATION_GUIDE.md

#### 2. Cliente duplicado

```json
{
  "person": {
    "person_id": 100  // ID diferente al esperado
  }
}
```

**Causa**: Ya existía una persona con el mismo DNI.

**Solución**: Verificar con SQL: `SELECT * FROM person WHERE document_number = '12345678'`

#### 3. Error 404 - Reserva no encontrada

**Causa**: La reserva no se creó en la BD.

**Solución**:
1. Revisar logs del servidor
2. Usar consultas SQL para ver última reserva creada
3. Verificar que el chatbot ejecutó `create_reservation` correctamente

---

## 🔗 Endpoints Relacionados

### Obtener Reserva Simple

```
GET /api/reservas/:id
```

Retorna solo la información básica de la reserva, sin los detalles completos de cliente, persona y habitaciones.

### Obtener Reservas de un Cliente

```
GET /api/reservas/cliente/:clienteId
```

Retorna todas las reservas de un cliente específico.

### Verificar Conversación del Chatbot

```
GET /api/chatbot/conversation/:id
```

Retorna el historial completo de la conversación que generó la reserva.

---

## 💡 Casos de Uso

### 1. Verificación Post-Creación

Después de que el chatbot confirme una reserva:

```javascript
// El chatbot retorna:
{
  "message": "¡Reserva confirmada! Su número es #123",
  "reservationCreated": 123
}

// Inmediatamente verificar:
const verification = await verifyReservation(123);
console.log('✅ Reserva verificada:', verification);
```

### 2. Testing de Integración

```javascript
describe('Chatbot Reservation Flow', () => {
  it('should create a complete reservation', async () => {
    // 1. Crear reserva con chatbot
    const chatResponse = await createReservationViaChat({
      message: "Quiero reservar del 15 al 20 de diciembre para 2 personas",
      // ... más mensajes
    });

    // 2. Verificar que se creó
    expect(chatResponse.reservationCreated).toBeDefined();

    // 3. Verificar completitud
    const verification = await verifyReservation(chatResponse.reservationCreated);

    expect(verification.reservation).toBeDefined();
    expect(verification.person.email).toBe('test@example.com');
    expect(verification.rooms.length).toBeGreaterThan(0);
    expect(verification.rooms[0].nights).toBe(5);
  });
});
```

### 3. Dashboard de Admin

```javascript
function ReservationVerificationPanel({ reservationId }) {
  const [verification, setVerification] = useState(null);

  useEffect(() => {
    verifyReservation(reservationId)
      .then(setVerification)
      .catch(console.error);
  }, [reservationId]);

  if (!verification) return <Loading />;

  return (
    <div className="verification-panel">
      <h2>Reserva #{verification.reservation.id}</h2>

      <Section title="Cliente">
        <p>{verification.person.name} {verification.person.first_surname}</p>
        <p>Email: {verification.person.email}</p>
        <p>Teléfono: {verification.person.phone_1}</p>
      </Section>

      <Section title="Habitaciones">
        {verification.rooms.map(room => (
          <RoomCard key={room.roomId} room={room} />
        ))}
      </Section>

      <Section title="Pagos">
        <p>Total: S/ {verification.reservation.subtotal}</p>
        {verification.payments.map(payment => (
          <PaymentCard key={payment.payment_id} payment={payment} />
        ))}
      </Section>
    </div>
  );
}
```

---

## 🔒 Seguridad

### Recomendaciones

1. **Autenticación**: En producción, este endpoint debería requerir autenticación
2. **Autorización**: Solo el cliente dueño de la reserva o un admin debería poder verificarla
3. **Rate Limiting**: Implementar límite de requests para prevenir abuso

### Ejemplo con Autenticación

```javascript
// Con JWT token
async function verifyReservation(reservationId, token) {
  const response = await fetch(`/api/reservas/${reservationId}/verify`, {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });

  return response.json();
}
```

---

## 📊 Métricas

El endpoint de verificación es útil para:

- **Testing automatizado**: Verificar que el flujo completo funciona
- **Debugging**: Identificar dónde falla el proceso de reserva
- **Auditoría**: Verificar integridad de datos
- **Soporte**: Ayudar a clientes con problemas en sus reservas
- **Analytics**: Analizar patrones de reservas creadas

---

## 🛠️ Troubleshooting

### El endpoint retorna 404

**Problema**: Reserva no existe en BD

**Soluciones**:
1. Verificar con SQL: `SELECT * FROM reservation ORDER BY reservation_id DESC LIMIT 1;`
2. Revisar logs del chatbot para ver si `create_reservation` se ejecutó
3. Verificar que el ID es correcto

### El endpoint retorna datos incompletos

**Problema**: Relaciones no creadas correctamente

**Soluciones**:
1. Verificar que existe el cliente: `SELECT * FROM client WHERE client_id = X`
2. Verificar que existe la persona: `SELECT * FROM person WHERE person_id = X`
3. Verificar habitaciones asignadas: `SELECT * FROM reservation_room WHERE reservation_id = X`

### Datos de persona incorrectos

**Problema**: Se reutilizó persona existente con datos viejos

**Solución**:
- El sistema actualiza los datos al crear la reserva, pero si el email/teléfono no coincide, verificar la lógica en `CreateReservaWithClientAndPayment`

---

## 📚 Referencias

- [DATABASE_VERIFICATION_GUIDE.md](./DATABASE_VERIFICATION_GUIDE.md) - Consultas SQL para verificación manual
- [CONVERSATION_STORAGE.md](./CONVERSATION_STORAGE.md) - Cómo se guardan las conversaciones
- [CHATBOT_V2_INTENT_DETECTION.md](./CHATBOT_V2_INTENT_DETECTION.md) - Cómo funciona la detección de intenciones

---

## 🎯 Próximos Pasos

Posibles mejoras al endpoint de verificación:

- [ ] Agregar campo `conversationId` buscando en `conversation_history`
- [ ] Incluir timeline de eventos de la reserva
- [ ] Agregar validaciones adicionales (fechas en pasado, etc.)
- [ ] Endpoint para verificar múltiples reservas a la vez
- [ ] Webhook de notificación cuando una reserva necesita atención

---

¿Necesitas ayuda? Revisa el [DATABASE_VERIFICATION_GUIDE.md](./DATABASE_VERIFICATION_GUIDE.md) para consultas SQL complementarias.
