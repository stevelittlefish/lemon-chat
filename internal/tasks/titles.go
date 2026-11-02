package tasks

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/debug"
	"github.com/stevelittlefish/lemon-chat/internal/llm"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

const titleInterval = 30 * time.Second

// Failed background title attempts back off exponentially per item, from
// titleRetryBase up to titleRetryMax, so a broken model server isn't hit
// every 30 s forever.
const (
	titleRetryBase = time.Minute
	titleRetryMax  = 6 * time.Hour
)

// CompletionAutoTitleMinTokens is the minimum number of completion tokens
// required to trigger automatic title generation for a completion.
const CompletionAutoTitleMinTokens = int64(50)

// ProviderFunc builds the provider for a model server, with the right
// protocol (chat-completions or Responses) and credentials (static key or
// OAuth). The server supplies it so tasks never build raw requests itself.
type ProviderFunc func(srv *config.ModelServer) llm.Provider

// Titler generates conversation and completion titles with the configured
// default model.
type Titler struct {
	st       *store.Store
	cfg      *config.Config
	provider ProviderFunc

	mu      sync.Mutex
	backoff map[string]titleBackoff // keyed "conv:<id>" / "comp:<id>"
}

type titleBackoff struct {
	failures    int
	nextAttempt time.Time
}

func NewTitler(st *store.Store, cfg *config.Config, provider ProviderFunc) *Titler {
	return &Titler{st: st, cfg: cfg, provider: provider, backoff: map[string]titleBackoff{}}
}

// Start runs the background worker that titles eligible untitled
// conversations and completions.
func (t *Titler) Start(onConvTitled func(int64, string), onCompTitled func(int64, string)) {
	go func() {
		t.generateTitles(onConvTitled)
		t.generateCompletionTitles(onCompTitled)
		ticker := time.NewTicker(titleInterval)
		defer ticker.Stop()
		for range ticker.C {
			t.generateTitles(onConvTitled)
			t.generateCompletionTitles(onCompTitled)
		}
	}()
}

// shouldAttempt reports whether the background worker may try key now.
func (t *Titler) shouldAttempt(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	b, ok := t.backoff[key]
	return !ok || !time.Now().Before(b.nextAttempt)
}

// recordResult clears key's backoff on success or extends it on failure.
func (t *Titler) recordResult(key string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err == nil {
		delete(t.backoff, key)
		return
	}
	b := t.backoff[key]
	delay := titleRetryBase << b.failures
	if delay > titleRetryMax || delay <= 0 {
		delay = titleRetryMax
	} else {
		b.failures++
	}
	b.nextAttempt = time.Now().Add(delay)
	t.backoff[key] = b
	log.Printf("title worker: %s failed, next attempt in %s", key, delay)
}

// titleModel is the resolved provider and model for title generation.
type titleModel struct {
	provider llm.Provider
	name     string
	timeout  time.Duration
}

// resolveTitleModel resolves the provider for the title-generation model (the
// configured default model). ok is false when generation can't proceed.
// skipDesc, if non-empty, is named in the log line emitted when no default
// model is configured; pass "" to skip silently.
func (t *Titler) resolveTitleModel(skipDesc string) (titleModel, bool) {
	modelName := t.cfg.Server.DefaultModel
	if modelName == "" {
		if skipDesc != "" {
			log.Printf("title worker: no default_model configured, skipping %s", skipDesc)
		}
		return titleModel{}, false
	}
	srv, err := t.cfg.ServerForModel(modelName)
	if err != nil {
		log.Printf("title worker: %v", err)
		return titleModel{}, false
	}
	return titleModel{
		provider: t.provider(srv),
		name:     modelName,
		timeout:  time.Duration(t.cfg.Server.ResponseTimeoutSeconds) * time.Second,
	}, true
}

// GenerateForConversation generates a title for convID and persists it.
// It runs the generation in a goroutine and calls onTitled on success.
func (t *Titler) GenerateForConversation(convID int64, onTitled func(int64, string)) {
	debug.Log("title: GenerateForConversation triggered for conversation %d", convID)
	go func() {
		m, ok := t.resolveTitleModel(fmt.Sprintf("conversation %d", convID))
		if !ok {
			return
		}
		debug.Log("title: generating title for conversation %d using model %q", convID, m.name)
		title, err := t.generateTitle(m, convID)
		if err != nil {
			log.Printf("title worker: conversation %d: %v", convID, err)
			return
		}
		if err := t.st.UpdateConversationTitle(convID, title); err != nil {
			log.Printf("title worker: update %d: %v", convID, err)
			return
		}
		log.Printf("title worker: titled conversation %d: %q", convID, title)
		if onTitled != nil {
			onTitled(convID, title)
		}
	}()
}

