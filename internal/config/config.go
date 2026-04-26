package config

import (
	"log"
	"os"
)

type Config struct {
	Port          string
	DBPath        string
	WhisperURL    string
	WhisperSecret string
	AdminKey      string // protects /admin endpoints
	EncryptSecret string // AES-256 key for encrypting tenant credentials
}

func Load() *Config {
	cfg := &Config{
		Port:          getEnv("PORT", "8080"),
		DBPath:        getEnv("DB_PATH", "./data/feedbackpulse.db"),
		WhisperURL:    getEnv("WHISPER_URL", "https://whisper-microservice.onrender.com"),
		WhisperSecret: getEnv("WHISPER_SECRET", ""),
		AdminKey:      mustEnv("ADMIN_KEY"),
		EncryptSecret: mustEnv("ENCRYPT_SECRET"),
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %q is not set", key)
	}
	return v
}
