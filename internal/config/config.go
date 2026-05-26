package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DBHost                    string
	DBPort                    string
	DBUser                    string
	DBPass                    string
	DBName                    string
	RedisAddr                 string
	RedisPassword             string
	RedisDB                   int
	RedisChannel              string
	Env                       string
	LogLevel                  string
	Port                      string
	JWTSecret                 string
	RunMigrations             bool
	MigrationsPath            string
	ReservationTTL            time.Duration
	ReservationExpireInterval time.Duration
	ReservationExpireBatch    int
	SlotGenerateInterval      time.Duration
	CORSAllowedOrigins        []string
}

func Load() (Config, error) {
	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid REDIS_DB: %w", err)
	}
	reservationTTLSeconds, err := strconv.Atoi(getEnv("RESERVATION_TTL_SECONDS", "300"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid RESERVATION_TTL_SECONDS: %w", err)
	}
	if reservationTTLSeconds <= 0 {
		return Config{}, fmt.Errorf("invalid RESERVATION_TTL_SECONDS: must be > 0")
	}
	reservationExpireIntervalSeconds, err := strconv.Atoi(getEnv("RESERVATION_EXPIRE_INTERVAL_SECONDS", "15"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid RESERVATION_EXPIRE_INTERVAL_SECONDS: %w", err)
	}
	if reservationExpireIntervalSeconds <= 0 {
		return Config{}, fmt.Errorf("invalid RESERVATION_EXPIRE_INTERVAL_SECONDS: must be > 0")
	}
	reservationExpireBatch, err := strconv.Atoi(getEnv("RESERVATION_EXPIRE_BATCH", "100"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid RESERVATION_EXPIRE_BATCH: %w", err)
	}
	if reservationExpireBatch <= 0 {
		return Config{}, fmt.Errorf("invalid RESERVATION_EXPIRE_BATCH: must be > 0")
	}
	slotGenerateIntervalSeconds, err := strconv.Atoi(getEnv("SLOT_GENERATE_INTERVAL_SECONDS", "60"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid SLOT_GENERATE_INTERVAL_SECONDS: %w", err)
	}
	if slotGenerateIntervalSeconds <= 0 {
		return Config{}, fmt.Errorf("invalid SLOT_GENERATE_INTERVAL_SECONDS: must be > 0")
	}
	corsAllowedOrigins := splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"))

	cfg := Config{
		DBHost:                    getEnv("DB_HOST", "localhost"),
		DBPort:                    getEnv("DB_PORT", "5432"),
		DBUser:                    getEnv("DB_USER", "booking"),
		DBPass:                    getEnv("DB_PASS", "booking"),
		DBName:                    getEnv("DB_NAME", "booking"),
		RedisAddr:                 getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:             getEnv("REDIS_PASSWORD", ""),
		RedisDB:                   redisDB,
		RedisChannel:              getEnv("REDIS_CHANNEL", "realtime:events"),
		Env:                       getEnv("APP_ENV", "dev"),
		LogLevel:                  getEnv("LOG_LEVEL", "info"),
		Port:                      getEnv("PORT", "8080"),
		JWTSecret:                 getEnv("JWT_SECRET", "booking-dev-secret"),
		RunMigrations:             strings.ToLower(getEnv("RUN_MIGRATIONS", "true")) != "false",
		MigrationsPath:            getEnv("MIGRATIONS_PATH", "file://migrations"),
		ReservationTTL:            time.Duration(reservationTTLSeconds) * time.Second,
		ReservationExpireInterval: time.Duration(reservationExpireIntervalSeconds) * time.Second,
		ReservationExpireBatch:    reservationExpireBatch,
		SlotGenerateInterval:      time.Duration(slotGenerateIntervalSeconds) * time.Second,
		CORSAllowedOrigins:        corsAllowedOrigins,
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

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
