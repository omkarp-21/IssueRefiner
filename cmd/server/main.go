package main

import (
	"context"
	"errors"
	"fmt"
	"issue-refiner/internal/auth"
	"issue-refiner/internal/config"
	"issue-refiner/internal/database"
	"issue-refiner/internal/github"
	"issue-refiner/internal/health"
	"issue-refiner/internal/jobs"
	"issue-refiner/internal/orgs"
	"issue-refiner/internal/providers"
	"issue-refiner/internal/repos"
	"issue-refiner/internal/sessions"
	"issue-refiner/internal/tokens"
	"issue-refiner/internal/users"
	"issue-refiner/internal/worker"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	log.Println("✓ Database Connected")

	router := gin.Default()

	// Dependencies
	userRepository := users.NewUserRepository(dbPool)
	userService := users.NewUserService(&userRepository)

	githubOAuth := github.NewOAuth(cfg.GitHub)
	githubClient := github.NewClient("https://api.github.com")

	tokenRepository := tokens.NewTokenRepository(dbPool)
	tokenService := tokens.NewTokenService(&tokenRepository)
	authService := auth.NewAuthService(&githubOAuth, &userService, &tokenService)

	sessionRepository := sessions.NewSessionRepository(dbPool)
	sessionService := sessions.NewSessionService(&sessionRepository)

	organizationRepository := orgs.NewOrganizationRepository(dbPool)
	organizationService := orgs.NewOrganizationService(&organizationRepository)
	authHandler := auth.NewAuthHandler(&authService, &sessionService, &organizationService)

	jobRepository := jobs.NewJobRepository(dbPool)

	repositoryRepository := repos.NewRepositoryRepository(dbPool)
	repositoryService := repos.NewRepositoryService(&repositoryRepository, &githubClient, &tokenService)
	repositoryHandler := repos.NewRepositoryHandler(&repositoryService)

	providerConfigRepository := providers.NewConfigRepository(dbPool)
	credentialStore := providers.NewCredentialStore(dbPool, cfg.Security.CredentialKey)
	providerRegistry := providers.DefaultRegistry()
	providerService := providers.NewProviderService(&providerConfigRepository, &credentialStore, &providerRegistry)
	providerHandler := providers.NewProviderHandler(&providerService)

	jobService := jobs.NewJobService(&jobRepository, &repositoryService, &providerService)

	workerPool := worker.NewPool(&jobService, cfg.Worker.WorkerCount, cfg.Worker.QueueSize, time.Duration(cfg.Worker.JobTimeoutSeconds)*time.Second)
	workerPool.Start()
	log.Printf("Worker pool started with %d workers", cfg.Worker.WorkerCount)

	jobHandler := jobs.NewJobHandler(&jobService, &workerPool)

	// Register Routes
	requireAuth := auth.RequireAuth(&sessionService, &userService, &organizationService)
	health.RegisterRoutes(router)
	authHandler.RegisterRoutes(router)
	jobHandler.RegisterRoutes(router, requireAuth)
	repositoryHandler.RegisterRoutes(router, requireAuth)
	providerHandler.RegisterRoutes(router, requireAuth, auth.RequireOwner())

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.ServerHost, cfg.Server.ServerPort),
		Handler: router,
	}

	log.Printf("Starting Server on %s:%d", cfg.Server.ServerHost, cfg.Server.ServerPort)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to Start Server: %v", err)
		}
	}()

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-stopCtx.Done()

	log.Println("Shutting down server...")
	deadline := time.Now().Add(10 * time.Second)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown failed: %v", err)
	}
	log.Println("Stopping worker pool...")
	remaining := time.Until(deadline)
	if remaining < 0 {
		remaining = 0
	}
	if ok := workerPool.StopWithTimeout(remaining); !ok {
		log.Println("Worker pool did not stop in time, continuing shutdown...")
	}
	dbPool.Close()
	log.Println("Server stopped")
}
