package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

type characterWithMessages struct {
	store.Character
	HiddenMessages []store.CharacterHiddenMessage `json:"hidden_messages"`
}

func (s *Server) handleListCharacters(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	chars, err := s.store.ListCharacters(user.ID, user.IsAdmin)
	if err != nil {
		internalError(w, err)
		return
	}
	if chars == nil {
		chars = []store.Character{}
	}
	writeJSON(w, http.StatusOK, chars)
}

func (s *Server) handleGetCharacter(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	char, err := s.store.GetCharacter(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	if char.Visibility == "private" && char.CreatedBy != user.ID && !user.IsAdmin {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	hiddenMsgs, err := s.store.ListCharacterHiddenMessages(id)
	if err != nil {
		internalError(w, err)
		return
	}
	if hiddenMsgs == nil {
		hiddenMsgs = []store.CharacterHiddenMessage{}
	}
	writeJSON(w, http.StatusOK, characterWithMessages{Character: *char, HiddenMessages: hiddenMsgs})
}

func (s *Server) handleCreateCharacter(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var req struct {
		Name           string                         `json:"name"`
		Model          string                         `json:"model"`
		SystemPrompt   *string                        `json:"system_prompt"`
		FirstMessage   *string                        `json:"first_message"`
		TitlePrompt    *string                        `json:"title_prompt"`
		Visibility     string                         `json:"visibility"`
		AutoTitle      bool                           `json:"auto_title"`
		HiddenMessages []store.CharacterHiddenMessage `json:"hidden_messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Name == "" || req.Model == "" {
		writeError(w, http.StatusBadRequest, "name and model are required")
		return
	}
	if !validVisibility(req.Visibility) {
		req.Visibility = "private"
	}
	char, err := s.store.CreateCharacter(req.Name, req.Model, req.SystemPrompt, req.FirstMessage, req.TitlePrompt, user.ID, req.Visibility, req.AutoTitle)
	if err != nil {
		internalError(w, err)
		return
	}
	if len(req.HiddenMessages) > 0 {
		if err := s.store.ReplaceCharacterHiddenMessages(char.ID, req.HiddenMessages); err != nil {
			internalError(w, err)
			return
		}
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
		internalError(w, err)
		return
	}

	isOwnerOrAdmin := existing.CreatedBy == user.ID || user.IsAdmin
	canEdit := isOwnerOrAdmin || existing.Visibility == "readwrite"
	if !canEdit {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	// Seed with existing values so partial updates work.
	req := struct {
		Name           string                          `json:"name"`
		Model          string                          `json:"model"`
		SystemPrompt   *string                         `json:"system_prompt"`
		FirstMessage   *string                         `json:"first_message"`
		TitlePrompt    *string                         `json:"title_prompt"`
		Visibility     string                          `json:"visibility"`
		AutoTitle      bool                            `json:"auto_title"`
		HiddenMessages *[]store.CharacterHiddenMessage `json:"hidden_messages"`
	}{
		Name:         existing.Name,
		Model:        existing.Model,
		SystemPrompt: existing.SystemPrompt,
		FirstMessage: existing.FirstMessage,
		TitlePrompt:  existing.TitlePrompt,
		Visibility:   existing.Visibility,
		AutoTitle:    existing.AutoTitle,
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Name == "" || req.Model == "" {
		writeError(w, http.StatusBadRequest, "name and model are required")
		return
	}
	// Only owner or admin may change visibility.
	if !isOwnerOrAdmin {
		req.Visibility = existing.Visibility
	} else if !validVisibility(req.Visibility) {
		req.Visibility = existing.Visibility
	}
	if err := s.store.UpdateCharacter(id, req.Name, req.Model, req.SystemPrompt, req.FirstMessage, req.TitlePrompt, req.Visibility, req.AutoTitle); err != nil {
		internalError(w, err)
		return
	}
	if req.HiddenMessages != nil {
		if err := s.store.ReplaceCharacterHiddenMessages(id, *req.HiddenMessages); err != nil {
			internalError(w, err)
			return
		}
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
		internalError(w, err)
		return
	}
	// Only owner or admin may delete.
	if existing.CreatedBy != user.ID && !user.IsAdmin {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if existing.AvatarFilename != nil {
		s.deleteAvatarFile(*existing.AvatarFilename)
	}
	if err := s.store.DeleteCharacter(id); err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleServeCharacterAvatar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	char, err := s.store.GetCharacter(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	if char.AvatarFilename == nil {
		writeError(w, http.StatusNotFound, "no avatar")
		return
	}
	s.serveAvatarFile(w, r, *char.AvatarFilename)
}

func (s *Server) handleUploadCharacterAvatar(w http.ResponseWriter, r *http.Request) {
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
		internalError(w, err)
		return
	}
	if existing.CreatedBy != user.ID && !user.IsAdmin && existing.Visibility != "readwrite" {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	data, ext, err := receiveAvatar(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	prefix := fmt.Sprintf("character-%d", id)
	filename, err := s.writeAvatar(prefix, data, ext)
	if err != nil {
		internalError(w, err)
		return
	}
	if err := s.store.SetCharacterAvatar(id, filename); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"has_avatar": true})
}

func (s *Server) handleDeleteCharacterAvatar(w http.ResponseWriter, r *http.Request) {
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
		internalError(w, err)
		return
	}
	if existing.CreatedBy != user.ID && !user.IsAdmin {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if existing.AvatarFilename != nil {
		s.deleteAvatarFile(*existing.AvatarFilename)
	}
	if err := s.store.ClearCharacterAvatar(id); err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validVisibility(v string) bool {
	return v == "private" || v == "readonly" || v == "readwrite"
}