// GenerateForCompletion generates a title for compID and persists it.
// It runs in a goroutine and calls onTitled on success. Always generates,
// regardless of whether a title already exists — callers should guard if needed.
func (t *Titler) GenerateForCompletion(compID int64, onTitled func(int64, string)) {
	debug.Log("title: GenerateForCompletion triggered for completion %d", compID)
	go func() {
		m, ok := t.resolveTitleModel(fmt.Sprintf("completion %d", compID))
		if !ok {
			return
		}

		userID, content, err := t.st.GetCompletionForTitle(compID)
		if err != nil {
			log.Printf("title worker: completion %d: %v", compID, err)
			return
		}
		if content == nil || *content == "" {
			log.Printf("title worker: completion %d has no content", compID)
			return
		}
		title, err := generateCompletionTitle(m, *content)
		if err != nil {
			log.Printf("title worker: completion %d: %v", compID, err)
			return
		}
		if err := t.st.UpdateCompletionTitle(compID, userID, title); err != nil {
			log.Printf("title worker: update completion %d: %v", compID, err)
			return
		}
		log.Printf("title worker: titled completion %d: %q", compID, title)
		if onTitled != nil {
			onTitled(compID, title)
		}
	}()
}

func (t *Titler) generateCompletionTitles(onTitled func(int64, string)) {
	debug.Log("title: running completion title job")
	m, ok := t.resolveTitleModel("")
	if !ok {
		return
	}

	ids, err := t.st.ListUntitledEligibleCompletions(CompletionAutoTitleMinTokens)
	if err != nil {
		log.Printf("title worker: list eligible completions: %v", err)
		return
	}
	if len(ids) > 0 {
		debug.Log("title: found %d untitled eligible completion(s): %v", len(ids), ids)
	} else {
		debug.Log("title: no untitled eligible completions")
	}

	for _, id := range ids {
		key := fmt.Sprintf("comp:%d", id)
		if !t.shouldAttempt(key) {
			continue
		}
		userID, content, err := t.st.GetCompletionForTitle(id)
		if err != nil || content == nil || *content == "" {
			continue
		}
		title, err := generateCompletionTitle(m, *content)
		t.recordResult(key, err)
		if err != nil {
			log.Printf("title worker: completion %d: %v", id, err)
			continue
		}
		if err := t.st.UpdateCompletionTitle(id, userID, title); err != nil {
			log.Printf("title worker: update completion %d: %v", id, err)
		} else {
			log.Printf("title worker: titled completion %d: %q", id, title)
			if onTitled != nil {
				onTitled(id, title)
			}
		}
	}
}

func generateCompletionTitle(m titleModel, content string) (string, error) {
	if len(content) > titleTranscriptLimit {
		content = content[:titleTranscriptLimit]
	}
	debug.Log("title: requesting completion title (model=%s)", m.name)
	return requestTitle(m, completionTitlePrompt, content)
}

func (t *Titler) generateTitles(onTitled func(int64, string)) {
	debug.Log("title: running conversation title job")
	m, ok := t.resolveTitleModel("")
	if !ok {
		return
	}

	ids, err := t.st.ListUntitledEligible()
	if err != nil {
		log.Printf("title worker: list eligible: %v", err)
		return
	}
	if len(ids) > 0 {
		debug.Log("title: found %d untitled eligible conversation(s): %v", len(ids), ids)
	} else {
		debug.Log("title: no untitled eligible conversations")
	}

	for _, id := range ids {
		key := fmt.Sprintf("conv:%d", id)
		if !t.shouldAttempt(key) {
			continue
		}
		title, err := t.generateTitle(m, id)
		t.recordResult(key, err)
		if err != nil {
			log.Printf("title worker: conversation %d: %v", id, err)
			continue
		}
		if err := t.st.UpdateConversationTitle(id, title); err != nil {
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

const completionTitlePrompt = "Generate a short title (at most 6 words) for the following text. Respond with only the title — no quotes, no trailing punctuation, no explanation."

func (t *Titler) generateTitle(m titleModel, convID int64) (string, error) {
	msgs, err := t.st.ListMessages(convID)
	if err != nil {
		return "", err
	}
	if len(msgs) == 0 {
		return "", fmt.Errorf("no messages")
	}

	transcript := buildTranscript(msgs)

	systemPrompt := defaultTitlePrompt
	if custom, err := t.st.GetConversationTitlePrompt(convID); err == nil && custom != "" {
		systemPrompt = custom
	}

	debug.Log("title: requesting conversation title (model=%s, conv=%d)", m.name, convID)
	return requestTitle(m, systemPrompt, transcript)
}

// requestTitle asks the model for a short title given a system prompt and the
// text to title, then normalises the response (trims surrounding whitespace
// and quotes). Returns an error if the model returns an empty title.
func requestTitle(m titleModel, systemPrompt, content string) (string, error) {
	msgs := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: content},
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	comp, err := m.provider.Stream(ctx, llm.Request{Model: m.name, Messages: msgs, MaxTokens: 20}, llm.Handler{})
	if err != nil {
		return "", err
	}

	title := strings.TrimSpace(comp.Content)
	title = strings.Trim(title, `"'`)
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("empty title from model")
	}
	return title, nil
}
