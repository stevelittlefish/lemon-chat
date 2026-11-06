package server

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/config"
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
	tutor, _ := st.CreateCharacter("Music Tutor", "m", nil, nil, nil, dad.ID, "readonly", false, nil)
	title := "Guitar practice"
	sonConv, _ = st.CreateConversation(son.ID, &title, nil, &tutor.ID)
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
	if !strings.Contains(out, fmt.Sprintf("#%d: \"Guitar practice\" with Music Tutor", sonConv.ID)) || !strings.Contains(out, "2 messages") || !strings.Contains(out, "Today is") {
		t.Fatalf("list = %q", out)
	}
	if strings.Contains(out, fmt.Sprintf("#%d:", current.ID)) {
		t.Fatalf("list includes the current conversation: %q", out)
	}

	// Word endings, case and extra unmatched words don't stop a match.
	out, err = executeHistorySearch(`{"query":"Tuned GUITARS ukulele"}`, tctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Matches 2 of 3 words (tuned, guitars), in the title") || !strings.Contains(out, "**tune**") {
		t.Fatalf("search = %q", out)
	}
	if out, _ := executeHistorySearch(`{"query":"secret"}`, tctx); !strings.Contains(out, "No conversations") {
		t.Fatalf("search matched a tool result: %q", out)
	}
	if _, err := executeHistorySearch(`{"query":"the 'and'"}`, tctx); err == nil {
		t.Fatal("a query of only stop words should be refused")
	}

	// Inside one conversation: matching messages with the numbers history_read uses.
	out, err = executeHistorySearch(fmt.Sprintf(`{"query":"string","conversation_id":%d}`, sonConv.ID), tctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[message 2] Music Tutor (assistant") || !strings.Contains(out, fmt.Sprintf("conversation_id=%d, start=1, count=8", sonConv.ID)) {
		t.Fatalf("search in conversation = %q", out)
	}

	out, err = executeHistoryRead(fmt.Sprintf(`{"conversation_id":%d}`, sonConv.ID), tctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[message 1] Sam (user") || !strings.Contains(out, "[message 2] Music Tutor") || strings.Contains(out, "secret") || !strings.Contains(out, "You have read all of it") {
		t.Fatalf("read = %q", out)
	}
	if _, err := executeHistoryRead(fmt.Sprintf(`{"conversation_id":%d}`, current.ID), tctx); err == nil {
		t.Fatal("read the current conversation")
	}
}

func TestHistoryFilters(t *testing.T) {
	st, _, son, _, sonConv := historyFixture(t)
	tctx := ToolContext{Store: st, UserID: son.ID}
	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	found := fmt.Sprintf("#%d:", sonConv.ID)

	for _, tc := range []struct {
		args string
		want bool
	}{
		{fmt.Sprintf(`{"since":%q}`, today), true},
		{fmt.Sprintf(`{"until":%q}`, today), true},
		{fmt.Sprintf(`{"since":%q}`, tomorrow), false},
		{fmt.Sprintf(`{"until":%q}`, yesterday), false},
		{`{"character":"tutor"}`, true},
	} {
		out, err := executeHistoryList(tc.args, tctx)
		if err != nil {
			t.Fatalf("list %s: %v", tc.args, err)
		}
		if strings.Contains(out, found) != tc.want {
			t.Errorf("list %s = %q; want found=%v", tc.args, out, tc.want)
		}
		search := strings.Replace(tc.args, "{", `{"query":"guitar",`, 1)
		out, err = executeHistorySearch(search, tctx)
		if err != nil {
			t.Fatalf("search %s: %v", search, err)
		}
		if strings.Contains(out, found) != tc.want {
			t.Errorf("search %s = %q; want found=%v", search, out, tc.want)
		}
	}

	if _, err := executeHistoryList(`{"since":"last week"}`, tctx); err == nil || !strings.Contains(err.Error(), "YYYY-MM-DD") {
		t.Errorf("bad date error = %v", err)
	}
	if _, err := executeHistoryList(fmt.Sprintf(`{"since":%q,"until":%q}`, today, yesterday), tctx); err == nil {
		t.Error("until before since was accepted")
	}
	if _, err := executeHistoryList(`{"character":"chef"}`, tctx); err == nil || !strings.Contains(err.Error(), "Music Tutor") {
		t.Errorf("unknown character error = %v; want the character names", err)
	}
}

func TestHistoryToolsOnlySeeOwnConversations(t *testing.T) {
	st, dad, son, dadConv, sonConv := historyFixture(t)
	st.CreateMessage(dadConv.ID, "user", "guitar strings for Sam's birthday", nil, nil, nil, nil, "")
	// Even an admin's regular history tools only see their own conversations.
	for _, u := range []*store.User{son, dad} {
		tctx := ToolContext{Store: st, UserID: u.ID}
		other := dadConv
		if u == dad {
			other = sonConv
		}
		out, err := executeHistoryList(`{"user":"someone"}`, tctx)
		if err != nil || strings.Contains(out, fmt.Sprintf("#%d:", other.ID)) || !strings.Contains(out, "(the person you are talking to)") {
			t.Fatalf("%s list = %q, %v", u.Username, out, err)
		}
		if out, _ := executeHistorySearch(`{"query":"guitar"}`, tctx); strings.Contains(out, fmt.Sprintf("#%d:", other.ID)) {
			t.Fatalf("%s search found another user's conversation: %q", u.Username, out)
		}
		if _, err := executeHistoryRead(fmt.Sprintf(`{"conversation_id":%d}`, other.ID), tctx); err == nil {
			t.Fatalf("%s read another user's conversation", u.Username)
		}
		if _, err := executeHistorySearch(fmt.Sprintf(`{"query":"guitar","conversation_id":%d}`, other.ID), tctx); err == nil {
			t.Fatalf("%s searched inside another user's conversation", u.Username)
		}
	}
}

func TestAdminHistoryTools(t *testing.T) {
	st, dad, son, dadConv, sonConv := historyFixture(t)
	st.CreateMessage(dadConv.ID, "user", "guitar strings for Sam's birthday", nil, nil, nil, nil, "")
	model := "m"
	current, _ := st.CreateConversation(dad.ID, nil, &model, nil)
	dadCtx := ToolContext{Store: st, UserID: dad.ID, ConversationID: current.ID}
	sonCtx := ToolContext{Store: st, UserID: son.ID}

	// Non-admins are refused, even if a model calls a tool it wasn't offered.
	for name, exec := range map[string]toolExecutor{
		"admin_list_users":     executeAdminListUsers,
		"admin_history_list":   executeAdminHistoryList,
		"admin_history_search": executeAdminHistorySearch,
		"admin_history_read":   executeAdminHistoryRead,
	} {
		args := fmt.Sprintf(`{"query":"guitar","conversation_id":%d}`, sonConv.ID)
		if _, err := exec(args, sonCtx); err == nil || !strings.Contains(err.Error(), "admins only") {
			t.Errorf("%s for a non-admin: err = %v", name, err)
		}
	}

	out, err := executeAdminListUsers(`{}`, dadCtx)
	if err != nil || !strings.Contains(out, "username: dad | display name: Dad | role: admin | the person you are talking to | 2 conversations") || !strings.Contains(out, "username: sam | display name: Sam | role: user | 1 conversation,") {
		t.Fatalf("list users = %q, %v", out, err)
	}

	// No user: everyone's conversations, each saying whose it is.
	out, err = executeAdminHistoryList(`{}`, dadCtx)
	if err != nil || !strings.Contains(out, "conversations of all users") ||
		!strings.Contains(out, fmt.Sprintf("#%d (Sam (sam)): \"Guitar practice\"", sonConv.ID)) || !strings.Contains(out, fmt.Sprintf("#%d (Dad (dad)):", dadConv.ID)) {
		t.Fatalf("admin list all = %q, %v", out, err)
	}
	out, err = executeAdminHistorySearch(`{"query":"guitar"}`, dadCtx)
	if err != nil || !strings.Contains(out, fmt.Sprintf("#%d (Sam", sonConv.ID)) || !strings.Contains(out, fmt.Sprintf("#%d (Dad", dadConv.ID)) {
		t.Fatalf("admin search all = %q, %v", out, err)
	}

	// One user, by username or display name.
	out, err = executeAdminHistoryList(`{"user":"Sam"}`, dadCtx)
	if err != nil || !strings.Contains(out, "conversations of Sam (sam)") || strings.Contains(out, fmt.Sprintf("#%d ", dadConv.ID)) {
		t.Fatalf("admin list sam = %q, %v", out, err)
	}
	if _, err := executeAdminHistoryList(`{"user":"nobody"}`, dadCtx); err == nil || !strings.Contains(err.Error(), "Sam (sam)") {
		t.Fatalf("unknown user error = %v; want the list of users", err)
	}
	if _, err := executeAdminHistoryList(`{"character":"chef"}`, dadCtx); err == nil || !strings.Contains(err.Error(), "Music Tutor") {
		t.Fatalf("unknown character error = %v", err)
	}

	// Any conversation by id, with admin_ names in the hints.
	out, err = executeAdminHistorySearch(fmt.Sprintf(`{"query":"guitar","conversation_id":%d}`, sonConv.ID), dadCtx)
	if err != nil || !strings.Contains(out, "of Sam (sam)") || !strings.Contains(out, "using admin_history_read") {
		t.Fatalf("admin search in conversation = %q, %v", out, err)
	}
	out, err = executeAdminHistoryRead(fmt.Sprintf(`{"conversation_id":%d}`, sonConv.ID), dadCtx)
	if err != nil || !strings.Contains(out, "[message 1] Sam (user") {
		t.Fatalf("admin read = %q, %v", out, err)
	}
	if _, err := executeAdminHistoryRead(fmt.Sprintf(`{"conversation_id":%d}`, current.ID), dadCtx); err == nil {
		t.Fatal("admin read the current conversation")
	}
}

func TestHistoryReadStartAndCount(t *testing.T) {
	st := memoryTestStore(t)
	u, _ := st.CreateUser("u", nil, false, nil)
	model := "m"
	conv, _ := st.CreateConversation(u.ID, nil, &model, nil)
	for i := 1; i <= 10; i++ {
		st.CreateMessage(conv.ID, "user", fmt.Sprintf("line %d", i), nil, nil, nil, nil, "")
	}
	tctx := ToolContext{Store: st, UserID: u.ID}
	out, err := executeHistoryRead(fmt.Sprintf(`{"conversation_id":%d,"start":4,"count":3}`, conv.ID), tctx)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "line 3\n") || !strings.Contains(out, "[message 4]") || !strings.Contains(out, "[message 6]") || strings.Contains(out, "[message 7]") ||
		!strings.Contains(out, "Showing messages 4–6 of 10") || !strings.Contains(out, "start=7") {
		t.Fatalf("window = %q", out)
	}
	out, _ = executeHistoryRead(fmt.Sprintf(`{"conversation_id":%d,"start":9}`, conv.ID), tctx)
	if !strings.Contains(out, "Showing messages 9–10 of 10; that is the end") {
		t.Fatalf("tail = %q", out)
	}
	if _, err := executeHistoryRead(fmt.Sprintf(`{"conversation_id":%d,"start":11}`, conv.ID), tctx); err == nil {
		t.Fatal("start past the end was accepted")
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
	if !strings.Contains(out, "Showing messages 1–2 of 5") || !strings.Contains(out, "start=3") {
		t.Fatalf("first page = %q", out[len(out)-200:])
	}

	// One message bigger than the limit is cut off rather than skipped.
	conv2, _ := st.CreateConversation(u.ID, nil, &model, nil)
	st.CreateMessage(conv2.ID, "user", strings.Repeat("y", historyReadMaxChars+10), nil, nil, nil, nil, "")
	out, _ = executeHistoryRead(fmt.Sprintf(`{"conversation_id":%d}`, conv2.ID), tctx)
	if !strings.Contains(out, "[message cut off: 10 more characters]") {
		t.Fatalf("oversized message = %q", out[len(out)-200:])
	}
}

func TestAdminHistoryToolsWithheldFromNonAdmins(t *testing.T) {
	defs := ToolDefsForCharacter([]string{"history", "admin_history"})
	if len(defs) != 7 {
		t.Fatalf("history + admin_history expand to %d tools", len(defs))
	}
	var names []string
	for _, d := range withoutAdminTools(defs) {
		names = append(names, d.Function.Name)
	}
	if strings.Join(names, ",") != "history_list,history_search,history_read" {
		t.Fatalf("non-admin tools = %v", names)
	}
	// The regular tools have no way to name another user.
	for _, d := range defs[:3] {
		if _, ok := d.Function.Parameters.Properties["user"]; ok {
			t.Errorf("%s has a user argument", d.Function.Name)
		}
	}
	if _, ok := toolRegistry["admin_history_list"].Function.Parameters.Properties["user"]; !ok {
		t.Error("admin_history_list has no user argument")
	}
}

func TestHistorySearchRanksByWordsMatched(t *testing.T) {
	st := memoryTestStore(t)
	u, _ := st.CreateUser("u", nil, false, nil)
	model := "m"
	one, _ := st.CreateConversation(u.ID, nil, &model, nil)
	st.CreateMessage(one.ID, "user", "Chess openings", nil, nil, nil, nil, "")
	st.CreateMessage(one.ID, "user", "More chess", nil, nil, nil, nil, "")
	both, _ := st.CreateConversation(u.ID, nil, &model, nil)
	st.CreateMessage(both.ID, "user", "Chess club tournament", nil, nil, nil, nil, "")

	matches, err := st.SearchConversations(u.ID, 0, "chess tournament", store.HistoryFilter{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 || matches[0].ID != both.ID || matches[1].ID != one.ID || matches[1].MessageHits != 2 {
		t.Fatalf("matches = %+v", matches)
	}

	// The index follows deletions.
	st.DeleteConversation(both.ID, u.ID)
	if matches, _ := st.SearchConversations(u.ID, 0, "tournament", store.HistoryFilter{}, 10); len(matches) != 0 {
		t.Fatalf("deleted conversation still found: %+v", matches)
	}
}

func TestGetToolsHidesAdminToolsFromNonAdmins(t *testing.T) {
	InitTools(&config.Config{})
	s := &Server{}
	for _, tc := range []struct {
		admin bool
		want  bool
	}{{false, false}, {true, true}} {
		req := asUser(httptest.NewRequest("GET", "/api/tools", nil), &store.User{ID: 1, IsAdmin: tc.admin})
		w := httptest.NewRecorder()
		s.handleGetTools(w, req)
		var tools []ToolMeta
		if err := json.Unmarshal(w.Body.Bytes(), &tools); err != nil {
			t.Fatal(err)
		}
		got := false
		for _, tool := range tools {
			if tool.ID == "admin_history" {
				got = true
			}
		}
		if got != tc.want {
			t.Errorf("admin=%v: admin_history listed = %v", tc.admin, got)
		}
	}
}
