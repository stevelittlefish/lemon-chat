package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

func (s *Server) handleListCompletions(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	items, err := s.store.ListCompletions(user.ID)
	if err != nil {
		internalError(w, err)
		return
	}
	if items == nil {
		items = []store.Completion{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateCompletion(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var req struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Model == "" {
		writeError(w, http.StatusBadRequest, "model is required")
		return
	}
	item, err := s.store.CreateCompletion(user.ID, req.Model)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleUpdateCompletion(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Title *string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Title == nil {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if err := s.store.UpdateCompletionTitle(id, user.ID, *req.Title); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRegenerateCompletionTitle(w http.ResponseWriter, r *http.Request) {
	// TODO: implement background title generation once completions have content
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleDeleteCompletion(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.store.DeleteCompletion(id, user.ID); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
