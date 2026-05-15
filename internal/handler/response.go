package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type APIResponse struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
	Meta    *APIMeta  `json:"meta,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIMeta struct {
	Total  int64 `json:"total,omitempty"`
	Offset int   `json:"offset,omitempty"`
	Limit  int   `json:"limit,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	resp := APIResponse{Success: status >= 200 && status < 300, Data: data}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("encode json response", "error", err)
	}
}

func Raw(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("encode raw response", "error", err)
	}
}

func Error(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	resp := APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("encode error response", "error", err)
	}
}

func PaginatedJSON(w http.ResponseWriter, status int, data any, total int64, offset, limit int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	resp := APIResponse{
		Success: true,
		Data:    data,
		Meta:    &APIMeta{Total: total, Offset: offset, Limit: limit},
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("encode paginated response", "error", err)
	}
}

func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
