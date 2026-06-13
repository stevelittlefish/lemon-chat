package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/debug"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

const titleInterval = 30 * time.Second

// CompletionAutoTitleMinTokens is the minimum number of completion tokens
// required to trigger automatic title generation for a completion.
const CompletionAutoTitleMinTokens = int64(50)

func StartTitleWorker(st *store.Store, cfg *config.Config, onConvTitled func(int64, string), onCompTitled func(int64, string)) {
	go func() {
		generateTitles(st, cfg, onConvTitled)
		generateCompletionTitles(st, cfg, onCompTitled)
		ticker := time.NewTicker(titleInterval)
		defer ticker.Stop()
		for range ticker.C {
			generateTitles(st, cfg, onConvTitled)
			generateCompletionTitles(st, cfg, onCompTitled)
		}
	}()
}

// GenerateTitleForConversation generates a title for convID and persists it.
// It runs the generation in a goroutine and calls onTitled on success.
func GenerateTitleForConversation(st *store.Store, cfg *config.Config, convID int64, onTitled func(int64, string)) {
	debug.Log("title: GenerateTitleForConversation triggered for conversation %d", convID)
	go func() {
		modelName := cfg.Server.DefaultModel
		if modelName == "" {
			log.Printf("title worker: no default_model configured, skipping conversation %d", convID)
			return
		}
		srv, err := cfg.ServerForModel(modelName)
		if err != nil {
			log.Printf("title worker: %v", err)
			return
		}
		chatURL := srv.APIBase + "/chat/completions"
		debug.Log("title: generating title for conversation %d using model %q at %s", convID, modelName, chatURL)
		responseTimeout := time.Duration(cfg.Server.ResponseTimeoutSeconds) * time.Second
		title, err := generateTitle(st, chatURL, modelName, srv.APIKey, convID, responseTimeout)
		if err != nil {
			log.Printf("title worker: conversation %d: %v", convID, err)
			return
		}
		if err := st.UpdateConversationTitle(convID, title); err != nil {
			log.Printf("title worker: update %d: %v", convID, err)
			return
		}
		log.Printf("title worker: titled conversation %d: %q", convID, title)
		if onTitled != nil {
			onTitled(convID, title)
		}
	}()
}

// GenerateTitleForCompletion generates a title for compID and persists it.
// It runs in a goroutine and calls onTitled on success. Always generates,
// regardless of whether a title already exists — callers should guard if needed.
func GenerateTitleForCompletion(st *store.Store, cfg *config.Config, compID int64, onTitled func(int64, string)) {
	debug.Log("title: GenerateTitleForCompletion triggered for completion %d", compID)
	go func() {
		modelName := cfg.Server.DefaultModel
		if modelName == "" {
			log.Printf("title worker: no default_model configured, skipping completion %d", compID)
			return
		}
		srv, err := cfg.ServerForModel(modelName)
		if err != nil {
			log.Printf("title worker: %v", err)
			return
		}
		chatURL := srv.APIBase + "/chat/completions"
		responseTimeout := time.Duration(cfg.Server.ResponseTimeoutSeconds) * time.Second

		userID, content, err := st.GetCompletionForTitle(compID)
		if err != nil {
			log.Printf("title worker: completion %d: %v", compID, err)
			return
		}
		if content == nil || *content == "" {
			log.Printf("title worker: completion %d has no content", compID)
			return
		}
		title, err := generateCompletionTitle(*content, chatURL, modelName, srv.APIKey, responseTimeout)
		if err != nil {
			log.Printf("title worker: completion %d: %v", compID, err)
			return
		}
		if err := st.UpdateCompletionTitle(compID, userID, title); err != nil {
			log.Printf("title worker: update completion %d: %v", compID, err)
			return
		}
		log.Printf("title worker: titled completion %d: %q", compID, title)
		if onTitled != nil {
			onTitled(compID, title)
		}
	}()
}

func generateCompletionTitles(st *store.Store, cfg *config.Config, onTitled func(int64, string)) {
	debug.Log("title: running completion title job")
	modelName := cfg.Server.DefaultModel
	if modelName == "" {
		return
	}
	srv, err := cfg.ServerForModel(modelName)
	if err != nil {
		log.Printf("title worker: %v", err)
		return
	}
	chatURL := srv.APIBase + "/chat/completions"
	responseTimeout := time.Duration(cfg.Server.ResponseTimeoutSeconds) * time.Second

	ids, err := st.ListUntitledEligibleCompletions(CompletionAutoTitleMinTokens)
	if err != nil {
		log.Printf("title worker: list eligible completions: %v", err)
		return
	}
	if len(ids) > 0 {
		log.Printf("title worker: found %d untitled eligible completion(s): %v", len(ids), ids)
	} else {
		debug.Log("title: no untitled eligible completions")
	}

	for _, id := range ids {
		userID, content, err := st.GetCompletionForTitle(id)
		if err != nil || content == nil || *content == "" {
			continue
		}
		title, err := generateCompletionTitle(*content, chatURL, modelName, srv.APIKey, responseTimeout)
		if err != nil {
			log.Printf("title worker: completion %d: %v", id, err)
			continue
		}
		if err := st.UpdateCompletionTitle(id, userID, title); err != nil {
			log.Printf("title worker: update completion %d: %v", id, err)
		} else {
			log.Printf("title worker: titled completion %d: %q", id, title)
			if onTitled != nil {
				onTitled(id, title)
			}
		}
	}
}

