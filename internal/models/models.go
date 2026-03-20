package models

import "time"

type User struct {
	Username      string  `json:"username"`
	PasswordHash  string  `json:"password_hash"` // было json:"-" — хеш не читался из Supabase
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

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	IsAdmin     bool   `json:"is_admin"`
	Username    string `json:"username"`
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

type UpdateProfileRequest struct {
	NewPassword *string `json:"new_password"`
	UserData    *string `json:"user_data"`
	ShowAllTab  *bool   `json:"show_all_tab"`
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

type GoogleMobileRequest struct {
	IDToken string `json:"id_token"`
}

// MeResponse — то что отдаётся клиенту. PasswordHash здесь намеренно отсутствует.
type MeResponse struct {
	Username     string  `json:"username"`
	IsAdmin      bool    `json:"is_admin"`
	ReadPoems    []int64 `json:"read_poems"`
	PinnedPoemID *int64  `json:"pinned_poem_id"`
	ShowAllTab   bool    `json:"show_all_tab"`
	UserData     string  `json:"user_data"`
}
