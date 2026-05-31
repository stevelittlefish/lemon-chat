package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/stevelittlefish/lemon-chat/internal/store"
	"github.com/stevelittlefish/lemon-chat/internal/tasks"
)

func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	convs, err := s.store.ListConversations(user.ID)
	if err != nil {
		internalError(w, err)
		return
	}
	if convs == nil {
		convs = []store.Conversation{}
	}
	writeJSON(w, http.StatusOK, convs)
}

func (s *Server) handleCreateConversation(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var req struct {
		Title       *string `json:"title"`
		Model       *string `json:"model"`
		CharacterID *int64  `json:"character_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if (req.Model == nil) == (req.CharacterID == nil) {
		writeError(w, http.StatusBadRequest, "exactly one of model or character_id is required")
		return
	}
	conv, err := s.store.CreateConversation(user.ID, req.Title, req.Model, req.CharacterID)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, conv)
}

func (s *Server) handleRegenerateTitle(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := s.store.GetConversation(id, user.ID); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		internalError(w, err)
		return
	}
	tasks.GenerateTitleForConversation(s.store, s.cfg, id, func(convID int64, title string) {
		s.hub.BroadcastTitleUpdate(convID, title)
	})
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleForkConversation(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		MessageID int64 `json:"message_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MessageID == 0 {
		writeError(w, http.StatusBadRequest, "message_id required")
		return
	}
	conv, err := s.store.ForkConversation(id, user.ID, req.MessageID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, conv)
}

func (s *Server) handleDeleteConversation(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.store.DeleteConversation(id, user.ID); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
