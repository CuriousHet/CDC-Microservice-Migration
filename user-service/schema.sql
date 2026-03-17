-- Schema for the independent User Service
CREATE TABLE IF NOT EXISTS users (
    id INT PRIMARY KEY, -- Using the original ID from Monolith
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
