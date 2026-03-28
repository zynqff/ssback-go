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

// ── POST /api/auth/send_otp ───────────────────────────────────────────────────
// Отправляет OTP код на email через Supabase Auth.
// is_new=true → регистрация (нужен username), false → вход.
func SendOTP(w http.ResponseWriter, r *http.Request) {
	var req models.SendOTPRequest
	if err := decode(r, &req); err != nil || req.Email == "" {
		writeError(w, 400, "email required")
		return
	}

	if req.IsNew {
		// Регистрация — проверяем что username и email свободны
		if req.Username == "" {
			writeError(w, 400, "username required for registration")
			return
		}
		if len([]rune(req.Username)) < 2 {
			writeError(w, 400, "никнейм слишком короткий (минимум 2 символа)")
			return
		}

		var byUsername models.User
		_ = db.DB.SelectOne("user", map[string]string{"username": req.Username}, &byUsername)
		if byUsername.Username != "" {
			writeError(w, 409, "этот никнейм уже занят")
			return
		}

		var byEmail models.User
		_ = db.DB.SelectOne("user", map[string]string{"email": req.Email}, &byEmail)
		if byEmail.Username != "" {
			writeError(w, 409, "этот email уже зарегистрирован")
			return
		}
	} else {
		// Вход — проверяем что пользователь существует
		var existing models.User
		_ = db.DB.SelectOne("user", map[string]string{"email": req.Email}, &existing)
		if existing.Username == "" {
			writeError(w, 404, "пользователь с таким email не найден")
			return
		}
	}

	// Отправляем OTP через Supabase Auth
	if err := services.SendSupabaseOTP(req.Email); err != nil {
		slog.Error("failed to send OTP", "email", req.Email, "err", err)
		writeError(w, 500, "ошибка отправки кода. Попробуйте позже.")
		return
	}

	writeJSON(w, 200, map[string]bool{"success": true})
}

// ── POST /api/auth/verify_otp ─────────────────────────────────────────────────
// Проверяет OTP код. При успехе создаёт пользователя (если регистрация) и возвращает JWT.
func VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req models.VerifyOTPRequest
	if err := decode(r, &req); err != nil || req.Email == "" || req.Token == "" {
		writeError(w, 400, "email and token required")
		return
	}

	// Проверяем код через Supabase Auth
	if err := services.VerifySupabaseOTP(req.Email, req.Token); err != nil {
		slog.Warn("OTP verification failed", "email", req.Email, "err", err)
		writeError(w, 401, "неверный или истёкший код")
		return
	}

	// Получаем пользователя из нашей таблицы
	var user models.User
	_ = db.DB.SelectOne("user", map[string]string{"email": req.Email}, &user)

	// Если пользователя нет — это регистрация через send_otp с is_new=true
	// username был передан при send_otp и уже проверен на уникальность
	// Но мы его не сохранили тогда — нужен второй запрос с username
	// Поэтому при регистрации клиент должен передать username и здесь
	if user.Username == "" {
		// Пользователь не найден — ошибка (регистрация делается через /api/auth/register_otp)
		writeError(w, 404, "пользователь не найден. Пройдите регистрацию.")
		return
	}

	token, err := services.CreateAccessToken(user.Username, user.IsAdmin)
	if err != nil {
		writeError(w, 500, "token error")
		return
	}

	writeJSON(w, 200, models.LoginResponse{
		AccessToken: token,
		IsAdmin:     user.IsAdmin,
		Username:    user.Username,
	})
}

