package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/debug"
	"github.com/stevelittlefish/lemon-chat/internal/llm"
	"github.com/stevelittlefish/lemon-chat/internal/store"
	"github.com/stevelittlefish/lemon-chat/internal/tasks"
)

type chatMsg struct {
	Role       string `json:"role"`
	Content    string `json:"content,omitempty"`
	ToolCalls  []any  `json:"tool_calls,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// buildChatMsgs assembles the LLM message payload from an optional prefix
// (e.g. character system prompt + hidden messages) and the stored message history.
// It is the single source of truth used by both the send handler and the context
// preview endpoint, so the two are guaranteed to produce identical output.
func buildChatMsgs(prefix []chatMsg, history []store.Message) []chatMsg {
	msgs := append([]chatMsg(nil), prefix...)
	for _, m := range history {
		msg := chatMsg{Role: m.Role, Content: m.Content}
		if m.ToolCalls != nil {
			var tc []any
			if json.Unmarshal([]byte(*m.ToolCalls), &tc) == nil {
				msg.ToolCalls = tc
			}
		}
		if m.ToolCallID != "" {
			msg.ToolCallID = m.ToolCallID
		}
		msgs = append(msgs, msg)
	}
	return msgs
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
	id, ok := pathID(w, r)
	if !ok {
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
		Status   string `json:"status,omitempty"`
		Error    string `json:"error,omitempty"`
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
				Status:   a.Status,
				Error:    a.Error,
			}
		}
	}

	// Walk messages: tool-result messages (role="tool") are hidden but their content
	// is attached to the preceding assistant tool-call message. All other messages
	// are shown as-is.
	type pendingResult struct{ viewIdx, tiIdx int }
	pendingByCallID := map[string]pendingResult{}
	views := make([]msgView, 0, len(all))

	for i := range all {
		m := all[i]
		if m.Role == "tool" {
			if p, ok := pendingByCallID[m.ToolCallID]; ok {
				views[p.viewIdx].ToolInteractions[p.tiIdx].Result = m.Content
			}
			continue
		}
		mv := msgView{Message: m}
		if m.ToolCalls != nil {
			var tc []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			}
			if jsonErr := json.Unmarshal([]byte(*m.ToolCalls), &tc); jsonErr == nil {
				viewIdx := len(views)
				for tiIdx, c := range tc {
					var argsVal any
					json.Unmarshal([]byte(c.Function.Arguments), &argsVal) //nolint:errcheck
					mv.ToolInteractions = append(mv.ToolInteractions, toolInteraction{
						ID:         c.ID,
						Name:       c.Function.Name,
						Args:       argsVal,
						Attachment: attByCallID[c.ID],
					})
					pendingByCallID[c.ID] = pendingResult{viewIdx, tiIdx}
				}
			}
		}
		views = append(views, mv)
	}

	writeJSON(w, http.StatusOK, views)
}

type messageToolLoop struct {
	server          *Server
	ctx             context.Context
	writer          http.ResponseWriter
	flusher         http.Flusher
	provider        llm.Provider
	request         llm.Request
	messages        []chatMsg
	wireLog         io.Writer
	onStart         func()
	committed       func() bool
	persistError    func() error
	conversationID  int64
	assistantName   string
	characterID     *int64
	toolContext     ToolContext
	maxLoops        int
	responseStarted *time.Time
}

// run streams model turns and executes requested tools until the model returns
// a final response or the configured tool-loop limit is reached. The bool is
// false only when the caller must stop because an HTTP error has been written.
func (loop *messageToolLoop) run() (string, *store.MessageStats, bool) {
	var finalContent string
	var finalStats *store.MessageStats

	for iteration := 0; iteration < loop.maxLoops; iteration++ {
		h := llm.Handler{
			ErrorLog: loop.server.modelErrors,
			OnStart:  loop.onStart,
			OnText: func(delta string) {
				if !loop.committed() {
					return
				}
				b, _ := json.Marshal(map[string]string{"delta": delta})
				fmt.Fprintf(loop.writer, "data: %s\n\n", b)
				loop.flusher.Flush()
			},
		}
		if loop.wireLog != nil {
			h.WireLog = loop.wireLog
		}

		loop.request.Messages = loop.messages
		comp, err := loop.provider.Stream(loop.ctx, loop.request, h)
		if persistErr := loop.persistError(); persistErr != nil {
			internalError(loop.writer, persistErr)
			return "", nil, false
		}
		if !loop.committed() {
			writeError(loop.writer, http.StatusBadGateway, "model unreachable")
			return "", nil, false
		}
		if err != nil {
			log.Printf("messages: stream error for conv %d model=%q: %v", loop.conversationID, loop.request.Model, err)
			writeSSEError(loop.writer, "model error")
			loop.flusher.Flush()
			break
		}

		var usageStats *store.MessageStats
		if comp.Usage != nil {
			usageStats = &store.MessageStats{
				PromptTokens:     int64(comp.Usage.PromptTokens),
				CompletionTokens: int64(comp.Usage.CompletionTokens),
				CachedTokens:     int64(comp.Usage.CachedTokens()),
				TotalTimeMS:      time.Since(*loop.responseStarted).Milliseconds(),
			}
		}

		if comp.FinishReason == "tool_calls" && len(comp.ToolCalls) > 0 {
			loop.persistToolCall(comp.Content, comp.ToolCalls)
			for _, tc := range comp.ToolCalls {
				loop.executeTool(tc)
			}

			newTurnEvt, _ := json.Marshal(map[string]any{"new_turn": map[string]any{"name": loop.assistantName}})
			fmt.Fprintf(loop.writer, "data: %s\n\n", newTurnEvt)
			loop.flusher.Flush()

			if iteration < loop.maxLoops-1 {
				continue
			}
			log.Printf("messages: tool call loop limit reached for conv %d", loop.conversationID)
		}

		finalContent = comp.Content
		finalStats = usageStats
		break
	}

	return finalContent, finalStats, true
}

func (loop *messageToolLoop) persistToolCall(content string, toolCalls []llm.ToolCall) {
	type toolCallRecord struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}

	records := make([]toolCallRecord, len(toolCalls))
	for i, tc := range toolCalls {
		records[i].ID = tc.ID
		records[i].Type = "function"
		records[i].Function.Name = tc.Name
		records[i].Function.Arguments = tc.Arguments
	}
	toolCallsJSON, _ := json.Marshal(records)
	toolCallsStr := string(toolCallsJSON)
	if _, err := loop.server.store.CreateMessage(loop.conversationID, "assistant", content, loop.characterID, &loop.assistantName, nil, &toolCallsStr, ""); err != nil {
		log.Printf("messages: failed to persist tool-call message for conv %d: %v", loop.conversationID, err)
	}
	var calls []any
	json.Unmarshal(toolCallsJSON, &calls) //nolint:errcheck
	loop.messages = append(loop.messages, chatMsg{Role: "assistant", Content: content, ToolCalls: calls})
}

func (loop *messageToolLoop) executeTool(tc llm.ToolCall) {
	logArgs := tc.Arguments
	if len(logArgs) > 200 {
		logArgs = logArgs[:200] + "…"
	}
	log.Printf("Calling tool name=%q args=%s conversation_id=%d", tc.Name, logArgs, loop.conversationID)

	var argsValue any
	json.Unmarshal([]byte(tc.Arguments), &argsValue) //nolint:errcheck
	callEvent, _ := json.Marshal(map[string]any{"tool_call": map[string]any{
		"id": tc.ID, "name": tc.Name, "args": argsValue,
	}})
	fmt.Fprintf(loop.writer, "data: %s\n\n", callEvent)
	loop.flusher.Flush()

	toolContext := loop.toolContext
	toolContext.ToolCallID = tc.ID
	result, err := ExecuteTool(tc.Name, tc.Arguments, toolContext)
	if err != nil {
		result = "error: " + err.Error()
	}
	logResult := result
	if len(logResult) > 200 {
		logResult = logResult[:200] + "…"
	}
	log.Printf("Tool result name=%q conversation_id=%d result=%q", tc.Name, loop.conversationID, logResult)

	resultEvent, _ := json.Marshal(map[string]any{"tool_result": map[string]any{
		"id": tc.ID, "result": result,
	}})
	fmt.Fprintf(loop.writer, "data: %s\n\n", resultEvent)
	loop.flusher.Flush()

	var attachment AttachmentResult
	if jsonErr := json.Unmarshal([]byte(result), &attachment); jsonErr == nil && attachment.AttachmentID != 0 {
		attachmentEvent, _ := json.Marshal(map[string]any{"attachment": map[string]any{
			"id":           attachment.AttachmentID,
			"tool_call_id": tc.ID,
			"title":        attachment.Title,
			"filename":     attachment.Filename,
			"mime_type":    attachment.MimeType,
			"background":   attachment.Background,
			"status":       attachment.Status,
		}})
		fmt.Fprintf(loop.writer, "data: %s\n\n", attachmentEvent)
		loop.flusher.Flush()
	}

	if _, err := loop.server.store.CreateMessage(loop.conversationID, "tool", result, nil, nil, nil, nil, tc.ID); err != nil {
		log.Printf("messages: failed to persist tool result for conv %d: %v", loop.conversationID, err)
	}
	loop.messages = append(loop.messages, chatMsg{Role: "tool", Content: result, ToolCallID: tc.ID})
}

const (
	titleTriggerNone          = ""
	titleTriggerCharacterAuto = "character-auto-title"
	titleTriggerThirdResponse = "third-assistant-response"
)

func conversationTitleTrigger(conv *store.Conversation, character *store.Character, history []store.Message) (reason string, userMessages, assistantMessages int) {
	userMessages = 1 // Include the user message persisted after history was loaded.
	for _, message := range history {
		switch message.Role {
		case "user":
			userMessages++
		case "assistant":
			assistantMessages++
		}
	}
	if conv.Title != nil {
		return titleTriggerNone, userMessages, assistantMessages
	}
	if character != nil && character.AutoTitle && userMessages == 1 {
		return titleTriggerCharacterAuto, userMessages, assistantMessages
	}
	if assistantMessages >= 2 {
		return titleTriggerThirdResponse, userMessages, assistantMessages
	}
	return titleTriggerNone, userMessages, assistantMessages
}

func (s *Server) triggerConversationTitle(conv *store.Conversation, character *store.Character, history []store.Message) {
	reason, userMessages, assistantMessages := conversationTitleTrigger(conv, character, history)
	if character != nil && character.AutoTitle && conv.Title == nil {
		debug.Log("title trigger (character auto-title): conv=%d userMsgCount=%d autoTitle=%v titleIsNil=%v", conv.ID, userMessages, character.AutoTitle, conv.Title == nil)
	}
	if reason != titleTriggerCharacterAuto && conv.Title == nil {
		debug.Log("title trigger (3rd assistant): conv=%d assistantMsgCount=%d (need >=2 to fire)", conv.ID, assistantMessages)
	}
	if reason != titleTriggerNone {
		tasks.GenerateTitleForConversation(s.store, s.cfg, conv.ID, s.hub.BroadcastTitleUpdate)
	}
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	convID, ok := pathID(w, r)
	if !ok {
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

	log.Printf("Sending message conversation_id=%d user_id=%d", convID, user.ID)

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

	// Build message history for model (user message not yet persisted).
	history, err := s.store.ListMessages(convID)
	if err != nil {
		internalError(w, err)
		return
	}

	chatMsgs = buildChatMsgs(chatMsgs, history)
	chatMsgs = append(chatMsgs, chatMsg{Role: "user", Content: req.Content})

	// Determine tool definitions for this character.
	var activeToolDefs []toolDef
	if usedCharacter != nil && len(usedCharacter.Tools) > 0 {
		activeToolDefs = ToolDefsForCharacter(usedCharacter.Tools)
	}

	responseTimeout := time.Duration(s.cfg.Server.ResponseTimeoutSeconds) * time.Second

	// Provider hides the wire protocol (chat-completions vs Responses) behind one
	// streaming call, so the tool loop below is protocol-blind.
	provider := llm.NewProvider(s.modelClient, modelServer, s.tokenSource(modelServer), s.oauthAccountID(modelServer))
	var toolsArg any
	if len(activeToolDefs) > 0 {
		toolsArg = activeToolDefs
	}

	// Single timeout context spanning all iterations.
	ctx, cancelResp := context.WithTimeout(r.Context(), responseTimeout)
	defer cancelResp()

	var tokenLog *os.File
	if s.cfg.Server.TokenLog {
		logPath := filepath.Join(s.cfg.Server.DataDir, "model_tokens.log")
		if f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			tokenLog = f
			defer tokenLog.Close()
			fmt.Fprintf(tokenLog, "\n=== conv=%d model=%s time=%s ===\n", convID, modelName, time.Now().Format(time.RFC3339))
		} else {
			log.Printf("token log: could not open %s: %v", logPath, err)
		}
	}

	flusher := w.(http.Flusher)

	// SSE is committed lazily on the first successful model response (the
	// provider's OnStart). This preserves the old behaviour: an unreachable model
	// yields a clean error and no orphaned user message, rather than a half-open
	// stream. persistErr captures a rare failure to persist the user message.
	var startTime time.Time
	committed := false
	var persistErr error
	commit := func() {
		if committed || persistErr != nil {
			return
		}
		if _, err := s.store.CreateMessage(convID, "user", req.Content, nil, user.DisplayName, nil, nil, ""); err != nil {
			persistErr = err
			return
		}
		committed = true
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")
		startTime = time.Now()
		nameJSON, _ := json.Marshal(map[string]string{"name": assistantName})
		fmt.Fprintf(w, "data: %s\n\n", nameJSON)
		flusher.Flush()
	}

	toolLoop := messageToolLoop{
		server:          s,
		ctx:             ctx,
		writer:          w,
		flusher:         flusher,
		provider:        provider,
		request:         llm.Request{Model: modelName, Tools: toolsArg, CacheKey: fmt.Sprintf("lemon-conv-%d", convID)},
		messages:        chatMsgs,
		wireLog:         tokenLog,
		onStart:         commit,
		committed:       func() bool { return committed },
		persistError:    func() error { return persistErr },
		conversationID:  convID,
		assistantName:   assistantName,
		characterID:     usedCharacterID,
		maxLoops:        s.cfg.Server.MaxToolLoops,
		responseStarted: &startTime,
		toolContext: ToolContext{
			ModelName:       modelName,
			ModelServer:     modelServer,
			ResponseTimeout: responseTimeout,
			SearXNGURL:      s.cfg.SearXNG.URL,
			Timezone:        s.cfg.Server.Timezone,
			UserID:          user.ID,
			ConversationID:  convID,
			Store:           s.store,
			DataDir:         s.cfg.Server.DataDir,
			Hub:             s.hub,
		},
	}
	finalContent, finalStats, ok := toolLoop.run()
	if !ok {
		return
	}

	if finalStats != nil {
		statsJSON, _ := json.Marshal(map[string]any{
			"stats": map[string]int64{
				"prompt_tokens":     finalStats.PromptTokens,
				"completion_tokens": finalStats.CompletionTokens,
				"cached_tokens":     finalStats.CachedTokens,
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

		s.triggerConversationTitle(conv, usedCharacter, history)
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func (s *Server) handleGetMessageContext(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	convID, ok := pathID(w, r)
	if !ok {
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

	var prefix []chatMsg

	if conv.CharacterID != nil {
		var char *store.Character
		char, prefix = s.resolveCharacter(w, user, *conv.CharacterID, prefix)
		if char == nil {
			return
		}
	}

	chatMsgs := buildChatMsgs(prefix, history)
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
	convID, ok := pathID(w, r)
	if !ok {
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
		writeError(w, http.StatusUnprocessableEntity, "character has no first message")
		return
	}

	// If a character override was provided, update the conversation to use it.
	if req.CharacterID != nil {
		if err := s.store.UpdateConversationAfterMessage(convID, nil, charID); err != nil {
			internalError(w, err)
			return
		}
	}

	log.Printf("Sending first message conversation_id=%d user_id=%d", convID, user.ID)
	content := substituteVars(*char.FirstMessage, char.Name, resolveUserName(user))
	msg, err := s.store.CreateMessage(convID, "assistant", content, charID, &char.Name, nil, nil, "")
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}
