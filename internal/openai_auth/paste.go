package openai_auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// The paste-the-code flow exists because OpenAI's redirect allow-list pins the
// callback to http://localhost:1455/auth/callback and cannot be pointed at a LAN
// address. When lemon-chat runs on a home-network box but the browser is on a
// different machine, the loopback callback can never reach the server. Instead:
//
//  1. Begin() returns an authorize URL to open and a PendingLogin to hold.
//  2. The human opens the URL, signs in, and is redirected to the (dead)
//     localhost:1455 URL, which won't load — but the address bar carries
//     ?code=…&state=…
//  3. They copy that URL (or the bare code) back into lemon-chat.
//  4. ParsePasted() extracts the code/state and Complete() exchanges it.
//
// It is a second driver over the same PKCE flow as Login, sharing the crypto and
// token exchange; the redirect_uri is the same fixed loopback value.

// PendingLogin carries the in-flight secrets of a paste-the-code login between
// Begin and Complete: the PKCE verifier proving this client started the flow and
// the state to match the redirect against. It is held in-process (keyed to the
// admin's session) and never reaches the wire.
type PendingLogin struct {
	Verifier string
	State    string
}

// Begin starts a paste-the-code login for a caller whose browser is on another
// machine. It mints PKCE + state and returns the authorize URL to open plus the
// pending secrets to hold until Complete.
func Begin() (authURL string, pending PendingLogin, err error) {
	pk, err := newPKCE()
	if err != nil {
		return "", PendingLogin{}, err
	}
	state, err := randomURLSafe(24)
	if err != nil {
		return "", PendingLogin{}, err
	}
	return authorizeURL(pk.Challenge, state), PendingLogin{Verifier: pk.Verifier, State: state}, nil
}

// Complete finishes a paste-the-code login: it checks the returned state against
// the pending login (a bare-code paste carries none; the PKCE verifier guards the
// exchange either way) and swaps the code for tokens.
func Complete(ctx context.Context, client *http.Client, code, state string, p PendingLogin) (Tokens, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return Tokens{}, fmt.Errorf("openai_auth: no authorization code")
	}
	if state != "" && p.State != "" && state != p.State {
		return Tokens{}, fmt.Errorf("openai_auth: state mismatch — start the sign-in again")
	}
	return exchangeCode(ctx, client, code, p.Verifier)
}

// ParsePasted pulls the authorization code (and state, if present) out of
// whatever the human pasted: the whole redirected URL
// (http://localhost:1455/auth/callback?code=…&state=…) or a bare code snipped
// out of it. A pasted error redirect is surfaced as an error.
func ParsePasted(s string) (code, state string, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "", fmt.Errorf("openai_auth: nothing pasted")
	}
	// No query form at all ⇒ the human copied just the code; take it verbatim.
	if !strings.ContainsAny(s, "?=") {
		return s, "", nil
	}
	raw := s
	if u, e := url.Parse(s); e == nil && u.RawQuery != "" {
		raw = u.RawQuery
	} else if i := strings.IndexByte(s, '?'); i >= 0 {
		raw = s[i+1:]
	}
	vals, e := url.ParseQuery(raw)
	if e != nil {
		return "", "", fmt.Errorf("openai_auth: could not read the pasted URL: %w", e)
	}
	if errStr := vals.Get("error"); errStr != "" {
		if d := vals.Get("error_description"); d != "" {
			errStr = d
		}
		return "", "", fmt.Errorf("openai_auth: sign-in was refused: %s", errStr)
	}
	code = vals.Get("code")
	if code == "" {
		return "", "", fmt.Errorf("openai_auth: no code in the pasted URL")
	}
	return code, vals.Get("state"), nil
}