// ── POST /api/auth/register_otp ───────────────────────────────────────────────
// Регистрация: проверяет OTP + создаёт пользователя с username.
func RegisterOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Token    string `json:"token"`
		Username string `json:"username"`
	}
	if err := decode(r, &req); err != nil || req.Email == "" || req.Token == "" || req.Username == "" {
		writeError(w, 400, "email, token and username required")
		return
	}
	if len([]rune(req.Username)) < 2 {
		writeError(w, 400, "никнейм слишком короткий")
		return
	}

	// Финальная проверка что никнейм и email ещё свободны
	var byUsername models.User
	_ = db.DB.SelectOne("user", map[string]string{"username": req.Username}, &byUsername)
	if byUsername.Username != "" {
		writeError(w, 409, "этот никнейм уже занят")
		return
	}
	var byEmail models.User
	_ = db.DB.SelectOne("user", map[string]string{"email": req.Email}, &byEmail)
	if byEmail.Username != "" {
		writeError(w, 409, "этот email уже зарегистрирован")
		return
	}

	// Проверяем OTP
	if err := services.VerifySupabaseOTP(req.Email, req.Token); err != nil {
		slog.Warn("OTP verification failed on register", "email", req.Email, "err", err)
		writeError(w, 401, "неверный или истёкший код")
		return
	}

	// Создаём пользователя
	if err := db.DB.Insert("user", map[string]interface{}{
		"username":        req.Username,
		"email":           req.Email,
		"is_admin":        false,
		"read_poems_json": []int64{},
		"show_all_tab":    false,
		"user_data":       "",
	}, nil); err != nil {
		slog.Error("failed to create user", "email", req.Email, "err", err)
		writeError(w, 500, "ошибка создания пользователя")
		return
	}

	token, err := services.CreateAccessToken(req.Username, false)
	if err != nil {
		writeError(w, 500, "token error")
		return
	}

	writeJSON(w, 201, models.LoginResponse{
		AccessToken: token,
		IsAdmin:     false,
		Username:    req.Username,
	})
}

// ── POST /api/auth/resolve_email ──────────────────────────────────────────────
// Возвращает email по username (для входа по никнейму).
func ResolveEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
	}
	if err := decode(r, &req); err != nil || req.Username == "" {
		writeError(w, 400, "username required")
		return
	}

	var user models.User
	_ = db.DB.SelectOne("user", map[string]string{"username": req.Username}, &user)
	if user.Email == "" {
		writeError(w, 404, "пользователь не найден")
		return
	}
	writeJSON(w, 200, map[string]string{"email": user.Email})
}

// ── POST /api/logout ──────────────────────────────────────────────────────────
func Logout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]bool{"success": true})
}

// ── POST /api/google/mobile-auth ─────────────────────────────────────────────
func GoogleMobileAuth(w http.ResponseWriter, r *http.Request) {
	var req models.GoogleMobileRequest
	if err := decode(r, &req); err != nil || req.IDToken == "" {
		writeError(w, 400, "id_token required")
		return
	}

	if config.C.GoogleClientID == "" {
		writeError(w, 500, "google auth not configured")
		return
	}

	email, err := verifyGoogleIDToken(r.Context(), req.IDToken, config.C.GoogleClientID)
	if err != nil {
		slog.Warn("google id_token verification failed", "err", err)
		writeError(w, 401, "invalid id_token")
		return
	}

	// Upsert пользователя
	username := upsertGoogleUser(email)

	token, err := services.CreateAccessToken(username, false)
	if err != nil {
		writeError(w, 500, "token error")
		return
	}
	writeJSON(w, 200, models.LoginResponse{
		AccessToken: token,
		IsAdmin:     false,
		Username:    username,
	})
}

func verifyGoogleIDToken(ctx context.Context, rawToken, audience string) (string, error) {
	payload, err := idtoken.Validate(ctx, rawToken, audience)
	if err != nil {
		return "", err
	}
	email, _ := payload.Claims["email"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	if email == "" || !emailVerified {
		return "", http.ErrNoCookie
	}
	return email, nil
}

// upsertGoogleUser создаёт пользователя если не существует, возвращает username.
func upsertGoogleUser(email string) string {
	var existing models.User
	_ = db.DB.SelectOne("user", map[string]string{"email": email}, &existing)
	if existing.Username != "" {
		return existing.Username
	}

	// Новый пользователь — username = email (можно потом сменить)
	_ = db.DB.Insert("user", map[string]interface{}{
		"username":        email,
		"email":           email,
		"is_admin":        false,
		"read_poems_json": []int64{},
		"show_all_tab":    false,
		"user_data":       "",
	}, nil)

	return email
}
