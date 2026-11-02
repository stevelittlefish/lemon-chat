package server

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/llm"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

func viewImageStore(t *testing.T) (*store.Store, int64, int64) {
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
	other, err := st.CreateConversation(user.ID, nil, &model, nil)
	if err != nil {
		t.Fatal(err)
	}
	return st, conv.ID, other.ID
}

func TestViewImageExecutor(t *testing.T) {
	st, convID, otherID := viewImageStore(t)
	old := comfyPollInterval
	comfyPollInterval = 5 * time.Millisecond
	t.Cleanup(func() { comfyPollInterval = old })
	tctx := ToolContext{Store: st, ConversationID: convID, ModelVision: true}

	if _, err := executeViewImage(`{}`, tctx); err == nil || !strings.Contains(err.Error(), "no images") {
		t.Errorf("empty conversation: %v", err)
	}

	first, _ := st.CreateAttachment("c1", convID, "Generated image", "image.png", "image/png", "attachments/a/image.png")
	latest, _ := st.CreateAttachment("c2", convID, "Generated image", "image.png", "image/png", "attachments/b/image.png")
	doc, _ := st.CreateAttachment("c3", convID, "Notes", "notes.md", "text/markdown", "attachments/c/notes.md")
	foreign, _ := st.CreateAttachment("c4", otherID, "Generated image", "image.png", "image/png", "attachments/d/image.png")

	for _, tt := range []struct {
		args   string
		wantID int64
		errHas string
	}{
		{`{}`, latest.ID, ""},
		{`{"image_id":null}`, latest.ID, ""},
		{`{"image_id":` + jsonInt(first.ID) + `}`, first.ID, ""},
		{`{"image_id":` + jsonInt(doc.ID) + `}`, 0, "no image"},
		{`{"image_id":` + jsonInt(foreign.ID) + `}`, 0, "no image"},
		{`{"image_id":99999}`, 0, "no image"},
	} {
		out, err := executeViewImage(tt.args, tctx)
		if tt.errHas != "" {
			if err == nil || !strings.Contains(err.Error(), tt.errHas) {
				t.Errorf("%s: err = %v, want %q", tt.args, err, tt.errHas)
			}
			continue
		}
		if err != nil || viewedImageID(out) != tt.wantID {
			t.Errorf("%s: out=%s err=%v, want image %d", tt.args, out, err, tt.wantID)
		}
		if strings.Contains(out, "attachment_id") {
			t.Errorf("result must not look like a new attachment: %s", out)
		}
	}

	tctx.ModelVision = false
	if _, err := executeViewImage(`{}`, tctx); err == nil {
		t.Error("non-vision model should be refused")
	}
}

func TestViewImageWaitsForPendingImage(t *testing.T) {
	st, convID, _ := viewImageStore(t)
	old := comfyPollInterval
	comfyPollInterval = 5 * time.Millisecond
	t.Cleanup(func() { comfyPollInterval = old })

	pending, _ := st.CreatePendingAttachment("c1", convID, "Generated image", "image.png", "image/png")
	go func() {
		time.Sleep(30 * time.Millisecond)
		st.FinaliseAttachment(pending.ID, "attachments/a/image.png") //nolint:errcheck
	}()
	out, err := executeViewImage(`{}`, ToolContext{Store: st, ConversationID: convID, ModelVision: true})
	if err != nil || viewedImageID(out) != pending.ID {
		t.Fatalf("out=%s err=%v", out, err)
	}

	failed, _ := st.CreatePendingAttachment("c2", convID, "Generated image", "image.png", "image/png")
	st.FailAttachment(failed.ID, "ComfyUI exploded") //nolint:errcheck
	if _, err := executeViewImage(`{"image_id":`+jsonInt(failed.ID)+`}`, ToolContext{Store: st, ConversationID: convID, ModelVision: true}); err == nil || !strings.Contains(err.Error(), "exploded") {
		t.Errorf("failed image: %v", err)
	}
}

