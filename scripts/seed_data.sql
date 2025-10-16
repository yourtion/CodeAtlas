-- Seed data for CodeAtlas development environment
-- This script creates sample repositories, files, and relationships for testing

-- Insert sample repositories
INSERT INTO repositories (id, name, url, description, language, created_at, updated_at) VALUES
('550e8400-e29b-41d4-a716-446655440001', 'sample-go-api', 'https://github.com/example/sample-go-api', 'Sample Go REST API project', 'Go', NOW(), NOW()),
('550e8400-e29b-41d4-a716-446655440002', 'sample-frontend', 'https://github.com/example/sample-frontend', 'Sample Svelte frontend application', 'JavaScript', NOW(), NOW()),
('550e8400-e29b-41d4-a716-446655440003', 'sample-microservice', 'https://github.com/example/sample-microservice', 'Sample microservice architecture', 'Go', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Insert sample files for Go API repository
INSERT INTO files (id, repository_id, path, language, content, created_at, updated_at) VALUES
('650e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440001', 'main.go', 'Go', 
'package main

import (
    "github.com/gin-gonic/gin"
    "log"
)

func main() {
    r := gin.Default()
    r.GET("/health", healthCheck)
    r.GET("/api/users", getUsers)
    log.Fatal(r.Run(":8080"))
}

func healthCheck(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
}

func getUsers(c *gin.Context) {
    c.JSON(200, gin.H{"users": []string{"alice", "bob"}})
}', NOW(), NOW()),

('650e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440001', 'models/user.go', 'Go',
'package models

type User struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"-"`
}

func (u *User) Validate() error {
    if u.Name == "" {
        return errors.New("name is required")
    }
    return nil
}', NOW(), NOW()),

('650e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440001', 'handlers/user_handler.go', 'Go',
'package handlers

import (
    "github.com/gin-gonic/gin"
    "sample-go-api/models"
)

type UserHandler struct {
    db *Database
}

func (h *UserHandler) GetUser(c *gin.Context) {
    id := c.Param("id")
    user, err := h.db.FindUserByID(id)
    if err != nil {
        c.JSON(404, gin.H{"error": "user not found"})
        return
    }
    c.JSON(200, user)
}', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Insert sample files for Frontend repository
INSERT INTO files (id, repository_id, path, language, content, created_at, updated_at) VALUES
('650e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440002', 'src/App.svelte', 'JavaScript',
'<script>
  import { onMount } from "svelte";
  import UserList from "./components/UserList.svelte";
  
  let users = [];
  
  onMount(async () => {
    const response = await fetch("http://localhost:8080/api/users");
    users = await response.json();
  });
</script>

<main>
  <h1>User Management</h1>
  <UserList {users} />
</main>', NOW(), NOW()),

('650e8400-e29b-41d4-a716-446655440005', '550e8400-e29b-41d4-a716-446655440002', 'src/components/UserList.svelte', 'JavaScript',
'<script>
  export let users = [];
</script>

<div class="user-list">
  {#each users as user}
    <div class="user-card">
      <h3>{user.name}</h3>
      <p>{user.email}</p>
    </div>
  {/each}
</div>', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Insert sample symbols
INSERT INTO symbols (id, file_id, name, kind, signature, line_start, line_end, created_at) VALUES
('750e8400-e29b-41d4-a716-446655440001', '650e8400-e29b-41d4-a716-446655440001', 'main', 'function', 'func main()', 7, 12, NOW()),
('750e8400-e29b-41d4-a716-446655440002', '650e8400-e29b-41d4-a716-446655440001', 'healthCheck', 'function', 'func healthCheck(c *gin.Context)', 14, 16, NOW()),
('750e8400-e29b-41d4-a716-446655440003', '650e8400-e29b-41d4-a716-446655440001', 'getUsers', 'function', 'func getUsers(c *gin.Context)', 18, 20, NOW()),
('750e8400-e29b-41d4-a716-446655440004', '650e8400-e29b-41d4-a716-446655440002', 'User', 'struct', 'type User struct', 3, 8, NOW()),
('750e8400-e29b-41d4-a716-446655440005', '650e8400-e29b-41d4-a716-446655440002', 'Validate', 'method', 'func (u *User) Validate() error', 10, 15, NOW()),
('750e8400-e29b-41d4-a716-446655440006', '650e8400-e29b-41d4-a716-446655440003', 'UserHandler', 'struct', 'type UserHandler struct', 6, 8, NOW()),
('750e8400-e29b-41d4-a716-446655440007', '650e8400-e29b-41d4-a716-446655440003', 'GetUser', 'method', 'func (h *UserHandler) GetUser(c *gin.Context)', 10, 18, NOW())
ON CONFLICT (id) DO NOTHING;

-- Insert sample dependencies (file-level)
INSERT INTO dependencies (id, source_file_id, target_file_id, dependency_type, created_at) VALUES
('850e8400-e29b-41d4-a716-446655440001', '650e8400-e29b-41d4-a716-446655440003', '650e8400-e29b-41d4-a716-446655440002', 'import', NOW()),
('850e8400-e29b-41d4-a716-446655440002', '650e8400-e29b-41d4-a716-446655440001', '650e8400-e29b-41d4-a716-446655440003', 'import', NOW())
ON CONFLICT (id) DO NOTHING;

-- Insert sample vector embeddings (mock data)
-- Note: In production, these would be generated by actual embedding models
INSERT INTO embeddings (id, file_id, symbol_id, embedding, created_at) VALUES
('950e8400-e29b-41d4-a716-446655440001', '650e8400-e29b-41d4-a716-446655440001', '750e8400-e29b-41d4-a716-446655440001', 
 array_fill(0.1::float, ARRAY[1536]), NOW()),
('950e8400-e29b-41d4-a716-446655440002', '650e8400-e29b-41d4-a716-446655440001', '750e8400-e29b-41d4-a716-446655440002',
 array_fill(0.2::float, ARRAY[1536]), NOW()),
('950e8400-e29b-41d4-a716-446655440003', '650e8400-e29b-41d4-a716-446655440002', '750e8400-e29b-41d4-a716-446655440004',
 array_fill(0.3::float, ARRAY[1536]), NOW())
ON CONFLICT (id) DO NOTHING;

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
