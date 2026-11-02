package tasks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/llm"
)

// stubProvider returns a fixed completion and records the request.
type stubProvider struct {
	content string
	err     error
	got     llm.Request
}

func (p *stubProvider) Stream(_ context.Context, req llm.Request, _ llm.Handler) (llm.Completion, error) {
	p.got = req
	return llm.Completion{Content: p.content}, p.err
}

func (p *stubProvider) ListModels(context.Context) ([]string, error) { return nil, nil }

func stubModel(p llm.Provider) titleModel {
	return titleModel{provider: p, name: "model", timeout: 5 * time.Second}
}

func TestRequestTitleTrimsWhitespaceAndQuotes(t *testing.T) {
	p := &stubProvider{content: "  \"My Great Title\" \n"}
	got, err := requestTitle(stubModel(p), "sys", "content")
	if err != nil {
		t.Fatalf("requestTitle returned error: %v", err)
	}
	if got != "My Great Title" {
		t.Errorf("requestTitle = %q, want %q", got, "My Great Title")
	}
	if p.got.Model != "model" || p.got.MaxTokens != 20 {
		t.Errorf("request = %+v", p.got)
	}
}

func TestRequestTitleEmptyIsError(t *testing.T) {
	if _, err := requestTitle(stubModel(&stubProvider{content: "   \n  "}), "sys", "content"); err == nil {
		t.Fatal("expected an error for empty title, got nil")
	}
}

func TestTitleBackoff(t *testing.T) {
	tl := NewTitler(nil, nil, nil)
	key := "conv:1"
	if !tl.shouldAttempt(key) {
		t.Fatal("fresh key should be attempted")
	}
	tl.recordResult(key, errors.New("boom"))
	if tl.shouldAttempt(key) {
		t.Fatal("key attempted again immediately after failure")
	}
	first := tl.backoff[key].nextAttempt
	tl.recordResult(key, errors.New("boom"))
	if !tl.backoff[key].nextAttempt.After(first) {
		t.Fatal("backoff did not grow")
	}

	for i := 0; i < 100; i++ {
		tl.recordResult(key, errors.New("boom"))
	}
	if d := time.Until(tl.backoff[key].nextAttempt); d > titleRetryMax {
		t.Fatalf("backoff %s exceeds max %s", d, titleRetryMax)
	}

	tl.recordResult(key, nil)
	if !tl.shouldAttempt(key) {
		t.Fatal("success did not clear backoff")
	}
}
