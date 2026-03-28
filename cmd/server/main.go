package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/ssback/internal/config"
	"github.com/ssback/internal/db"
	"github.com/ssback/internal/handlers"
	"github.com/ssback/internal/middleware"
)

func main() {
	config.Load()
	db.Init()

	authLimiter := middleware.NewRateLimiter(10, time.Minute)
	aiLimiter   := middleware.NewRateLimiter(20, time.Minute)

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	allowedOrigins := config.C.AllowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:*", "http://127.0.0.1:*"}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
	}))

	// ── Public ────────────────────────────────────────────────────────────────
	r.Get("/api/config", handlers.GetAppConfig)
	r.Get("/api/poems", handlers.GetPoems)
	r.Post("/api/logout", handlers.Logout)

	// OTP (email-код через Supabase Auth)
	r.With(authLimiter.Handler).Post("/api/auth/send_otp", handlers.SendOTP)
	r.With(authLimiter.Handler).Post("/api/auth/verify_otp", handlers.VerifyOTP)
	r.With(authLimiter.Handler).Post("/api/auth/register_otp", handlers.RegisterOTP)
	r.Post("/api/auth/resolve_email", handlers.ResolveEmail)

	// Google OAuth
	r.With(authLimiter.Handler).Post("/api/google/mobile-auth", handlers.GoogleMobileAuth)

	// ── Authenticated ─────────────────────────────────────────────────────────
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth)

		r.Get("/api/me", handlers.GetMe)
		r.Post("/api/profile", handlers.UpdateProfile)
		r.Post("/api/change_username", handlers.ChangeUsername)
		r.Post("/api/change_email", handlers.ChangeEmail)
		r.Post("/api/toggle_read", handlers.ToggleRead)
		r.Post("/api/toggle_pin", handlers.TogglePin)

		r.With(aiLimiter.Handler).Post("/api/ai/chat", handlers.AIChat)
		r.Post("/api/ai/verify_key", handlers.AIVerifyKey)

		// ── Admin ──────────────────────────────────────────────────────────
		r.Group(func(r chi.Router) {
			r.Use(middleware.AdminOnly)

			r.Post("/api/poems", handlers.AddPoem)
			r.Put("/api/poems/{id}", handlers.EditPoem)
			r.Delete("/api/poems/{id}", handlers.DeletePoem)

			r.Post("/api/ai/generate_key", handlers.AIGenerateKey)
			r.Get("/api/ai/keys", handlers.AIGetKeys)
			r.Post("/api/ai/disable_key", handlers.AIDisableKey)
		})
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"message":"Сборник Стихов API (Go)","status":"ok"}`)
	})

	addr := ":" + config.C.Port
	fmt.Printf("🚀 Server running on %s\n", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		panic(err)
	}
}
