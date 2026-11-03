package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

// memoryOwner resolves whose memories a settings-page request is for: the
// caller's own, or with ?user_id= (admins only) another user's.
func (s *Server) memoryOwner(w http.ResponseWriter, r *http.Request) (int64, bool) {
	user := currentUser(r)
	raw := r.URL.Query().Get("user_id")
	if raw == "" {
		return user.ID, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return 0, false
	}
	if id != user.ID && !user.IsAdmin {
		writeError(w, http.StatusForbidden, "forbidden")
		return 0, false
	}
	if _, err := s.store.UserByID(id); notFoundOr500(w, err) {
		return 0, false
	}
	return id, true
}

// memoryForRequest loads the memory named in the path, if the caller owns it
// or is an admin.
func (s *Server) memoryForRequest(w http.ResponseWriter, r *http.Request) (*store.Memory, bool) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return nil, false
	}
	m, err := s.store.GetMemory(id)
	if err == nil && m.UserID != user.ID && !user.IsAdmin {
		err = store.ErrNotFound
	}
	if notFoundOr500(w, err) {
		return nil, false
	}
	return m, true
}

func (s *Server) handleListMemories(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := s.memoryOwner(w, r)
	if !ok {
		return
	}
	if user := currentUser(r); ownerID != user.ID {
		log.Printf("Listing memories user_id=%d admin_id=%d", ownerID, user.ID)
	}
	memories, err := s.store.ListMemories(ownerID)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, memories)
}

func (s *Server) handleCreateMemory(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := s.memoryOwner(w, r)
	if !ok {
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	text, err := validMemoryText(req.Text)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	m, err := s.store.CreateMemory(ownerID, text, nil)
	if err != nil {
		internalError(w, err)
		return
	}
	logMemoryAction(r, "Creating", m.ID, ownerID)
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) handleUpdateMemory(w http.ResponseWriter, r *http.Request) {
	m, ok := s.memoryForRequest(w, r)
	if !ok {
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	text, err := validMemoryText(req.Text)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	logMemoryAction(r, "Updating", m.ID, m.UserID)
	if err := s.store.UpdateMemory(m.ID, m.UserID, text); notFoundOr500(w, err) {
		return
	}
	updated, err := s.store.GetMemory(m.ID)
	if notFoundOr500(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteMemory(w http.ResponseWriter, r *http.Request) {
	m, ok := s.memoryForRequest(w, r)
	if !ok {
		return
	}
	logMemoryAction(r, "Deleting", m.ID, m.UserID)
	if err := s.store.DeleteMemory(m.ID, m.UserID); notFoundOr500(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// logMemoryAction logs a settings-page change, naming the admin when they
// act on another user's memories.
func logMemoryAction(r *http.Request, verb string, memoryID, ownerID int64) {
	user := currentUser(r)
	if ownerID == user.ID {
		log.Printf("%s memory id=%d user_id=%d", verb, memoryID, ownerID)
		return
	}
	log.Printf("%s memory id=%d user_id=%d admin_id=%d", verb, memoryID, ownerID, user.ID)
}
