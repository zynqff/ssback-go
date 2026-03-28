package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	SupabaseURL        string
	SupabaseKey        string // anon key — для чтения таблиц
	SupabaseServiceKey string // service_role key — для Auth Admin API (OTP)
	GroqAPIKey         string
	GoogleClientID     string
	GoogleClientSecret string
	SecretKey          string
	Port               string
	AllowedOrigins     []string
}

var C Config

func Load() {
	_ = godotenv.Load()

	C = Config{
		SupabaseURL:        mustGet("SUPABASE_URL"),
		SupabaseKey:        mustGet("SUPABASE_KEY"),
		SupabaseServiceKey: getOrDefault("SUPABASE_SERVICE_KEY", ""),
		GroqAPIKey:         os.Getenv("GROQ_API_KEY"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		SecretKey:          mustGet("SECRET_KEY"),
		Port:               getOrDefault("PORT", "8000"),
		AllowedOrigins:     parseList(os.Getenv("ALLOWED_ORIGINS")),
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

func parseList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
