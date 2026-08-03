package config

import (
	"fmt"
	"os"
	"strconv"
)

type ServerConfig struct {
	ServerHost string
	ServerPort int
}

type DatabaseConfig struct {
	DatabaseHost     string
	DatabasePort     int
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string
	DatabaseSSLMode  string
}

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func LoadFromEnv() (*Config, error) {
	cfg := &Config{}

	// Server
	cfg.Server.ServerHost = getEnvOrDefault("SERVER_HOST", "0.0.0.0")
	serverPort, err := strconv.Atoi(getEnvOrDefault("SERVER_PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_PORT: %w", err)
	}
	cfg.Server.ServerPort = serverPort

	// Database
	cfg.Database.DatabaseHost = getEnvOrDefault("DB_HOST", "localhost")
	dbPort, err := strconv.Atoi(getEnvOrDefault("DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}
	cfg.Database.DatabasePort = dbPort
	cfg.Database.DatabaseUser = getEnvOrDefault("DB_USER", "postgres")
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		return nil, fmt.Errorf("Please Provide DB Password.")
	}
	cfg.Database.DatabasePassword = dbPassword
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		return nil, fmt.Errorf("Please Provide DB Name.")
	}
	cfg.Database.DatabaseName = dbName
	cfg.Database.DatabaseSSLMode = getEnvOrDefault("DB_SSLMODE", "disable")

	return cfg, nil
}
