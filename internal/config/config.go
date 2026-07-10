// Package config provides environment-based configuration for PetroSync.
// All fields are explicit — no magic defaults buried in struct tags.
package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config holds all application configuration sourced from environment variables.
type Config struct {
	// Server
	AppEnv  string `envconfig:"APP_ENV" default:"development"`
	AppPort int    `envconfig:"APP_PORT" default:"8080"`

	// Database
	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`

	// Valkey
	ValkeyAddr string `envconfig:"VALKEY_ADDR" default:"localhost:6379"`

	// JWT
	JWTSecret string `envconfig:"JWT_SECRET" required:"true"`

	// Telegram Bot
	TelegramBotToken string `envconfig:"TELEGRAM_BOT_TOKEN"`

	// Garage (S3-compatible object storage)
	GarageEndpoint  string `envconfig:"GARAGE_ENDPOINT" required:"true"`
	GarageAccessKey string `envconfig:"GARAGE_ACCESS_KEY" required:"true"`
	GarageSecretKey string `envconfig:"GARAGE_SECRET_KEY" required:"true"`
	GarageBucket    string `envconfig:"GARAGE_BUCKET" default:"petrosync"`

	// Derived
	JWTExpiry       time.Duration // computed: 30 minutes
	RefreshTokenTTL time.Duration // computed: 30 days
	SessionTTL      time.Duration // computed: 8 hours
}

// Load reads configuration from environment variables.
// It attempts to load a .env file first (silently ignores if missing).
func Load() (*Config, error) {
	// Best-effort .env load — ignore error if file doesn't exist
	_ = godotenv.Load()

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	// Derived durations
	cfg.JWTExpiry = 30 * time.Minute
	cfg.RefreshTokenTTL = 30 * 24 * time.Hour
	cfg.SessionTTL = 8 * time.Hour

	return &cfg, nil
}
