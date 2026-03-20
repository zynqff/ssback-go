package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type visitor struct {
	count    int
	lastSeen time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int
	window   time.Duration
}

// NewRateLimiter создаёт лимитер: не более limit запросов за window на один IP
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
	}
	// Фоновая очистка устаревших записей каждые 5 минут
	go func() {
		for range time.Tick(5 * time.Minute) {
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.lastSeen) > rl.window {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := realIP(r)

		rl.mu.Lock()
		v, ok := rl.visitors[ip]
		if !ok || time.Since(v.lastSeen) > rl.window {
			rl.visitors[ip] = &visitor{count: 1, lastSeen: time.Now()}
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}
		v.lastSeen = time.Now()
		v.count++
		count := v.count
		rl.mu.Unlock()

		if count > rl.limit {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w,
				`{"error":"слишком много запросов, попробуйте позже"}`,
				http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// realIP достаёт реальный IP с учётом Render.com proxy-заголовков
func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For может содержать цепочку: "client, proxy1, proxy2"
		return strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
