package handlers

import (
	"net/http"

	"github.com/ssback/internal/db"
	"github.com/ssback/internal/middleware"
	"github.com/ssback/internal/models"
)

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

	if req.NewPassword != nil {
		if len(*req.NewPassword) < 4 {
			writeError(w, 400, "пароль не менее 4 символов")
			return
		}
		hash, err := hashPassword(*req.NewPassword)
		if err != nil {
			writeError(w, 500, "hash error")
			return
		}
		update["password_hash"] = hash
	}

	if req.UserData != nil {
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
