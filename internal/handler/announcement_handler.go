package handler

import (
	"net/http"

	"github.com/anthropic/oidc-platform/internal/handler/middleware"
	"github.com/anthropic/oidc-platform/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AnnouncementHandler struct{ service *service.AnnouncementService }

func NewAnnouncementHandler(s *service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{service: s}
}

func (h *AnnouncementHandler) PublicList(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, true)
}

func (h *AnnouncementHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, false)
}

func (h *AnnouncementHandler) list(w http.ResponseWriter, r *http.Request, publishedOnly bool) {
	w.Header().Set("Cache-Control", "no-store")
	items, err := h.service.List(r.Context(), publishedOnly)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, items)
}

func (h *AnnouncementHandler) Create(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, uuid.Nil)
}

func (h *AnnouncementHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil || id == uuid.Nil {
		Error(w, http.StatusBadRequest, "invalid_id", "invalid announcement ID")
		return
	}
	h.save(w, r, id)
}

func (h *AnnouncementHandler) save(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	adminID, err := middleware.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 256*1024)
	var input service.AnnouncementInput
	if err := DecodeJSON(r, &input); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	item, err := h.service.Save(r.Context(), id, adminID, input)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	status := http.StatusOK
	if id == uuid.Nil {
		status = http.StatusCreated
	}
	JSON(w, status, item)
}

func (h *AnnouncementHandler) Delete(w http.ResponseWriter, r *http.Request) {
	adminID, err := middleware.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil || id == uuid.Nil {
		Error(w, http.StatusBadRequest, "invalid_id", "invalid announcement ID")
		return
	}
	if err := h.service.Delete(r.Context(), id, adminID); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
