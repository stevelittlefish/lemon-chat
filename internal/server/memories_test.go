package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

func memoryTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestMemoryToolsSaveUpdateForget(t *testing.T) {
	st := memoryTestStore(t)
	alice, _ := st.CreateUser("alice", nil, false, nil)
	bob, _ := st.CreateUser("bob", nil, false, nil)
	model := "m"
	conv, _ := st.CreateConversation(alice.ID, nil, &model, nil)
	tctx := ToolContext{Store: st, UserID: alice.ID, ConversationID: conv.ID}

	out, err := executeMemorySave(`{"text":"  Has a cat called Biscuit "}`, tctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Saved as #") || !strings.Contains(out, ": Has a cat called Biscuit") {
		t.Fatalf("save without loaded memories should include the list, got %q", out)
	}
	tctx.MemoriesLoaded = true
	out, err = executeMemorySave(`{"text":"Works as a software engineer"}`, tctx)
	if err != nil || strings.Contains(out, "Memories about") {
		t.Fatalf("save with loaded memories = %q, %v; want no list", out, err)
	}

	memories, _ := st.ListMemories(alice.ID)
	if len(memories) != 2 || memories[0].Text != "Has a cat called Biscuit" || *memories[0].ConversationID != conv.ID {
		t.Fatalf("memories = %+v", memories)
	}
	cat := memories[0].ID

	if _, err := executeMemoryUpdate(fmt.Sprintf(`{"id":%d,"text":"Has cats called Biscuit and Crumb"}`, cat), tctx); err != nil {
		t.Fatal(err)
	}
	// Bob can't touch Alice's memories.
	bobCtx := ToolContext{Store: st, UserID: bob.ID}
	if _, err := executeMemoryForget(fmt.Sprintf(`{"id":%d}`, cat), bobCtx); err == nil {
		t.Fatal("bob forgot alice's memory")
	}
	if out, _ := executeMemoryList("{}", bobCtx); out != "No memories saved yet." {
		t.Fatalf("bob's list = %q", out)
	}
	if out, _ := executeMemoryList("{}", tctx); !strings.Contains(out, "Biscuit and Crumb") {
		t.Fatalf("update didn't stick: %q", out)
	}
	if _, err := executeMemoryForget(fmt.Sprintf(`{"id":%d}`, cat), tctx); err != nil {
		t.Fatal(err)
	}

	if _, err := executeMemorySave(`{"text":"`+strings.Repeat("é", memoryMaxChars+1)+`"}`, tctx); err == nil {
		t.Fatal("over-long memory saved")
	}
	if _, err := executeMemorySave(`{"text":" "}`, tctx); err == nil {
		t.Fatal("empty memory saved")
	}

	// Deleting the conversation keeps the memory, without the link.
	if _, err := st.DeleteConversation(conv.ID, alice.ID); err != nil {
		t.Fatal(err)
	}
	memories, _ = st.ListMemories(alice.ID)
	if len(memories) != 1 || memories[0].ConversationID != nil {
		t.Fatalf("after deleting the conversation, memories = %+v", memories)
	}
}

func TestMemoryHandlersAccess(t *testing.T) {
	st := memoryTestStore(t)
	s := &Server{store: st}
	alice, _ := st.CreateUser("alice", nil, false, nil)
	bob, _ := st.CreateUser("bob", nil, false, nil)
	admin, _ := st.CreateUser("admin", nil, true, nil)
	m, _ := st.CreateMemory(alice.ID, "Likes tea", nil)

	do := func(h http.HandlerFunc, method, target, body string, u *store.User, id int64) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, target, strings.NewReader(body))
		if id != 0 {
			r.SetPathValue("id", fmt.Sprint(id))
		}
		w := httptest.NewRecorder()
		h(w, asUser(r, u))
		return w
	}

	if w := do(s.handleListMemories, "GET", fmt.Sprintf("/api/memories?user_id=%d", alice.ID), "", bob, 0); w.Code != http.StatusForbidden {
		t.Errorf("bob listing alice's memories: %d", w.Code)
	}
	if w := do(s.handleUpdateMemory, "PATCH", "/", `{"text":"x"}`, bob, m.ID); w.Code != http.StatusNotFound {
		t.Errorf("bob editing alice's memory: %d", w.Code)
	}
	if w := do(s.handleDeleteMemory, "DELETE", "/", "", bob, m.ID); w.Code != http.StatusNotFound {
		t.Errorf("bob deleting alice's memory: %d", w.Code)
	}

	w := do(s.handleListMemories, "GET", fmt.Sprintf("/api/memories?user_id=%d", alice.ID), "", admin, 0)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Likes tea") {
		t.Fatalf("admin listing alice's memories: %d %s", w.Code, w.Body)
	}
	if w := do(s.handleCreateMemory, "POST", fmt.Sprintf("/api/memories?user_id=%d", alice.ID), `{"text":"Has a dog"}`, admin, 0); w.Code != http.StatusCreated {
		t.Fatalf("admin adding to alice's memories: %d %s", w.Code, w.Body)
	}
	if w := do(s.handleUpdateMemory, "PATCH", "/", `{"text":"Likes green tea"}`, admin, m.ID); w.Code != http.StatusOK {
		t.Fatalf("admin editing alice's memory: %d %s", w.Code, w.Body)
	}
	if w := do(s.handleCreateMemory, "POST", "/api/memories", `{"text":"Own memory"}`, bob, 0); w.Code != http.StatusCreated {
		t.Fatalf("bob adding his own memory: %d %s", w.Code, w.Body)
	}

	memories, _ := st.ListMemories(alice.ID)
	if len(memories) != 2 || memories[0].Text != "Likes green tea" || memories[1].Text != "Has a dog" {
		t.Fatalf("alice's memories = %+v", memories)
	}
	if w := do(s.handleDeleteMemory, "DELETE", "/", "", admin, m.ID); w.Code != http.StatusNoContent {
		t.Fatalf("admin deleting alice's memory: %d", w.Code)
	}
	if bobs, _ := st.ListMemories(bob.ID); len(bobs) != 1 {
		t.Fatalf("bob's memories = %+v", bobs)
	}
}

