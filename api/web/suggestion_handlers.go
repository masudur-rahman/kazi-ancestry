package web

import (
	"net/http"

	"github.com/masudur-rahman/kazi-ancestry/models"
	"github.com/masudur-rahman/kazi-ancestry/services/all"

	"github.com/go-chi/chi/v5"
)

// HandleSubmitSuggestion records a proposed edit for admin review.
func HandleSubmitSuggestion(w http.ResponseWriter, r *http.Request) {
	var sug models.Suggestion
	if err := ReadJSON(r, &sug); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if sug.PersonID == "" || sug.Payload == "" {
		WriteError(w, http.StatusBadRequest, "bad_request", "personId and payload are required")
		return
	}
	if u := userFromContext(r.Context()); u != nil {
		sug.SubmittedBy = u.Email
	}
	if err := all.GetServices().Suggestion.Submit(&sug); err != nil {
		WriteServiceError(w, "submit_error", err)
		return
	}
	WriteJSON(w, http.StatusCreated, sug)
}

// HandleListSuggestions returns the review inbox (admin).
func HandleListSuggestions(w http.ResponseWriter, r *http.Request) {
	list, err := all.GetServices().Suggestion.List()
	if err != nil {
		WriteServiceError(w, "list_error", err)
		return
	}
	WriteJSON(w, http.StatusOK, list)
}

// HandleApproveSuggestion / HandleRejectSuggestion update a suggestion's status (admin).
func HandleApproveSuggestion(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := all.GetServices().Suggestion.Approve(id); err != nil {
		WriteServiceError(w, "approve_error", err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"id": id, "status": "approved"})
}

func HandleRejectSuggestion(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := all.GetServices().Suggestion.Reject(id); err != nil {
		WriteServiceError(w, "reject_error", err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"id": id, "status": "rejected"})
}
