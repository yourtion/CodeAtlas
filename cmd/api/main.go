package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/yourtionguo/CodeAtlas/internal/api"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

func main() {
	// Initialize database connection
	db, err := models.NewDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Create API server
	server := api.NewServer(db)

	// Create Gin router
	r := gin.Default()

	// Register routes
	server.RegisterRoutes(r)

	// Start server
	log.Println("Starting CodeAtlas API server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}