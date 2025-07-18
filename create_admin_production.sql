-- Script to create admin user in production database
-- Run this in your Railway PostgreSQL database

-- First, check if the admin user exists
SELECT username, role, status FROM users WHERE username = 'admin';

-- If the user doesn't exist, run this INSERT:
-- Password is already hashed for 'Headstart1'
INSERT INTO users (username, password, role, status, registration_date, created_at)
VALUES (
    'admin',
    '$2a$10$YourHashedPasswordHere', -- This will be replaced with actual hash
    'manager',
    'active',
    CURRENT_DATE,
    CURRENT_TIMESTAMP
);

-- If the user exists but you need to reset the password, use this UPDATE:
-- UPDATE users 
-- SET password = '$2a$10$YourHashedPasswordHere',
--     role = 'manager',
--     status = 'active'
-- WHERE username = 'admin';