package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string `env:"DB_HOST" envDefault:"localhost"`
	DBPort     string `env:"DB_PORT" envDefault:"5432"`
	DBUser     string `env:"DB_USER" envDefault:"postgres"`
	DBPassword string `env:"DB_PASSWORD" envDefault:"password"`
	DBName     string `env:"DB_NAME" envDefault:"auth_db"`

	JWTSecret      string        `env:"JWT_SECRET" envDefault:"your-super-secret-key"`
	JWTIssuer      string        `env:"JWT_ISSUER" envDefault:"auth-app"`
	JWTExpiryHours time.Duration `env:"JWT_EXPIRY_HOURS" envDefault:"24h"`

	ServerPort string `env:"SERVER_PORT" envDefault:"8080"`
	ServerHost string `env:"SERVER_HOST" envDefault:"0.0.0.0"`
	Env        string `env:"ENVIRONMENT" envDefault:"development"`

	EnableSwagger bool `env:"ENABLE_SWAGGER" envDefault:"true"`
}

func Load() *Config {
	// Загружаем .env файл
	_ = godotenv.Load()

	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "auth_db"),

		JWTSecret:      getEnv("JWT_SECRET", "your-super-secret-key"),
		JWTIssuer:      getEnv("JWT_ISSUER", "auth-app"),
		JWTExpiryHours: getEnvAsDuration("JWT_EXPIRY_HOURS", 24*time.Hour),

		ServerPort: getEnv("SERVER_PORT", "8080"),
		ServerHost: getEnv("SERVER_HOST", "0.0.0.0"),
		Env:        getEnv("ENVIRONMENT", "development"),

		EnableSwagger: getEnvAsBool("ENABLE_SWAGGER", true),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}
