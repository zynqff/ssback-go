package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ssback/internal/config"
)

// SendSupabaseOTP — отправляет OTP письмо через Supabase Auth Admin API.
// Supabase сам генерирует код и отправляет письмо через настроенный SMTP.
// Требует SUPABASE_SERVICE_KEY (service_role key).
func SendSupabaseOTP(email string) error {
	if config.C.SupabaseServiceKey == "" {
		return fmt.Errorf("SUPABASE_SERVICE_KEY not set")
	}

	url := config.C.SupabaseURL + "/auth/v1/otp"

	body, _ := json.Marshal(map[string]interface{}{
		"email":       email,
		"create_user": false, // мы управляем созданием юзера сами
	})

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("apikey", config.C.SupabaseServiceKey)
	req.Header.Set("Authorization", "Bearer "+config.C.SupabaseServiceKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase otp error %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

// VerifySupabaseOTP — проверяет OTP код через Supabase Auth Admin API.
// Возвращает email пользователя при успехе.
func VerifySupabaseOTP(email, token string) error {
	if config.C.SupabaseServiceKey == "" {
		return fmt.Errorf("SUPABASE_SERVICE_KEY not set")
	}

	url := config.C.SupabaseURL + "/auth/v1/verify"

	body, _ := json.Marshal(map[string]interface{}{
		"email": email,
		"token": token,
		"type":  "magiclink", // Supabase OTP использует тип magiclink
	})

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("apikey", config.C.SupabaseServiceKey)
	req.Header.Set("Authorization", "Bearer "+config.C.SupabaseServiceKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("invalid or expired otp: %s", string(b))
	}
	return nil
}
