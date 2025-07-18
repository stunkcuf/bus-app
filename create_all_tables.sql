-- Complete database schema for HS-Bus Fleet Management System
-- Run this in your Railway PostgreSQL database

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    username VARCHAR(50) PRIMARY KEY,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('manager', 'driver')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('active', 'pending')),
    registration_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create sessions table
CREATE TABLE IF NOT EXISTS sessions (
    token VARCHAR(255) PRIMARY KEY,
    username VARCHAR(50) NOT NULL REFERENCES users(username) ON DELETE CASCADE,
    csrf_token VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create buses table
CREATE TABLE IF NOT EXISTS buses (
    bus_id VARCHAR(50) PRIMARY KEY,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'maintenance', 'out_of_service')),
    model VARCHAR(100),
    capacity INTEGER,
    oil_status VARCHAR(20) DEFAULT 'good' CHECK (oil_status IN ('good', 'due_soon', 'overdue')),
    tire_status VARCHAR(20) DEFAULT 'good' CHECK (tire_status IN ('good', 'due_soon', 'overdue')),
    maintenance_notes TEXT,
    current_mileage INTEGER DEFAULT 0,
    last_oil_change INTEGER DEFAULT 0,
    last_tire_service INTEGER DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create vehicles table
CREATE TABLE IF NOT EXISTS vehicles (
    vehicle_id VARCHAR(50) PRIMARY KEY,
    model VARCHAR(100),
    description TEXT,
    year INTEGER,
    tire_size VARCHAR(50),
    license VARCHAR(50),
    oil_status VARCHAR(20) DEFAULT 'good' CHECK (oil_status IN ('good', 'due_soon', 'overdue')),
    tire_status VARCHAR(20) DEFAULT 'good' CHECK (tire_status IN ('good', 'due_soon', 'overdue')),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'maintenance', 'out_of_service')),
    maintenance_notes TEXT,
    serial_number VARCHAR(100),
    base VARCHAR(100),
    service_interval INTEGER,
    current_mileage INTEGER DEFAULT 0,
    last_oil_change INTEGER DEFAULT 0,
    last_tire_service INTEGER DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create routes table
CREATE TABLE IF NOT EXISTS routes (
    route_id VARCHAR(50) PRIMARY KEY,
    route_name VARCHAR(100) NOT NULL,
    description TEXT,
    positions JSONB DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create students table
CREATE TABLE IF NOT EXISTS students (
    student_id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    locations JSONB DEFAULT '[]',
    phone_number VARCHAR(20),
    alt_phone_number VARCHAR(20),
    guardian VARCHAR(100),
    pickup_time TIME,
    dropoff_time TIME,
    position_number INTEGER,
    route_id VARCHAR(50) REFERENCES routes(route_id) ON DELETE SET NULL,
    driver VARCHAR(50) REFERENCES users(username) ON DELETE SET NULL,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create route_assignments table
CREATE TABLE IF NOT EXISTS route_assignments (
    id SERIAL PRIMARY KEY,
    driver VARCHAR(50) NOT NULL REFERENCES users(username) ON DELETE CASCADE,
    bus_id VARCHAR(50) NOT NULL REFERENCES buses(bus_id) ON DELETE CASCADE,
    route_id VARCHAR(50) NOT NULL REFERENCES routes(route_id) ON DELETE CASCADE,
    assigned_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(driver, route_id),
    UNIQUE(bus_id, route_id)
);

-- Create maintenance log tables
CREATE TABLE IF NOT EXISTS bus_maintenance_logs (
    id SERIAL PRIMARY KEY,
    bus_id VARCHAR(50) NOT NULL REFERENCES buses(bus_id) ON DELETE CASCADE,
    date DATE NOT NULL,
    category VARCHAR(50) NOT NULL CHECK (category IN ('oil_change', 'tire_service', 'inspection', 'repair', 'other')),
    notes TEXT,
    mileage INTEGER,
    cost DECIMAL(10, 2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS vehicle_maintenance_logs (
    id SERIAL PRIMARY KEY,
    vehicle_id VARCHAR(50) NOT NULL REFERENCES vehicles(vehicle_id) ON DELETE CASCADE,
    date DATE NOT NULL,
    category VARCHAR(50) NOT NULL CHECK (category IN ('oil_change', 'tire_service', 'inspection', 'repair', 'other')),
    notes TEXT,
    mileage INTEGER,
    cost DECIMAL(10, 2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create driver logs table
CREATE TABLE IF NOT EXISTS driver_logs (
    id SERIAL PRIMARY KEY,
    driver VARCHAR(50) NOT NULL REFERENCES users(username) ON DELETE CASCADE,
    bus_id VARCHAR(50) NOT NULL REFERENCES buses(bus_id) ON DELETE CASCADE,
    route_id VARCHAR(50) NOT NULL REFERENCES routes(route_id) ON DELETE CASCADE,
    date DATE NOT NULL,
    period VARCHAR(20) NOT NULL CHECK (period IN ('morning', 'afternoon')),
    departure_time TIME,
    arrival_time TIME,
    begin_mileage DOUBLE PRECISION,
    end_mileage DOUBLE PRECISION,
    attendance JSONB DEFAULT '[]',
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_bus_maintenance_logs_bus_id ON bus_maintenance_logs(bus_id);
CREATE INDEX IF NOT EXISTS idx_vehicle_maintenance_logs_vehicle_id ON vehicle_maintenance_logs(vehicle_id);
CREATE INDEX IF NOT EXISTS idx_students_route_id ON students(route_id);
CREATE INDEX IF NOT EXISTS idx_route_assignments_driver ON route_assignments(driver);
CREATE INDEX IF NOT EXISTS idx_driver_logs_driver ON driver_logs(driver);
CREATE INDEX IF NOT EXISTS idx_driver_logs_date ON driver_logs(date);

-- Now create the admin user
INSERT INTO users (username, password, role, status, registration_date, created_at)
VALUES (
    'admin',
    '$2a$10$9wIkdWfdRvoLrIkLa7czSucxMxk1Do7t/022UO4y1oLiz0VHZg29e',
    'manager',
    'active',
    CURRENT_DATE,
    CURRENT_TIMESTAMP
) ON CONFLICT (username) DO UPDATE
SET password = EXCLUDED.password,
    role = EXCLUDED.role,
    status = EXCLUDED.status;

-- Verify everything was created
SELECT 'Users table' as table_name, COUNT(*) as row_count FROM users
UNION ALL
SELECT 'Admin user exists' as table_name, COUNT(*) as row_count FROM users WHERE username = 'admin';