package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ssback/internal/config"
	"github.com/ssback/internal/db"
	"github.com/ssback/internal/models"
	"github.com/ssback/internal/services"
)

// POST /api/login
func Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := decode(r, &req); err != nil {
		writeError(w, 400, "bad request")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, 400, "заполните все поля")
		return
	}

	var user models.User
	if err := db.DB.SelectOne("user", map[string]string{"username": req.Username}, &user); err != nil || user.Username == "" {
		writeError(w, 401, "неверный логин или пароль")
		return
	}
	if !services.CheckPassword(req.Password, user.PasswordHash) {
		writeError(w, 401, "неверный логин или пароль")
		return
	}
	token, err := services.CreateAccessToken(user.Username, user.IsAdmin)
	if err != nil {
		slog.Error("failed to create token", "err", err)
		writeError(w, 500, "token error")
		return
	}
	writeJSON(w, 200, models.LoginResponse{AccessToken: token, IsAdmin: user.IsAdmin, Username: user.Username})
}

// POST /api/register
func Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := decode(r, &req); err != nil {
		writeError(w, 400, "bad request")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, 400, "заполните все поля")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, 400, "пароль не менее 8 символов")
		return
	}

	var existing models.User
	_ = db.DB.SelectOne("user", map[string]string{"username": req.Username}, &existing)
	if existing.Username != "" {
		writeError(w, 409, "пользователь уже существует")
		return
	}

	hash, err := services.HashPassword(req.Password)
	if err != nil {
		slog.Error("failed to hash password", "err", err)
		writeError(w, 500, "hash error")
		return
	}

	if err := db.DB.Insert("user", map[string]string{
		"username":      req.Username,
		"password_hash": hash,
	}, nil); err != nil {
		slog.Error("failed to insert user", "username", req.Username, "err", err)
		writeError(w, 500, "db error")
		return
	}
	writeJSON(w, 201, map[string]bool{"success": true})
}

// POST /api/logout
func Logout(w http.ResponseWriter, r *http.Request) {
	// Мобильный клиент сам удаляет токен из secure storage.
	// Бэкенд просто подтверждает запрос.
	writeJSON(w, 200, map[string]bool{"success": true})
}

// POST /api/google/mobile-auth
func GoogleMobileAuth(w http.ResponseWriter, r *http.Request) {
	var req models.GoogleMobileRequest
	if err := decode(r, &req); err != nil || req.IDToken == "" {
		writeError(w, 400, "id_token required")
		return
	}

	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + req.IDToken)
	if err != nil {
		slog.Error("google tokeninfo request failed", "err", err)
		writeError(w, 500, "google verification failed")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		writeError(w, 401, "invalid id_token")
		return
	}

	var info struct {
		Email         string `json:"email"`
		Aud           string `json:"aud"`
		EmailVerified string `json:"email_verified"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &info); err != nil || info.Email == "" {
		writeError(w, 401, "could not parse token info")
		return
	}

	if info.EmailVerified != "true" {
		writeError(w, 401, "email not verified")
		return
	}

	if config.C.GoogleClientID != "" && info.Aud != config.C.GoogleClientID {
		writeError(w, 401, "token audience mismatch")
		return
	}

	upsertGoogleUser(info.Email)

	token, err := services.CreateAccessToken(info.Email, false)
	if err != nil {
		slog.Error("failed to create token for google user", "email", info.Email, "err", err)
		writeError(w, 500, "token error")
		return
	}
	writeJSON(w, 200, models.LoginResponse{
		AccessToken: token,
		IsAdmin:     false,
		Username:    info.Email,
	})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func upsertGoogleUser(email string) {
	var existing models.User
	_ = db.DB.SelectOne("user", map[string]string{"username": email}, &existing)
	if existing.Username == "" {
		hash, _ := services.HashPassword("oauth_user_" + email)
		if err := db.DB.Insert("user", map[string]string{
			"username":      email,
			"password_hash": hash,
		}, nil); err != nil {
			slog.Error("failed to upsert google user", "email", email, "err", err)
		}
	}
}

