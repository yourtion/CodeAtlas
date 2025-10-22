package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/api/handlers"
	"github.com/yourtionguo/CodeAtlas/internal/api/middleware"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// ServerConfig holds server configuration
type ServerConfig struct {
	EnableAuth     bool
	AuthTokens     []string
	CORSOrigins    []string
}

// Server represents the API server
type Server struct {
	db                   *models.DB
	config               *ServerConfig
	repoRepository       *models.RepositoryRepository
	fileRepository       *models.FileRepository
	indexHandler         *handlers.IndexHandler
	repoHandler          *handlers.RepositoryHandler
	searchHandler        *handlers.SearchHandler
	relationshipHandler  *handlers.RelationshipHandler
}

// NewServer creates a new API server
func NewServer(db *models.DB, config *ServerConfig) *Server {
	if config == nil {
		config = &ServerConfig{
			EnableAuth:  false,
			AuthTokens:  []string{},
			CORSOrigins: []string{"*"},
		}
	}

	return &Server{
		db:                  db,
		config:              config,
		repoRepository:      models.NewRepositoryRepository(db),
		fileRepository:      models.NewFileRepository(db),
		indexHandler:        handlers.NewIndexHandler(db),
		repoHandler:         handlers.NewRepositoryHandler(db),
		searchHandler:       handlers.NewSearchHandler(db),
		relationshipHandler: handlers.NewRelationshipHandler(db),
	}
}

// SetupRouter creates and configures the Gin router with all middleware and routes
func (s *Server) SetupRouter() *gin.Engine {
	// Create router
	r := gin.New()

	// Add recovery middleware
	r.Use(gin.Recovery())

	// Add logging middleware
	r.Use(middleware.Logging())

	// Add CORS middleware
	corsConfig := middleware.NewCORSConfig(s.config.CORSOrigins)
	r.Use(middleware.CORS(corsConfig))

	// Add authentication middleware
	authConfig := middleware.NewAuthConfig(s.config.EnableAuth, s.config.AuthTokens)
	r.Use(middleware.Auth(authConfig))

	// Register routes
	s.RegisterRoutes(r)

	return r
}

// RegisterRoutes registers all API routes
func (s *Server) RegisterRoutes(r *gin.Engine) {
	// Health check endpoint (no auth required)
	r.GET("/health", s.healthCheck)

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Index endpoint
		v1.POST("/index", s.indexHandler.Index)

		// Repository endpoints
		v1.GET("/repositories", s.repoHandler.GetAll)
		v1.GET("/repositories/:id", s.repoHandler.GetByID)
		v1.POST("/repositories", s.createRepository)

		// Search endpoint
		v1.POST("/search", s.searchHandler.Search)

		// Relationship endpoints
		v1.GET("/symbols/:id/callers", s.relationshipHandler.GetCallers)
		v1.GET("/symbols/:id/callees", s.relationshipHandler.GetCallees)
		v1.GET("/symbols/:id/dependencies", s.relationshipHandler.GetDependencies)
		v1.GET("/files/:id/symbols", s.relationshipHandler.GetFileSymbols)

		// File endpoints
		v1.POST("/files", s.createFile)

		// Commit endpoints
		v1.POST("/commits", s.createCommit)
	}
}

// healthCheck handles health check requests
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "CodeAtlas API server is running",
	})
}

// createRepository handles repository creation
func (s *Server) createRepository(c *gin.Context) {
	var req struct {
		Name   string `json:"name" binding:"required"`
		URL    string `json:"url"`
		Branch string `json:"branch"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Branch == "" {
		req.Branch = "main"
	}

	repo := &models.Repository{
		RepoID:   uuid.New().String(),
		Name:     req.Name,
		URL:      req.URL,
		Branch:   req.Branch,
		Metadata: map[string]interface{}{},
	}

	ctx := context.Background()
	if err := s.repoRepository.Create(ctx, repo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, repo)
}

// getRepository retrieves a repository by ID
func (s *Server) getRepository(c *gin.Context) {
	repoID := c.Param("id")
	if repoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository ID"})
		return
	}

	ctx := context.Background()
	repo, err := s.repoRepository.GetByID(ctx, repoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if repo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		return
	}

	c.JSON(http.StatusOK, repo)
}

// createFile handles file creation
func (s *Server) createFile(c *gin.Context) {
	var req struct {
		RepositoryID string `json:"repository_id" binding:"required"`
		Path         string `json:"path" binding:"required"`
		Content      string `json:"content"`
		Language     string `json:"language"`
		Size         int64  `json:"size"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	file := &models.File{
		FileID:   uuid.New().String(),
		RepoID:   req.RepositoryID,
		Path:     req.Path,
		Language: req.Language,
		Size:     req.Size,
		Checksum: "", // TODO: Calculate checksum from content
	}

	ctx := context.Background()
	if err := s.fileRepository.Create(ctx, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, file)
}

// createCommit handles commit creation
func (s *Server) createCommit(c *gin.Context) {
	var req struct {
		RepositoryID int    `json:"repository_id" binding:"required"`
		Hash         string `json:"hash" binding:"required"`
		Author       string `json:"author" binding:"required"`
		Email        string `json:"email" binding:"required"`
		Message      string `json:"message" binding:"required"`
		Timestamp    string `json:"timestamp" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse timestamp
	// TODO: Implement proper timestamp parsing

	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}
