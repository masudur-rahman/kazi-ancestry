package web

import (
	"encoding/json"
	"net/http"

	"github.com/masudur-rahman/kazi-ancestry/infra/logr"
	"github.com/masudur-rahman/kazi-ancestry/models"
)

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON sends a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// WriteError sends a JSON error response and logs it.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	logr.DefaultLogger.Errorw("web_error", "status", status, "code", code, "message", message)
	WriteJSON(w, status, errorResponse{Code: code, Message: message})
}

// WriteServiceError maps a service error to its HTTP status (via StatusError).
func WriteServiceError(w http.ResponseWriter, code string, err error) {
	status, msg := models.ParseStatusError(err)
	WriteError(w, status, code, msg)
}

// ReadJSON decodes a JSON request body into target.
func ReadJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}
