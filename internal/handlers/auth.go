package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"google.golang.org/api/idtoken"

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
	writeJSON(w, 200, map[string]bool{"success": true})
}

// POST /api/google/mobile-auth
func GoogleMobileAuth(w http.ResponseWriter, r *http.Request) {
	var req models.GoogleMobileRequest
	if err := decode(r, &req); err != nil || req.IDToken == "" {
		writeError(w, 400, "id_token required")
		return
	}

	if config.C.GoogleClientID == "" {
		slog.Error("GOOGLE_CLIENT_ID not configured")
		writeError(w, 500, "google auth not configured")
		return
	}

	email, err := verifyGoogleIDToken(r.Context(), req.IDToken, config.C.GoogleClientID)
	if err != nil {
		slog.Warn("google id_token verification failed", "err", err)
		writeError(w, 401, "invalid id_token")
		return
	}

	upsertGoogleUser(email)

	token, err := services.CreateAccessToken(email, false)
	if err != nil {
		slog.Error("failed to create token for google user", "email", email, "err", err)
		writeError(w, 500, "token error")
		return
	}
	writeJSON(w, 200, models.LoginResponse{
		AccessToken: token,
		IsAdmin:     false,
		Username:    email,
	})
}

// verifyGoogleIDToken проверяет подпись токена локально через официальную
// библиотеку google.golang.org/api/idtoken (уже есть в go.mod).
// Кэширует публичные ключи Google — не делает HTTP-запрос на каждый вход.
func verifyGoogleIDToken(ctx context.Context, rawToken, audience string) (string, error) {
	payload, err := idtoken.Validate(ctx, rawToken, audience)
	if err != nil {
		return "", err
	}
	email, _ := payload.Claims["email"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	if email == "" || !emailVerified {
		return "", http.ErrNoCookie // generic sentinel — caller logs it
	}
	return email, nil
}

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
