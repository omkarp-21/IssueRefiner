package main

import (
	"context"
	"fmt"
	"issue-refiner/internal/config"
	"issue-refiner/internal/database"
	"issue-refiner/internal/health"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load Config file
	if err := godotenv.Load(); err != nil {
		err = godotenv.Load("../../.env")
		if err != nil {
			log.Fatalf("Warning: Could not load .env file: %v", err)
		}
	}

	// Load Config variables
	log.Println("Loading Configuration...")
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("Failed to load Configuration: %v", err)
	}
	log.Println("✓ Configuration loaded")

	// Connect DB
	log.Println("Connecting to Database...")
	startupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbPool, err := database.New(startupCtx, cfg.Database)
	if err != nil {
		log.Fatalf("Unable to Establish DB Connection: %v", err)
	}
	defer dbPool.Close()
	log.Println("✓ Database Connected")

	router := gin.Default()

	// Register Routes
	health.RegisterRoutes(router)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.ServerHost, cfg.Server.ServerPort),
		Handler: router,
	}

	log.Printf("Starting Server on %s:%d", cfg.Server.ServerHost, cfg.Server.ServerPort)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to Start Server: %v", err)
	}
}
