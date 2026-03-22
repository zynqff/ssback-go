package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ssback/internal/db"
	"github.com/ssback/internal/middleware"
	"github.com/ssback/internal/models"
	"github.com/ssback/internal/services"
)

// POST /api/ai/chat
func AIChat(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req models.ChatRequest
	if err := decode(r, &req); err != nil || req.Prompt == "" {
		writeError(w, 400, "prompt required")
		return
	}
	if len([]rune(req.Prompt)) > 2000 {
		writeError(w, 400, "промпт слишком длинный (максимум 2000 символов)")
		return
	}

	// Проверяем ai_enabled из app_config
	var aiEnabledRow struct {
		Value string `json:"value"`
	}
	_ = db.DB.SelectOne("app_config", map[string]string{"key": "ai_enabled"}, &aiEnabledRow)
	if aiEnabledRow.Value == "false" {
		writeError(w, 503, "AI чат временно отключён администратором")
		return
	}

	// Проверяем ai_daily_limit из app_config (не распространяется на админа)
	if !claims.IsAdmin {
		var limitRow struct {
			Value string `json:"value"`
		}
		_ = db.DB.SelectOne("app_config", map[string]string{"key": "ai_daily_limit"}, &limitRow)
		if limitRow.Value != "" && limitRow.Value != "0" {
			var limit int
			fmt.Sscanf(limitRow.Value, "%d", &limit)
			if limit > 0 {
				today := time.Now().UTC().Format("2006-01-02")
				count, err := services.GetUserAIChatCountToday(claims.Subject, today)
				if err == nil && count >= limit {
					writeError(w, 429, fmt.Sprintf("достигнут дневной лимит AI (%d сообщений)", limit))
					return
				}
			}
		}
	}

	hasAccess := claims.IsAdmin

	if !hasAccess {
		var user models.User
		_ = db.DB.SelectOne("user", map[string]string{"username": claims.Subject}, &user)
		if user.UserGeminiKey != "" && services.ValidateAndUseKey(user.UserGeminiKey) {
			hasAccess = true
		}
	}

	if !hasAccess {
		writeError(w, 403, "нет доступа к AI. Введите действующий ключ в профиле")
		return
	}

	history, err := services.GetChatHistory(claims.Subject)
	if err != nil {
		history = []models.ChatMessage{}
	}

	response, err := services.GetGroqResponse(req.Prompt, history)
	if err != nil {
		slog.Error("groq request failed", "user", claims.Subject, "err", err)
		writeError(w, 500, "ошибка AI, попробуйте позже")
		return
	}

	_ = services.SaveChatMessage(claims.Subject, "user", req.Prompt)
	_ = services.SaveChatMessage(claims.Subject, "assistant", response)

	writeJSON(w, 200, map[string]string{"response": response})
}

// POST /api/ai/verify_key
func AIVerifyKey(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req models.VerifyKeyRequest
	if err := decode(r, &req); err != nil || req.Key == "" {
		writeError(w, 400, "key required")
		return
	}

	if !services.ValidateAndUseKey(req.Key) {
		writeError(w, 403, "неверный или просроченный ключ")
		return
	}

	if err := db.DB.Update("user", map[string]string{"username": claims.Subject}, map[string]interface{}{
		"user_gemini_key": req.Key,
	}); err != nil {
		slog.Error("failed to save user key", "user", claims.Subject, "err", err)
		writeError(w, 500, "db error")
		return
	}
	writeJSON(w, 200, map[string]bool{"success": true})
}

// POST /api/ai/generate_key (admin)
func AIGenerateKey(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req models.GenerateKeyRequest
	_ = decode(r, &req)

	var expiresAt *time.Time
	if req.ExpiresInHours > 0 {
		t := time.Now().UTC().Add(time.Duration(req.ExpiresInHours) * time.Hour)
		expiresAt = &t
	}

	var dailyLimit *int
	if req.DailyLimit > 0 {
		dailyLimit = &req.DailyLimit
	}

	key, err := services.SaveAPIKey(claims.Subject, expiresAt, dailyLimit)
	if err != nil {
		slog.Error("failed to generate AI key", "admin", claims.Subject, "err", err)
		writeError(w, 500, "failed to generate key")
		return
	}
	writeJSON(w, 200, map[string]string{"key": key})
}

// GET /api/ai/keys (admin)
func AIGetKeys(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	keys, err := services.GetKeysForAdmin(claims.Subject)
	if err != nil {
		slog.Error("failed to get AI keys", "admin", claims.Subject, "err", err)
		writeError(w, 500, "db error")
		return
	}
	writeJSON(w, 200, keys)
}

// POST /api/ai/disable_key (admin)
func AIDisableKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key string `json:"key"`
	}
	if err := decode(r, &req); err != nil || req.Key == "" {
		writeError(w, 400, "key required")
		return
	}
	if err := services.DisableKey(req.Key); err != nil {
		slog.Error("failed to disable AI key", "key", req.Key, "err", err)
		writeError(w, 500, "db error")
		return
	}
	writeJSON(w, 200, map[string]bool{"success": true})
}
