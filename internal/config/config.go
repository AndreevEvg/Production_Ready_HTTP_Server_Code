package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config хранит все настройки приложения
type Config struct {
	Environment string 
	ServiceName string 

	// HTTP Server
	ServerPort string
	ServerReadTimeout time.Duration 
	ServerWriteTimeout time.Duration 
	ServerIdleTimeout time.Duration

	// Database
	DatabaseURL string 
	DatabaseMaxConns int 
	DatabaseIdleConns int 
	DatabaseConnLifetime time.Duration

	// Redis
	RedisURL string 
	RedisPassword string 
	RedisDB int 

	// Rate Limiting
	RateLimitRequests int 
	RateLimitDuration time.Duration

	// Security
	JWTSecret string 
	CORSAllowedOrigins []string 
}

// Load загружает конфигурацию из переменных окружения
func Load() (*Config, error) {
	// Загружаем .env файл если он существует
	_ = godotenv.Load()

	cfg := &Config{
		Environment: getEnv("ENVIRONMENT", "production"),
		ServiceName: getEnv("SERVICE_NAME", "myapp"),

		// Сервер
		ServerPort: getEnv("SERVER_PORT", "8080"),
		ServerReadTimeout: getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
		ServerWriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
		ServerIdleTimeout: getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),

		// База данных
		DatabaseURL: getEnv("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable"),
		DatabaseMaxConns: getIntEnv("DATABASE_MAX_CONNS", 25),
		DatabaseIdleConns: getIntEnv("DATABASE_IDLE_CONNS", 5),
		DatabaseConnLifetime: getDurationEnv("DATABASE_CONN_LIFETIME", 5*time.Minute),

		// Redis
		RedisURL: getEnv("REDIS_URL", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB: getIntEnv("REDIS_DB", 0),

		// Rate Limiting
		RateLimitRequests: getIntEnv("RATE_LIMIT_REQUESTS", 100),
		RateLimitDuration: getDurationEnv("RATE_LIMIT_DURATION", 1*time.Minute),

		// Security
		JWTSecret: getEnv("JWT_SECRET", "your-secret-key"),
		CORSAllowedOrigins: getSliceEnv("CORS_ALLOWED_ORIGINS", []string{"*"}),
	}

	// Валидация критичных настроек
	if cfg.Environment == "production" && cfg.JWTSecret == "your-secret-key" {
		return nil, fmt.Errorf("JWT_SECRET must be changed in production")
	}

	return cfg, nil
}

// Вспомогательные функции для чтения переменных окружения
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal 
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Простое разделение по запятой, для прода нужно более сложное
		return []string{value}
	}
	return defaultValue
}