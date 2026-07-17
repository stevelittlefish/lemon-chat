package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/stevelittlefish/lemon-chat/internal/config"
)

// This file is the provider seam: one neutral streaming interface that hides the
// wire protocol behind a factory. Today two surfaces are implemented — OpenAI
// chat-completions and the OpenAI Responses API (Codex) — which share a
// transport and differ only in request body and stream shape, so both are served
// by httpProvider. Genuinely different providers (native Ollama, Anthropic) will
// be added as separate Provider implementations without touching call sites.
//
// See docs/provider_abstraction.md for the design and rationale.

// Request is a protocol-neutral generation request. Messages and Tools stay in
// the OpenAI chat-completions JSON shape (any value that marshals to it), so
// callers assemble them as they do today; each provider lowers them to its own
// wire body. Prior reasoning/thinking is never carried here — it is persisted
// for display but excluded from model input.
type Request struct {
	Model           string
	Messages        any      // chat-completions-shaped messages (e.g. []chatMsg)
	Tools           any      // chat-completions-shaped tool defs, or nil
	MaxTokens       int      // 0 ⇒ omit
	Temperature     *float64 // nil ⇒ omit (Codex rejects it regardless)
	ReasoningEffort string   // "", "low", "medium", "high" — reserved; not yet populated
}

// ToolCall is one completed function call parsed from the stream.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

// Handler receives streaming events. All callbacks are optional and invoked
// synchronously on the decode goroutine.
type Handler struct {
	// OnStart fires once the model responds successfully (2xx), before any delta.
	// It is the "connection established" signal a caller uses to commit to a
	// streamed response (e.g. write SSE headers) only after the model is reachable.
	OnStart func()
	// OnText delivers assistant text deltas.
	OnText func(delta string)
	// OnThinking delivers reasoning/thinking deltas. Reserved: no provider
	// populates it yet (see the streaming-thinking backlog item).
	OnThinking func(delta string)
	// OnRawFrame, if set, receives every raw chat-completions data line (for
	// Responses servers, the post-conversion frame). Used for token logging.
	OnRawFrame func(data string)
}

// Provider streams one assistant turn and lists reachable models. Concrete
// implementations lower a Request to their wire protocol.
type Provider interface {
	Stream(ctx context.Context, req Request, h Handler) (Completion, error)
	ListModels(ctx context.Context) ([]string, error)
}

// NewProvider builds the provider for a model server. token resolves the bearer
// (static api_key or refreshing OAuth) at call time; accountID is the
// chatgpt-account-id header for OAuth/Responses servers ("" otherwise).
func NewProvider(client *http.Client, srv *config.ModelServer, token config.TokenSource, accountID string) Provider {
	return NewOpenAIProvider(client, srv.APIBase, srv.UsesResponses(), token, accountID)
}

// NewOpenAIProvider builds an OpenAI chat-completions or Responses provider from
// primitives, for callers that don't hold a *config.ModelServer (e.g. the
// research engine's writer/worker endpoints). responses selects the Responses
// surface; token may be nil (no Authorization header).
func NewOpenAIProvider(client *http.Client, apiBase string, responses bool, token config.TokenSource, accountID string) Provider {
	if token == nil {
		token = config.StaticToken("")
	}
	return &httpProvider{
		client:    client,
		base:      apiBase,
		responses: responses,
		token:     token,
		accountID: accountID,
	}
}

// httpProvider serves the OpenAI chat-completions and Responses surfaces, which
// share transport and differ only in body + stream shape.
type httpProvider struct {
	client    *http.Client
	base      string
	responses bool
	token     config.TokenSource
	accountID string
}

func (p *httpProvider) endpoint() string {
	if p.responses {
		return p.base + "/responses"
	}
	return p.base + "/chat/completions"
}

