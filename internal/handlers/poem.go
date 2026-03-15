package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ssback/internal/db"
	"github.com/ssback/internal/models"
	"github.com/ssback/internal/services"
)

// GET /api/poems
func GetPoems(w http.ResponseWriter, r *http.Request) {
	var poems []models.Poem
	if err := db.DB.Select("poem", nil, &poems); err != nil {
		writeError(w, 500, "db error")
		return
	}
	if poems == nil {
		poems = []models.Poem{}
	}
	services.ProcessPoems(poems)
	writeJSON(w, 200, map[string]interface{}{"success": true, "poems": poems})
}

// POST /api/poems (admin)
func AddPoem(w http.ResponseWriter, r *http.Request) {
	var req models.PoemCreate
	if err := decode(r, &req); err != nil {
		writeError(w, 400, "bad request")
		return
	}
	if req.Title == "" || req.Author == "" || req.Text == "" {
		writeError(w, 400, "все поля должны быть заполнены")
		return
	}

	var existing models.Poem
	_ = db.DB.SelectOne("poem", map[string]string{"title": req.Title}, &existing)
	if existing.Title != "" {
		writeError(w, 409, "стих с таким названием уже существует")
		return
	}

	var inserted []models.Poem
	if err := db.DB.Insert("poem", req, &inserted); err != nil || len(inserted) == 0 {
		writeError(w, 500, "db error")
		return
	}
	p := inserted[0]
	services.ProcessPoem(&p)
	writeJSON(w, 201, map[string]interface{}{"success": true, "poem": p})
}

// PUT /api/poems/{title} (admin)
func EditPoem(w http.ResponseWriter, r *http.Request) {
	originalTitle := chi.URLParam(r, "title")

	var req models.PoemCreate
	if err := decode(r, &req); err != nil {
		writeError(w, 400, "bad request")
		return
	}
	if req.Title == "" || req.Author == "" || req.Text == "" {
		writeError(w, 400, "все поля должны быть заполнены")
		return
	}

	var existing models.Poem
	_ = db.DB.SelectOne("poem", map[string]string{"title": originalTitle}, &existing)
	if existing.Title == "" {
		writeError(w, 404, "стих не найден")
		return
	}

	if req.Title != originalTitle {
		var conflict models.Poem
		_ = db.DB.SelectOne("poem", map[string]string{"title": req.Title}, &conflict)
		if conflict.Title != "" {
			writeError(w, 409, "стих с новым названием уже существует")
			return
		}
	}

	if err := db.DB.Update("poem", map[string]string{"title": originalTitle}, req); err != nil {
		writeError(w, 500, "db error")
		return
	}
	writeJSON(w, 200, map[string]bool{"success": true})
}

// DELETE /api/poems/{title} (admin)
func DeletePoem(w http.ResponseWriter, r *http.Request) {
	title := chi.URLParam(r, "title")

	var existing models.Poem
	_ = db.DB.SelectOne("poem", map[string]string{"title": title}, &existing)
	if existing.Title == "" {
		writeError(w, 404, "стих не найден")
		return
	}

	if err := db.DB.Delete("poem", map[string]string{"title": title}); err != nil {
		writeError(w, 500, "db error")
		return
	}
	writeJSON(w, 200, map[string]bool{"success": true})
}
