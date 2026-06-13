package server

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/stevelittlefish/lemon-chat/internal/store"
	"github.com/stevelittlefish/lemon-chat/internal/tasks"
)

const defaultConversationPageSize = 30

func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)

	limit := defaultConversationPageSize
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	// Fetch one extra to detect whether more pages exist.
	convs, err := s.store.ListConversations(user.ID, limit+1, offset)
	if err != nil {
		internalError(w, err)
		return
	}
	hasMore := len(convs) > limit
	if hasMore {
		convs = convs[:limit]
	}
	if convs == nil {
		convs = []store.Conversation{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"conversations": convs,
		"has_more":      hasMore,
	})
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
	log.Printf("Creating conversation user_id=%d", user.ID)
	conv, err := s.store.CreateConversation(user.ID, req.Title, req.Model, req.CharacterID)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, conv)
}

func (s *Server) handleUpdateConversation(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req struct {
		Title *string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if _, err := s.store.GetConversation(id, user.ID); notFoundOr500(w, err) {
		return
	}
	if req.Title != nil {
		log.Printf("Updating conversation title id=%d user_id=%d", id, user.ID)
		if err := s.store.UpdateConversationTitle(id, *req.Title); err != nil {
			internalError(w, err)
			return
		}
		s.hub.BroadcastTitleUpdate(id, *req.Title)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRegenerateTitle(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, err := s.store.GetConversation(id, user.ID); notFoundOr500(w, err) {
		return
	}
	log.Printf("Regenerating title conversation_id=%d user_id=%d", id, user.ID)
	tasks.GenerateTitleForConversation(s.store, s.cfg, id, func(convID int64, title string) {
		s.hub.BroadcastTitleUpdate(convID, title)
	})
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleForkConversation(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req struct {
		MessageID int64 `json:"message_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MessageID == 0 {
		writeError(w, http.StatusBadRequest, "message_id required")
		return
	}
	log.Printf("Forking conversation id=%d user_id=%d", id, user.ID)
	conv, err := s.store.ForkConversation(id, user.ID, req.MessageID)
	if notFoundOr500(w, err) {
		return
	}
	s.hub.BroadcastConversationListChanged()
	writeJSON(w, http.StatusCreated, conv)
}

func (s *Server) handleImportConversation(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)

	var req struct {
		Model       *string `json:"model"`
		CharacterID *int64  `json:"character_id"`
		Messages    []struct {
			Role       string           `json:"role"`
			Content    string           `json:"content"`
			ToolCalls  []map[string]any `json:"tool_calls"`
			ToolCallID string           `json:"tool_call_id"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if (req.Model == nil) == (req.CharacterID == nil) {
		writeError(w, http.StatusBadRequest, "exactly one of model or character_id is required")
		return
	}

	const attachmentPlaceholder = "[Attachment not available: the file referenced here was not included in the JSON export and cannot be displayed.]"

	var msgs []store.ImportMessage
	for _, m := range req.Messages {
		if m.Role == "system" {
			continue
		}

		content := m.Content

		if m.Role == "tool" && content != "" {
			var att AttachmentResult
			if jsonErr := json.Unmarshal([]byte(content), &att); jsonErr == nil && att.AttachmentID != 0 {
				content = attachmentPlaceholder
			}
		}

		var toolCallsStr *string
		if len(m.ToolCalls) > 0 {
			b, err := json.Marshal(m.ToolCalls)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid tool_calls")
				return
			}
			tc := string(b)
			toolCallsStr = &tc
		}

		msgs = append(msgs, store.ImportMessage{
			Role:       m.Role,
			Content:    content,
			ToolCalls:  toolCallsStr,
			ToolCallID: m.ToolCallID,
		})
	}

	if len(msgs) == 0 {
		writeError(w, http.StatusBadRequest, "no messages to import")
		return
	}

	log.Printf("Importing conversation user_id=%d messages=%d", user.ID, len(msgs))
	conv, err := s.store.ImportConversation(user.ID, req.Model, req.CharacterID, msgs)
	if err != nil {
		internalError(w, err)
		return
	}
	s.hub.BroadcastConversationListChanged()
	writeJSON(w, http.StatusCreated, conv)
}

func (s *Server) handleDeleteConversation(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	log.Printf("Deleting conversation id=%d user_id=%d", id, user.ID)
	attPaths, err := s.store.DeleteConversation(id, user.ID)
	if notFoundOr500(w, err) {
		return
	}
	for _, p := range attPaths {
		os.RemoveAll(filepath.Join(s.cfg.Server.DataDir, filepath.Dir(p)))
	}
	w.WriteHeader(http.StatusNoContent)
}
