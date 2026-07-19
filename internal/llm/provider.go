package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

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

// StatusError is a non-2xx response from a model server. It is returned instead
// of a bare fmt.Errorf so callers can distinguish a permanent rejection (a bad
// model name, a malformed request) from a transient one worth retrying.
type StatusError struct {
	Status int
	Body   string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("model server returned %d: %s", e.Status, e.Body)
}

// Transient reports whether the status is worth retrying: any 5xx, plus the two
// 4xx codes that mean "try again" rather than "this request is wrong".
func (e *StatusError) Transient() bool {
	return e.Status >= 500 || e.Status == http.StatusRequestTimeout || e.Status == http.StatusTooManyRequests
}

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
	// CacheKey is a stable identifier (e.g. per conversation) sent as the
	// Responses prompt_cache_key so a conversation's requests route to the same
	// prompt cache. Empty ⇒ omitted. Only used on the Responses surface.
	CacheKey string
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
	// Responses servers, the post-conversion frame). Useful to consumers that
	// need the provider-neutral stream; WireLog captures the actual HTTP stream.
	OnRawFrame func(data string)
	// WireLog receives a replay-oriented HTTP transcript: the request URL, safe
	// headers, exact JSON body, response status and headers, and raw response
	// body. Credential-bearing headers are redacted. Callers own and close it.
	WireLog io.Writer
	// ErrorLog, if set, receives the same redacted transcript as WireLog but only
	// when the request fails (transport error, non-2xx, or a stream error) — so a
	// long-lived process can leave it on and only ever write failures to disk.
	// The transcript is buffered in memory (capped at errorLogCapBytes) and
	// flushed here in a single Write on failure. Callers own it and must make
	// concurrent Writes safe. Independent of WireLog; either, both, or neither.
	ErrorLog io.Writer
}

// errorLogCapBytes bounds the in-memory transcript buffer kept per request for
// the on-error dump. Transcripts are normally a few KB; the cap only matters for
// pathologically long streams, where the most recent bytes (which carry the
// failing frame) are kept and the head is dropped with a truncation marker.
const errorLogCapBytes = 1 << 20 // 1 MiB

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
		if req.CacheKey != "" {
			extra["prompt_cache_key"] = req.CacheKey
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
	// The Codex backend only engages prompt caching when a stable session-id
	// header is present (the body prompt_cache_key alone is not enough — verified
	// against the live endpoint). The header name is "session-id" (hyphen), matching
	// the codex-rs client (codex-api build_session_headers). Send the request's
	// cache key as the session id.
	if p.responses && req.CacheKey != "" {
		hreq.Header.Set("session-id", req.CacheKey)
	}

	// transcript is the always-captured sink: WireLog (if any) plus an in-memory
	// capped buffer that is flushed to ErrorLog only on failure. flushOnError
	// does that flush; it is a no-op when ErrorLog is nil.
	transcript, errBuf := h.transcriptSinks()
	flushOnError := func(cause error) {
		if h.ErrorLog == nil || errBuf == nil {
			return
		}
		var dump bytes.Buffer
		fmt.Fprintf(&dump, "\n===== model error model=%s endpoint=%s: %v =====\n",
			req.Model, p.endpoint(), cause)
		dump.Write(errBuf.Bytes())
		fmt.Fprint(&dump, "\n===== end model error =====\n")
		h.ErrorLog.Write(dump.Bytes())
	}

	writeWireRequest(transcript, hreq, payload)

	resp, err := p.client.Do(hreq)
	if err != nil {
		writeWireError(transcript, err)
		wrapped := fmt.Errorf("model request failed: %w", err)
		flushOnError(wrapped)
		return Completion{}, wrapped
	}
	defer resp.Body.Close()
	writeWireResponse(transcript, resp)
	if transcript != nil {
		resp.Body = io.NopCloser(io.TeeReader(resp.Body, transcript))
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		writeWireEnd(transcript)
		wrapped := error(&StatusError{Status: resp.StatusCode, Body: fmt.Sprintf("%.300s", respBody)})
		flushOnError(wrapped)
		return Completion{}, wrapped
	}
	if h.OnStart != nil {
		h.OnStart()
	}

	var body io.Reader = resp.Body
	if p.responses {
		body = ResponsesToChatSSE(resp.Body)
	}
	completion, err := readChatCompletionsStreamFull(body, h)
	writeWireEnd(transcript)
	if err != nil {
		flushOnError(err)
	}
	return completion, err
}

