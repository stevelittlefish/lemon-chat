package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

func (s *Server) handleListCharacters(w http.ResponseWriter, r *http.Request) {
	chars, err := s.store.ListCharacters()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if chars == nil {
		chars = []store.Character{}
	}
	writeJSON(w, http.StatusOK, chars)
}

func (s *Server) handleCreateCharacter(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var req struct {
		Name         string  `json:"name"`
		Model        string  `json:"model"`
		SystemPrompt *string `json:"system_prompt"`
		FirstMessage *string `json:"first_message"`
		AllowEditing bool    `json:"allow_editing"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Name == "" || req.Model == "" {
		writeError(w, http.StatusBadRequest, "name and model are required")
		return
	}
	char, err := s.store.CreateCharacter(req.Name, req.Model, req.SystemPrompt, req.FirstMessage, user.ID, req.AllowEditing)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, char)
}

func (s *Server) handleUpdateCharacter(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := s.store.GetCharacter(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	// Owner, admin, or allow_editing may edit.
	canEdit := existing.CreatedBy == user.ID || user.IsAdmin || existing.AllowEditing
	if !canEdit {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	// Seed with existing values so partial updates work.
	req := struct {
		Name         string  `json:"name"`
		Model        string  `json:"model"`
		SystemPrompt *string `json:"system_prompt"`
		FirstMessage *string `json:"first_message"`
		AllowEditing bool    `json:"allow_editing"`
	}{
		Name:         existing.Name,
		Model:        existing.Model,
		SystemPrompt: existing.SystemPrompt,
		FirstMessage: existing.FirstMessage,
		AllowEditing: existing.AllowEditing,
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Name == "" || req.Model == "" {
		writeError(w, http.StatusBadRequest, "name and model are required")
		return
	}
	// Only owner or admin may toggle allow_editing.
	if existing.CreatedBy != user.ID && !user.IsAdmin {
		req.AllowEditing = existing.AllowEditing
	}
	if err := s.store.UpdateCharacter(id, req.Name, req.Model, req.SystemPrompt, req.FirstMessage, req.AllowEditing); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteCharacter(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := s.store.GetCharacter(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	// Only owner or admin may delete.
	if existing.CreatedBy != user.ID && !user.IsAdmin {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := s.store.DeleteCharacter(id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
