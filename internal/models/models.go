package models

import "time"

type User struct {
	Username        string   `json:"username"`
	PasswordHash    string   `json:"password_hash,omitempty"`
	IsAdmin         bool     `json:"is_admin"`
	ReadPoemsJSON   []string `json:"read_poems_json"`
	PinnedPoemTitle *string  `json:"pinned_poem_title"`
	ShowAllTab      bool     `json:"show_all_tab"`
	UserData        string   `json:"user_data"`
	UserGeminiKey   string   `json:"user_gemini_key,omitempty"`
}

type Poem struct {
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

// Request/response types

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
	Title string `json:"title"`
}

type ToggleResponse struct {
	Success     bool    `json:"success"`
	Action      string  `json:"action"`
	PinnedTitle *string `json:"pinned_title,omitempty"`
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

type MeResponse struct {
	Username        string   `json:"username"`
	IsAdmin         bool     `json:"is_admin"`
	ReadPoems       []string `json:"read_poems"`
	PinnedPoemTitle *string  `json:"pinned_poem_title"`
	ShowAllTab      bool     `json:"show_all_tab"`
	UserData        string   `json:"user_data"`
}
