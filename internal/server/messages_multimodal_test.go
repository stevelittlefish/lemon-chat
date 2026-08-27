package server

import (
	"encoding/json"
	"testing"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

// A text-only chatMsg must marshal exactly as before (plain string content),
// so existing model requests are byte-for-byte unchanged.
func TestChatMsgMarshalTextOnly(t *testing.T) {
	raw, err := json.Marshal(chatMsg{Role: "user", Content: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"content":"hello","role":"user"}` {
		t.Fatalf("unexpected encoding: %s", raw)
	}

	// Empty content is omitted (assistant tool-call turns rely on this).
	raw, _ = json.Marshal(chatMsg{Role: "assistant", ToolCalls: []any{map[string]any{"id": "c1"}}})
	var m map[string]any
	json.Unmarshal(raw, &m)
	if _, ok := m["content"]; ok {
		t.Fatalf("empty content should be omitted: %s", raw)
	}
}

// When ContentParts is set, content is a multimodal array (text + image_url).
func TestChatMsgMarshalMultimodal(t *testing.T) {
	msg := chatMsg{Role: "user", ContentParts: userContentParts("look", []any{imagePart("data:image/png;base64,AAAA")})}
	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	parts, ok := m["content"].([]any)
	if !ok || len(parts) != 2 {
		t.Fatalf("content should be a 2-element array: %s", raw)
	}
	if parts[0].(map[string]any)["type"] != "text" {
		t.Errorf("first part should be text: %v", parts[0])
	}
	img := parts[1].(map[string]any)
	if img["type"] != "image_url" {
		t.Errorf("second part should be image_url: %v", img)
	}
	if img["image_url"].(map[string]any)["url"] != "data:image/png;base64,AAAA" {
		t.Errorf("image url wrong: %v", img)
	}
}

// buildChatMsgs attaches image parts only to the user message identified by the
// imagesFor callback, leaving other messages as plain-string content.
func TestBuildChatMsgsImages(t *testing.T) {
	history := []store.Message{
		{ID: 1, Role: "user", Content: "with image"},
		{ID: 2, Role: "assistant", Content: "reply"},
		{ID: 3, Role: "user", Content: "no image"},
	}
	imagesFor := func(id int64) []any {
		if id == 1 {
			return []any{imagePart("data:image/png;base64,AAAA")}
		}
		return nil
	}
	msgs := buildChatMsgs(nil, history, imagesFor)
	if msgs[0].ContentParts == nil {
		t.Fatal("message 1 should have image content parts")
	}
	if msgs[0].Content != "" {
		t.Errorf("message 1 string content should be cleared, got %q", msgs[0].Content)
	}
	if msgs[1].ContentParts != nil || msgs[2].ContentParts != nil {
		t.Error("only message 1 should carry image parts")
	}
	if msgs[2].Content != "no image" {
		t.Errorf("message 3 content wrong: %q", msgs[2].Content)
	}
}
