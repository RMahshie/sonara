package config

import (
	"github.com/spf13/viper"
	"github.com/rs/zerolog/log"
	"strings"
)

// Config holds all configuration for the application
type Config struct {
	Database   DatabaseConfig
	Server     ServerConfig
	AWS        AWSConfig
	OpenAI     OpenAIConfig
	Processing ProcessingConfig
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port           string
	Env            string
	AllowedOrigins []string
}

// AWSConfig holds AWS/S3 configuration
type AWSConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	S3Bucket        string
	S3Endpoint      string
}

// OpenAIConfig holds OpenAI configuration
type OpenAIConfig struct {
	APIKey string
}

// ProcessingConfig holds processing service configuration
type ProcessingConfig struct {
	PythonCmd string
}

// Load loads configuration from environment variables and .env files
func Load() (*Config, error) {
	// Set defaults
	viper.SetDefault("DATABASE_URL", "postgres://sonara:localdev@localhost:5432/sonara_dev?sslmode=disable")
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("ENVIRONMENT", "dev")
	viper.SetDefault("AWS_REGION", "us-east-1")
	viper.SetDefault("AWS_ACCESS_KEY_ID", "")
	viper.SetDefault("AWS_SECRET_ACCESS_KEY", "")
	viper.SetDefault("S3_BUCKET", "sonara-audio")
	viper.SetDefault("S3_ENDPOINT", "")
	viper.SetDefault("ALLOWED_ORIGINS", "https://sonara.up.railway.app,http://localhost:5173,http://localhost:3000")
	viper.SetDefault("OPENAI_API_KEY", "")
	viper.SetDefault("PYTHON_CMD", "python3")

	// Read from .env files based on environment
	env := viper.GetString("ENVIRONMENT")
	if env == "" {
		env = "dev" // Use "dev" to match .env.dev filename
	}

	// Try to read .env file for the current environment
	viper.SetConfigName(".env." + env)
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	// Read .env file (ignore error if file doesn't exist)
	_ = viper.ReadInConfig() // Ignore error - file may not exist

	// Environment variables override .env file values
	viper.AutomaticEnv()

	// Bind specific environment variable names
	viper.BindEnv("DATABASE_URL")
	viper.BindEnv("PORT")
	viper.BindEnv("ENVIRONMENT")
	viper.BindEnv("AWS_REGION")
	viper.BindEnv("AWS_ACCESS_KEY_ID")
	viper.BindEnv("AWS_SECRET_ACCESS_KEY")
	viper.BindEnv("S3_BUCKET")
	viper.BindEnv("S3_ENDPOINT")
	viper.BindEnv("ALLOWED_ORIGINS")
	viper.BindEnv("OPENAI_API_KEY")
	viper.BindEnv("PYTHON_CMD")

	var config Config
	config.Database.URL = viper.GetString("DATABASE_URL")
	config.Server.Port = viper.GetString("PORT")
	config.Server.Env = viper.GetString("ENVIRONMENT")
	config.Server.AllowedOrigins = strings.Split(viper.GetString("ALLOWED_ORIGINS"), ",")
	config.AWS.Region = viper.GetString("AWS_REGION")
	config.AWS.AccessKeyID = viper.GetString("AWS_ACCESS_KEY_ID")
	config.AWS.SecretAccessKey = viper.GetString("AWS_SECRET_ACCESS_KEY")
	config.AWS.S3Bucket = viper.GetString("S3_BUCKET")
	config.AWS.S3Endpoint = viper.GetString("S3_ENDPOINT")
	config.OpenAI.APIKey = viper.GetString("OPENAI_API_KEY")
	config.Processing.PythonCmd = viper.GetString("PYTHON_CMD")

	// Add this debug logging
	log.Info().
	Strs("allowed_origins", config.Server.AllowedOrigins).
	Int("origin_count", len(config.Server.AllowedOrigins)).
	Msg("CORS configuration debug")

	return &config, nil
}

// GetStringOrDefault returns the value from viper if set, otherwise returns the default
func GetStringOrDefault(envVar, def string) string {
	if viper.IsSet(envVar) {
		return viper.GetString(envVar)
	}
	return def
}
