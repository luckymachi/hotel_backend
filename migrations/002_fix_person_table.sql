-- Migración para asegurar que la tabla person existe con todos los campos necesarios
-- Esta migración maneja tanto el caso de tabla "persona" (español) como "person" (inglés)

-- Paso 1: Si existe tabla "persona" y no existe "person", renombrarla
DO $$
BEGIN
    -- Verificar si existe tabla persona y no existe person
    IF EXISTS (SELECT FROM pg_tables WHERE tablename = 'persona')
       AND NOT EXISTS (SELECT FROM pg_tables WHERE tablename = 'person') THEN

        ALTER TABLE persona RENAME TO person;
        ALTER TABLE person RENAME COLUMN personaid TO person_id;
        ALTER TABLE person RENAME COLUMN nombre TO name;
        ALTER TABLE person RENAME COLUMN primerapellido TO first_surname;
        ALTER TABLE person RENAME COLUMN segundoapellido TO second_surname;
        ALTER TABLE person RENAME COLUMN numerodocumento TO document_number;
        ALTER TABLE person RENAME COLUMN genero TO gender;
        ALTER TABLE person RENAME COLUMN correo TO email;
        ALTER TABLE person RENAME COLUMN telefono_1 TO phone_1;
        ALTER TABLE person RENAME COLUMN telefono_2 TO phone_2;
        ALTER TABLE person RENAME COLUMN ciudadreferencia TO reference_city;
        ALTER TABLE person RENAME COLUMN paisreferencia TO reference_country;
        ALTER TABLE person RENAME COLUMN activo TO active;
        ALTER TABLE person RENAME COLUMN fechacreacion TO creation_date;

        RAISE NOTICE 'Tabla persona renombrada a person y columnas traducidas al inglés';
    END IF;
END $$;

-- Paso 2: Agregar campo birth_date si no existe
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT FROM information_schema.columns
        WHERE table_name = 'person' AND column_name = 'birth_date'
    ) THEN
        ALTER TABLE person ADD COLUMN birth_date TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP - INTERVAL '18 years');
        RAISE NOTICE 'Campo birth_date agregado a tabla person';
    END IF;
END $$;

-- Paso 3: Actualizar constraints si es necesario
DO $$
BEGIN
    -- Asegurar que email sea unique
    IF NOT EXISTS (
        SELECT FROM pg_constraint
        WHERE conname = 'person_email_key'
    ) THEN
        ALTER TABLE person ADD CONSTRAINT person_email_key UNIQUE (email);
        RAISE NOTICE 'Constraint UNIQUE agregado a email';
    END IF;
END $$;

-- Paso 4: Actualizar referencias en otras tablas
DO $$
BEGIN
    -- Si existe tabla cliente con personaid, renombrar a person_id
    IF EXISTS (
        SELECT FROM information_schema.columns
        WHERE table_name = 'cliente' AND column_name = 'personaid'
    ) AND EXISTS (
        SELECT FROM pg_tables WHERE tablename = 'cliente'
    ) THEN
        ALTER TABLE cliente RENAME COLUMN personaid TO person_id;
        RAISE NOTICE 'Columna personaid renombrada a person_id en tabla cliente';
    END IF;

    -- Si existe tabla client con personaid, renombrar a person_id
    IF EXISTS (
        SELECT FROM information_schema.columns
        WHERE table_name = 'client' AND column_name = 'personaid'
    ) THEN
        ALTER TABLE client RENAME COLUMN personaid TO person_id;
        RAISE NOTICE 'Columna personaid renombrada a person_id en tabla client';
    END IF;
END $$;

-- Paso 5: Crear índices para mejorar performance
CREATE INDEX IF NOT EXISTS idx_person_document_number ON person(document_number);
CREATE INDEX IF NOT EXISTS idx_person_email ON person(email);

-- Verificación final
DO $$
DECLARE
    table_exists BOOLEAN;
    column_count INTEGER;
BEGIN
    -- Verificar que tabla person existe
    SELECT EXISTS (
        SELECT FROM pg_tables WHERE tablename = 'person'
    ) INTO table_exists;

    IF NOT table_exists THEN
        RAISE EXCEPTION 'ERROR: Tabla person no existe después de la migración';
    END IF;

    -- Contar columnas esperadas
    SELECT COUNT(*) INTO column_count
    FROM information_schema.columns
    WHERE table_name = 'person'
    AND column_name IN ('person_id', 'name', 'first_surname', 'second_surname',
                        'document_number', 'gender', 'email', 'phone_1', 'phone_2',
                        'reference_city', 'reference_country', 'active',
                        'creation_date', 'birth_date');

    IF column_count < 14 THEN
        RAISE WARNING 'Advertencia: Tabla person tiene solo % de 14 columnas esperadas', column_count;
    ELSE
        RAISE NOTICE '✅ Migración completada exitosamente. Tabla person tiene todas las columnas necesarias.';
    END IF;
END $$;
