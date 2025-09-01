package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// Server represents the API server
type Server struct {
	db *models.DB
}

// NewServer creates a new API server
func NewServer(db *models.DB) *Server {
	return &Server{db: db}
}

// RegisterRoutes registers all API routes
func (s *Server) RegisterRoutes(r *gin.Engine) {
	// Health check endpoint
	r.GET("/health", s.healthCheck)

	// Repository endpoints
	r.POST("/api/v1/repositories", s.createRepository)
	r.GET("/api/v1/repositories/:id", s.getRepository)
	
	// File endpoints
	r.POST("/api/v1/files", s.createFile)
	
	// Commit endpoints
	r.POST("/api/v1/commits", s.createCommit)
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
		Name string `json:"name" binding:"required"`
		URL  string `json:"url"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	repo, err := s.db.CreateRepository(req.Name, req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, repo)
}

// getRepository retrieves a repository by ID
func (s *Server) getRepository(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository ID"})
		return
	}
	
	repo, err := s.db.GetRepositoryByID(id)
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
		RepositoryID int    `json:"repository_id" binding:"required"`
		Path         string `json:"path" binding:"required"`
		Content      string `json:"content"`
		Language     string `json:"language"`
		Size         int    `json:"size"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	file, err := s.db.CreateFile(req.RepositoryID, req.Path, req.Content, req.Language, req.Size)
	if err != nil {
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