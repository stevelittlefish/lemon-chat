package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stevelittlefish/lemon-chat/internal/config"
)

func TestImageInputSupport(t *testing.T) {
	mux := http.NewServeMux()
	// OpenRouter-style listing: architecture present for two models, absent for one.
	mux.HandleFunc("GET /v1/models", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[
			{"id":"seeing","architecture":{"input_modalities":["text","image"]}},
			{"id":"blind","architecture":{"input_modalities":["text"]}},
			{"id":"ollama-model"},
			{"id":"silent"}]}`))
	})
	// Ollama's native show endpoint knows about one model only.
	mux.HandleFunc("POST /api/show", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Model string }
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Model != "ollama-model" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"capabilities":["completion","vision"]}`))
	})
	// Codex catalog.
	mux.HandleFunc("GET /codex/models", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("chatgpt-account-id") != "acct" {
			http.Error(w, "no account", http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`{"models":[{"slug":"gpt-x","input_modalities":["text","image"]},{"slug":"gpt-text","input_modalities":["text"]}]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	got, err := ImageInputSupport(context.Background(), srv.Client(), &config.ModelServer{APIBase: srv.URL + "/v1"}, "", "",
		[]string{"seeing", "blind", "ollama-model", "silent"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"seeing": true, "blind": false, "ollama-model": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v (silent must be absent)", got, want)
	}
	for id, v := range want {
		if got[id] != v {
			t.Errorf("%s = %v, want %v", id, got[id], v)
		}
	}

	codex := &config.ModelServer{APIBase: srv.URL + "/codex", API: config.APIResponses}
	got, err = ImageInputSupport(context.Background(), srv.Client(), codex, "tok", "acct", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got["gpt-x"] || got["gpt-text"] || len(got) != 2 {
		t.Errorf("codex = %v", got)
	}
}
