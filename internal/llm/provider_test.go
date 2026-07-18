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
