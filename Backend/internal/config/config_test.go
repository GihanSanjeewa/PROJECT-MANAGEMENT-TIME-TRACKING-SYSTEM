package config_test

import (
	"testing"
	"time"

	"project-management-system/internal/config"
)

func TestConfigValidation_Success(t *testing.T) {
	cfg := &config.Config{
		Port:              "8080",
		Environment:       "test",
		DatabaseDSN:       "postgres://postgres:password@localhost:5432/test_db",
		RedisAddr:         "localhost:6379",
		JWTSecret:         "this-is-a-valid-secure-key-with-at-least-32-chars",
		WorkHoursPerDay:   8.0,
		DBMaxConns:        20,
		DBMinConns:        5,
		DBMaxConnLifetime: time.Hour,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

func TestConfigValidation_WeakJWTSecret(t *testing.T) {
	cfg := &config.Config{
		Port:            "8080",
		DatabaseDSN:     "postgres://test",
		JWTSecret:       "short-secret", // < 32 characters
		WorkHoursPerDay: 8.0,
		DBMaxConns:      10,
		DBMinConns:      2,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation failure for weak JWT secret (< 32 chars), got nil")
	}
}

func TestConfigValidation_InvalidWorkHours(t *testing.T) {
	cfg := &config.Config{
		Port:            "8080",
		DatabaseDSN:     "postgres://test",
		JWTSecret:       "this-is-a-valid-secure-key-with-at-least-32-chars",
		WorkHoursPerDay: 25.0, // > 24 hours
		DBMaxConns:      10,
		DBMinConns:      2,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation failure for work hours > 24, got nil")
	}
}
