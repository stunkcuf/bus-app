-- Railway Database Fix Script
-- This script ensures the admin user exists with the correct password

-- First, check what tables exist
SELECT table_name FROM information_schema.tables 
WHERE table_schema = 'public' 
ORDER BY table_name;

-- Check if users table exists and has data
SELECT COUNT(*) as user_count FROM users;

-- Check if admin exists
SELECT username, role, status, created_at 
FROM users 
WHERE username = 'admin';

-- If admin doesn't exist, create it
-- Password is already hashed for 'Headstart1' 
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

-- Verify admin was created/updated
SELECT username, role, status, 
       CASE WHEN password LIKE '$2a$%' THEN 'bcrypt hashed' ELSE 'plain text' END as password_type,
       created_at 
FROM users 
WHERE username = 'admin';

-- Show all users
SELECT username, role, status, created_at 
FROM users 
ORDER BY created_at;