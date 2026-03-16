package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	SupabaseURL    string
	SupabaseKey    string
	GroqAPIKey     string
	GoogleClientID string
	GoogleClientSecret string
	SecretKey      string
	AdminUsernames []string
	AdminPasswords []string
	Port           string
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
		AdminUsernames:     splitEnv("ADMIN_USERNAMES"),
		AdminPasswords:     splitEnv("ADMIN_PASSWORDS"),
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

func splitEnv(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	return strings.Split(v, ",")
}

// AdminsMap returns map[username]password for virtual admins
func (c *Config) AdminsMap() map[string]string {
	m := make(map[string]string)
	for i, u := range c.AdminUsernames {
		if i < len(c.AdminPasswords) {
			m[u] = c.AdminPasswords[i]
		}
	}
	return m
}
