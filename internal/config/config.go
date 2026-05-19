package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DBHost         string
	DBPort         string
	DBUser         string
	DBPass         string
	DBName         string
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	RedisChannel   string
	Env            string
	LogLevel       string
	Port           string
	JWTSecret      string
	RunMigrations  bool
	MigrationsPath string
}

func Load() (Config, error) {
	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid REDIS_DB: %w", err)
	}

	cfg := Config{
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "booking"),
		DBPass:         getEnv("DB_PASS", "booking"),
		DBName:         getEnv("DB_NAME", "booking"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		RedisDB:        redisDB,
		RedisChannel:   getEnv("REDIS_CHANNEL", "realtime:events"),
		Env:            getEnv("APP_ENV", "dev"),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		Port:           getEnv("PORT", "8080"),
		JWTSecret:      getEnv("JWT_SECRET", "booking-dev-secret"),
		RunMigrations:  strings.ToLower(getEnv("RUN_MIGRATIONS", "true")) != "false",
		MigrationsPath: getEnv("MIGRATIONS_PATH", "file://migrations"),
	}

	if cfg.Port == "" {
		return Config{}, fmt.Errorf("PORT is required")
	}
	if cfg.RedisAddr == "" {
		return Config{}, fmt.Errorf("REDIS_ADDR is required")
	}
	if cfg.RedisChannel == "" {
		return Config{}, fmt.Errorf("REDIS_CHANNEL is required")
	}
	if cfg.Env == "" {
		return Config{}, fmt.Errorf("APP_ENV is required")
	}
	if cfg.LogLevel == "" {
		return Config{}, fmt.Errorf("LOG_LEVEL is required")
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
