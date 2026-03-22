package handlers

import (
	"net/http"
	"net/url"

	"github.com/ssback/internal/db"
)

// GET /api/config — публичный endpoint, без авторизации
func GetAppConfig(w http.ResponseWriter, r *http.Request) {
	q := url.Values{}
	q.Set("select", "*")

	var rows []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := db.DB.SelectWithQuery("app_config", q, &rows); err != nil {
		writeError(w, 500, "failed to load config")
		return
	}

	cfg := make(map[string]string, len(rows))
	for _, row := range rows {
		cfg[row.Key] = row.Value
	}

	writeJSON(w, 200, cfg)
}
