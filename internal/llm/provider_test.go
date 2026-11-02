package llm

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stevelittlefish/lemon-chat/internal/config"
)

func TestProviderWireLogCapturesRawTranscriptAndRedactsCredentials(t *testing.T) {
	const rawEvent = `{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":12,"output_tokens":3,"total_tokens":15,"input_tokens_details":{"cached_tokens":8}}}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("session-id"); got != "conversation-73" {
			t.Errorf("session-id = %q", got)
		}
		if got := r.Header.Get("session_id"); got != "" {
			t.Errorf("unexpected session_id header = %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %s\n\n", rawEvent)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(server.Client(), server.URL, true, config.StaticToken("oauth-secret"), "account-secret")
	var transcript bytes.Buffer
	_, err := provider.Stream(context.Background(), Request{
		Model:    "gpt-test",
		Messages: []map[string]any{{"role": "user", "content": "cache this exact prompt"}},
		CacheKey: "conversation-73",
	}, Handler{WireLog: &transcript})
	if err != nil {
		t.Fatal(err)
	}

	got := transcript.String()
	for _, want := range []string{
		"POST " + server.URL + "/responses",
		"Authorization: [redacted]",
		"Chatgpt-Account-Id: [redacted]",
		"Session-Id: conversation-73",
		`"prompt_cache_key":"conversation-73"`,
		`"text":"cache this exact prompt"`,
		"200 OK",
		"data: " + rawEvent,
		"--- end response ---",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("transcript missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "oauth-secret") || strings.Contains(got, "account-secret") {
		t.Fatalf("transcript leaked credentials:\n%s", got)
	}
}

func TestProviderErrorLogWritesOnlyOnFailure(t *testing.T) {
	// First a success (must not write to ErrorLog), then a 500 (must).
	var fail bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"detail":"boom"}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %s\n\n", `{"type":"response.completed","response":{"status":"completed"}}`)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(server.Client(), server.URL, true, config.StaticToken("oauth-secret"), "acct")
	var errLog bytes.Buffer
	req := Request{Model: "gpt-test", Messages: []map[string]any{{"role": "user", "content": "hi"}}}

	if _, err := provider.Stream(context.Background(), req, Handler{ErrorLog: &errLog}); err != nil {
		t.Fatalf("success call errored: %v", err)
	}
	if errLog.Len() != 0 {
		t.Fatalf("ErrorLog written on success:\n%s", errLog.String())
	}

	fail = true
	if _, err := provider.Stream(context.Background(), req, Handler{ErrorLog: &errLog}); err == nil {
		t.Fatal("expected error on 500")
	}
	got := errLog.String()
	for _, want := range []string{"model error", "500", "boom", "end model error"} {
		if !strings.Contains(got, want) {
			t.Errorf("error dump missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "oauth-secret") {
		t.Fatalf("error dump leaked credentials:\n%s", got)
	}
}

func TestCapBufferKeepsTailAndMarksTruncation(t *testing.T) {
	c := &capBuffer{cap: 10}
	c.Write([]byte("abcdef"))
	c.Write([]byte("ghijklmn")) // total 14, cap 10 → drop first 4 ("abcd")
	got := string(c.Bytes())
	if !strings.HasSuffix(got, "efghijklmn") {
		t.Errorf("tail not retained: %q", got)
	}
	if !strings.Contains(got, "truncated") {
		t.Errorf("missing truncation marker: %q", got)
	}
}

// failingReader yields data and then a read error, like a dropped connection.
type failingReader struct {
	data []byte
	err  error
}

func (f *failingReader) Read(p []byte) (int, error) {
	if len(f.data) == 0 {
		return 0, f.err
	}
	n := copy(p, f.data)
	f.data = f.data[n:]
	return n, nil
}

func TestChatStreamReturnsPartialOutputWithError(t *testing.T) {
	body := &failingReader{
		data: []byte(`data: {"choices":[{"delta":{"content":"Half an ans"}}]}` + "\n\n"),
		err:  fmt.Errorf("connection reset"),
	}
	comp, err := readChatCompletionsStreamFull(body, Handler{})
	if err == nil {
		t.Fatal("mid-stream failure reported as success")
	}
	if comp.Content != "Half an ans" {
		t.Fatalf("partial content = %q", comp.Content)
	}
}

func TestChatStreamIgnoresOutOfRangeToolIndex(t *testing.T) {
	sse := `data: {"choices":[{"delta":{"tool_calls":[{"index":1000000000,"id":"x","function":{"name":"evil"}}]}}]}` + "\n\n" +
		`data: {"choices":[{"delta":{"tool_calls":[{"index":-1,"id":"y","function":{"name":"evil"}}]}}]}` + "\n\n" +
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"ok","function":{"name":"get_time","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}` + "\n\n" +
		"data: [DONE]\n\n"
	comp, err := readChatCompletionsStreamFull(strings.NewReader(sse), Handler{})
	if err != nil {
		t.Fatal(err)
	}
	if len(comp.ToolCalls) != 1 || comp.ToolCalls[0].Name != "get_time" {
		t.Fatalf("tool calls = %+v, want only get_time", comp.ToolCalls)
	}
}

func TestProviderErrorLogWritesOnMidStreamFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %s\n\n", `{"choices":[{"delta":{"content":"partial"}}]}`)
		w.(http.Flusher).Flush()
		// Drop the connection mid-stream (chunked body never terminated).
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			conn.Close()
		}
	}))
	defer server.Close()

	provider := NewOpenAIProvider(server.Client(), server.URL, false, config.StaticToken("k"), "")
	var errLog bytes.Buffer
	comp, err := provider.Stream(context.Background(), Request{Model: "m", Messages: []map[string]any{{"role": "user", "content": "hi"}}}, Handler{ErrorLog: &errLog})
	if err == nil {
		t.Fatal("expected error for dropped stream")
	}
	if comp.Content != "partial" {
		t.Fatalf("partial content = %q", comp.Content)
	}
	if !strings.Contains(errLog.String(), "model error") {
		t.Fatalf("mid-stream failure not written to error log:\n%s", errLog.String())
	}
}
