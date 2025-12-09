-- Migration to add icon_key and status columns to service table
-- Date: 2025-12-06
-- Description: Adds icon_key (varchar) and status (integer) columns to service table

ALTER TABLE service
ADD COLUMN IF NOT EXISTS icon_key varchar(50);

ALTER TABLE service
ADD COLUMN IF NOT EXISTS status integer DEFAULT 1 NOT NULL; -- 1: disponible, 0: no disponible

COMMENT ON COLUMN service.icon_key IS 'Descripción del icono del servicio para el frontend';
COMMENT ON COLUMN service.status IS 'Estado de disponibilidad del servicio (1: disponible, 0: no disponible)';
