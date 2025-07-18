-- Create import history table to track all Excel imports
CREATE TABLE IF NOT EXISTS import_history (
    id SERIAL PRIMARY KEY,
    import_id VARCHAR(50) UNIQUE NOT NULL,
    import_type VARCHAR(20) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    total_rows INTEGER DEFAULT 0,
    successful_rows INTEGER DEFAULT 0,
    failed_rows INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    warning_count INTEGER DEFAULT 0,
    summary TEXT,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    duration INTERVAL GENERATED ALWAYS AS (end_time - start_time) STORED,
    imported_by INTEGER REFERENCES users(id),
    rollback_available BOOLEAN DEFAULT TRUE,
    rollback_expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient querying
CREATE INDEX idx_import_history_import_type ON import_history(import_type);
CREATE INDEX idx_import_history_start_time ON import_history(start_time DESC);
CREATE INDEX idx_import_history_imported_by ON import_history(imported_by);

-- Create import errors table to store detailed error information
CREATE TABLE IF NOT EXISTS import_errors (
    id SERIAL PRIMARY KEY,
    import_id VARCHAR(50) REFERENCES import_history(import_id) ON DELETE CASCADE,
    row_number INTEGER,
    column_name VARCHAR(100),
    sheet_name VARCHAR(100),
    error_type VARCHAR(50),
    error_message TEXT,
    error_value TEXT,
    severity VARCHAR(20) DEFAULT 'error', -- error, warning, info
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for efficient error lookup
CREATE INDEX idx_import_errors_import_id ON import_errors(import_id);

-- Add import_id column to existing tables for tracking which import created each record
ALTER TABLE mileage_records ADD COLUMN IF NOT EXISTS import_id VARCHAR(50);
ALTER TABLE ecse_students ADD COLUMN IF NOT EXISTS import_id VARCHAR(50);
ALTER TABLE students ADD COLUMN IF NOT EXISTS import_id VARCHAR(50);
ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS import_id VARCHAR(50);
ALTER TABLE agency_vehicles ADD COLUMN IF NOT EXISTS import_id VARCHAR(50);
ALTER TABLE school_buses ADD COLUMN IF NOT EXISTS import_id VARCHAR(50);
ALTER TABLE program_staff ADD COLUMN IF NOT EXISTS import_id VARCHAR(50);

-- Create indexes for import tracking
CREATE INDEX IF NOT EXISTS idx_mileage_records_import_id ON mileage_records(import_id);
CREATE INDEX IF NOT EXISTS idx_ecse_students_import_id ON ecse_students(import_id);
CREATE INDEX IF NOT EXISTS idx_students_import_id ON students(import_id);
CREATE INDEX IF NOT EXISTS idx_vehicles_import_id ON vehicles(import_id);
CREATE INDEX IF NOT EXISTS idx_agency_vehicles_import_id ON agency_vehicles(import_id);
CREATE INDEX IF NOT EXISTS idx_school_buses_import_id ON school_buses(import_id);
CREATE INDEX IF NOT EXISTS idx_program_staff_import_id ON program_staff(import_id);