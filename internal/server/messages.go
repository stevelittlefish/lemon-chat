package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/debug"
	"github.com/stevelittlefish/lemon-chat/internal/store"
	"github.com/stevelittlefish/lemon-chat/internal/tasks"
)

type chatMsg struct {
	Role       string `json:"role"`
	Content    string `json:"content,omitempty"`
	ToolCalls  []any  `json:"tool_calls,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

type streamToolCall struct {
	id       string
	name     string
	argsJSON strings.Builder
}

func resolveUserName(user *store.User) string {
	if user.DisplayName != nil && *user.DisplayName != "" {
		return *user.DisplayName
	}
	return user.Username
}

func substituteVars(s, charName, userName string) string {
	s = strings.ReplaceAll(s, "{{char}}", charName)
	s = strings.ReplaceAll(s, "{{user}}", userName)
	return s
}

// resolveCharacter fetches a character, checks visibility for the given user,
// and prepends its system prompt and hidden messages to msgs.
// Returns nil, nil after writing the HTTP error if anything fails.
func (s *Server) resolveCharacter(w http.ResponseWriter, user *store.User, charID int64, msgs []chatMsg) (*store.Character, []chatMsg) {
	char, err := s.store.GetCharacter(charID)
	if err != nil {
		internalError(w, err)
		return nil, nil
	}
	if char.Visibility == "private" && char.CreatedBy != user.ID && !user.IsAdmin {
		writeError(w, http.StatusForbidden, "forbidden")
		return nil, nil
	}
	userName := resolveUserName(user)
	if char.SystemPrompt != nil {
		msgs = append(msgs, chatMsg{Role: "system", Content: substituteVars(*char.SystemPrompt, char.Name, userName)})
	}
	hiddenMsgs, err := s.store.ListCharacterHiddenMessages(char.ID)
	if err != nil {
		internalError(w, err)
		return nil, nil
	}
	for _, hm := range hiddenMsgs {
		msgs = append(msgs, chatMsg{Role: hm.Role, Content: substituteVars(hm.Content, char.Name, userName)})
	}
	return char, msgs
}


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
	all, err := s.store.ListMessages(id)
	if err != nil {
		internalError(w, err)
		return
	}

	type attachmentMeta struct {
		ID       int64  `json:"id"`
		Title    string `json:"title"`
		Filename string `json:"filename"`
		MimeType string `json:"mime_type"`
	}
	type toolInteraction struct {
		ID         string          `json:"id,omitempty"`
		Name       string          `json:"name"`
		Args       any             `json:"args,omitempty"`
		Result     string          `json:"result,omitempty"`
		Attachment *attachmentMeta `json:"attachment,omitempty"`
	}
	type msgView struct {
		store.Message
		ToolInteractions []toolInteraction `json:"tool_interactions,omitempty"`
	}

	// Load attachments for this conversation and index by tool_call_id.
	attachments, attErr := s.store.ListAttachmentsByConversation(id)
	attByCallID := map[string]*attachmentMeta{}
	if attErr == nil {
		for i := range attachments {
			a := &attachments[i]
			attByCallID[a.ToolCallID] = &attachmentMeta{
				ID:       a.ID,
				Title:    a.Title,
				Filename: a.Filename,
				MimeType: a.MimeType,
			}
		}
	}

	// Walk all messages, collecting tool interactions and attaching them to the
	// visible assistant message that consumed them.
	var pending []toolInteraction
	pendingIdxByID := map[string]int{} // tool call ID → index in pending
	views := make([]msgView, 0, len(all))

	for i := range all {
		m := all[i]
		if m.Role == "tool" {
			if idx, ok := pendingIdxByID[m.ToolCallID]; ok {
				pending[idx].Result = m.Content
			}
			continue
		}
		if m.Role == "assistant" && m.Content == "" && m.ToolCalls != nil {
			var tc []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			}
			if jsonErr := json.Unmarshal([]byte(*m.ToolCalls), &tc); jsonErr == nil {
				for _, c := range tc {
					var argsVal any
					json.Unmarshal([]byte(c.Function.Arguments), &argsVal) //nolint:errcheck
					ti := toolInteraction{
						ID:         c.ID,
						Name:       c.Function.Name,
						Args:       argsVal,
						Attachment: attByCallID[c.ID],
					}
					pendingIdxByID[c.ID] = len(pending)
					pending = append(pending, ti)
				}
			}
			continue
		}
		mv := msgView{Message: m}
		if len(pending) > 0 {
			mv.ToolInteractions = pending
			pending = nil
			pendingIdxByID = map[string]int{}
		}
		views = append(views, mv)
	}

	writeJSON(w, http.StatusOK, views)
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

	var chatMsgs []chatMsg

	// Resolve which model/character to use: request override takes precedence over conversation.
	var modelName, assistantName string
	var usedModel *string
	var usedCharacterID *int64
	var usedCharacter *store.Character

	if req.CharacterID != nil {
		var char *store.Character
		char, chatMsgs = s.resolveCharacter(w, user, *req.CharacterID, chatMsgs)
		if char == nil {
			return
		}
		usedCharacterID = req.CharacterID
		usedCharacter = char
		modelName = char.Model
		assistantName = char.Name
	} else if req.Model != nil {
		usedModel = req.Model
		modelName = *req.Model
		assistantName = modelName
	} else if conv.CharacterID != nil {
		var char *store.Character
		char, chatMsgs = s.resolveCharacter(w, user, *conv.CharacterID, chatMsgs)
		if char == nil {
			return
		}
		usedCharacterID = conv.CharacterID
		usedCharacter = char
		modelName = char.Model
		assistantName = char.Name
	} else {
		usedModel = conv.Model
		modelName = *conv.Model
		assistantName = modelName
	}

	modelServer, err := s.cfg.ServerForModel(modelName)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unknown model")
		return
	}
	chatURL := modelServer.APIBase + "/chat/completions"

	// Persist user message.
	if _, err := s.store.CreateMessage(convID, "user", req.Content, nil, user.DisplayName, nil, nil, ""); err != nil {
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
		msg := chatMsg{Role: m.Role, Content: m.Content}
		if m.ToolCalls != nil {
			var tc []any
			if jsonErr := json.Unmarshal([]byte(*m.ToolCalls), &tc); jsonErr == nil {
				msg.ToolCalls = tc
				msg.Content = ""
			}
		}
		if m.ToolCallID != "" {
			msg.ToolCallID = m.ToolCallID
		}
		chatMsgs = append(chatMsgs, msg)
	}

	// Determine tool definitions for this character.
	var activeToolDefs []toolDef
	if usedCharacter != nil && len(usedCharacter.Tools) > 0 {
		activeToolDefs = ToolDefsForCharacter(usedCharacter.Tools)
	}

	responseTimeout := time.Duration(s.cfg.Server.ResponseTimeoutSeconds) * time.Second

	// payloadMap is rebuilt each loop iteration when chatMsgs grows.
	payloadMap := map[string]any{
		"model":          modelName,
		"messages":       chatMsgs,
		"stream":         true,
		"stream_options": map[string]any{"include_usage": true},
	}
	if len(activeToolDefs) > 0 {
		payloadMap["tools"] = activeToolDefs
	}

	// Single timeout context spanning all iterations.
	ctx, cancelResp := context.WithTimeout(r.Context(), responseTimeout)
	defer cancelResp()

	doRequest := func() (*http.Response, error) {
		payloadMap["messages"] = chatMsgs
		p, _ := json.Marshal(payloadMap)
		hreq, err := http.NewRequestWithContext(ctx, "POST", chatURL, bytes.NewReader(p))
		if err != nil {
			return nil, err
		}
		hreq.Header.Set("Content-Type", "application/json")
		if modelServer.APIKey != "" {
			hreq.Header.Set("Authorization", "Bearer "+modelServer.APIKey)
		}
		return s.modelClient.Do(hreq)
	}

	startTime := time.Now()
	resp, err := doRequest()
	if err != nil {
		writeError(w, http.StatusBadGateway, "model unreachable")
		return
	}

	// Committed to SSE from this point.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher := w.(http.Flusher)

	nameJSON, _ := json.Marshal(map[string]string{"name": assistantName})
	fmt.Fprintf(w, "data: %s\n\n", nameJSON)
	flusher.Flush()

	const maxToolLoops = 5
	var finalContent string
	var finalStats *store.MessageStats

	for loop := 0; loop < maxToolLoops; loop++ {
		if loop > 0 {
			resp, err = doRequest()
			if err != nil {
				writeSSEError(w, "model unreachable")
				flusher.Flush()
				break
			}
		}

		var fullContent strings.Builder
		var usageStats *store.MessageStats
		var pendingCalls []*streamToolCall
		var finishReason string

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 1<<20), 1<<20)
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
						Content   string `json:"content"`
						ToolCalls []struct {
							Index    int    `json:"index"`
							ID       string `json:"id"`
							Function struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							} `json:"function"`
						} `json:"tool_calls"`
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
			choice := chunk.Choices[0]
			if choice.FinishReason != nil {
				finishReason = *choice.FinishReason
			}
			if text := choice.Delta.Content; text != "" {
				fullContent.WriteString(text)
				delta, _ := json.Marshal(map[string]string{"delta": text})
				fmt.Fprintf(w, "data: %s\n\n", delta)
				flusher.Flush()
			}
			for _, tc := range choice.Delta.ToolCalls {
				for len(pendingCalls) <= tc.Index {
					pendingCalls = append(pendingCalls, &streamToolCall{})
				}
				if tc.ID != "" {
					pendingCalls[tc.Index].id = tc.ID
				}
				if tc.Function.Name != "" {
					pendingCalls[tc.Index].name = tc.Function.Name
				}
				pendingCalls[tc.Index].argsJSON.WriteString(tc.Function.Arguments)
			}
		}
		if scanErr := scanner.Err(); scanErr != nil {
			log.Printf("messages: SSE scanner error for conv %d: %v", convID, scanErr)
		}
		resp.Body.Close()

		if finishReason == "tool_calls" && len(pendingCalls) > 0 {
			// Persist the assistant tool-call message (role=assistant, content="").
			type tcRecord struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			}
			records := make([]tcRecord, len(pendingCalls))
			for i, tc := range pendingCalls {
				records[i].ID = tc.id
				records[i].Type = "function"
				records[i].Function.Name = tc.name
				records[i].Function.Arguments = tc.argsJSON.String()
			}
			toolCallsJSON, _ := json.Marshal(records)
			toolCallsStr := string(toolCallsJSON)
			if _, err := s.store.CreateMessage(convID, "assistant", "", usedCharacterID, &assistantName, nil, &toolCallsStr, ""); err != nil {
				log.Printf("messages: failed to persist tool-call message for conv %d: %v", convID, err)
			}
			var tc2 []any
			json.Unmarshal(toolCallsJSON, &tc2) //nolint:errcheck
			chatMsgs = append(chatMsgs, chatMsg{Role: "assistant", ToolCalls: tc2})

			// Execute each tool and persist the result.
			for _, tc := range pendingCalls {
				logArgs := tc.argsJSON.String()
				if len(logArgs) > 200 {
					logArgs = logArgs[:200] + "…"
				}
				log.Printf("Calling tool name=%q args=%s conversation_id=%d", tc.name, logArgs, convID)
				var argsVal any
				json.Unmarshal([]byte(tc.argsJSON.String()), &argsVal) //nolint:errcheck
				evtJSON, _ := json.Marshal(map[string]any{"tool_call": map[string]any{
					"id":   tc.id,
					"name": tc.name,
					"args": argsVal,
				}})
				fmt.Fprintf(w, "data: %s\n\n", evtJSON)
				flusher.Flush()

				tctx := ToolContext{
					ModelName:       modelName,
					ModelServer:     modelServer,
					ResponseTimeout: responseTimeout,
					SearXNGURL:      s.cfg.SearXNG.URL,
					ComfyUIURL:      s.cfg.ComfyUI.URL,
					ComfyUIWorkflow: s.cfg.ComfyUI.Workflow,
					ToolCallID:      tc.id,
					ConversationID:  convID,
					Store:           s.store,
				}
				result, execErr := ExecuteTool(tc.name, tc.argsJSON.String(), tctx)
				if execErr != nil {
					result = "error: " + execErr.Error()
				}
				logResult := result
				if len(logResult) > 200 {
					logResult = logResult[:200] + "…"
				}
				log.Printf("Tool result name=%q conversation_id=%d result=%q", tc.name, convID, logResult)

				resultEvt, _ := json.Marshal(map[string]any{"tool_result": map[string]any{
					"id":     tc.id,
					"result": result,
				}})
				fmt.Fprintf(w, "data: %s\n\n", resultEvt)
				flusher.Flush()

				// If the tool produced an attachment, emit a dedicated SSE event.
				var attResult AttachmentResult
				if jsonErr := json.Unmarshal([]byte(result), &attResult); jsonErr == nil && attResult.AttachmentID != 0 {
					attEvt, _ := json.Marshal(map[string]any{"attachment": map[string]any{
						"id":           attResult.AttachmentID,
						"tool_call_id": tc.id,
						"title":        attResult.Title,
						"filename":     attResult.Filename,
						"mime_type":    attResult.MimeType,
					}})
					fmt.Fprintf(w, "data: %s\n\n", attEvt)
					flusher.Flush()
				}

				if _, err := s.store.CreateMessage(convID, "tool", result, nil, nil, nil, nil, tc.id); err != nil {
					log.Printf("messages: failed to persist tool result for conv %d: %v", convID, err)
				}
				chatMsgs = append(chatMsgs, chatMsg{Role: "tool", Content: result, ToolCallID: tc.id})
			}

			if loop < maxToolLoops-1 {
				continue
			}
			log.Printf("messages: tool call loop limit reached for conv %d", convID)
		}

		finalContent = fullContent.String()
		finalStats = usageStats
		break
	}

	if finalStats != nil {
		statsJSON, _ := json.Marshal(map[string]any{
			"stats": map[string]int64{
				"prompt_tokens":     finalStats.PromptTokens,
				"completion_tokens": finalStats.CompletionTokens,
				"total_time_ms":     finalStats.TotalTimeMS,
			},
		})
		fmt.Fprintf(w, "data: %s\n\n", statsJSON)
		flusher.Flush()
	}

	// Persist completed assistant message and update conversation model/character.
	if finalContent != "" {
		prevUpdatedAt, parseErr := time.Parse(time.RFC3339, conv.UpdatedAt)
		if parseErr != nil {
			log.Printf("messages: conv %d: failed to parse updated_at %q: %v", convID, conv.UpdatedAt, parseErr)
		}
		msg, err := s.store.CreateMessage(convID, "assistant", finalContent, usedCharacterID, &assistantName, finalStats, nil, "")
		if err2 := s.store.UpdateConversationAfterMessage(convID, usedModel, usedCharacterID); err2 != nil {
			log.Printf("messages: failed to update conversation %d after message: %v", convID, err2)
		}
		if time.Since(prevUpdatedAt) > 2*time.Minute {
			s.hub.BroadcastConversationListChanged()
		}
		if err == nil {
			idJSON, _ := json.Marshal(map[string]int64{"message_id": msg.ID})
			fmt.Fprintf(w, "data: %s\n\n", idJSON)
			flusher.Flush()
		}

		titleTriggered := false

		// Trigger auto-title on the first completed exchange when the character requests it.
		if usedCharacter != nil && usedCharacter.AutoTitle && conv.Title == nil {
			userMsgCount := 0
			for _, m := range history {
				if m.Role == "user" {
					userMsgCount++
				}
			}
			debug.Log("title trigger (character auto-title): conv=%d userMsgCount=%d autoTitle=%v titleIsNil=%v", convID, userMsgCount, usedCharacter.AutoTitle, conv.Title == nil)
			if userMsgCount == 1 {
				tasks.GenerateTitleForConversation(s.store, s.cfg, convID, s.hub.BroadcastTitleUpdate)
				titleTriggered = true
			}
		}

		// Trigger title generation on the third assistant response for any untitled conversation.
		if !titleTriggered && conv.Title == nil {
			assistantMsgCount := 0
			for _, m := range history {
				if m.Role == "assistant" {
					assistantMsgCount++
				}
			}
			debug.Log("title trigger (3rd assistant): conv=%d assistantMsgCount=%d (need >=2 to fire)", convID, assistantMsgCount)
			if assistantMsgCount >= 2 {
				tasks.GenerateTitleForConversation(s.store, s.cfg, convID, s.hub.BroadcastTitleUpdate)
			}
		}
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func (s *Server) handleGetMessageContext(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	convID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	msgID, err := strconv.ParseInt(r.PathValue("msgId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid message id")
		return
	}

	conv, err := s.store.GetConversation(convID, user.ID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	allMsgs, err := s.store.ListMessages(convID)
	if err != nil {
		internalError(w, err)
		return
	}

	// Collect messages up to and including the target message.
	var history []store.Message
	for _, m := range allMsgs {
		history = append(history, m)
		if m.ID == msgID {
			break
		}
	}
	if len(history) == 0 || history[len(history)-1].ID != msgID {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}

	var chatMsgs []chatMsg

	if conv.CharacterID != nil {
		char, err := s.store.GetCharacter(*conv.CharacterID)
		if err == nil {
			userName := resolveUserName(user)
			if char.SystemPrompt != nil {
				chatMsgs = append(chatMsgs, chatMsg{Role: "system", Content: substituteVars(*char.SystemPrompt, char.Name, userName)})
			}
			hiddenMsgs, err := s.store.ListCharacterHiddenMessages(char.ID)
			if err == nil {
				for _, hm := range hiddenMsgs {
					chatMsgs = append(chatMsgs, chatMsg{Role: hm.Role, Content: substituteVars(hm.Content, char.Name, userName)})
				}
			}
		}
	}

	for _, m := range history {
		msg := chatMsg{Role: m.Role, Content: m.Content}
		if m.ToolCalls != nil {
			var tc []any
			if jsonErr := json.Unmarshal([]byte(*m.ToolCalls), &tc); jsonErr == nil {
				msg.ToolCalls = tc
				msg.Content = ""
			}
		}
		if m.ToolCallID != "" {
			msg.ToolCallID = m.ToolCallID
		}
		chatMsgs = append(chatMsgs, msg)
	}
	if chatMsgs == nil {
		chatMsgs = []chatMsg{}
	}

	writeJSON(w, http.StatusOK, map[string]any{"messages": chatMsgs})
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

	content := substituteVars(*char.FirstMessage, char.Name, resolveUserName(user))
	msg, err := s.store.CreateMessage(convID, "assistant", content, charID, &char.Name, nil, nil, "")
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}