func (p *httpProvider) Stream(ctx context.Context, req Request, h Handler) (Completion, error) {
	key, err := p.token(ctx)
	if err != nil {
		return Completion{}, err
	}

	var payload []byte
	if p.responses {
		extra := map[string]any{}
		if req.MaxTokens > 0 {
			extra["max_tokens"] = req.MaxTokens
		}
		if payload, err = BuildResponsesBody(req.Model, req.Messages, req.Tools, extra, true); err != nil {
			return Completion{}, err
		}
	} else {
		body := map[string]any{
			"model":          req.Model,
			"messages":       req.Messages,
			"stream":         true,
			"stream_options": map[string]any{"include_usage": true},
		}
		if req.Tools != nil {
			body["tools"] = req.Tools
		}
		if req.MaxTokens > 0 {
			body["max_tokens"] = req.MaxTokens
		}
		if req.Temperature != nil {
			body["temperature"] = *req.Temperature
		}
		payload, _ = json.Marshal(body)
	}

	hreq, err := http.NewRequestWithContext(ctx, "POST", p.endpoint(), bytes.NewReader(payload))
	if err != nil {
		return Completion{}, fmt.Errorf("build request: %w", err)
	}
	hreq.Header.Set("Content-Type", "application/json")
	hreq.Header.Set("Accept", "text/event-stream")
	if key != "" {
		hreq.Header.Set("Authorization", "Bearer "+key)
	}
	if p.accountID != "" {
		hreq.Header.Set("chatgpt-account-id", p.accountID)
	}

	resp, err := p.client.Do(hreq)
	if err != nil {
		return Completion{}, fmt.Errorf("model request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return Completion{}, fmt.Errorf("model server returned %d: %.300s", resp.StatusCode, respBody)
	}
	if h.OnStart != nil {
		h.OnStart()
	}

	var body io.Reader = resp.Body
	if p.responses {
		body = ResponsesToChatSSE(resp.Body)
	}
	return readChatCompletionsStreamFull(body, h)
}

func (p *httpProvider) ListModels(ctx context.Context) ([]string, error) {
	// The ChatGPT/Codex Responses backend exposes no model-enumeration endpoint,
	// so return the built-in roster (no network, no auth needed).
	if p.responses {
		out := append([]string(nil), CodexModels...)
		sort.Strings(out)
		return out, nil
	}
	key, err := p.token(ctx)
	if err != nil {
		return nil, err
	}
	return ListModels(ctx, p.client, p.base, key)
}

// CodexModels is the built-in roster returned for a Responses/Codex server, which
// has no enumeration endpoint. It's a curated starter list — edit to taste; the
// authoritative model IDs are whatever your ChatGPT/Codex plan accepts, and the
// roster here is only a discovery convenience (any ID can be used in a [[model]]
// entry regardless of whether it appears here).
var CodexModels = []string{
	"gpt-5.6-codex",
	"gpt-5.5-codex",
	"gpt-5.4-codex",
	"gpt-5.3-codex",
	"gpt-5.2-codex",
	"gpt-5.1-codex",
	"gpt-5.1-codex-mini",
	"gpt-5.1-codex-max",
}

// readChatCompletionsStreamFull parses a chat-completions SSE body into text,
// tool calls, usage, and finish reason. It is the single parser both surfaces
// use — the Responses stream is converted to chat-completions frames first.
func readChatCompletionsStreamFull(body io.Reader, h Handler) (Completion, error) {
	var text []byte
	var usage *Usage
	var finishReason string
	type pendingCall struct {
		id, name string
		args     []byte
	}
	var calls []*pendingCall

	err := ScanSSE(body, func(data string) error {
		if h.OnRawFrame != nil {
			h.OnRawFrame(data)
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
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage *Usage `json:"usage"`
		}
		if json.Unmarshal([]byte(data), &chunk) != nil {
			return nil
		}
		if chunk.Usage != nil {
			usage = chunk.Usage
		}
		if len(chunk.Choices) == 0 {
			return nil
		}
		choice := chunk.Choices[0]
		if choice.FinishReason != "" {
			finishReason = choice.FinishReason
		}
		if d := choice.Delta.Content; d != "" {
			text = append(text, d...)
			if h.OnText != nil {
				h.OnText(d)
			}
		}
		for _, tc := range choice.Delta.ToolCalls {
			for len(calls) <= tc.Index {
				calls = append(calls, &pendingCall{})
			}
			if tc.ID != "" {
				calls[tc.Index].id = tc.ID
			}
			if tc.Function.Name != "" {
				calls[tc.Index].name = tc.Function.Name
			}
			calls[tc.Index].args = append(calls[tc.Index].args, tc.Function.Arguments...)
		}
		return nil
	})
	if err != nil && len(text) == 0 && len(calls) == 0 {
		return Completion{}, fmt.Errorf("stream read failed: %w", err)
	}
	comp := Completion{Content: string(text), Usage: usage, FinishReason: finishReason}
	for _, c := range calls {
		comp.ToolCalls = append(comp.ToolCalls, ToolCall{ID: c.id, Name: c.name, Arguments: string(c.args)})
	}
	return comp, nil
}
