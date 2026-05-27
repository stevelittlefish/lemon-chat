package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

func (s *Server) handleListPersonas(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	personas, err := s.store.ListPersonas(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if personas == nil {
		personas = []store.Persona{}
	}
	writeJSON(w, http.StatusOK, personas)
}

func (s *Server) handleCreatePersona(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var req struct {
		Name         string  `json:"name"`
		Description  *string `json:"description"`
		SystemPrompt string  `json:"system_prompt"`
		IsGlobal     bool    `json:"is_global"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Name == "" || req.SystemPrompt == "" {
		writeError(w, http.StatusBadRequest, "name and system_prompt required")
		return
	}
	// Only admins can create global personas.
	if req.IsGlobal && !user.IsAdmin {
		req.IsGlobal = false
	}
	persona, err := s.store.CreatePersona(req.Name, req.Description, req.SystemPrompt, user.ID, req.IsGlobal)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, persona)
}

func (s *Server) handleUpdatePersona(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := s.store.GetPersona(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	// Only the creator or an admin can edit.
	if (existing.CreatedBy == nil || *existing.CreatedBy != user.ID) && !user.IsAdmin {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req struct {
		Name         string  `json:"name"`
		Description  *string `json:"description"`
		SystemPrompt string  `json:"system_prompt"`
		IsGlobal     bool    `json:"is_global"`
	}
	// Seed with existing values so partial updates work.
	req.Name = existing.Name
	req.Description = existing.Description
	req.SystemPrompt = existing.SystemPrompt
	req.IsGlobal = existing.IsGlobal

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.IsGlobal && !user.IsAdmin {
		req.IsGlobal = false
	}
	if err := s.store.UpdatePersona(id, req.Name, req.Description, req.SystemPrompt, req.IsGlobal); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeletePersona(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := s.store.GetPersona(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if (existing.CreatedBy == nil || *existing.CreatedBy != user.ID) && !user.IsAdmin {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := s.store.DeletePersona(id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
