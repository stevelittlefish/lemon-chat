package searx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchParsesResults(t *testing.T) {
	var gotPath, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"title":"First","url":"https://a.test","content":"hello"},{"title":"Second","url":"https://b.test","content":""}]}`))
	}))
	defer srv.Close()

	// Trailing slash on baseURL must be trimmed; page 0 must become pageno=1.
	results, err := Search(context.Background(), srv.URL+"/", "go lang", 0)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if gotUA != "lemon-chat/1.0" {
		t.Errorf("User-Agent = %q, want lemon-chat/1.0", gotUA)
	}
	want := "/search?q=go+lang&format=json&pageno=1"
	if gotPath != want {
		t.Errorf("request path = %q, want %q", gotPath, want)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].Title != "First" || results[0].URL != "https://a.test" || results[0].Content != "hello" {
		t.Errorf("results[0] = %+v", results[0])
	}
}

func TestSearchPageNumber(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer srv.Close()

	if _, err := Search(context.Background(), srv.URL, "x", 3); err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if want := "/search?q=x&format=json&pageno=3"; gotPath != want {
		t.Errorf("request path = %q, want %q", gotPath, want)
	}
}

func TestSearchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	if _, err := Search(context.Background(), srv.URL, "x", 1); err == nil {
		t.Fatal("expected an error on HTTP 500, got nil")
	}
}

func TestSearchBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>not json</html>`))
	}))
	defer srv.Close()

	if _, err := Search(context.Background(), srv.URL, "x", 1); err == nil {
		t.Fatal("expected an error on non-JSON response, got nil")
	}
}
