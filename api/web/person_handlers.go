package web

import (
	"net/http"

	"github.com/masudur-rahman/kazi-ancestry/models"
	"github.com/masudur-rahman/kazi-ancestry/services/all"

	"github.com/go-chi/chi/v5"
)

// personRequest is the editable shape accepted from the client (tags as array).
type personRequest struct {
	ID       string   `json:"id"`
	ParentID *string  `json:"parentId"`
	Position int      `json:"position"`
	Name     string   `json:"name"`
	Origin   string   `json:"origin"`
	Alias    string   `json:"alias"`
	Spouse   string   `json:"spouse"`
	Birth    string   `json:"birth"`
	Death    string   `json:"death"`
	Note     string   `json:"note"`
	Tags     []string `json:"tags"`
}

func (pr personRequest) toModel() models.Person {
	p := models.Person{
		ID: pr.ID, ParentID: pr.ParentID, Position: pr.Position, Name: pr.Name,
		Origin: pr.Origin, Alias: pr.Alias, Spouse: pr.Spouse,
		Birth: pr.Birth, Death: pr.Death, Note: pr.Note,
	}
	p.SetTags(pr.Tags)
	return p
}

// HandleCreatePerson adds a person, server-assigning a slug id when absent.
func HandleCreatePerson(w http.ResponseWriter, r *http.Request) {
	var req personRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Name == "" {
		WriteError(w, http.StatusBadRequest, "bad_request", "name is required")
		return
	}
	p := req.toModel()
	p.ID = "" // force server-side slug generation
	if err := all.GetServices().Person.Create(&p); err != nil {
		WriteServiceError(w, "create_error", err)
		return
	}
	WriteJSON(w, http.StatusCreated, p)
}

// HandleUpdatePerson edits an existing person.
func HandleUpdatePerson(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req personRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	p := req.toModel()
	p.ID = id
	if err := all.GetServices().Person.Update(id, &p); err != nil {
		WriteServiceError(w, "update_error", err)
		return
	}
	WriteJSON(w, http.StatusOK, p)
}

// HandleDeletePerson removes a person.
func HandleDeletePerson(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := all.GetServices().Person.Delete(id); err != nil {
		WriteServiceError(w, "delete_error", err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"id": id})
}

// reorderRequest is the new sibling order under a common parent.
type reorderRequest struct {
	ParentID string   `json:"parentId"`
	Order    []string `json:"order"`
}

// HandleReorderPerson rewrites the sibling order under a parent (admin-only).
func HandleReorderPerson(w http.ResponseWriter, r *http.Request) {
	var req reorderRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.ParentID == "" || len(req.Order) == 0 {
		WriteError(w, http.StatusBadRequest, "bad_request", "parentId and order are required")
		return
	}
	if err := all.GetServices().Person.Reorder(req.ParentID, req.Order); err != nil {
		WriteServiceError(w, "reorder_error", err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"parentId": req.ParentID})
}
