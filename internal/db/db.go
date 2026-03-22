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

func (c *Client) Select(table string, filters map[string]string, result interface{}) error {
	q := url.Values{}
	q.Set("select", "*")
	for k, v := range filters {
		q.Set(k, "eq."+v)
	}
	return c.do("GET", "/"+table, q, nil, result, "")
}

// SelectWithQuery — позволяет передать произвольные query-параметры (gte, lte, in и т.д.)
func (c *Client) SelectWithQuery(table string, q url.Values, result interface{}) error {
	if q == nil {
		q = url.Values{}
	}
	if q.Get("select") == "" {
		q.Set("select", "*")
	}
	return c.do("GET", "/"+table, q, nil, result, "")
}

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
		return nil
	}
	return json.Unmarshal(arr[0], result)
}

func (c *Client) Insert(table string, data interface{}, result interface{}) error {
	prefer := ""
	if result != nil {
		prefer = "return=representation"
	}
	return c.do("POST", "/"+table, nil, data, result, prefer)
}

func (c *Client) Update(table string, filters map[string]string, data interface{}) error {
	q := url.Values{}
	for k, v := range filters {
		q.Set(k, "eq."+v)
	}
	return c.do("PATCH", "/"+table, q, data, nil, "return=minimal")
}

// UpdateAtomic — оптимистичная блокировка для AI-счётчика.
func (c *Client) UpdateAtomic(table string, filters map[string]string, expectedUsageToday string, data interface{}) error {
	q := url.Values{}
	for k, v := range filters {
		q.Set(k, "eq."+v)
	}
	q.Set("usage_today", "eq."+expectedUsageToday)

	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", c.baseURL+"/"+table+"?"+q.Encode(), bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal,count=exact")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase error %d: %s", resp.StatusCode, string(body))
	}

	contentRange := resp.Header.Get("Content-Range")
	if contentRange == "*/0" || contentRange == "" {
		return fmt.Errorf("atomic update: row was modified concurrently")
	}
	return nil
}

func (c *Client) Delete(table string, filters map[string]string) error {
	q := url.Values{}
	for k, v := range filters {
		q.Set(k, "eq."+v)
	}
	return c.do("DELETE", "/"+table, q, nil, nil, "")
}

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

// DeleteOldChatHistory удаляет сообщения чата сверх keepCount на пользователя.
func (c *Client) DeleteOldChatHistory(username string, keepCount int) error {
	q := url.Values{}
	q.Set("select", "id")
	q.Set("username", "eq."+username)
	q.Set("order", "created_at.desc")
	q.Set("offset", fmt.Sprintf("%d", keepCount))

	var rows []struct {
		ID int64 `json:"id"`
	}
	if err := c.do("GET", "/ai_chat_history", q, nil, &rows, ""); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = fmt.Sprintf("%d", row.ID)
	}

	dq := url.Values{}
	dq.Set("id", "in.("+joinCSV(ids)+")")
	return c.do("DELETE", "/ai_chat_history", dq, nil, nil, "")
}

func joinCSV(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}
