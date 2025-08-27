-- Initialize the database
CREATE DATABASE IF NOT EXISTS jan_api_gateway;

-- Create the user if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'jan_user') THEN
        CREATE ROLE jan_user WITH LOGIN PASSWORD 'jan_password';
    END IF;
END
$$;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE jan_api_gateway TO jan_user;
GRANT ALL PRIVILEGES ON SCHEMA public TO jan_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO jan_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO jan_user;

-- Set default privileges for future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO jan_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO jan_user;
