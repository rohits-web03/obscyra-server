package config

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Region          string
	PublicBaseURL   string
}

type Config struct {
	DB_URL      string
	Port        string
	JWTSecret   string
	Environment string
	CorsConfig  cors.Options
	R2          R2Config
}

var Envs = initConfig()

func initConfig() Config {
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	log.Println("Running in development mode, loading", envFile)
	if err := godotenv.Load(envFile); err != nil {
		log.Println("No", envFile, "file found")
	}

	return Config{
		DB_URL:      getEnv("DB_URL", ""),
		Port:        getEnv("PORT", "8080"),
		JWTSecret:   getEnv("JWT_SECRET", "not-so-secret-now-is-it?"),
		Environment: getEnv("ENV", "development"),
		CorsConfig:  CorsConfig(),
		R2: R2Config{
			AccountID:       getEnv("R2_ACCOUNT_ID", ""),
			AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
			BucketName:      getEnv("R2_BUCKET_NAME", ""),
			Region:          getEnv("R2_REGION", "auto"),
			PublicBaseURL:   getEnv("R2_PUBLIC_BASE_URL", ""),
		},
	}
}

// Gets the env by key or fallbacks
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func CorsConfig() cors.Options {
	return cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173","https://obscyra.vercel.app"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}
}
