package tasks

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// titleServer returns an httptest server that replies with content as the
// assistant message of a single chat-completion choice.
func titleServer(t *testing.T, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := `{"choices":[{"message":{"content":` + jsonString(content) + `}}]}`
		_, _ = w.Write([]byte(resp))
	}))
}

// jsonString quotes s as a JSON string literal.
func jsonString(s string) string {
	var b []byte
	b = append(b, '"')
	for _, r := range s {
		switch r {
		case '"':
			b = append(b, '\\', '"')
		case '\\':
			b = append(b, '\\', '\\')
		case '\n':
			b = append(b, '\\', 'n')
		default:
			b = append(b, string(r)...)
		}
	}
	b = append(b, '"')
	return string(b)
}

func TestRequestTitleTrimsWhitespaceAndQuotes(t *testing.T) {
	srv := titleServer(t, "  \"My Great Title\" \n")
	defer srv.Close()

	got, err := requestTitle("sys", "content", srv.URL, "model", "", 5*time.Second)
	if err != nil {
		t.Fatalf("requestTitle returned error: %v", err)
	}
	if got != "My Great Title" {
		t.Errorf("requestTitle = %q, want %q", got, "My Great Title")
	}
}

func TestRequestTitleEmptyIsError(t *testing.T) {
	srv := titleServer(t, "   \n  ")
	defer srv.Close()

	if _, err := requestTitle("sys", "content", srv.URL, "model", "", 5*time.Second); err == nil {
		t.Fatal("expected an error for empty title, got nil")
	}
}
