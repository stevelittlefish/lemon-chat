package server

import (
	"testing"
	"time"
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
