package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ssback/internal/config"
	"github.com/ssback/internal/db"
	"github.com/ssback/internal/models"
	"github.com/ssback/internal/services"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func googleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     config.C.GoogleClientID,
		ClientSecret: config.C.GoogleClientSecret,
		RedirectURL:  "",
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

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
	if len(req.Password) < 4 {
		writeError(w, 400, "пароль не менее 4 символов")
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
		writeError(w, 500, "hash error")
		return
	}

	if err := db.DB.Insert("user", map[string]string{
		"username":      req.Username,
		"password_hash": hash,
	}, nil); err != nil {
		writeError(w, 500, "db error")
		return
	}
	writeJSON(w, 201, map[string]bool{"success": true})
}

// GET /api/logout
func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:    "access_token",
		Value:   "",
		Expires: time.Unix(0, 0),
		MaxAge:  -1,
		Path:    "/",
	})
	writeJSON(w, 200, map[string]bool{"success": true})
}

// GET /auth/google/login
func GoogleLogin(w http.ResponseWriter, r *http.Request) {
	cfg := googleOAuthConfig()
	cfg.RedirectURL = fmt.Sprintf("%s://%s/auth/google/callback", scheme(r), r.Host)
	url := cfg.AuthCodeURL("state-token", oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GET /auth/google/callback
func GoogleCallback(w http.ResponseWriter, r *http.Request) {
	cfg := googleOAuthConfig()
	cfg.RedirectURL = fmt.Sprintf("%s://%s/auth/google/callback", scheme(r), r.Host)

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, "/login?error=google_auth_failed", http.StatusSeeOther)
		return
	}

	oauthToken, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		http.Redirect(w, r, "/login?error=google_auth_failed", http.StatusSeeOther)
		return
	}

	email, err := getGoogleEmail(oauthToken.AccessToken)
	if err != nil || email == "" {
		http.Redirect(w, r, "/login?error=google_auth_failed", http.StatusSeeOther)
		return
	}

	upsertGoogleUser(email)

	token, err := services.CreateAccessToken(email, false)
	if err != nil {
		http.Redirect(w, r, "/login?error=token_error", http.StatusSeeOther)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "Bearer " + token,
		HttpOnly: true,
		MaxAge:   86400,
		Path:     "/",
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
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

	// Проверяем что email подтверждён
	if info.EmailVerified != "true" {
		writeError(w, 401, "email not verified")
		return
	}

	// Проверяем что токен выдан для нашего приложения
	if config.C.GoogleClientID != "" && info.Aud != config.C.GoogleClientID {
		writeError(w, 401, "token audience mismatch")
		return
	}

	upsertGoogleUser(info.Email)

	token, err := services.CreateAccessToken(info.Email, false)
	if err != nil {
		writeError(w, 500, "token error")
		return
	}
	writeJSON(w, 200, models.LoginResponse{
		AccessToken: token,
		IsAdmin:     false,
		Username:    info.Email,
	})
}

// helpers

func upsertGoogleUser(email string) {
	var existing models.User
	_ = db.DB.SelectOne("user", map[string]string{"username": email}, &existing)
	if existing.Username == "" {
		hash, _ := services.HashPassword("oauth_user_" + email)
		_ = db.DB.Insert("user", map[string]string{
			"username":      email,
			"password_hash": hash,
		}, nil)
	}
}

func getGoogleEmail(accessToken string) (string, error) {
	req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var info struct {
		Email string `json:"email"`
	}
	json.NewDecoder(resp.Body).Decode(&info)
	return info.Email, nil
}

func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
		return fwd
	}
	return "http"
}
