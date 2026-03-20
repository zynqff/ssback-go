package db

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ssback/internal/config"
)

// Client is a minimal Supabase REST client
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

var DB *Client

func Init() {
	DB = &Client{
		baseURL: config.C.SupabaseURL + "/rest/v1",
		apiKey:  config.C.SupabaseKey,
		http:    &http.Client{},
	}
}

func (c *Client) do(method, path string, query url.Values, body interface{}, result interface{}, prefer string) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequest(method, u, bodyReader)
	if err != nil {
		return err
	}

	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if prefer != "" {
		req.Header.Set("Prefer", prefer)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase error %d: %s", resp.StatusCode, string(b))
	}

	if result != nil && resp.StatusCode != 204 {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// Select returns rows from table with optional eq filters
func (c *Client) Select(table string, filters map[string]string, result interface{}) error {
	q := url.Values{}
	q.Set("select", "*")
	for k, v := range filters {
		q.Set(k, "eq."+v)
	}
	return c.do("GET", "/"+table, q, nil, result, "")
}

// SelectOne returns a single row or nil
func (c *Client) SelectOne(table string, filters map[string]string, result interface{}) error {
	q := url.Values{}
	q.Set("select", "*")
	for k, v := range filters {
		q.Set(k, "eq."+v)
	}
	q.Set("limit", "1")

	var rows json.RawMessage
	err := c.do("GET", "/"+table, q, nil, &rows, "")
	if err != nil {
		return err
	}

	var arr []json.RawMessage
	if err := json.Unmarshal(rows, &arr); err != nil {
		return err
	}
	if len(arr) == 0 {
		return nil // not found — caller checks
	}
	return json.Unmarshal(arr[0], result)
}

// Insert inserts a row and optionally returns inserted data
func (c *Client) Insert(table string, data interface{}, result interface{}) error {
	prefer := ""
	if result != nil {
		prefer = "return=representation"
	}
	return c.do("POST", "/"+table, nil, data, result, prefer)
}

// Update updates rows matching filters
func (c *Client) Update(table string, filters map[string]string, data interface{}) error {
	q := url.Values{}
	for k, v := range filters {
		q.Set(k, "eq."+v)
	}
	return c.do("PATCH", "/"+table, q, data, nil, "return=minimal")
}

// UpdateAtomic обновляет строку только если usage_today совпадает с ожидаемым.
// Это оптимистичная блокировка против race condition при параллельных запросах AI.
// Если кто-то успел изменить счётчик раньше — Supabase не найдёт строку и вернёт ошибку.
func (c *Client) UpdateAtomic(table string, filters map[string]string, expectedUsageToday string, data interface{}) error {
	q := url.Values{}
	for k, v := range filters {
		q.Set(k, "eq."+v)
	}
	// Доп. условие: обновляем только если usage_today всё ещё равен ожидаемому
	q.Set("usage_today", "eq."+expectedUsageToday)

	// Supabase с Prefer: return=minimal + count=exact вернёт 0 строк если условие не совпало
	req, err := http.NewRequest("PATCH", c.baseURL+"/"+table+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}

	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req2, err := http.NewRequest("PATCH", c.baseURL+"/"+table+"?"+q.Encode(), bytes.NewReader(b))
	if err != nil {
		return err
	}
	req2.Header.Set("apikey", c.apiKey)
	req2.Header.Set("Authorization", "Bearer "+c.apiKey)
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Prefer", "return=minimal,count=exact")
	_ = req

	resp, err := c.http.Do(req2)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase error %d: %s", resp.StatusCode, string(body))
	}

	// Проверяем Content-Range: если 0 строк обновлено — кто-то опередил нас
	contentRange := resp.Header.Get("Content-Range")
	if contentRange == "*/0" || contentRange == "" {
		return fmt.Errorf("atomic update: row was modified concurrently")
	}
	return nil
}

// Delete deletes rows matching filters
func (c *Client) Delete(table string, filters map[string]string) error {
	q := url.Values{}
	for k, v := range filters {
		q.Set(k, "eq."+v)
	}
	return c.do("DELETE", "/"+table, q, nil, nil, "")
}

// SelectOrdered returns rows ordered by a column
func (c *Client) SelectOrdered(table string, filters map[string]string, orderBy string, desc bool, limit int, result interface{}) error {
	q := url.Values{}
	q.Set("select", "*")
	for k, v := range filters {
		q.Set(k, "eq."+v)
	}
	dir := "asc"
	if desc {
		dir = "desc"
	}
	q.Set("order", orderBy+"."+dir)
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	return c.do("GET", "/"+table, q, nil, result, "")
}
