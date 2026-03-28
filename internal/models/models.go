package models

import "time"

type User struct {
	Username      string  `json:"username"`
	Email         string  `json:"email"`
	IsAdmin       bool    `json:"is_admin"`
	ReadPoemsJSON []int64 `json:"read_poems_json"`
	PinnedPoemID  *int64  `json:"pinned_poem_id"`
	ShowAllTab    bool    `json:"show_all_tab"`
	UserData      string  `json:"user_data"`
	UserGeminiKey string  `json:"user_gemini_key,omitempty"`
}

type Poem struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Text      string `json:"text"`
	LineCount int    `json:"line_count,omitempty"`
}

type AIKey struct {
	Key           string     `json:"key"`
	GeneratedBy   string     `json:"generated_by"`
	ExpiresAt     *time.Time `json:"expires_at"`
	DailyLimit    *int       `json:"daily_limit"`
	IsActive      bool       `json:"is_active"`
	UsageToday    int        `json:"usage_today"`
	LastUsageDate *string    `json:"last_usage_date"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ── Auth ──────────────────────────────────────────────────────────────────────

type SendOTPRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"` // только при регистрации (is_new=true)
	IsNew    bool   `json:"is_new"`
}

type VerifyOTPRequest struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type GoogleMobileRequest struct {
	IDToken string `json:"id_token"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	IsAdmin     bool   `json:"is_admin"`
	Username    string `json:"username"`
}

// ── User ──────────────────────────────────────────────────────────────────────

type MeResponse struct {
	Username     string  `json:"username"`
	IsAdmin      bool    `json:"is_admin"`
	ReadPoems    []int64 `json:"read_poems"`
	PinnedPoemID *int64  `json:"pinned_poem_id"`
	ShowAllTab   bool    `json:"show_all_tab"`
	UserData     string  `json:"user_data"`
}

type UpdateProfileRequest struct {
	UserData   *string `json:"user_data"`
	ShowAllTab *bool   `json:"show_all_tab"`
}

type ChangeUsernameRequest struct {
	NewUsername string `json:"new_username"`
}

type ChangeEmailRequest struct {
	NewEmail string `json:"new_email"`
}

type ToggleRequest struct {
	PoemID int64 `json:"poem_id"`
}

type ToggleResponse struct {
	Success      bool   `json:"success"`
	Action       string `json:"action"`
	PinnedPoemID *int64 `json:"pinned_poem_id,omitempty"`
}

type PoemCreate struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	Text   string `json:"text"`
}

type ChatRequest struct {
	Prompt string `json:"prompt"`
}

type GenerateKeyRequest struct {
	ExpiresInHours int `json:"expires_in_hours"`
	DailyLimit     int `json:"daily_limit"`
}

type VerifyKeyRequest struct {
	Key string `json:"key"`
}
