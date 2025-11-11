-- Tabla para almacenar tokens de encuestas
CREATE TABLE survey_token (
    token_id SERIAL PRIMARY KEY,
    token VARCHAR(64) UNIQUE NOT NULL,
    reservation_id INTEGER NOT NULL REFERENCES reservation(reservation_id),
    client_id INTEGER NOT NULL REFERENCES client(client_id),
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Índices
    CONSTRAINT unique_token_per_reservation UNIQUE (reservation_id)
);

CREATE INDEX idx_survey_token_token ON survey_token(token);
CREATE INDEX idx_survey_token_reservation ON survey_token(reservation_id);
CREATE INDEX idx_survey_token_expires ON survey_token(expires_at);

COMMENT ON TABLE survey_token IS 'Tokens for satisfaction survey links sent via email';
COMMENT ON COLUMN survey_token.token IS 'Unique token for survey access';
COMMENT ON COLUMN survey_token.expires_at IS 'Token expiration date (e.g., 30 days after checkout)';
COMMENT ON COLUMN survey_token.used IS 'Whether the survey has been completed';