// fakeChatServer is a chat-completions endpoint that always answers "ok" and
// records each request's messages.
func fakeChatServer(t *testing.T) (*httptest.Server, func() [][]map[string]any) {
	t.Helper()
	var mu sync.Mutex
	var requests [][]map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Messages []map[string]any `json:"messages"`
		}
		json.Unmarshal(body, &req) //nolint:errcheck
		mu.Lock()
		requests = append(requests, req.Messages)
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"ok"},"finish_reason":"stop"}]}`+"\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	return srv, func() [][]map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return requests
	}
}

func TestLoadMemoriesAtStart(t *testing.T) {
	chat, requests := fakeChatServer(t)
	st := memoryTestStore(t)
	cfg := &config.Config{
		ModelServers: []config.ModelServer{{Name: "local", APIBase: chat.URL}},
		Models:       []config.Model{{Name: "m", ModelServer: "local"}},
	}
	cfg.Server.DataDir = t.TempDir()
	cfg.Server.ResponseTimeoutSeconds = 30
	cfg.Server.MaxToolLoops = 5
	s := New(cfg, st, NewHub(st))

	user, _ := st.CreateUser("alice", nil, false, nil)
	if _, err := st.CreateMemory(user.ID, "Has a cat called Biscuit", nil); err != nil {
		t.Fatal(err)
	}
	char, _ := st.CreateCharacter("Helper", "m", nil, nil, nil, user.ID, "private", false, []string{"memory"})
	if err := st.SetCharacterLoadMemories(char.ID, true); err != nil {
		t.Fatal(err)
	}
	conv, _ := st.CreateConversation(user.ID, nil, nil, &char.ID)

	send := func(text string) string {
		r := httptest.NewRequest("POST", "/", strings.NewReader(`{"content":"`+text+`"}`))
		r.SetPathValue("id", fmt.Sprint(conv.ID))
		w := httptest.NewRecorder()
		s.handleSendMessage(w, asUser(r, user))
		if w.Code != http.StatusOK {
			t.Fatalf("send %q: %d %s", text, w.Code, w.Body)
		}
		return w.Body.String()
	}

	body := send("hello")
	if !strings.Contains(body, `"tool_call"`) || !strings.Contains(body, "Biscuit") {
		t.Fatalf("first send should stream the loaded memories, got:\n%s", body)
	}
	send("again")

	reqs := requests()
	if len(reqs) != 2 {
		t.Fatalf("got %d model requests, want 2", len(reqs))
	}
	roles := func(msgs []map[string]any) string {
		var out []string
		for _, m := range msgs {
			out = append(out, m["role"].(string))
		}
		return strings.Join(out, ",")
	}
	// The memory_list round sits right after the first user message, and the
	// second turn replays it unchanged.
	if got := roles(reqs[0]); got != "user,assistant,tool" {
		t.Fatalf("first request roles = %s", got)
	}
	if got := roles(reqs[1]); got != "user,assistant,tool,assistant,user" {
		t.Fatalf("second request roles = %s", got)
	}
	first, _ := json.Marshal(reqs[0])
	replayed, _ := json.Marshal(reqs[1][:3])
	if string(first) != string(replayed) {
		t.Fatalf("second request changed the prefix:\n%s\n%s", first, replayed)
	}
	if c, _ := reqs[0][2]["content"].(string); !strings.Contains(c, "#1: Has a cat called Biscuit") {
		t.Fatalf("tool result = %q", c)
	}

	msgs, _ := st.ListMessages(conv.ID)
	calls := 0
	for _, m := range msgs {
		if m.ToolCallID == memoryAutoloadCallID(conv.ID) {
			calls++
		}
	}
	if calls != 1 {
		t.Fatalf("stored %d memory_list results, want 1", calls)
	}
}
