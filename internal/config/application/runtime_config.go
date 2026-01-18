package application

import (
	"os"
	"strings"
)

// RuntimeConfig holds all runtime configuration from CLI flags, environment variables, and .env file
type RuntimeConfig struct {
	// API Configuration
	APIKey  string
	APIPort string

	// Development Mode
	DevMode bool

	// Logging Configuration
	LogLevel  string
	LogFormat string
	LogOutput string

	// Database Configuration
	DBPath string

	// Config file path
	ConfigPath string
}

// LoadRuntimeConfig loads configuration with precedence: CLI flags > env vars > .env file > defaults
func LoadRuntimeConfig(apiKey, port, logLevel, logFormat, logOutput, dbPath, configPath string, devMode bool) *RuntimeConfig {
	cfg := &RuntimeConfig{
		APIKey:     getValue(apiKey, "MEERKAT_API_KEY", ""),
		APIPort:    getValue(port, "MEERKAT_API_PORT", "8080"),
		DevMode:    devMode || getBoolEnv("MEERKAT_DEV_MODE", false),
		LogLevel:   getValue(logLevel, "MEERKAT_LOG_LEVEL", "INFO"),
		LogFormat:  getValue(logFormat, "MEERKAT_LOG_FORMAT", "text"),
		LogOutput:  getValue(logOutput, "MEERKAT_LOG_OUTPUT", "stdout"),
		DBPath:     getValue(dbPath, "MEERKAT_DB_PATH", "observations.db"),
		ConfigPath: configPath,
	}

	return cfg
}

// getValue returns the first non-empty value from CLI flag, env var, or default
func getValue(cliValue, envKey, defaultValue string) string {
	if cliValue != "" {
		return cliValue
	}
	if envValue := os.Getenv(envKey); envValue != "" {
		return envValue
	}
	return defaultValue
}

// getBoolEnv gets a boolean environment variable
func getBoolEnv(key string, defaultValue bool) bool {
	value := strings.ToLower(os.Getenv(key))
	if value == "true" || value == "1" || value == "yes" {
		return true
	}
	if value == "false" || value == "0" || value == "no" {
		return false
	}
	return defaultValue
}

// Validate checks that required configuration is present
func (c *RuntimeConfig) Validate() error {
	if c.APIKey == "" {
		return &ConfigError{Field: "api-key", Message: "API key is required (set MEERKAT_API_KEY or use --api-key flag)"}
	}
	if c.ConfigPath == "" {
		return &ConfigError{Field: "config", Message: "Config file path is required"}
	}
	return nil
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}

