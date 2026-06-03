package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/debug"
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

func (s *Server) handleGetCompletion(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := s.store.GetCompletion(id, user.ID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
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
		Title   *string `json:"title"`
		Content *string `json:"content"`
		Model   *string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Title == nil && req.Content == nil && req.Model == nil {
		writeError(w, http.StatusBadRequest, "title, content, or model required")
		return
	}
	if req.Title != nil {
		if err := s.store.UpdateCompletionTitle(id, user.ID, *req.Title); errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		} else if err != nil {
			internalError(w, err)
			return
		}
	}
	if req.Content != nil {
		if err := s.store.UpdateCompletionContent(id, user.ID, *req.Content); errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		} else if err != nil {
			internalError(w, err)
			return
		}
	}
	if req.Model != nil {
		if *req.Model == "" {
			writeError(w, http.StatusBadRequest, "model cannot be empty")
			return
		}
		if err := s.store.UpdateCompletionModel(id, user.ID, *req.Model); errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		} else if err != nil {
			internalError(w, err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRegenerateCompletionTitle(w http.ResponseWriter, r *http.Request) {
	// TODO: implement background title generation once completions have content
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleUndoCompletion(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	content, err := s.store.UndoCompletion(id, user.ID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusConflict, "no snapshot to undo")
		return
	} else if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"content": content})
}

func (s *Server) handleRedoCompletion(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	content, err := s.store.RedoCompletion(id, user.ID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusConflict, "no snapshot to redo")
		return
	} else if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"content": content})
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

func (s *Server) handleRunCompletion(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	comp, err := s.store.GetCompletion(id, user.ID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		internalError(w, err)
		return
	}

	var req struct {
		Content     string   `json:"content"`
		MaxTokens   *int     `json:"max_tokens"`
		Temperature *float64 `json:"temperature"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// Snapshot current content for undo before overwriting it.
	if comp.Content != nil {
		if err := s.store.SetPrevContent(id, user.ID, *comp.Content); err != nil {
			log.Printf("completions: failed to set prev_content for %d: %v", id, err)
		}
	}

	// Save the prompt content before running.
	if err := s.store.UpdateCompletionContent(id, user.ID, req.Content); err != nil {
		internalError(w, err)
		return
	}

	modelServer, err := s.cfg.ServerForModel(comp.Model)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unknown model")
		return
	}
	completionURL := modelServer.APIBase + "/completions"

	modelPayload := map[string]any{
		"model":  comp.Model,
		"prompt": req.Content,
		"stream": true,
		"stream_options": map[string]any{
			"include_usage": true,
		},
	}
	if req.MaxTokens != nil {
		modelPayload["max_tokens"] = *req.MaxTokens
	}
	if req.Temperature != nil {
		modelPayload["temperature"] = *req.Temperature
	}
	payload, _ := json.Marshal(modelPayload)
	debug.Log("completions: POST %s", completionURL)
	debug.Log("completions: payload %s", payload)

	responseTimeout := time.Duration(s.cfg.ResponseTimeoutSeconds) * time.Second
	ctx, cancelResp := context.WithTimeout(r.Context(), responseTimeout)
	defer cancelResp()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", completionURL, bytes.NewReader(payload))
	if err != nil {
		writeError(w, http.StatusBadGateway, "model unreachable")
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if modelServer.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+modelServer.APIKey)
	}

	resp, err := s.modelClient.Do(httpReq)
	if err != nil {
		debug.Log("completions: model request failed: %v", err)
		writeError(w, http.StatusBadGateway, "model unreachable")
		return
	}
	defer resp.Body.Close()
	debug.Log("completions: model responded status=%d", resp.StatusCode)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher := w.(http.Flusher)

	var generated strings.Builder
	var chunkCount int
	var promptTokens, completionTokens int64
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := line[6:]
		if data == "[DONE]" {
			debug.Log("completions: received [DONE] after %d chunks, %d generated chars", chunkCount, generated.Len())
			break
		}
		var chunk struct {
			Choices []struct {
				Text         string  `json:"text"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int64 `json:"prompt_tokens"`
				CompletionTokens int64 `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			debug.Log("completions: failed to parse chunk: %v — raw: %q", err, data)
			continue
		}
		if chunk.Usage != nil {
			promptTokens = chunk.Usage.PromptTokens
			completionTokens = chunk.Usage.CompletionTokens
			debug.Log("completions: usage: prompt=%d completion=%d", promptTokens, completionTokens)
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		if text := chunk.Choices[0].Text; text != "" {
			chunkCount++
			generated.WriteString(text)
			delta, _ := json.Marshal(map[string]string{"delta": text})
			fmt.Fprintf(w, "data: %s\n\n", delta)
			flusher.Flush()
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("completions: SSE scanner error for completion %d: %v", id, err)
	}

	// Save the final content (prompt + generated).
	if generated.Len() > 0 {
		finalContent := req.Content + generated.String()
		debug.Log("completions: saving final content (%d prompt + %d generated = %d total chars)", len(req.Content), generated.Len(), len(finalContent))
		if err := s.store.UpdateCompletionContent(id, user.ID, finalContent); err != nil {
			log.Printf("completions: failed to save final content for %d: %v", id, err)
		}
	}

	if promptTokens > 0 || completionTokens > 0 {
		if err := s.store.UpdateTokenCounts(id, user.ID, promptTokens, completionTokens); err != nil {
			log.Printf("completions: failed to save token counts for %d: %v", id, err)
		}
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}