// transcriptSinks returns the combined writer that the wire-transcript helpers
// write to, plus the capped buffer feeding ErrorLog (nil when ErrorLog is off).
// The returned writer is nil when neither WireLog nor ErrorLog is set, so the
// helpers stay no-ops and no capture happens.
func (h Handler) transcriptSinks() (io.Writer, *capBuffer) {
	var sinks []io.Writer
	var errBuf *capBuffer
	if h.WireLog != nil {
		sinks = append(sinks, h.WireLog)
	}
	if h.ErrorLog != nil {
		errBuf = &capBuffer{cap: errorLogCapBytes}
		sinks = append(sinks, errBuf)
	}
	switch len(sinks) {
	case 0:
		return nil, nil
	case 1:
		return sinks[0], errBuf
	default:
		return io.MultiWriter(sinks...), errBuf
	}
}

// capBuffer is an io.Writer that retains at most cap bytes, keeping the most
// recent when it overflows (the failing frame is at the end of a stream). It is
// written from a single goroutine per request, so it needs no locking.
type capBuffer struct {
	buf       []byte
	cap       int
	truncated int64
}

func (c *capBuffer) Write(p []byte) (int, error) {
	n := len(p)
	c.buf = append(c.buf, p...)
	if len(c.buf) > c.cap {
		drop := len(c.buf) - c.cap
		c.truncated += int64(drop)
		c.buf = append(c.buf[:0], c.buf[drop:]...)
	}
	return n, nil
}

// Bytes returns the retained transcript, prefixed with a marker when earlier
// bytes were dropped to stay under the cap.
func (c *capBuffer) Bytes() []byte {
	if c.truncated == 0 {
		return c.buf
	}
	prefix := fmt.Sprintf("[... %d earlier byte(s) truncated to stay under cap ...]\n", c.truncated)
	return append([]byte(prefix), c.buf...)
}

func writeWireRequest(w io.Writer, req *http.Request, payload []byte) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "\n--- request ---\n%s %s\n", req.Method, req.URL.String())
	writeWireHeaders(w, req.Header)
	fmt.Fprintf(w, "\n%s\n", payload)
}

func writeWireResponse(w io.Writer, resp *http.Response) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "\n--- response ---\n%s\n", resp.Status)
	writeWireHeaders(w, resp.Header)
	fmt.Fprintln(w)
}

func writeWireHeaders(w io.Writer, headers http.Header) {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		values := headers.Values(key)
		if sensitiveWireHeader(key) {
			values = []string{"[redacted]"}
		}
		for _, value := range values {
			fmt.Fprintf(w, "%s: %s\n", key, value)
		}
	}
}

func sensitiveWireHeader(key string) bool {
	switch strings.ToLower(key) {
	case "authorization", "proxy-authorization", "cookie", "set-cookie", "x-api-key", "chatgpt-account-id":
		return true
	default:
		return false
	}
}

func writeWireError(w io.Writer, err error) {
	if w != nil {
		fmt.Fprintf(w, "\n--- transport error ---\n%s\n", err)
	}
}

func writeWireEnd(w io.Writer) {
	if w != nil {
		fmt.Fprintln(w, "\n--- end response ---")
	}
}

func (p *httpProvider) ListModels(ctx context.Context) ([]string, error) {
	key, err := p.token(ctx)
	if err != nil {
		return nil, err
	}
	if p.responses {
		return listCodexModels(ctx, p.client, p.base, key, p.accountID)
	}
	return ListModels(ctx, p.client, p.base, key)
}

// listCodexModels fetches the Codex model catalog (GET {base}/models) using the
// OAuth token and account header. Unlike the OpenAI-compatible /models
// ({"data":[{"id":…}]}), the Codex backend returns {"models":[{"slug":…}]}, so
// it needs its own parse. The list is live and authoritative — the ids depend on
// the linked account's plan.
// codexClientVersion is sent as the required client_version query param on the
// Codex /models catalog request. The backend requires the field; any valid
// semver is accepted.
const codexClientVersion = "0.0.0"

func listCodexModels(ctx context.Context, client *http.Client, apiBase, token, accountID string) ([]string, error) {
	url := strings.TrimRight(apiBase, "/") + "/models?client_version=" + codexClientVersion
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if accountID != "" {
		req.Header.Set("chatgpt-account-id", accountID)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("model request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("model server returned %d: %.300s", resp.StatusCode, body)
	}
	var result struct {
		Models []struct {
			Slug string `json:"slug"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	models := make([]string, 0, len(result.Models))
	for _, m := range result.Models {
		if m.Slug != "" {
			models = append(models, m.Slug)
		}
	}
	sort.Strings(models)
	return models, nil
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
