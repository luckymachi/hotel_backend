-- Migración para agregar soporte de estado de reserva en conversation_history
-- Fecha: 2025-11-20
-- Descripción: Agrega columna para almacenar el estado de una reserva en progreso durante conversaciones del chatbot

ALTER TABLE conversation_history
ADD COLUMN IF NOT EXISTS reservation_state JSONB;

-- Crear índice para búsquedas eficientes de conversaciones con reservas en progreso
CREATE INDEX IF NOT EXISTS idx_conversation_reservation
ON conversation_history ((reservation_state IS NOT NULL))
WHERE reservation_state IS NOT NULL;
