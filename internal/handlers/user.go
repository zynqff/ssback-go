package handlers

import (
	"net/http"

	"github.com/ssback/internal/db"
	"github.com/ssback/internal/middleware"
	"github.com/ssback/internal/models"
	"github.com/ssback/internal/services"
)

const maxUserDataBytes = 2000

// GET /api/me
func GetMe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var user models.User
	if err := db.DB.SelectOne("user", map[string]string{"username": claims.Subject}, &user); err != nil || user.Username == "" {
		writeError(w, 404, "пользователь не найден")
		return
	}

	reads := user.ReadPoemsJSON
	if reads == nil {
		reads = []int64{}
	}

	writeJSON(w, 200, models.MeResponse{
		Username:     user.Username,
		IsAdmin:      user.IsAdmin,
		ReadPoems:    reads,
		PinnedPoemID: user.PinnedPoemID,
		ShowAllTab:   user.ShowAllTab,
		UserData:     user.UserData,
	})
}

// POST /api/profile
func UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req models.UpdateProfileRequest
	if err := decode(r, &req); err != nil {
		writeError(w, 400, "bad request")
		return
	}

	update := map[string]interface{}{}

	if req.UserData != nil {
		if len(*req.UserData) > maxUserDataBytes {
			writeError(w, 400, "заметки слишком длинные (максимум 2000 символов)")
			return
		}
		update["user_data"] = *req.UserData
	}

	if req.ShowAllTab != nil {
		update["show_all_tab"] = *req.ShowAllTab
	}

	if len(update) > 0 {
		if err := db.DB.Update("user", map[string]string{"username": claims.Subject}, update); err != nil {
			writeError(w, 500, "db error")
			return
		}
	}

	writeJSON(w, 200, map[string]bool{"success": true})
}

// POST /api/change_username
func ChangeUsername(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req models.ChangeUsernameRequest
	if err := decode(r, &req); err != nil || req.NewUsername == "" {
		writeError(w, 400, "new_username required")
		return
	}
	if len([]rune(req.NewUsername)) < 2 {
		writeError(w, 400, "никнейм слишком короткий")
		return
	}
	if req.NewUsername == claims.Subject {
		writeError(w, 400, "это уже ваш никнейм")
		return
	}

	var existing models.User
	_ = db.DB.SelectOne("user", map[string]string{"username": req.NewUsername}, &existing)
	if existing.Username != "" {
		writeError(w, 409, "этот никнейм уже занят")
		return
	}

	if err := db.DB.Update("user", map[string]string{"username": claims.Subject}, map[string]interface{}{
		"username": req.NewUsername,
	}); err != nil {
		writeError(w, 500, "db error")
		return
	}

	// Выдаём новый токен с новым username
	token, err := services.CreateAccessToken(req.NewUsername, claims.IsAdmin)
	if err != nil {
		writeError(w, 500, "token error")
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"success":      true,
		"access_token": token,
		"username":     req.NewUsername,
	})
}

// POST /api/change_email
func ChangeEmail(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req models.ChangeEmailRequest
	if err := decode(r, &req); err != nil || req.NewEmail == "" {
		writeError(w, 400, "new_email required")
		return
	}

	var existing models.User
	_ = db.DB.SelectOne("user", map[string]string{"email": req.NewEmail}, &existing)
	if existing.Username != "" {
		writeError(w, 409, "этот email уже используется")
		return
	}

	if err := db.DB.Update("user", map[string]string{"username": claims.Subject}, map[string]interface{}{
		"email": req.NewEmail,
	}); err != nil {
		writeError(w, 500, "db error")
		return
	}

	writeJSON(w, 200, map[string]bool{"success": true})
}

// POST /api/toggle_read
func ToggleRead(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req models.ToggleRequest
	if err := decode(r, &req); err != nil || req.PoemID == 0 {
		writeError(w, 400, "poem_id required")
		return
	}

	var user models.User
	if err := db.DB.SelectOne("user", map[string]string{"username": claims.Subject}, &user); err != nil || user.Username == "" {
		writeError(w, 404, "пользователь не найден")
		return
	}

	reads := user.ReadPoemsJSON
	if reads == nil {
		reads = []int64{}
	}

	action := "marked"
	newReads := []int64{}
	found := false
	for _, id := range reads {
		if id == req.PoemID {
			found = true
			action = "unmarked"
		} else {
			newReads = append(newReads, id)
		}
	}
	if !found {
		newReads = append(newReads, req.PoemID)
	}

	if err := db.DB.Update("user", map[string]string{"username": claims.Subject}, map[string]interface{}{
		"read_poems_json": newReads,
	}); err != nil {
		writeError(w, 500, "db error")
		return
	}

	writeJSON(w, 200, models.ToggleResponse{Success: true, Action: action})
}

// POST /api/toggle_pin
func TogglePin(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req models.ToggleRequest
	if err := decode(r, &req); err != nil || req.PoemID == 0 {
		writeError(w, 400, "poem_id required")
		return
	}

	var user models.User
	if err := db.DB.SelectOne("user", map[string]string{"username": claims.Subject}, &user); err != nil || user.Username == "" {
		writeError(w, 404, "пользователь не найден")
		return
	}

	var newPinned *int64
	action := "pinned"
	if user.PinnedPoemID != nil && *user.PinnedPoemID == req.PoemID {
		action = "unpinned"
		newPinned = nil
	} else {
		id := req.PoemID
		newPinned = &id
	}

	if err := db.DB.Update("user", map[string]string{"username": claims.Subject}, map[string]interface{}{
		"pinned_poem_id": newPinned,
	}); err != nil {
		writeError(w, 500, "db error")
		return
	}

	writeJSON(w, 200, models.ToggleResponse{Success: true, Action: action, PinnedPoemID: newPinned})
}
