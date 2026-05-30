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
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/store"
	"github.com/stevelittlefish/lemon-chat/internal/tasks"
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
		internalError(w, err)
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
		Content     string  `json:"content"`
		Model       *string `json:"model"`
		CharacterID *int64  `json:"character_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content required")
		return
	}
	if req.Model != nil && req.CharacterID != nil {
		writeError(w, http.StatusBadRequest, "at most one of model or character_id may be specified")
		return
	}

	type chatMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var chatMsgs []chatMsg

	// Resolve which model/character to use: request override takes precedence over conversation.
	var modelName, assistantName string
	var usedModel *string
	var usedCharacterID *int64
	var usedCharacter *store.Character

	if req.CharacterID != nil {
		char, err := s.store.GetCharacter(*req.CharacterID)
		if err != nil {
			internalError(w, err)
			return
		}
		usedCharacterID = req.CharacterID
		usedCharacter = char
		modelName = char.Model
		assistantName = char.Name
		if char.SystemPrompt != nil {
			chatMsgs = append(chatMsgs, chatMsg{Role: "system", Content: *char.SystemPrompt})
		}
		hiddenMsgs, err := s.store.ListCharacterHiddenMessages(char.ID)
		if err != nil {
			internalError(w, err)
			return
		}
		for _, hm := range hiddenMsgs {
			chatMsgs = append(chatMsgs, chatMsg{Role: hm.Role, Content: hm.Content})
		}
	} else if req.Model != nil {
		usedModel = req.Model
		modelName = *req.Model
		assistantName = modelName
	} else if conv.CharacterID != nil {
		char, err := s.store.GetCharacter(*conv.CharacterID)
		if err != nil {
			internalError(w, err)
			return
		}
		usedCharacterID = conv.CharacterID
		usedCharacter = char
		modelName = char.Model
		assistantName = char.Name
		if char.SystemPrompt != nil {
			chatMsgs = append(chatMsgs, chatMsg{Role: "system", Content: *char.SystemPrompt})
		}
		hiddenMsgs, err := s.store.ListCharacterHiddenMessages(char.ID)
		if err != nil {
			internalError(w, err)
			return
		}
		for _, hm := range hiddenMsgs {
			chatMsgs = append(chatMsgs, chatMsg{Role: hm.Role, Content: hm.Content})
		}
	} else {
		usedModel = conv.Model
		modelName = *conv.Model
		assistantName = modelName
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
	if _, err := s.store.CreateMessage(convID, "user", req.Content, nil, nil); err != nil {
		internalError(w, err)
		return
	}

	// Build message history for model.
	history, err := s.store.ListMessages(convID)
	if err != nil {
		internalError(w, err)
		return
	}

	for _, m := range history {
		chatMsgs = append(chatMsgs, chatMsg{Role: m.Role, Content: m.Content})
	}

	payload, _ := json.Marshal(map[string]any{
		"model":    modelName,
		"messages": chatMsgs,
		"stream":   true,
		"stream_options": map[string]any{
			"include_usage": true,
		},
	})

	startTime := time.Now()
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

	nameJSON, _ := json.Marshal(map[string]string{"name": assistantName})
	fmt.Fprintf(w, "data: %s\n\n", nameJSON)
	flusher.Flush()

	// OpenAI-compatible streaming: each line is "data: <json>" or "data: [DONE]"
	var fullContent string
	var usageStats *store.MessageStats
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := line[6:]
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int64 `json:"prompt_tokens"`
				CompletionTokens int64 `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.Usage != nil {
			usageStats = &store.MessageStats{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTimeMS:      time.Since(startTime).Milliseconds(),
			}
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

	if usageStats != nil {
		statsJSON, _ := json.Marshal(map[string]any{
			"stats": map[string]int64{
				"prompt_tokens":     usageStats.PromptTokens,
				"completion_tokens": usageStats.CompletionTokens,
				"total_time_ms":     usageStats.TotalTimeMS,
			},
		})
		fmt.Fprintf(w, "data: %s\n\n", statsJSON)
		flusher.Flush()
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	// Persist completed assistant message and update conversation model/character.
	if fullContent != "" {
		_, _ = s.store.CreateMessage(convID, "assistant", fullContent, &assistantName, usageStats)
		_ = s.store.UpdateConversationAfterMessage(convID, usedModel, usedCharacterID)

		// Trigger auto-title on the first completed exchange when the character requests it.
		// Count user messages in history — if this is the only one, it's the first exchange.
		if usedCharacter != nil && usedCharacter.AutoTitle && conv.Title == nil {
			userMsgCount := 0
			for _, m := range history {
				if m.Role == "user" {
					userMsgCount++
				}
			}
			if userMsgCount == 1 {
				tasks.GenerateTitleForConversation(s.store, s.cfg, convID, s.hub.BroadcastTitleUpdate)
			}
		}

		// Trigger title generation on the third assistant response for any untitled conversation.
		if conv.Title == nil {
			assistantMsgCount := 0
			for _, m := range history {
				if m.Role == "assistant" {
					assistantMsgCount++
				}
			}
			if assistantMsgCount >= 2 {
				tasks.GenerateTitleForConversation(s.store, s.cfg, convID, s.hub.BroadcastTitleUpdate)
			}
		}
	}
}

func writeSSEError(w io.Writer, msg string) {
	errJSON, _ := json.Marshal(map[string]string{"error": msg})
	fmt.Fprintf(w, "data: %s\n\n", errJSON)
}

func (s *Server) handleFirstMessage(w http.ResponseWriter, r *http.Request) {
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
		CharacterID *int64 `json:"character_id"`
	}
	json.NewDecoder(r.Body).Decode(&req) // body is optional

	msgs, err := s.store.ListMessages(convID)
	if err != nil {
		internalError(w, err)
		return
	}
	if len(msgs) > 0 {
		writeError(w, http.StatusConflict, "conversation already has messages")
		return
	}

	charID := conv.CharacterID
	if req.CharacterID != nil {
		charID = req.CharacterID
	}
	if charID == nil {
		writeError(w, http.StatusBadRequest, "no character")
		return
	}

	char, err := s.store.GetCharacter(*charID)
	if err != nil {
		writeError(w, http.StatusNotFound, "character not found")
		return
	}
	if char.Visibility == "private" && char.CreatedBy != user.ID && !user.IsAdmin {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if char.FirstMessage == nil {
		writeError(w, http.StatusBadRequest, "character has no first message")
		return
	}

	// If a character override was provided, update the conversation to use it.
	if req.CharacterID != nil {
		if err := s.store.UpdateConversationAfterMessage(convID, nil, charID); err != nil {
			internalError(w, err)
			return
		}
	}

	msg, err := s.store.CreateMessage(convID, "assistant", *char.FirstMessage, &char.Name, nil)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}
