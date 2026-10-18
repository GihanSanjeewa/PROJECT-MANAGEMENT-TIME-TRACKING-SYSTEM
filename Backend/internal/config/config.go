package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port             string
	Environment      string
	DatabaseDSN      string
	RedisAddr        string
	JWTSecret        string
	WorkHoursPerDay  float64
	DBMaxConns       int
	DBMinConns       int
	DBMaxConnLifetime time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		Environment:       getEnv("ENV", "development"),
		DatabaseDSN:       getEnv("DB_DSN", "postgres://postgres:postgres@localhost:5432/pulseflow_pm?sslmode=disable"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		JWTSecret:         getEnv("JWT_SECRET", "super-secret-pulseflow-jwt-key-minimum-32-chars!"),
		WorkHoursPerDay:   getEnvFloat("WORK_HOURS_PER_DAY", 8.0),
		DBMaxConns:        getEnvInt("DB_MAX_CONNS", 25),
		DBMinConns:        getEnvInt("DB_MIN_CONNS", 5),
		DBMaxConnLifetime: time.Hour,
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Port) == "" {
		return fmt.Errorf("PORT cannot be empty")
	}
	if strings.TrimSpace(c.DatabaseDSN) == "" {
		return fmt.Errorf("DB_DSN cannot be empty")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long for security (got %d chars)", len(c.JWTSecret))
	}
	if c.WorkHoursPerDay <= 0 || c.WorkHoursPerDay > 24 {
		return fmt.Errorf("WORK_HOURS_PER_DAY must be between 1.0 and 24.0 (got %.2f)", c.WorkHoursPerDay)
	}
	if c.DBMaxConns < c.DBMinConns {
		return fmt.Errorf("DB_MAX_CONNS (%d) cannot be less than DB_MIN_CONNS (%d)", c.DBMaxConns, c.DBMinConns)
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val, ok := os.LookupEnv(key); ok {
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			return floatVal
		}
	}
	return defaultVal
}
