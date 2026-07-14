package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestScanSSE(t *testing.T) {
	body := strings.Join([]string{
		"data: {\"n\":1}",
		"",
		": comment line ignored",
		"data: {\"n\":2}",
		"data: [DONE]",
		"data: {\"n\":3}", // after [DONE], must not be delivered
	}, "\n")

	var got []string
	err := ScanSSE(strings.NewReader(body), func(data string) error {
		got = append(got, data)
		return nil
	})
	if err != nil {
		t.Fatalf("ScanSSE returned error: %v", err)
	}
	want := []string{`{"n":1}`, `{"n":2}`}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("ScanSSE delivered %v, want %v", got, want)
	}
}

func TestScanSSEStopsOnCallbackError(t *testing.T) {
	body := "data: a\ndata: b\ndata: c\n"
	var got []string
	sentinel := io.EOF
	err := ScanSSE(strings.NewReader(body), func(data string) error {
		got = append(got, data)
		if data == "b" {
			return sentinel
		}
		return nil
	})
	if err != sentinel {
		t.Fatalf("ScanSSE error = %v, want %v", err, sentinel)
	}
	if strings.Join(got, "|") != "a|b" {
		t.Fatalf("ScanSSE delivered %v, want [a b]", got)
	}
}

func TestChatComplete(t *testing.T) {
	var gotPayload map[string]any
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewDecoder(r.Body).Decode(&gotPayload)
		w.Write([]byte(`{"choices":[{"message":{"content":"hello world"}}],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12,"cost":0.00125}}`))
	}))
	defer srv.Close()

	content, err := ChatComplete(context.Background(), http.DefaultClient, srv.URL, "secret-key", "test-model",
		[]Message{{Role: "user", Content: "hi"}},
		map[string]any{"max_tokens": 20})
	if err != nil {
		t.Fatalf("ChatComplete returned error: %v", err)
	}
	if content != "hello world" {
		t.Fatalf("content = %q, want %q", content, "hello world")
	}
	if gotAuth != "Bearer secret-key" {
		t.Fatalf("Authorization = %q, want %q", gotAuth, "Bearer secret-key")
	}
	if gotPayload["stream"] != false {
		t.Fatalf("payload stream = %v, want false", gotPayload["stream"])
	}
	if gotPayload["model"] != "test-model" {
		t.Fatalf("payload model = %v, want test-model", gotPayload["model"])
	}
	if gotPayload["max_tokens"] != float64(20) {
		t.Fatalf("payload max_tokens = %v, want 20", gotPayload["max_tokens"])
	}
}

func TestChatCompleteWithUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"priced"},"finish_reason":"length"}],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12,"cost":0.00125,"completion_tokens_details":{"reasoning_tokens":1}}}`))
	}))
	defer srv.Close()
	result, err := ChatCompleteWithUsage(context.Background(), http.DefaultClient, srv.URL, "", "model", []Message{}, nil)
	if err != nil || result.UsageCost() == nil || *result.UsageCost() != 0.00125 {
		t.Fatalf("result = %+v, err = %v", result, err)
	}
	if result.FinishReason != "length" || !strings.Contains(string(result.RawUsage), "reasoning_tokens") {
		t.Fatalf("completion metadata not retained: %+v raw=%s", result, result.RawUsage)
	}
}

func TestChatCompleteStream(t *testing.T) {
	var gotPayload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotPayload)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hello \"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"world\"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":2,\"total_tokens\":6,\"cost\":0.0007}}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	var deltas []string
	result, err := ChatCompleteStreamWithUsage(context.Background(), http.DefaultClient, srv.URL, "", "test-model",
		[]Message{{Role: "user", Content: "hi"}}, map[string]any{"max_tokens": 20},
		func(delta string) { deltas = append(deltas, delta) })
	content := result.Content
	if err != nil {
		t.Fatalf("ChatCompleteStream returned error: %v", err)
	}
	if content != "hello world" || strings.Join(deltas, "") != content {
		t.Fatalf("content = %q, deltas = %v", content, deltas)
	}
	if result.UsageCost() == nil || *result.UsageCost() != 0.0007 {
		t.Fatalf("usage = %+v, want cost 0.0007", result.Usage)
	}
	if result.FinishReason != "stop" || !strings.Contains(string(result.RawUsage), "prompt_tokens") {
		t.Fatalf("stream metadata not retained: %+v raw=%s", result, result.RawUsage)
	}
	if gotPayload["stream"] != true {
		t.Fatalf("payload stream = %v, want true", gotPayload["stream"])
	}
}

func TestChatCompleteNoAuthWhenKeyEmpty(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"choices":[{"message":{"content":"x"}}]}`))
	}))
	defer srv.Close()

	if _, err := ChatComplete(context.Background(), http.DefaultClient, srv.URL, "", "m", []Message{{Role: "user", Content: "hi"}}, nil); err != nil {
		t.Fatalf("ChatComplete returned error: %v", err)
	}
	if gotAuth != "" {
		t.Fatalf("Authorization = %q, want empty", gotAuth)
	}
}

func TestCompletionTruncated(t *testing.T) {
	usageAtLimit := &Usage{CompletionTokens: 8192}
	for _, tc := range []struct {
		name       string
		completion Completion
		maxTokens  int
		want       bool
	}{
		{"length", Completion{FinishReason: "length"}, 0, true},
		{"provider max tokens", Completion{FinishReason: "max_tokens"}, 0, true},
		{"usage fallback", Completion{Usage: usageAtLimit}, 8192, true},
		{"ordinary stop", Completion{FinishReason: "stop", Usage: usageAtLimit}, 8192, false},
		{"below limit", Completion{Usage: &Usage{CompletionTokens: 100}}, 8192, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.completion.Truncated(tc.maxTokens); got != tc.want {
				t.Fatalf("Truncated(%d) = %t, want %t", tc.maxTokens, got, tc.want)
			}
		})
	}
}

func TestChatCompleteErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	defer srv.Close()

	if _, err := ChatComplete(context.Background(), http.DefaultClient, srv.URL, "", "m", []Message{}, nil); err == nil {
		t.Fatal("ChatComplete error = nil, want non-nil for 500 status")
	} else if ErrorHTTPStatus(err) != http.StatusInternalServerError {
		t.Fatalf("ErrorHTTPStatus = %d, want 500", ErrorHTTPStatus(err))
	}
}

func TestChatCompleteNoChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	if _, err := ChatComplete(context.Background(), http.DefaultClient, srv.URL, "", "m", []Message{}, nil); err == nil {
		t.Fatal("ChatComplete error = nil, want non-nil for empty choices")
	}
}
