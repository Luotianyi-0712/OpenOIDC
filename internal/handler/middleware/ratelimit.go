package middleware

import (
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/anthropic/oidc-platform/internal/port"
)

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := indexByte(xff, ','); i > 0 {
			return trimSpace(xff[:i])
		}
		return trimSpace(xff)
	}
	if rip := r.Header.Get("X-Real-IP"); rip != "" {
		return rip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// RateLimit uses fixed parameters.
func RateLimit(cache port.Cache, requests int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			key := "ratelimit:" + r.URL.Path + ":" + ip
			n, err := cache.IncrementRateLimit(r.Context(), key, window)
			if err == nil && int(n) > requests {
				w.Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"rate_limited","message":"too many requests"}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// DynamicRateLimit reads rate limit parameters from settings at runtime.
// Falls back to defaultRequests/defaultWindow if settings are not configured.
func DynamicRateLimit(cache port.Cache, settingsRepo port.SettingsRepository, defaultRequests int, defaultWindow time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests := defaultRequests
			window := defaultWindow

			if settingsRepo != nil {
				if v, err := settingsRepo.Get(r.Context(), "rate_limit_requests"); err == nil && v.Value != "" {
					if n, err := strconv.Atoi(v.Value); err == nil && n > 0 {
						requests = n
					}
				}
				if v, err := settingsRepo.Get(r.Context(), "rate_limit_window_seconds"); err == nil && v.Value != "" {
					if n, err := strconv.Atoi(v.Value); err == nil && n > 0 {
						window = time.Duration(n) * time.Second
					}
				}
			}

			ip := clientIP(r)
			key := "ratelimit:" + r.URL.Path + ":" + ip
			n, err := cache.IncrementRateLimit(r.Context(), key, window)
			if err == nil && int(n) > requests {
				w.Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"rate_limited","message":"too many requests"}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
