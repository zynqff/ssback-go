package handlers

import (
	"net/http"

	"github.com/ssback/internal/db"
	"github.com/ssback/internal/middleware"
	"github.com/ssback/internal/models"
	"github.com/ssback/internal/services"
)

// GET /api/me
func GetMe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	if services.IsVirtualAdmin(claims.Username) {
		writeJSON(w, 200, models.MeResponse{
			Username:        claims.Username,
			IsAdmin:         true,
			ReadPoems:       services.GetVirtualAdminReadPoems(claims.Username),
			PinnedPoemTitle: services.GetVirtualAdminPinnedPoem(claims.Username),
			ShowAllTab:      false,
			UserData:        "",
		})
		return
	}

	var user models.User
	if err := db.DB.SelectOne("user", map[string]string{"username": claims.Username}, &user); err != nil || user.Username == "" {
		writeError(w, 404, "пользователь не найден")
		return
	}

	reads := user.ReadPoemsJSON
	if reads == nil {
		reads = []string{}
	}

	writeJSON(w, 200, models.MeResponse{
		Username:        user.Username,
		IsAdmin:         user.IsAdmin,
		ReadPoems:       reads,
		PinnedPoemTitle: user.PinnedPoemTitle,
		ShowAllTab:      user.ShowAllTab,
		UserData:        user.UserData,
	})
}

// POST /api/profile
func UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	if services.IsVirtualAdmin(claims.Username) {
		writeError(w, 403, "настройки профиля недоступны для виртуальных администраторов")
		return
	}

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
		hash, err := services.HashPassword(*req.NewPassword)
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
		if err := db.DB.Update("user", map[string]string{"username": claims.Username}, update); err != nil {
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
	if err := decode(r, &req); err != nil || req.Title == "" {
		writeError(w, 400, "title required")
		return
	}

	if services.IsVirtualAdmin(claims.Username) {
		action := services.ToggleVirtualAdminRead(claims.Username, req.Title)
		writeJSON(w, 200, models.ToggleResponse{Success: true, Action: action})
		return
	}

	var user models.User
	if err := db.DB.SelectOne("user", map[string]string{"username": claims.Username}, &user); err != nil || user.Username == "" {
		writeError(w, 404, "пользователь не найден")
		return
	}

	reads := user.ReadPoemsJSON
	if reads == nil {
		reads = []string{}
	}

	action := "marked"
	newReads := []string{}
	found := false
	for _, t := range reads {
		if t == req.Title {
			found = true
			action = "unmarked"
		} else {
			newReads = append(newReads, t)
		}
	}
	if !found {
		newReads = append(newReads, req.Title)
	}

	if err := db.DB.Update("user", map[string]string{"username": claims.Username}, map[string]interface{}{
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
	if err := decode(r, &req); err != nil || req.Title == "" {
		writeError(w, 400, "title required")
		return
	}

	if services.IsVirtualAdmin(claims.Username) {
		action, pinned := services.ToggleVirtualAdminPinned(claims.Username, req.Title)
		writeJSON(w, 200, models.ToggleResponse{Success: true, Action: action, PinnedTitle: pinned})
		return
	}

	var user models.User
	if err := db.DB.SelectOne("user", map[string]string{"username": claims.Username}, &user); err != nil || user.Username == "" {
		writeError(w, 404, "пользователь не найден")
		return
	}

	var newPinned *string
	action := "pinned"
	if user.PinnedPoemTitle != nil && *user.PinnedPoemTitle == req.Title {
		action = "unpinned"
		newPinned = nil
	} else {
		t := req.Title
		newPinned = &t
	}

	if err := db.DB.Update("user", map[string]string{"username": claims.Username}, map[string]interface{}{
		"pinned_poem_title": newPinned,
	}); err != nil {
		writeError(w, 500, "db error")
		return
	}

	writeJSON(w, 200, models.ToggleResponse{Success: true, Action: action, PinnedTitle: newPinned})
}
