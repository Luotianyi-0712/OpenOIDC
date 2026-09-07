package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anthropic/oidc-platform/internal/adapter/sqlite"
	"github.com/anthropic/oidc-platform/internal/config"
	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/handler"
	"github.com/anthropic/oidc-platform/internal/router"
	"github.com/anthropic/oidc-platform/internal/service"
	"github.com/google/uuid"
)

func announcementTestRouter(t *testing.T) http.Handler {
	t.Helper()
	ctx := context.Background()
	db, err := sqlite.NewDB(ctx, filepath.Join(t.TempDir(), "announcements.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for i := 0; i < 2; i++ {
		if err := sqlite.RunMigrations(db); err != nil {
			t.Fatal(err)
		}
	}
	users, sessions := sqlite.NewUserRepo(db), sqlite.NewSessionRepo(db)
	for _, role := range []string{domain.RoleAdmin, domain.RoleUser} {
		user := &domain.User{ID: uuid.New(), Email: string(role) + "@example.test", Role: role, Status: domain.UserStatusActive, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
		if err := users.Create(ctx, user); err != nil {
			t.Fatal(err)
		}
		if err := sessions.Create(ctx, &domain.UserSession{ID: uuid.New(), UserID: user.ID, SessionToken: string(role), ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	return router.NewRouter(router.Deps{
		AnnouncementHandler: handler.NewAnnouncementHandler(service.NewAnnouncementService(sqlite.NewAnnouncementRepo(db), sqlite.NewAuditRepo(db))),
		SessionService:      service.NewSessionService(sessions, users, &config.Config{}), UserRepo: users,
	})
}

func announcementRequest(t *testing.T, router http.Handler, method, path, token string, body any, status int) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(string(data)))
	if token != "" {
		r.AddCookie(&http.Cookie{Name: "oidc_session", Value: token})
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != status {
		t.Fatalf("%s %s: got %d, want %d: %s", method, path, w.Code, status, w.Body.String())
	}
	return w
}

func TestAnnouncementCRUDAndDraftIsolation(t *testing.T) {
	r := announcementTestRouter(t)
	input := service.AnnouncementInput{Title: " Release notes ", Content: "# Update\n\n**Markdown** and [links](https://example.test)", DisplayMode: "both", Dismissible: true, Scrolling: true}
	created := announcementRequest(t, r, "POST", "/admin/announcements", "admin", input, http.StatusCreated)
	var result struct {
		Data domain.Announcement `json:"data"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	a := result.Data
	if a.ID == uuid.Nil || a.Revision == uuid.Nil || a.Title != "Release notes" || !a.Dismissible || !a.Scrolling {
		t.Fatalf("unexpected announcement: %+v", a)
	}
	public := announcementRequest(t, r, "GET", "/announcements", "", nil, http.StatusOK)
	if strings.Contains(public.Body.String(), "Release notes") || !strings.Contains(public.Body.String(), `"data":[]`) {
		t.Fatal("draft leaked or list was not an array")
	}
	if public.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("public response can retain withdrawn announcements")
	}
	admin := announcementRequest(t, r, "GET", "/admin/announcements", "admin", nil, http.StatusOK)
	if !strings.Contains(admin.Body.String(), "Release notes") {
		t.Fatal("admin cannot see draft")
	}
	input.IsPublished = true
	updated := announcementRequest(t, r, "PUT", "/admin/announcements/"+a.ID.String(), "admin", input, http.StatusOK)
	if err := json.Unmarshal(updated.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Data.Revision == a.Revision || !result.Data.CreatedAt.Equal(a.CreatedAt) {
		t.Fatal("update did not rotate revision or preserve creation time")
	}
	public = announcementRequest(t, r, "GET", "/announcements", "", nil, http.StatusOK)
	if !strings.Contains(public.Body.String(), "Release notes") {
		t.Fatal("published announcement missing")
	}
	input.IsPublished = false
	announcementRequest(t, r, "PUT", "/admin/announcements/"+a.ID.String(), "admin", input, http.StatusOK)
	public = announcementRequest(t, r, "GET", "/announcements", "", nil, http.StatusOK)
	if strings.Contains(public.Body.String(), "Release notes") {
		t.Fatal("withdrawn announcement leaked")
	}
	announcementRequest(t, r, "DELETE", "/admin/announcements/"+a.ID.String(), "admin", nil, http.StatusOK)
	announcementRequest(t, r, "DELETE", "/admin/announcements/"+a.ID.String(), "admin", nil, http.StatusNotFound)
	announcementRequest(t, r, "PUT", "/admin/announcements/"+a.ID.String(), "admin", input, http.StatusNotFound)
}

func TestAnnouncementPermissionsAndValidation(t *testing.T) {
	r := announcementTestRouter(t)
	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		path := "/admin/announcements"
		if method == "PUT" || method == "DELETE" {
			path += "/" + uuid.NewString()
		}
		announcementRequest(t, r, method, path, "", nil, http.StatusUnauthorized)
		announcementRequest(t, r, method, path, "user", nil, http.StatusForbidden)
	}
	for _, input := range []service.AnnouncementInput{
		{Title: " ", Content: "body", DisplayMode: "both"},
		{Title: strings.Repeat("x", 161), Content: "body", DisplayMode: "both"},
		{Title: "title", Content: " ", DisplayMode: "both"},
		{Title: "title", Content: strings.Repeat("x", 65537), DisplayMode: "both"},
		{Title: "title", Content: "body", DisplayMode: "invalid"},
	} {
		announcementRequest(t, r, "POST", "/admin/announcements", "admin", input, http.StatusBadRequest)
	}
	announcementRequest(t, r, "POST", "/admin/announcements", "admin", map[string]any{"title": "title", "content": "body", "display_mode": "both", "id": uuid.NewString()}, http.StatusBadRequest)
	announcementRequest(t, r, "PUT", "/admin/announcements/not-an-id", "admin", nil, http.StatusBadRequest)
	announcementRequest(t, r, "DELETE", "/admin/announcements/"+uuid.Nil.String(), "admin", nil, http.StatusBadRequest)
	for _, mode := range []string{"banner", "modal", "both"} {
		announcementRequest(t, r, "POST", "/admin/announcements", "admin", service.AnnouncementInput{Title: mode, Content: "body", DisplayMode: mode}, http.StatusCreated)
	}
}
