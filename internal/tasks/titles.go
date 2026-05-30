package tasks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

const titleInterval = 30 * time.Second

func StartTitleWorker(st *store.Store, cfg *config.Config, onTitled func(int64, string)) {
	go func() {
		generateTitles(st, cfg, onTitled)
		ticker := time.NewTicker(titleInterval)
		defer ticker.Stop()
		for range ticker.C {
			generateTitles(st, cfg, onTitled)
		}
	}()
}

// GenerateTitleForConversation generates a title for convID and persists it.
// It runs the generation in a goroutine and calls onTitled on success.
func GenerateTitleForConversation(st *store.Store, cfg *config.Config, convID int64, onTitled func(int64, string)) {
	go func() {
		modelName := cfg.ModelServer.Default
		if modelName == "" {
			return
		}
		chatURL := strings.TrimRight(cfg.ModelServer.APIBase, "/") + "/chat/completions"
		title, err := generateTitle(st, chatURL, modelName, convID)
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

func generateTitles(st *store.Store, cfg *config.Config, onTitled func(int64, string)) {
	modelName := cfg.ModelServer.Default
	if modelName == "" {
		return
	}
	chatURL := strings.TrimRight(cfg.ModelServer.APIBase, "/") + "/chat/completions"

	ids, err := st.ListUntitledEligible()
	if err != nil {
		log.Printf("title worker: list eligible: %v", err)
		return
	}

	for _, id := range ids {
		title, err := generateTitle(st, chatURL, modelName, id)
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

func generateTitle(st *store.Store, chatURL, modelName string, convID int64) (string, error) {
	msgs, err := st.ListMessages(convID)
	if err != nil {
		return "", err
	}
	if len(msgs) == 0 {
		return "", fmt.Errorf("no messages")
	}

	type chatMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	out := []chatMsg{
		{
			Role:    "system",
			Content: "Generate a short title (at most 6 words) for the following conversation. Respond with only the title — no quotes, no trailing punctuation, no explanation.",
		},
	}

	var total int
	for _, m := range msgs {
		content := m.Content
		if total+len(content) > 3000 {
			content = content[:3000-total]
		}
		out = append(out, chatMsg{Role: m.Role, Content: content})
		total += len(content)
		if total >= 3000 {
			break
		}
	}

	payload, _ := json.Marshal(map[string]any{
		"model":      modelName,
		"messages":   out,
		"stream":     false,
		"max_tokens": 20,
	})

	resp, err := http.Post(chatURL, "application/json", bytes.NewReader(payload)) //nolint:gosec
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
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
