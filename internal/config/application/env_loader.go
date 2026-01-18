package application

import (
	"os"

	"github.com/joho/godotenv"
	"meerkat-v0/internal/infrastructure/logger"
)

// LoadEnvFile loads environment variables from a .env file
// If envFile is empty, it attempts to load .env from the current directory
// Returns true if a file was loaded, false otherwise
func LoadEnvFile(logger *logger.Logger, envFile string) bool {
	// If no file specified, try default .env in current directory
	if envFile == "" {
		envFile = ".env"
	}

	// Check if file exists
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		logger.Debug("No .env file found", "path", envFile)
		return false
	}

	// Load the .env file
	err := godotenv.Load(envFile)
	if err != nil {
		logger.Warn("Failed to load .env file", "path", envFile, "err", err)
		return false
	}

	logger.Debug("Loaded .env file", "path", envFile)
	return true
}

