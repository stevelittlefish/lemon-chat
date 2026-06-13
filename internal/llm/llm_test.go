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
		w.Write([]byte(`{"choices":[{"message":{"content":"hello world"}}]}`))
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

func TestChatCompleteErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	defer srv.Close()

	if _, err := ChatComplete(context.Background(), http.DefaultClient, srv.URL, "", "m", []Message{}, nil); err == nil {
		t.Fatal("ChatComplete error = nil, want non-nil for 500 status")
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
