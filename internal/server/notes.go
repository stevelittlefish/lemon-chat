package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

// checkNoteAccess verifies the note is visible to user. Admins bypass the
// check. Returns false and writes the error response if access is denied.
func (s *Server) checkNoteAccess(w http.ResponseWriter, noteID int64, user *store.User) bool {
	if user.IsAdmin {
		return true
	}
	visible, err := s.store.IsNoteVisibleToUser(noteID, user.ID)
	if err != nil {
		internalError(w, err)
		return false
	}
	if !visible {
		writeError(w, http.StatusNotFound, "not found")
		return false
	}
	return true
}

func (s *Server) handleListNotes(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	prefix := r.URL.Query().Get("prefix")
	notes, err := s.store.ListUserVisibleNotes(prefix, user.ID)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, notes)
}

func (s *Server) handleGetNote(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	n, err := s.store.GetNoteByID(id)
	if notFoundOr500(w, err) {
		return
	}
	if !s.checkNoteAccess(w, n.ID, user) {
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *Server) handleUpsertNote(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var req struct {
		Key      string `json:"key"`
		Value    string `json:"value"`
		ReadOnly *bool  `json:"read_only"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}
	if err := store.ValidateNoteKey(req.Key); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Derive scope FKs from the key prefix.
	var userID *int64
	var convID *int64
	switch req.Key[0] {
	case 'u':
		userID = &user.ID
	case 'c':
		writeError(w, http.StatusBadRequest, "conversation-scoped notes cannot be created from the settings UI")
		return
	}

	log.Printf("Saving note key=%q user_id=%d read_only=%v", req.Key, user.ID, req.ReadOnly)
	n, err := s.store.UpsertNoteAdmin(req.Key, req.Value, req.ReadOnly, userID, convID)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *Server) handleDeleteNote(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	n, err := s.store.GetNoteByID(id)
	if notFoundOr500(w, err) {
		return
	}
	if !s.checkNoteAccess(w, n.ID, user) {
		return
	}
	if n.ReadOnly {
		writeError(w, http.StatusForbidden, "note is read-only")
		return
	}

	log.Printf("Deleting note id=%d key=%q user_id=%d", id, n.Key, user.ID)
	if err := s.store.DeleteNoteByID(id); err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSetNoteReadOnly(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var req struct {
		ReadOnly bool `json:"read_only"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	n, err := s.store.GetNoteByID(id)
	if notFoundOr500(w, err) {
		return
	}
	if !s.checkNoteAccess(w, n.ID, user) {
		return
	}

	log.Printf("Toggling read-only note id=%d key=%q read_only=%v user_id=%d", id, n.Key, req.ReadOnly, user.ID)
	if err := s.store.SetNoteReadOnly(id, req.ReadOnly); err != nil {
		internalError(w, err)
		return
	}
	updated, err := s.store.GetNoteByID(id)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
