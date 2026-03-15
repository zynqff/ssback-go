package handlers

import (
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

	hasAccess := claims.IsAdmin

	if !hasAccess {
		var user models.User
		_ = db.DB.SelectOne("user", map[string]string{"username": claims.Username}, &user)
		if user.UserGeminiKey != "" && services.ValidateAndUseKey(user.UserGeminiKey) {
			hasAccess = true
		}
	}

	if !hasAccess {
		writeError(w, 403, "нет доступа к AI. Введите действующий ключ в профиле")
		return
	}

	history, err := services.GetChatHistory(claims.Username)
	if err != nil {
		history = []models.ChatMessage{}
	}

	response, err := services.GetGroqResponse(req.Prompt, history)
	if err != nil {
		writeError(w, 500, "groq error: "+err.Error())
		return
	}

	_ = services.SaveChatMessage(claims.Username, "user", req.Prompt)
	_ = services.SaveChatMessage(claims.Username, "assistant", response)

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

	if err := db.DB.Update("user", map[string]string{"username": claims.Username}, map[string]interface{}{
		"user_gemini_key": req.Key,
	}); err != nil {
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

	key, err := services.SaveAPIKey(claims.Username, expiresAt, dailyLimit)
	if err != nil {
		writeError(w, 500, "failed to generate key")
		return
	}
	writeJSON(w, 200, map[string]string{"key": key})
}

// GET /api/ai/keys (admin)
func AIGetKeys(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	keys, err := services.GetKeysForAdmin(claims.Username)
	if err != nil {
		writeError(w, 500, "db error")
		return
	}
	writeJSON(w, 200, keys)
}

// POST /api/ai/disable_key/{key} (admin)
func AIDisableKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key string `json:"key"`
	}
	if err := decode(r, &req); err != nil || req.Key == "" {
		writeError(w, 400, "key required")
		return
	}
	if err := services.DisableKey(req.Key); err != nil {
		writeError(w, 500, "db error")
		return
	}
	writeJSON(w, 200, map[string]bool{"success": true})
}