func generateCompletionTitle(content, chatURL, modelName, apiKey string, responseTimeout time.Duration) (string, error) {
	if len(content) > titleTranscriptLimit {
		content = content[:titleTranscriptLimit]
	}

	type chatMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	msgs := []chatMsg{
		{Role: "system", Content: "Generate a short title (at most 6 words) for the following text. Respond with only the title — no quotes, no trailing punctuation, no explanation."},
		{Role: "user", Content: content},
	}
	payload, _ := json.Marshal(map[string]any{
		"model":      modelName,
		"messages":   msgs,
		"stream":     false,
		"max_tokens": 20,
	})

	debug.Log("title: completion title payload: %s", payload)
	debug.Log("title: POST %s (model=%s, completion title)", chatURL, modelName)
	ctx, cancel := context.WithTimeout(context.Background(), responseTimeout)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", chatURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("POST %s: %w", chatURL, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("POST %s: %w", chatURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("model server returned %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	title := strings.TrimSpace(result.Choices[0].Message.Content)
	title = strings.Trim(title, `"'`)
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("empty title from model")
	}
	return title, nil
}

func generateTitles(st *store.Store, cfg *config.Config, onTitled func(int64, string)) {
	debug.Log("title: running conversation title job")
	modelName := cfg.Server.DefaultModel
	if modelName == "" {
		return
	}
	srv, err := cfg.ServerForModel(modelName)
	if err != nil {
		log.Printf("title worker: %v", err)
		return
	}
	chatURL := srv.APIBase + "/chat/completions"
	responseTimeout := time.Duration(cfg.Server.ResponseTimeoutSeconds) * time.Second

	ids, err := st.ListUntitledEligible()
	if err != nil {
		log.Printf("title worker: list eligible: %v", err)
		return
	}
	if len(ids) > 0 {
		log.Printf("title worker: found %d untitled eligible conversation(s): %v", len(ids), ids)
	} else {
		debug.Log("title: no untitled eligible conversations")
	}

	for _, id := range ids {
		title, err := generateTitle(st, chatURL, modelName, srv.APIKey, id, responseTimeout)
		if err != nil {
			log.Printf("title worker: conversation %d: %v", id, err)
			continue
		}
		if err := st.UpdateConversationTitle(id, title); err != nil {
			log.Printf("title worker: update %d: %v", id, err)
		} else {
			log.Printf("title worker: titled conversation %d: %q", id, title)
			if onTitled != nil {
				onTitled(id, title)
			}
		}
	}
}

const titleTranscriptLimit = 10000

func buildTranscript(msgs []store.Message) string {
	var sb strings.Builder
	for _, m := range msgs {
		switch m.Role {
		case "user":
			sb.WriteString("User: ")
		case "assistant":
			sb.WriteString("Assistant: ")
		default:
			continue
		}
		sb.WriteString(m.Content)
		sb.WriteString("\n\n")
		if sb.Len() >= titleTranscriptLimit {
			break
		}
	}
	transcript := sb.String()
	if len(transcript) > titleTranscriptLimit {
		transcript = transcript[:titleTranscriptLimit]
	}
	return strings.TrimSpace(transcript)
}

const defaultTitlePrompt = "Generate a short title (at most 6 words) for the following conversation. Respond with only the title — no quotes, no trailing punctuation, no explanation."

func generateTitle(st *store.Store, chatURL, modelName, apiKey string, convID int64, responseTimeout time.Duration) (string, error) {
	msgs, err := st.ListMessages(convID)
	if err != nil {
		return "", err
	}
	if len(msgs) == 0 {
		return "", fmt.Errorf("no messages")
	}

	transcript := buildTranscript(msgs)

	systemPrompt := defaultTitlePrompt
	if custom, err := st.GetConversationTitlePrompt(convID); err == nil && custom != "" {
		systemPrompt = custom
	}

	type chatMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	out := []chatMsg{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: transcript},
	}

	payload, _ := json.Marshal(map[string]any{
		"model":      modelName,
		"messages":   out,
		"stream":     false,
		"max_tokens": 20,
	})

	debug.Log("title: payload for conversation %d: %s", convID, payload)
	debug.Log("title: POST %s (model=%s, conv=%d)", chatURL, modelName, convID)
	ctx, cancel := context.WithTimeout(context.Background(), responseTimeout)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", chatURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("POST %s: %w", chatURL, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("POST %s: %w", chatURL, err)
	}
	defer resp.Body.Close()
	debug.Log("title: response status %d for conversation %d", resp.StatusCode, convID)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	debug.Log("title: response body for conversation %d: %s", convID, body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("model server returned %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	title := strings.TrimSpace(result.Choices[0].Message.Content)
	title = strings.Trim(title, `"'`)
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("empty title from model")
	}
	return title, nil
}
