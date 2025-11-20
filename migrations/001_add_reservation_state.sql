-- Migration to add reservation state support in conversation_history
-- Date: 2025-11-20
-- Description: Adds column to store reservation in progress state during chatbot conversations

-- Add reservation_state column to conversation_history
ALTER TABLE conversation_history
ADD COLUMN IF NOT EXISTS reservation_state JSONB;

-- Create index for efficient searches of conversations with reservations in progress
CREATE INDEX IF NOT EXISTS idx_conversation_reservation
ON conversation_history ((reservation_state IS NOT NULL))
WHERE reservation_state IS NOT NULL;

-- Add comments for documentation
COMMENT ON COLUMN conversation_history.reservation_state IS 'Stores the state of a reservation in progress as JSON (dates, guests, room type, personal data, step)';
COMMENT ON TABLE conversation_history IS 'Stores complete chatbot conversation history including messages and reservation state';
