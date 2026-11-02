package server

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/llm"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

// fakeProvider replays scripted turns and records whether tools were offered
// and the messages each turn was sent.
type fakeProvider struct {
	turns        []fakeTurn
	calls        int
	toolsOffered []bool
	sent         [][]chatMsg
}

type fakeTurn struct {
	comp llm.Completion
	err  error
}

func (f *fakeProvider) Stream(ctx context.Context, req llm.Request, h llm.Handler) (llm.Completion, error) {
	f.toolsOffered = append(f.toolsOffered, req.Tools != nil)
	if msgs, ok := req.Messages.([]chatMsg); ok {
		f.sent = append(f.sent, append([]chatMsg(nil), msgs...))
	}
	if h.OnStart != nil {
		h.OnStart()
	}
	turn := f.turns[f.calls]
	f.calls++
	if h.OnText != nil && turn.comp.Content != "" {
		h.OnText(turn.comp.Content)
	}
	return turn.comp, turn.err
}

func (f *fakeProvider) ListModels(context.Context) ([]string, error) { return nil, nil }

func newTestToolLoop(t *testing.T, ctx context.Context, p llm.Provider, maxLoops int) (*messageToolLoop, *httptest.ResponseRecorder) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	user, err := st.CreateUser("u", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	model := "m"
	conv, err := st.CreateConversation(user.ID, nil, &model, nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	started := time.Now()
	return &messageToolLoop{
		server:          &Server{store: st},
		ctx:             ctx,
		writer:          w,
		flusher:         w,
		provider:        p,
		request:         llm.Request{Model: "m", Tools: []any{map[string]any{"type": "function"}}},
		onStart:         func() {},
		committed:       func() bool { return true },
		persistError:    func() error { return nil },
		conversationID:  conv.ID,
		assistantName:   "m",
		maxLoops:        maxLoops,
		responseStarted: &started,
	}, w
}

func toolTurn(id string) fakeTurn {
	return fakeTurn{comp: llm.Completion{FinishReason: "tool_calls", ToolCalls: []llm.ToolCall{{ID: id, Name: "no_such_tool", Arguments: "{}"}}}}
}

func TestToolLoopForcesFinalAnswerAfterLimit(t *testing.T) {
	p := &fakeProvider{turns: []fakeTurn{
		toolTurn("a"),
		toolTurn("b"),
		{comp: llm.Completion{Content: "final answer", FinishReason: "stop"}},
	}}
	loop, _ := newTestToolLoop(t, context.Background(), p, 2)

	content, _, ok := loop.run()
	if !ok || content != "final answer" {
		t.Fatalf("content=%q ok=%v, want final answer", content, ok)
	}
	want := []bool{true, true, false}
	if fmt.Sprint(p.toolsOffered) != fmt.Sprint(want) {
		t.Fatalf("tools offered per turn = %v, want %v", p.toolsOffered, want)
	}
}

func TestToolLoopKeepsPartialAnswerAndWarns(t *testing.T) {
	p := &fakeProvider{turns: []fakeTurn{
		{comp: llm.Completion{Content: "half an ans"}, err: fmt.Errorf("stream interrupted")},
	}}
	loop, w := newTestToolLoop(t, context.Background(), p, 5)

	content, _, ok := loop.run()
	if !ok || content != "half an ans" {
		t.Fatalf("content=%q ok=%v, want partial content kept", content, ok)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"warning"`) || strings.Contains(body, `"error"`) {
		t.Fatalf("want a warning event and no error event, got:\n%s", body)
	}
}

func TestToolLoopDoesNotRunToolsFromBrokenStream(t *testing.T) {
	broken := toolTurn("a")
	broken.comp.Content = "let me check"
	broken.err = fmt.Errorf("stream interrupted")
	p := &fakeProvider{turns: []fakeTurn{broken}}
	loop, w := newTestToolLoop(t, context.Background(), p, 5)

	if _, _, ok := loop.run(); !ok {
		t.Fatal("run returned !ok")
	}
	if p.calls != 1 || strings.Contains(w.Body.String(), `"tool_call"`) {
		t.Fatalf("tools from a broken stream were run (calls=%d):\n%s", p.calls, w.Body.String())
	}
}

func TestToolLoopUserStopKeepsPartialSilently(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := &fakeProvider{turns: []fakeTurn{
		{comp: llm.Completion{Content: "stopped here"}, err: context.Canceled},
	}}
	loop, w := newTestToolLoop(t, ctx, p, 5)

	content, _, ok := loop.run()
	if !ok || content != "stopped here" {
		t.Fatalf("content=%q ok=%v", content, ok)
	}
	if strings.Contains(w.Body.String(), `"warning"`) || strings.Contains(w.Body.String(), `"error"`) {
		t.Fatalf("user stop produced a warning/error event:\n%s", w.Body.String())
	}
}
