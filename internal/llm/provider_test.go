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
		if got := r.Header.Get("session_id"); got != "conversation-73" {
			t.Errorf("session_id = %q", got)
		}
		if got := r.Header.Get("session-id"); got != "" {
			t.Errorf("unexpected session-id header = %q", got)
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
		"Session_id: conversation-73",
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
