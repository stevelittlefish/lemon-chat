package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/openai_auth"
)

// pendingLogins holds in-flight paste-the-code logins between the begin and
// complete requests, keyed by admin user id. The PKCE verifier + state never
// leave the process. Entries are short-lived; a new begin replaces any prior one
// for that user.
type pendingLogins struct {
	mu sync.Mutex
	m  map[int64]openai_auth.PendingLogin
}

func newPendingLogins() *pendingLogins {
	return &pendingLogins{m: map[int64]openai_auth.PendingLogin{}}
}

func (p *pendingLogins) put(userID int64, pl openai_auth.PendingLogin) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.m[userID] = pl
}

func (p *pendingLogins) take(userID int64) (openai_auth.PendingLogin, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	pl, ok := p.m[userID]
	delete(p.m, userID)
	return pl, ok
}

// handleOpenAIStatus reports whether an OpenAI account is linked.
func (s *Server) handleOpenAIStatus(w http.ResponseWriter, r *http.Request) {
	linked, accountID, expiry, err := s.oauth.Status()
	if err != nil {
		internalError(w, err)
		return
	}
	resp := map[string]any{"linked": linked, "account_id": accountID}
	if !expiry.IsZero() {
		resp["expiry"] = expiry.UTC().Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleOpenAILoginBegin starts a paste-the-code login: it returns the authorize
// URL to open and stashes the pending PKCE secrets for handleOpenAILoginComplete.
func (s *Server) handleOpenAILoginBegin(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	authURL, pending, err := openai_auth.Begin()
	if err != nil {
		internalError(w, err)
		return
	}
	s.pendingLogins.put(user.ID, pending)
	log.Printf("Beginning OpenAI account link user_id=%d", user.ID)
	writeJSON(w, http.StatusOK, map[string]string{"authorize_url": authURL})
}

// handleOpenAILoginComplete finishes a paste-the-code login: it parses whatever
// the admin pasted (the redirected URL or a bare code), exchanges it, and
// persists the shared token.
func (s *Server) handleOpenAILoginComplete(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var req struct {
		Pasted string `json:"pasted"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	pending, ok := s.pendingLogins.take(user.ID)
	if !ok {
		writeError(w, http.StatusBadRequest, "no login in progress — start again")
		return
	}
	code, state, err := openai_auth.ParsePasted(req.Pasted)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	tokens, err := openai_auth.Complete(r.Context(), s.modelClient, code, state, pending)
	if err != nil {
		log.Printf("OpenAI account link failed user_id=%d: %v", user.ID, err)
		writeError(w, http.StatusBadGateway, "sign-in failed: "+err.Error())
		return
	}
	if err := s.oauth.SetTokens(tokens); err != nil {
		internalError(w, err)
		return
	}
	log.Printf("Linked OpenAI account user_id=%d account_id=%q", user.ID, tokens.AccountID)
	writeJSON(w, http.StatusOK, map[string]any{"linked": true, "account_id": tokens.AccountID})
}

// handleOpenAIDisconnect removes the stored OpenAI token.
func (s *Server) handleOpenAIDisconnect(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if err := s.oauth.Unlink(); err != nil {
		internalError(w, err)
		return
	}
	log.Printf("Unlinked OpenAI account user_id=%d", user.ID)
	writeJSON(w, http.StatusOK, map[string]bool{"linked": false})
}
