package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	SupabaseURL        string
	SupabaseKey        string
	GroqAPIKey         string
	GoogleClientID     string
	GoogleClientSecret string
	SecretKey          string
	Port               string
}

var C Config

func Load() {
	_ = godotenv.Load()

	C = Config{
		SupabaseURL:        mustGet("SUPABASE_URL"),
		SupabaseKey:        mustGet("SUPABASE_KEY"),
		GroqAPIKey:         os.Getenv("GROQ_API_KEY"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		SecretKey:          mustGet("SECRET_KEY"),
		Port:               getOrDefault("PORT", "8000"),
	}
}

func mustGet(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required env variable not set: " + key)
	}
	return v
}

func getOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
