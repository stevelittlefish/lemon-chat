package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

func TestAdminDeleteUserRequiresTypedConfirmation(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	adminUser, err := st.CreateUser("admin", nil, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	victim, err := st.CreateUser("victim", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	cfg.Server.DataDir = t.TempDir()
	s := &Server{cfg: cfg, store: st, research: newResearchManager()}

	del := func(query string) int {
		r := httptest.NewRequest("DELETE", "/api/admin/users/x"+query, nil)
		r.SetPathValue("id", strconv.FormatInt(victim.ID, 10))
		r = r.WithContext(context.WithValue(r.Context(), ctxUser, adminUser))
		w := httptest.NewRecorder()
		s.handleAdminDeleteUser(w, r)
		return w.Code
	}

	for _, q := range []string{"", "?confirm=", "?confirm=Victim", "?confirm=admin"} {
		if code := del(q); code != http.StatusBadRequest {
			t.Fatalf("query %q: status %d, want 400", q, code)
		}
	}
	if _, err := st.UserByID(victim.ID); err != nil {
		t.Fatalf("user deleted without confirmation: %v", err)
	}

	if code := del("?confirm=victim"); code != http.StatusNoContent {
		t.Fatalf("confirmed delete: status %d, want 204", code)
	}
	if _, err := st.UserByID(victim.ID); err == nil {
		t.Fatal("user still exists after confirmed delete")
	}
}
