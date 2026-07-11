package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/redditimport"
	"github.com/stevelittlefish/lemon-chat/internal/research"
)

// TestSubscribeAfterFinish verifies that subscribing after finish() has already
// run returns a pre-closed channel (with the last event buffered), so the SSE
// handler receives [DONE] instead of hanging forever.
func TestSubscribeAfterFinish(t *testing.T) {
	run := &researchRun{cancel: func() {}, subs: map[chan []byte]struct{}{}}
	last := []byte(`{"status":"done"}`)

	run.finish(last)
	ch := run.subscribe()

	// Must receive the last event without blocking.
	select {
	case msg, ok := <-ch:
		if string(msg) != string(last) {
			t.Fatalf("got %q, want %q", msg, last)
		}
		_ = ok
	case <-time.After(time.Second):
		t.Fatal("subscribe after finish: timed out waiting for last event")
	}

	// Channel must be closed (second read returns zero value, ok=false).
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("channel should be closed after last event")
		}
	case <-time.After(time.Second):
		t.Fatal("subscribe after finish: channel not closed")
	}
}

func TestRedditHarnessPrepareGroupsAndRejectsURLs(t *testing.T) {
	s := &Server{cfg: &config.Config{}}
	body, _ := json.Marshal(redditHarnessRequest{Action: "prepare", URLs: []redditimport.RequestedPage{
		{URL: "https://reddit.com/r/test/comments/abc123/topic/", Title: "First"},
		{URL: "https://old.reddit.com/r/test/comments/abc123/topic/comment1/", Title: "Same thread"},
		{URL: "https://example.com/not-reddit", Title: "Rejected"},
	}})
	recorder := httptest.NewRecorder()
	s.handleRedditImportHarness(recorder, httptest.NewRequest(http.MethodPost, "/api/debug/reddit-import", bytes.NewReader(body)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var got struct {
		Request  redditimport.Request `json:"request"`
		Rejected []map[string]string  `json:"rejected"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Request.Pages) != 1 || len(got.Rejected) != 1 {
		t.Fatalf("grouped pages/rejections = %d/%d, want 1/1", len(got.Request.Pages), len(got.Rejected))
	}
}

func TestDebugOnlyReturnsNotFoundWhenDisabled(t *testing.T) {
	s := &Server{cfg: &config.Config{}}
	handler := s.debugOnly(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodGet, "/debug/reddit-import", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}

	s.cfg.Server.Debug = true
	recorder = httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodGet, "/debug/reddit-import", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("debug status = %d, want 204", recorder.Code)
	}
}

func TestFormatHarnessFindings(t *testing.T) {
	got := formatHarnessFindings([]research.Finding{{
		Title: "Captured thread", URL: "https://www.reddit.com/comments/abc123/",
		Summary: "Useful summary", Evidence: "Quoted evidence",
	}})
	for _, want := range []string{"## Finding 1", "Captured thread", "Useful summary", "Quoted evidence"} {
		if !strings.Contains(got, want) {
			t.Errorf("formatted finding missing %q:\n%s", want, got)
		}
	}
}
