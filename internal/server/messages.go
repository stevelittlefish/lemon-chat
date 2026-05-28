package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := s.store.GetConversation(id, user.ID); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	msgs, err := s.store.ListMessages(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if msgs == nil {
		msgs = []store.Message{}
	}
	writeJSON(w, http.StatusOK, msgs)
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	convID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	conv, err := s.store.GetConversation(convID, user.ID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content required")
		return
	}

	type chatMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var chatMsgs []chatMsg

	// Derive model from conversation; prepend system prompt if using a character.
	var modelName string
	if conv.CharacterID != nil {
		char, err := s.store.GetCharacter(*conv.CharacterID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		modelName = char.Model
		if char.SystemPrompt != nil {
			chatMsgs = append(chatMsgs, chatMsg{Role: "system", Content: *char.SystemPrompt})
		}
	} else {
		modelName = *conv.Model
	}

	known := false
	for _, m := range s.cfg.Models {
		if m.Name == modelName {
			known = true
			break
		}
	}
	if !known {
		writeError(w, http.StatusBadRequest, "unknown model")
		return
	}
	chatURL := strings.TrimRight(s.cfg.ModelServer.APIBase, "/") + "/chat/completions"

	// Persist user message.
	if _, err := s.store.CreateMessage(convID, "user", req.Content); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Build message history for model.
	history, err := s.store.ListMessages(convID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	for _, m := range history {
		chatMsgs = append(chatMsgs, chatMsg{Role: m.Role, Content: m.Content})
	}

	payload, _ := json.Marshal(map[string]any{
		"model":    modelName,
		"messages": chatMsgs,
		"stream":   true,
	})

	resp, err := http.Post(chatURL, "application/json", bytes.NewReader(payload)) //nolint:gosec
	if err != nil {
		writeError(w, http.StatusBadGateway, "model unreachable")
		return
	}
	defer resp.Body.Close()

	// Stream response back to client via SSE.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher := w.(http.Flusher)

	// OpenAI-compatible streaming: each line is "data: <json>" or "data: [DONE]"
	var fullContent string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := line[6:]
		if payload == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		if text := chunk.Choices[0].Delta.Content; text != "" {
			fullContent += text
			delta, _ := json.Marshal(map[string]string{"delta": text})
			fmt.Fprintf(w, "data: %s\n\n", delta)
			flusher.Flush()
		}
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	// Persist completed assistant message.
	if fullContent != "" {
		_, _ = s.store.CreateMessage(convID, "assistant", fullContent)
		_ = s.store.TouchConversation(convID)
	}
}

func writeSSEError(w io.Writer, msg string) {
	errJSON, _ := json.Marshal(map[string]string{"error": msg})
	fmt.Fprintf(w, "data: %s\n\n", errJSON)
}
