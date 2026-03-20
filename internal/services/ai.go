package services

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ssback/internal/config"
	"github.com/ssback/internal/db"
	"github.com/ssback/internal/models"
)

const groqModel = "moonshotai/kimi-k2-instruct"
const groqURL   = "https://api.groq.com/openai/v1/chat/completions"

// maxChatHistory — сколько сообщений хранить на пользователя в Supabase.
const maxChatHistory = 40

func GenerateAPIKey() string {
	b := make([]byte, 24)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

type dbAIKey struct {
	Key           string  `json:"key"`
	GeneratedBy   string  `json:"generated_by"`
	ExpiresAt     *string `json:"expires_at"`
	DailyLimit    *int    `json:"daily_limit"`
	IsActive      bool    `json:"is_active"`
	UsageToday    int     `json:"usage_today"`
	LastUsageDate *string `json:"last_usage_date"`
}

func SaveAPIKey(generatedBy string, expiresAt *time.Time, dailyLimit *int) (string, error) {
	key := GenerateAPIKey()
	row := map[string]interface{}{
		"key":          key,
		"generated_by": generatedBy,
		"is_active":    true,
		"usage_today":  0,
	}
	if expiresAt != nil {
		row["expires_at"] = expiresAt.Format(time.RFC3339)
	}
	if dailyLimit != nil {
		row["daily_limit"] = *dailyLimit
	}
	return key, db.DB.Insert("ai_keys", row, nil)
}

func ValidateAndUseKey(key string) bool {
	var k dbAIKey
	if err := db.DB.SelectOne("ai_keys", map[string]string{"key": key}, &k); err != nil {
		slog.Error("failed to select ai_key", "err", err)
		return false
	}
	if k.Key == "" || !k.IsActive {
		return false
	}

	if k.ExpiresAt != nil {
		exp, err := time.Parse(time.RFC3339, *k.ExpiresAt)
		if err == nil && time.Now().UTC().After(exp) {
			return false
		}
	}

	today := time.Now().UTC().Format("2006-01-02")

	usageToday := k.UsageToday
	if k.LastUsageDate == nil || *k.LastUsageDate != today {
		usageToday = 0
	}

	if k.DailyLimit != nil && usageToday >= *k.DailyLimit {
		return false
	}

	err := db.DB.UpdateAtomic(
		"ai_keys",
		map[string]string{"key": key},
		fmt.Sprintf("%d", usageToday),
		map[string]interface{}{
			"usage_today":     usageToday + 1,
			"last_usage_date": today,
		},
	)
	if err != nil {
		slog.Warn("ai key atomic update failed (possible race), denying", "key", key)
		return false
	}
	return true
}

func DisableKey(key string) error {
	return db.DB.Update("ai_keys", map[string]string{"key": key}, map[string]interface{}{"is_active": false})
}

func GetKeysForAdmin(adminUsername string) ([]dbAIKey, error) {
	var keys []dbAIKey
	err := db.DB.Select("ai_keys", map[string]string{"generated_by": adminUsername}, &keys)
	return keys, err
}

// ── Chat history ──────────────────────────────────────────────────────────────

type dbChatRow struct {
	Username  string `json:"username"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

func SaveChatMessage(username, role, content string) error {
	if err := db.DB.Insert("ai_chat_history", map[string]string{
		"username": username,
		"role":     role,
		"content":  content,
	}, nil); err != nil {
		return err
	}
	// Ротация: удаляем старые записи в фоне, не задерживаем ответ клиенту.
	go func() {
		if err := db.DB.DeleteOldChatHistory(username, maxChatHistory); err != nil {
			slog.Warn("chat history rotation failed", "user", username, "err", err)
		}
	}()
	return nil
}

func GetChatHistory(username string) ([]models.ChatMessage, error) {
	var rows []dbChatRow
	err := db.DB.SelectOrdered("ai_chat_history", map[string]string{"username": username}, "created_at", false, 20, &rows)
	if err != nil {
		return nil, err
	}
	msgs := make([]models.ChatMessage, len(rows))
	for i, r := range rows {
		msgs[i] = models.ChatMessage{Role: r.Role, Content: r.Content}
	}
	return msgs, nil
}

// ── Groq request ──────────────────────────────────────────────────────────────

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqRequest struct {
	Model       string        `json:"model"`
	Messages    []groqMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func GetGroqResponse(prompt string, history []models.ChatMessage) (string, error) {
	if config.C.GroqAPIKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY not set")
	}

	msgs := []groqMessage{
		{
			Role: "system",
			Content: "Ты — умный помощник на сайте «Сборник стихов». " +
				"Помогаешь пользователям находить стихи, обсуждать их содержание, " +
				"историю и смысл. Отвечай на русском языке, кратко и по делу.",
		},
	}
	for _, h := range history {
		msgs = append(msgs, groqMessage{Role: h.Role, Content: h.Content})
	}
	msgs = append(msgs, groqMessage{Role: "user", Content: prompt})

	payload := groqRequest{
		Model:       groqModel,
		Messages:    msgs,
		MaxTokens:   1024,
		Temperature: 0.7,
	}

	b, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", groqURL, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+config.C.GroqAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var gr groqResponse
	if err := json.Unmarshal(body, &gr); err != nil {
		return "", err
	}
	if gr.Error != nil {
		return "", fmt.Errorf("groq: %s", gr.Error.Message)
	}
	if len(gr.Choices) == 0 {
		return "", fmt.Errorf("groq: empty response")
	}
	return gr.Choices[0].Message.Content, nil
}