func jsonInt(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func fakeImageLoader(t *testing.T, available ...int64) func(int64) any {
	return func(id int64) any {
		for _, a := range available {
			if a == id {
				return imagePart("data:image/png;base64,ID" + jsonInt(id))
			}
		}
		return nil
	}
}

// After a round of tool calls that includes view_image, the next model turn
// gets the image as a user message following all of that round's tool results.
func TestToolLoopAttachesViewedImage(t *testing.T) {
	p := &fakeProvider{}
	loop, _ := newTestToolLoop(t, context.Background(), p, 5)
	att, err := loop.server.store.CreateAttachment("gen", loop.conversationID, "Generated image", "image.png", "image/png", "attachments/a/image.png")
	if err != nil {
		t.Fatal(err)
	}
	loop.toolContext = ToolContext{Store: loop.server.store, ConversationID: loop.conversationID, ModelVision: true}
	loop.viewedImage = fakeImageLoader(t, att.ID)
	p.turns = []fakeTurn{
		{comp: llm.Completion{FinishReason: "tool_calls", ToolCalls: []llm.ToolCall{
			{ID: "v", Name: "view_image", Arguments: `{"image_id":null}`},
			{ID: "x", Name: "no_such_tool", Arguments: "{}"},
		}}},
		{comp: llm.Completion{Content: "it's a lemon", FinishReason: "stop"}},
	}

	if content, _, ok := loop.run(); !ok || content != "it's a lemon" {
		t.Fatalf("content=%q ok=%v", content, ok)
	}
	sent := p.sent[1]
	n := len(sent)
	if n < 3 || sent[n-3].Role != "tool" || sent[n-2].Role != "tool" || sent[n-1].Role != "user" {
		t.Fatalf("want tool, tool, user at the end of the second turn, got %+v", sent)
	}
	parts, _ := json.Marshal(sent[n-1].ContentParts)
	if !strings.Contains(string(parts), "image_url") || !strings.Contains(string(parts), "ID"+jsonInt(att.ID)) {
		t.Errorf("viewed image not attached: %s", parts)
	}
}

// Reloading history re-creates the same follow-up message, after the whole
// run of tool results, and only when a loader is supplied.
func TestBuildChatMsgsReattachesViewedImages(t *testing.T) {
	calls := `[{"id":"v1","type":"function","function":{"name":"view_image","arguments":"{}"}},` +
		`{"id":"v2","type":"function","function":{"name":"view_image","arguments":"{}"}},` +
		`{"id":"g","type":"function","function":{"name":"get_time","arguments":"{}"}}]`
	history := []store.Message{
		{ID: 1, Role: "user", Content: "what did you draw?"},
		{ID: 2, Role: "assistant", ToolCalls: &calls},
		{ID: 3, Role: "tool", ToolCallID: "v1", Content: `{"image_id":7,"status":"shown"}`},
		{ID: 4, Role: "tool", ToolCallID: "v2", Content: `{"image_id":8,"status":"shown"}`},
		{ID: 5, Role: "tool", ToolCallID: "g", Content: `{"image_id":9,"status":"shown"}`}, // not view_image
		{ID: 6, Role: "assistant", Content: "a lemon"},
	}

	msgs := buildChatMsgs(nil, history, nil, fakeImageLoader(t, 7))
	roles := make([]string, len(msgs))
	for i, m := range msgs {
		roles[i] = m.Role
	}
	if got := strings.Join(roles, ","); got != "user,assistant,tool,tool,tool,user,assistant" {
		t.Fatalf("roles = %s", got)
	}
	parts, _ := json.Marshal(msgs[5].ContentParts)
	if !strings.Contains(string(parts), "ID7") || !strings.Contains(string(parts), "image_id=8 is no longer available") || strings.Contains(string(parts), "image_id=9") {
		t.Errorf("follow-up parts = %s", parts)
	}

	if msgs := buildChatMsgs(nil, history, nil, nil); len(msgs) != len(history) {
		t.Errorf("without a loader (non-vision model) no images should be added, got %d messages", len(msgs))
	}
}
