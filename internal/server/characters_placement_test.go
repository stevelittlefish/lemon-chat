package server

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestCharacterPlacementRules(t *testing.T) {
	s, st := restrictedFixture(t)
	admin, _ := st.CreateUser("admin", nil, true, nil)
	kid, _ := st.CreateUser("kid", nil, false, nil)
	shared, _ := st.CreateCharacter("Shared", "free", nil, nil, nil, admin.ID, "readonly", false, nil)
	private, _ := st.CreateCharacter("Private", "free", nil, nil, nil, admin.ID, "private", false, nil)
	def, _ := st.CreateCharacter("Default", "free", nil, nil, nil, admin.ID, "readonly", false, nil)
	if err := st.SetDefaultCharacter(def.ID); err != nil {
		t.Fatal(err)
	}

	place := func(id int64, body string) (int, string) {
		r := httptest.NewRequest("PATCH", "/x", strings.NewReader(body))
		r.SetPathValue("id", strconv.FormatInt(id, 10))
		w := httptest.NewRecorder()
		s.handleSetCharacterPlacement(w, asUser(r, admin))
		return w.Code, w.Body.String()
	}

	if code, body := place(shared.ID, `{"recommended":true}`); code != http.StatusOK || !strings.Contains(body, `"is_recommended":true`) {
		t.Fatalf("recommend shared: %d %s", code, body)
	}
	if code, _ := place(private.ID, `{"recommended":true}`); code != http.StatusBadRequest {
		t.Errorf("recommend private: %d, want 400", code)
	}
	if code, _ := place(def.ID, `{"disabled":true}`); code != http.StatusBadRequest {
		t.Errorf("disable default: %d, want 400", code)
	}
	// Disabling clears recommended; omitted fields are kept otherwise.
	if code, body := place(shared.ID, `{"disabled":true}`); code != http.StatusOK || !strings.Contains(body, `"is_recommended":false`) {
		t.Fatalf("disable shared: %d %s", code, body)
	}
	got, _ := st.GetCharacter(shared.ID)
	if !got.IsDisabled || got.IsRecommended {
		t.Fatalf("stored placement: disabled=%v recommended=%v", got.IsDisabled, got.IsRecommended)
	}

	// A disabled character can't start a conversation or become the default...
	w := httptest.NewRecorder()
	body := `{"character_id":` + strconv.FormatInt(shared.ID, 10) + `}`
	s.handleCreateConversation(w, asUser(httptest.NewRequest("POST", "/api/conversations", strings.NewReader(body)), kid))
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "Shared is disabled") {
		t.Fatalf("new conversation with disabled character: %d %s", w.Code, w.Body)
	}
	r := httptest.NewRequest("POST", "/x", nil)
	r.SetPathValue("id", strconv.FormatInt(shared.ID, 10))
	w = httptest.NewRecorder()
	s.handleSetDefaultCharacter(w, asUser(r, admin))
	if w.Code != http.StatusBadRequest {
		t.Errorf("set disabled character as default: %d, want 400", w.Code)
	}

	// ...or be switched to from another conversation.
	conv, err := st.CreateConversation(kid.ID, nil, nil, &def.ID)
	if err != nil {
		t.Fatal(err)
	}
	r = httptest.NewRequest("POST", "/x", strings.NewReader(`{"content":"hi","character_id":`+strconv.FormatInt(shared.ID, 10)+`}`))
	r.SetPathValue("id", strconv.FormatInt(conv.ID, 10))
	w = httptest.NewRecorder()
	s.handleSendMessage(w, asUser(r, kid))
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "disabled") {
		t.Fatalf("switch to disabled character: %d %s", w.Code, w.Body)
	}

	// Re-enabling brings it back (recommended stays off until set again).
	if code, _ := place(shared.ID, `{"disabled":false}`); code != http.StatusOK {
		t.Errorf("re-enable: %d", code)
	}
}
