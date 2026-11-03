package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

// restrictedFixture: a paid server with one model opened to everyone, a free
// local server, and a shared character on the paid model.
func restrictedFixture(t *testing.T) (*Server, *store.Store) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	open := false
	cfg := &config.Config{
		ModelServers: []config.ModelServer{
			{Name: "paid", APIBase: "http://paid.invalid/v1", Restricted: true},
			{Name: "local", APIBase: "http://local.invalid/v1"},
		},
		Models: []config.Model{
			{Name: "big", DisplayName: "Big Model", ModelServer: "paid"},
			{Name: "cheap", ModelServer: "paid", Restricted: &open},
			{Name: "free", ModelServer: "local"},
		},
	}
	cfg.Server.DataDir = t.TempDir()
	return &Server{cfg: cfg, store: st, research: newResearchManager()}, st
}

func asUser(r *http.Request, u *store.User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), ctxUser, u))
}

func TestModelRestrictedInheritsAndOverrides(t *testing.T) {
	s, _ := restrictedFixture(t)
	for name, want := range map[string]bool{"big": true, "cheap": false, "free": false, "unknown": false} {
		if got := s.cfg.ModelRestricted(name); got != want {
			t.Errorf("ModelRestricted(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestModelListHidesRestrictedModels(t *testing.T) {
	s, st := restrictedFixture(t)
	kid, _ := st.CreateUser("kid", nil, false, nil)
	admin, _ := st.CreateUser("admin", nil, true, nil)
	trusted, _ := st.CreateUser("trusted", nil, false, nil)
	trusted.CanUseRestricted = true

	names := func(u *store.User) string {
		w := httptest.NewRecorder()
		s.handleModels(w, asUser(httptest.NewRequest("GET", "/api/models", nil), u))
		var got []struct{ Name string }
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, m := range got {
			out = append(out, m.Name)
		}
		return strings.Join(out, ",")
	}
	if got := names(kid); got != "cheap,free" {
		t.Errorf("kid sees %q, want cheap,free", got)
	}
	for _, u := range []*store.User{admin, trusted} {
		if got := names(u); got != "big,cheap,free" {
			t.Errorf("%s sees %q, want all models", u.Username, got)
		}
	}
}

func TestRestrictedModelRefusedForChatAndCharacters(t *testing.T) {
	s, st := restrictedFixture(t)
	kid, _ := st.CreateUser("kid", nil, false, nil)
	admin, _ := st.CreateUser("admin", nil, true, nil)

	// A new conversation on the paid model is refused with the display name.
	w := httptest.NewRecorder()
	s.handleCreateConversation(w, asUser(httptest.NewRequest("POST", "/api/conversations", strings.NewReader(`{"model":"big"}`)), kid))
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "Big Model is restricted") {
		t.Fatalf("create conversation: %d %s", w.Code, w.Body)
	}

	// Sending to an existing paid-model character is refused before any model call.
	char, err := st.CreateCharacter("Shared", "big", nil, nil, nil, admin.ID, "readwrite", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	conv, err := st.CreateConversation(kid.ID, nil, nil, &char.ID)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/x", strings.NewReader(`{"content":"hi"}`))
	r.SetPathValue("id", strconv.FormatInt(conv.ID, 10))
	w = httptest.NewRecorder()
	s.handleSendMessage(w, asUser(r, kid))
	if w.Code != http.StatusForbidden {
		t.Fatalf("send: %d %s", w.Code, w.Body)
	}

	// The character list flags it unavailable for the kid only.
	available := func(u *store.User) bool {
		w := httptest.NewRecorder()
		s.handleListCharacters(w, asUser(httptest.NewRequest("GET", "/api/characters", nil), u))
		var got []struct {
			ModelAvailable bool `json:"model_available"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || len(got) != 1 {
			t.Fatalf("list characters: %v %s", err, w.Body)
		}
		return got[0].ModelAvailable
	}
	if available(kid) || !available(admin) {
		t.Errorf("model_available: kid=%v admin=%v", available(kid), available(admin))
	}

	// Creating a character on the paid model is refused; on the open one it works.
	w = httptest.NewRecorder()
	s.handleCreateCharacter(w, asUser(httptest.NewRequest("POST", "/api/characters", strings.NewReader(`{"name":"Mine","model":"big"}`)), kid))
	if w.Code != http.StatusForbidden {
		t.Fatalf("create restricted character: %d", w.Code)
	}
	w = httptest.NewRecorder()
	s.handleCreateCharacter(w, asUser(httptest.NewRequest("POST", "/api/characters", strings.NewReader(`{"name":"Mine","model":"cheap"}`)), kid))
	if w.Code != http.StatusCreated {
		t.Fatalf("create open character: %d %s", w.Code, w.Body)
	}

	// Editing the shared character without changing its model is allowed;
	// switching another character to the paid model is not.
	edit := func(id int64, body string) int {
		r := httptest.NewRequest("PUT", "/x", strings.NewReader(body))
		r.SetPathValue("id", strconv.FormatInt(id, 10))
		w := httptest.NewRecorder()
		s.handleUpdateCharacter(w, asUser(r, kid))
		return w.Code
	}
	if code := edit(char.ID, `{"name":"Shared 2","model":"big"}`); code != http.StatusOK && code != http.StatusNoContent {
		t.Errorf("edit keeping restricted model: %d", code)
	}
	var mine struct{ ID int64 }
	_ = json.Unmarshal(w.Body.Bytes(), &mine)
	if code := edit(mine.ID, `{"name":"Mine","model":"big"}`); code != http.StatusForbidden {
		t.Errorf("switch own character to restricted model: %d", code)
	}
}

func TestAdminGrantsRestrictedAccess(t *testing.T) {
	s, st := restrictedFixture(t)
	admin, _ := st.CreateUser("admin", nil, true, nil)
	kid, _ := st.CreateUser("kid", nil, false, nil)

	r := httptest.NewRequest("PUT", "/x", strings.NewReader(`{"can_use_restricted":true}`))
	r.SetPathValue("id", strconv.FormatInt(kid.ID, 10))
	w := httptest.NewRecorder()
	s.handleAdminUpdateUser(w, asUser(r, admin))
	if w.Code != http.StatusNoContent {
		t.Fatalf("update: %d %s", w.Code, w.Body)
	}
	got, err := st.UserByID(kid.ID)
	if err != nil || !got.CanUseRestricted || got.IsAdmin {
		t.Fatalf("after grant: %+v, %v", got, err)
	}
	if !s.userCanUseModel(got, "big") {
		t.Error("granted user still refused")
	}

	w = httptest.NewRecorder()
	s.handleAdminCreateUser(w, asUser(httptest.NewRequest("POST", "/x", strings.NewReader(`{"username":"new"}`)), admin))
	if w.Code != http.StatusCreated || !strings.Contains(w.Body.String(), `"can_use_restricted":false`) {
		t.Fatalf("new users must default to no access: %d %s", w.Code, w.Body)
	}
}
