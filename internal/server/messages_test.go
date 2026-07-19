package server

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/llm"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

func TestConversationTitleTrigger(t *testing.T) {
	title := "Already titled"
	autoTitleCharacter := &store.Character{AutoTitle: true}

	tests := []struct {
		name      string
		conv      *store.Conversation
		character *store.Character
		history   []store.Message
		want      string
	}{
		{
			name:      "character requests title after first exchange",
			conv:      &store.Conversation{},
			character: autoTitleCharacter,
			want:      titleTriggerCharacterAuto,
		},
		{
			name:      "character auto title only applies to first user message",
			conv:      &store.Conversation{},
			character: autoTitleCharacter,
			history:   []store.Message{{Role: "user"}},
			want:      titleTriggerNone,
		},
		{
			name:    "third assistant response triggers title",
			conv:    &store.Conversation{},
			history: []store.Message{{Role: "assistant"}, {Role: "user"}, {Role: "assistant"}},
			want:    titleTriggerThirdResponse,
		},
		{
			name:      "third response fallback applies to auto title character",
			conv:      &store.Conversation{},
			character: autoTitleCharacter,
			history:   []store.Message{{Role: "user"}, {Role: "assistant"}, {Role: "tool"}, {Role: "assistant"}},
			want:      titleTriggerThirdResponse,
		},
		{
			name:    "fewer than two prior assistant responses does not trigger",
			conv:    &store.Conversation{},
			history: []store.Message{{Role: "user"}, {Role: "assistant"}},
			want:    titleTriggerNone,
		},
		{
			name:      "existing title disables automatic title",
			conv:      &store.Conversation{Title: &title},
			character: autoTitleCharacter,
			history:   []store.Message{{Role: "assistant"}, {Role: "assistant"}},
			want:      titleTriggerNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := conversationTitleTrigger(tt.conv, tt.character, tt.history); got != tt.want {
				t.Fatalf("conversationTitleTrigger() = %q, want %q", got, tt.want)
			}
		})
	}
}

type fakeMessageProvider struct {
	completion llm.Completion
	request    llm.Request
}

func (p *fakeMessageProvider) Stream(_ context.Context, req llm.Request, handler llm.Handler) (llm.Completion, error) {
	p.request = req
	handler.OnStart()
	handler.OnText(p.completion.Content)
	return p.completion, nil
}

func (p *fakeMessageProvider) ListModels(context.Context) ([]string, error) {
	return nil, nil
}

func TestMessageToolLoopStreamsFinalResponse(t *testing.T) {
	provider := &fakeMessageProvider{completion: llm.Completion{
		Content:      "Final answer",
		FinishReason: "stop",
		Usage:        &llm.Usage{PromptTokens: 12, CompletionTokens: 3},
	}}
	recorder := httptest.NewRecorder()
	committed := false
	started := time.Time{}
	messages := []chatMsg{{Role: "user", Content: "Question"}}

	loop := messageToolLoop{
		server:          &Server{},
		ctx:             context.Background(),
		writer:          recorder,
		flusher:         recorder,
		provider:        provider,
		request:         llm.Request{Model: "test-model"},
		messages:        messages,
		onStart:         func() { committed = true; started = time.Now() },
		committed:       func() bool { return committed },
		persistError:    func() error { return nil },
		maxLoops:        1,
		responseStarted: &started,
	}

	content, stats, ok := loop.run()
	if !ok {
		t.Fatal("messageToolLoop.run() reported an HTTP error")
	}
	if content != "Final answer" {
		t.Fatalf("content = %q, want %q", content, "Final answer")
	}
	if stats == nil || stats.PromptTokens != 12 || stats.CompletionTokens != 3 {
		t.Fatalf("stats = %#v, want prompt=12 completion=3", stats)
	}
	if !strings.Contains(recorder.Body.String(), `"delta":"Final answer"`) {
		t.Fatalf("stream = %q, want final answer delta", recorder.Body.String())
	}
	gotMessages, ok := provider.request.Messages.([]chatMsg)
	if !ok || len(gotMessages) != 1 || gotMessages[0].Role != messages[0].Role || gotMessages[0].Content != messages[0].Content {
		t.Fatalf("provider messages = %#v, want %#v", provider.request.Messages, messages)
	}
}
