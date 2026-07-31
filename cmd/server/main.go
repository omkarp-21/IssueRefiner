package main

import (
	"fmt"
	"issue-refiner/internal/config"
	"issue-refiner/internal/health"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		err = godotenv.Load("../../.env")
		if err != nil {
			log.Println("Warning: Could not load .env file")
		}
	}

	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("Failed to load Configuration: %v", err)
	}

	router := gin.Default()

	// Register Routes
	health.RegisterRoutes(router)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort),
		Handler: router,
	}

	log.Printf("Starting Server on %s:%d", cfg.ServerHost, cfg.ServerPort)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to Start Server: %v", err)
	}
}
