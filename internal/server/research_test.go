package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/redditimport"
	"github.com/stevelittlefish/lemon-chat/internal/research"
	"github.com/stevelittlefish/lemon-chat/internal/store"
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

func TestResearchTokenLimitsAreClampedPerModel(t *testing.T) {
	s := &Server{cfg: &config.Config{
		Models:   []config.Model{{Name: "writer", MaxOutputTokens: 16000}, {Name: "html", MaxOutputTokens: 24000}},
		Research: config.Research{SynthesisTokens: 8192, MemoryTokens: 6000, FinalReportTokens: 32768, SectionTokens: 12288, HTMLReportTokens: 32768},
	}}
	limits := s.researchTokenLimits("writer", "html")
	if limits.Synthesis != 8192 || limits.Memory != 6000 || limits.FinalReport != 16000 || limits.Section != 12288 || limits.HTMLReport != 24000 {
		t.Fatalf("unexpected effective limits: %+v", limits)
	}
}

func TestPendingRedditRequestMatchesDurableRequestID(t *testing.T) {
	pendingJSON, err := json.Marshal(research.PendingRedditRound{Request: redditimport.Request{
		Version: redditimport.SchemaVersion, RequestID: "request-1",
		Pages: []redditimport.RequestedPage{{URL: "https://www.reddit.com/comments/abc123/"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	requestID := "request-1"
	pendingText := string(pendingJSON)
	job := &store.ResearchJob{RedditRequestID: &requestID, PendingRedditRound: &pendingText}
	request, err := pendingRedditRequest(job)
	if err != nil || request.RequestID != requestID {
		t.Fatalf("request=%+v err=%v", request, err)
	}
	wrong := "request-2"
	job.RedditRequestID = &wrong
	if _, err := pendingRedditRequest(job); err == nil {
		t.Fatal("mismatched durable request ID was accepted")
	}
}

func newAwaitingRedditJob(t *testing.T) (*Server, *store.User, *store.ResearchJob) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	user, err := st.CreateUser("owner", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	job, err := st.CreateResearchJob(user.ID, "", "query", "model", "research", false, false, true, false, "", "", "", 3, 600, 0, "{}")
	if err != nil {
		t.Fatal(err)
	}
	request, err := redditimport.NewRequest("request-1", []redditimport.RequestedPage{{URL: "https://reddit.com/r/test/comments/abc123/topic/"}}, redditimport.CaptureLimits{})
	if err != nil {
		t.Fatal(err)
	}
	pending, _ := json.Marshal(research.PendingRedditRound{Round: 1, Request: request})
	if err := st.SetResearchJobAwaitingReddit(job.ID, request.RequestID, string(pending), 500); err != nil {
		t.Fatal(err)
	}
	job, err = st.GetResearchJob(job.ID, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	return &Server{store: st, research: newResearchManager()}, user, job
}

func requestAsUser(method, target string, user *store.User) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxUser, user))
	return req
}

func TestCancelAwaitingRedditJob(t *testing.T) {
	s, user, job := newAwaitingRedditJob(t)
	req := requestAsUser(http.MethodPost, "/api/research/42/cancel", user)
	req.SetPathValue("id", fmt.Sprint(job.ID))
	recorder := httptest.NewRecorder()
	s.handleCancelResearch(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	got, err := s.store.GetResearchJob(job.ID, user.ID)
	if err != nil || got.Status != store.ResearchStatusCancelled {
		t.Fatalf("cancelled job=%+v err=%v", got, err)
	}
}

func TestGetResearchTraceIsOwnerScoped(t *testing.T) {
	s, owner, job := newAwaitingRedditJob(t)
	other, err := s.store.CreateUser("trace-other", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.store.AppendResearchEvent(job.ID, "stop_decision", "deciding", 1, "YES — enough evidence", `{"stop":true}`); err != nil {
		t.Fatal(err)
	}
	call, err := s.store.BeginResearchLLMCall(job.ID, "deciding", "stop_decision", 1, "model", "https://models.example/v1", `[]`, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.store.CompleteResearchLLMCall(call.ID, 125, "YES", "stop", `{"total_tokens":12}`, nil, http.StatusOK, ""); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		user *store.User
		want int
	}{
		{name: "owner", user: owner, want: http.StatusOK},
		{name: "other user", user: other, want: http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := requestAsUser(http.MethodGet, "/api/research/42/trace", tc.user)
			req.SetPathValue("id", fmt.Sprint(job.ID))
			recorder := httptest.NewRecorder()
			s.handleGetResearchTrace(recorder, req)
			if recorder.Code != tc.want {
				t.Fatalf("status=%d body=%s, want %d", recorder.Code, recorder.Body.String(), tc.want)
			}
			if tc.want == http.StatusOK && (!strings.Contains(recorder.Body.String(), "stop_decision") || !strings.Contains(recorder.Body.String(), `\"total_tokens\":12`)) {
				t.Fatalf("trace response missing event or call: %s", recorder.Body.String())
			}
		})
	}
}

func TestRedditImportRejectsWrongOwnerAndStaleRequest(t *testing.T) {
	s, owner, job := newAwaitingRedditJob(t)
	other, err := s.store.CreateUser("other", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	response := redditimport.Response{Version: 1, RequestID: "stale-request", Pages: []redditimport.CapturedPage{{
		RequestedURL: "https://www.reddit.com/comments/abc123/", Post: redditimport.CapturedPost{Body: "synthetic"}, Complete: true,
	}}}
	body, _ := json.Marshal(response)

	for _, tc := range []struct {
		name string
		user *store.User
		want int
	}{
		{name: "wrong owner", user: other, want: http.StatusNotFound},
		{name: "stale request", user: owner, want: http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := requestAsUser(http.MethodPost, "/api/research/42/reddit-import", tc.user)
			req.Body = io.NopCloser(bytes.NewReader(body))
			req.SetPathValue("id", fmt.Sprint(job.ID))
			recorder := httptest.NewRecorder()
			s.handleResearchRedditImport(recorder, req)
			if recorder.Code != tc.want {
				t.Fatalf("status=%d body=%s, want %d", recorder.Code, recorder.Body.String(), tc.want)
			}
		})
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
