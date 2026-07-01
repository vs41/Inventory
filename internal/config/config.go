package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port           string
	DatabaseURL    string
	RedisAddr      string
	JWTSecret      string
	JWTExpiryHours int
}

func Load() Config {
	return Config{
		Port:           getEnv("APP_PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://freshtrack:freshtrack@localhost:5433/freshtrack?sslmode=disable"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		JWTSecret:      getEnv("JWT_SECRET", "dev_secret_change_me"),
		JWTExpiryHours: getEnvInt("JWT_EXPIRY_HOURS", 12),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
