-- Seed data for CodeAtlas development environment
-- This script creates sample repositories, files, and relationships for testing

-- Set search path to match schema initialization
LOAD 'age';
SET search_path = ag_catalog, "$user", public;

-- Insert sample repositories
INSERT INTO repositories (repo_id, name, url, branch, metadata, created_at, updated_at) VALUES
('550e8400-e29b-41d4-a716-446655440001', 'sample-go-api', 'https://github.com/example/sample-go-api', 'main',
 '{"description": "Sample Go REST API project", "language": "Go"}'::jsonb, NOW(), NOW()),
('550e8400-e29b-41d4-a716-446655440002', 'sample-frontend', 'https://github.com/example/sample-frontend', 'main',
 '{"description": "Sample Svelte frontend application", "language": "JavaScript"}'::jsonb, NOW(), NOW()),
('550e8400-e29b-41d4-a716-446655440003', 'sample-microservice', 'https://github.com/example/sample-microservice', 'main',
 '{"description": "Sample microservice architecture", "language": "Go"}'::jsonb, NOW(), NOW())
ON CONFLICT (repo_id) DO NOTHING;

-- Insert sample files for Go API repository
INSERT INTO files (file_id, repo_id, path, language, size, checksum, created_at, updated_at) VALUES
('650e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440001', 'main.go', 'Go', 512, 'abc123', NOW(), NOW()),
('650e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440001', 'models/user.go', 'Go', 256, 'def456', NOW(), NOW()),
('650e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440001', 'handlers/user_handler.go', 'Go', 384, 'ghi789', NOW(), NOW())
ON CONFLICT (file_id) DO NOTHING;

-- Insert sample files for Frontend repository
INSERT INTO files (file_id, repo_id, path, language, size, checksum, created_at, updated_at) VALUES
('650e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440002', 'src/App.svelte', 'JavaScript', 320, 'jkl012', NOW(), NOW()),
('650e8400-e29b-41d4-a716-446655440005', '550e8400-e29b-41d4-a716-446655440002', 'src/components/UserList.svelte', 'JavaScript', 256, 'mno345', NOW(), NOW())
ON CONFLICT (file_id) DO NOTHING;

-- Insert sample symbols
INSERT INTO symbols (symbol_id, file_id, name, kind, signature, start_line, end_line, start_byte, end_byte, docstring, created_at) VALUES
('750e8400-e29b-41d4-a716-446655440001', '650e8400-e29b-41d4-a716-446655440001', 'main', 'function', 'func main()', 7, 12, 100, 200, 'Main entry point', NOW()),
('750e8400-e29b-41d4-a716-446655440002', '650e8400-e29b-41d4-a716-446655440001', 'healthCheck', 'function', 'func healthCheck(c *gin.Context)', 14, 16, 250, 350, 'Health check endpoint', NOW()),
('750e8400-e29b-41d4-a716-446655440003', '650e8400-e29b-41d4-a716-446655440001', 'getUsers', 'function', 'func getUsers(c *gin.Context)', 18, 20, 400, 500, 'Get users endpoint', NOW()),
('750e8400-e29b-41d4-a716-446655440004', '650e8400-e29b-41d4-a716-446655440002', 'User', 'struct', 'type User struct', 3, 8, 50, 150, 'User model', NOW()),
('750e8400-e29b-41d4-a716-446655440005', '650e8400-e29b-41d4-a716-446655440002', 'Validate', 'method', 'func (u *User) Validate() error', 10, 15, 200, 300, 'Validate user data', NOW()),
('750e8400-e29b-41d4-a716-446655440006', '650e8400-e29b-41d4-a716-446655440003', 'UserHandler', 'struct', 'type UserHandler struct', 6, 8, 100, 150, 'User handler', NOW()),
('750e8400-e29b-41d4-a716-446655440007', '650e8400-e29b-41d4-a716-446655440003', 'GetUser', 'method', 'func (h *UserHandler) GetUser(c *gin.Context)', 10, 18, 200, 400, 'Get user by ID', NOW())
ON CONFLICT (symbol_id) DO NOTHING;

-- Verify data insertion
DO $$
DECLARE
    repo_count INTEGER;
    file_count INTEGER;
    symbol_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO repo_count FROM repositories;
    SELECT COUNT(*) INTO file_count FROM files;
    SELECT COUNT(*) INTO symbol_count FROM symbols;
    
    RAISE NOTICE '✅ Seed data loaded successfully:';
    RAISE NOTICE '   - Repositories: %', repo_count;
    RAISE NOTICE '   - Files: %', file_count;
    RAISE NOTICE '   - Symbols: %', symbol_count;
END $$;
