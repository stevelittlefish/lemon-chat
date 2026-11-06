package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

func historyFixture(t *testing.T) (st *store.Store, dad, son *store.User, dadConv, sonConv *store.Conversation) {
	t.Helper()
	st = memoryTestStore(t)
	dadName, sonName := "Dad", "Sam"
	dad, _ = st.CreateUser("dad", nil, true, &dadName)
	son, _ = st.CreateUser("sam", nil, false, &sonName)
	model := "m"
	dadConv, _ = st.CreateConversation(dad.ID, nil, &model, nil)
	title := "Guitar practice"
	sonConv, _ = st.CreateConversation(son.ID, &title, &model, nil)
	st.CreateMessage(sonConv.ID, "user", "How do I tune a guitar to drop D?", nil, nil, nil, nil, "")
	calls := `[{"id":"c1","type":"function","function":{"name":"searxng","arguments":"{}"}}]`
	st.CreateMessage(sonConv.ID, "assistant", "", nil, nil, nil, &calls, "")
	st.CreateMessage(sonConv.ID, "tool", "secret tool output", nil, nil, nil, nil, "c1")
	st.CreateMessage(sonConv.ID, "assistant", "Lower the low E string a whole step to D.", nil, nil, nil, nil, "")
	return
}

func TestHistoryToolsOwnConversations(t *testing.T) {
	st, _, son, _, sonConv := historyFixture(t)
	model := "m"
	current, _ := st.CreateConversation(son.ID, nil, &model, nil)
	tctx := ToolContext{Store: st, UserID: son.ID, ConversationID: current.ID}

	out, err := executeHistoryList(`{}`, tctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, fmt.Sprintf("#%d: \"Guitar practice\"", sonConv.ID)) || !strings.Contains(out, "2 messages") {
		t.Fatalf("list = %q", out)
	}
	if strings.Contains(out, fmt.Sprintf("#%d:", current.ID)) {
		t.Fatalf("list includes the current conversation: %q", out)
	}

	out, err = executeHistorySearch(`{"query":"DROP d"}`, tctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "1 matching messages") || !strings.Contains(out, "tune a guitar to drop D") {
		t.Fatalf("search = %q", out)
	}
	if out, _ := executeHistorySearch(`{"query":"secret"}`, tctx); !strings.Contains(out, "No conversations") {
		t.Fatalf("search matched a tool result: %q", out)
	}
	if out, _ := executeHistorySearch(`{"query":"100%"}`, tctx); !strings.Contains(out, "No conversations") {
		t.Fatalf("%% should be literal: %q", out)
	}

	out, err = executeHistoryRead(fmt.Sprintf(`{"conversation_id":%d}`, sonConv.ID), tctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Sam (user") || !strings.Contains(out, "whole step") || strings.Contains(out, "secret") || !strings.Contains(out, "[End of conversation.]") {
		t.Fatalf("read = %q", out)
	}
	if _, err := executeHistoryRead(fmt.Sprintf(`{"conversation_id":%d}`, current.ID), tctx); err == nil {
		t.Fatal("read the current conversation")
	}
}

func TestHistoryToolsCrossUserAccess(t *testing.T) {
	st, dad, son, dadConv, sonConv := historyFixture(t)
	sonCtx := ToolContext{Store: st, UserID: son.ID}
	dadCtx := ToolContext{Store: st, UserID: dad.ID, ConversationID: dadConv.ID}

	// A non-admin can't look at anyone else's history.
	if _, err := executeHistoryList(`{"user":"dad"}`, sonCtx); err == nil {
		t.Fatal("non-admin listed another user's conversations")
	}
	if _, err := executeHistoryRead(fmt.Sprintf(`{"conversation_id":%d}`, dadConv.ID), sonCtx); err == nil {
		t.Fatal("non-admin read another user's conversation")
	}
	// Naming yourself is fine.
	if _, err := executeHistoryList(`{"user":"Sam"}`, sonCtx); err != nil {
		t.Fatal(err)
	}

	// An admin can, by username or display name.
	out, err := executeHistoryList(`{"user":"sam"}`, dadCtx)
	if err != nil || !strings.Contains(out, "Guitar practice") {
		t.Fatalf("admin list = %q, %v", out, err)
	}
	out, err = executeHistorySearch(`{"query":"guitar","user":"Sam"}`, dadCtx)
	if err != nil || !strings.Contains(out, "Guitar practice") {
		t.Fatalf("admin search = %q, %v", out, err)
	}
	out, err = executeHistoryRead(fmt.Sprintf(`{"conversation_id":%d}`, sonConv.ID), dadCtx)
	if err != nil || !strings.Contains(out, "of Sam (sam)") {
		t.Fatalf("admin read = %q, %v", out, err)
	}
	if _, err := executeHistoryList(`{"user":"nobody"}`, dadCtx); err == nil || !strings.Contains(err.Error(), "Sam (sam)") {
		t.Fatalf("unknown user error = %v; want the list of users", err)
	}
}

func TestHistoryReadPages(t *testing.T) {
	st := memoryTestStore(t)
	u, _ := st.CreateUser("u", nil, false, nil)
	model := "m"
	conv, _ := st.CreateConversation(u.ID, nil, &model, nil)
	long := strings.Repeat("x", historyReadMaxChars/2)
	for range 5 {
		st.CreateMessage(conv.ID, "user", long, nil, nil, nil, nil, "")
	}
	tctx := ToolContext{Store: st, UserID: u.ID}
	out, err := executeHistoryRead(fmt.Sprintf(`{"conversation_id":%d}`, conv.ID), tctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Showing messages 1–2 of 5") || !strings.Contains(out, "offset=2") {
		t.Fatalf("first page = %q", out[len(out)-200:])
	}
	out, err = executeHistoryRead(fmt.Sprintf(`{"conversation_id":%d,"offset":4}`, conv.ID), tctx)
	if err != nil || !strings.Contains(out, "[5]") || !strings.Contains(out, "[End of conversation.]") {
		t.Fatalf("last page = %v", err)
	}

	// One message bigger than a page is cut off rather than skipped.
	conv2, _ := st.CreateConversation(u.ID, nil, &model, nil)
	st.CreateMessage(conv2.ID, "user", strings.Repeat("y", historyReadMaxChars+10), nil, nil, nil, nil, "")
	out, _ = executeHistoryRead(fmt.Sprintf(`{"conversation_id":%d}`, conv2.ID), tctx)
	if !strings.Contains(out, "[message cut off: 10 more characters]") {
		t.Fatalf("oversized message = %q", out[len(out)-200:])
	}
}

func TestWithoutHistoryUserParam(t *testing.T) {
	defs := ToolDefsForCharacter([]string{"history"})
	if len(defs) != 3 {
		t.Fatalf("history expands to %d tools", len(defs))
	}
	stripped := withoutHistoryUserParam(defs)
	for _, d := range stripped {
		if _, ok := d.Function.Parameters.Properties[historyUserParam]; ok {
			t.Errorf("%s still offers the user argument", d.Function.Name)
		}
	}
	// The registry itself is untouched.
	if _, ok := toolRegistry["history_list"].Function.Parameters.Properties[historyUserParam]; !ok {
		t.Fatal("stripping modified the registry")
	}
}
