package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	// Server Config
	ServerHost string
	ServerPort int
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func LoadFromEnv() (*Config, error) {
	cfg := &Config{}

	cfg.ServerHost = getEnvOrDefault("SERVER_HOST", "0.0.0.0")

	port, err := strconv.Atoi(getEnvOrDefault("SERVER_PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_PORT: %v", err)
	}
	cfg.ServerPort = port

	return cfg, nil
}
