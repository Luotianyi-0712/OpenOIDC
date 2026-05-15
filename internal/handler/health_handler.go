package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: rdb}
}

func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	Raw(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := map[string]any{}
	overall := true

	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			checks["database"] = map[string]any{"ok": false, "error": err.Error()}
			overall = false
		} else {
			checks["database"] = map[string]any{"ok": true}
		}
	}

	if h.redis != nil {
		if err := h.redis.Ping(ctx).Err(); err != nil {
			checks["redis"] = map[string]any{"ok": false, "error": err.Error()}
			overall = false
		} else {
			checks["redis"] = map[string]any{"ok": true}
		}
	}

	status := http.StatusOK
	if !overall {
		status = http.StatusServiceUnavailable
	}
	Raw(w, status, map[string]any{"status": status, "checks": checks})
}
