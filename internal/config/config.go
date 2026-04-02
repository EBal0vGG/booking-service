package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DBHost         string
	DBPort         string
	DBUser         string
	DBPass         string
	DBName         string
	Port           string
	JWTSecret      string
	RunMigrations  bool
	MigrationsPath string
}

func Load() (Config, error) {
	cfg := Config{
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "booking"),
		DBPass:         getEnv("DB_PASS", "booking"),
		DBName:         getEnv("DB_NAME", "booking"),
		Port:           getEnv("PORT", "8080"),
		JWTSecret:      getEnv("JWT_SECRET", "booking-dev-secret"),
		RunMigrations:  strings.ToLower(getEnv("RUN_MIGRATIONS", "true")) != "false",
		MigrationsPath: getEnv("MIGRATIONS_PATH", "file://migrations"),
	}

	if cfg.Port == "" {
		return Config{}, fmt.Errorf("PORT is required")
	}

	return cfg, nil
}

func (c Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser,
		c.DBPass,
		c.DBHost,
		c.DBPort,
		c.DBName,
	)
}

func getEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	return value
}
