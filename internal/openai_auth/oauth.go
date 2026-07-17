package openai_auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuth endpoint and client constants, mirroring the Codex CLI's public client.
const (
	issuer        = "https://auth.openai.com"
	authorizePath = "/oauth/authorize"
	tokenPath     = "/oauth/token"

	// clientID is OpenAI's public client identifier that unlocks
	// ChatGPT-subscription quota — identical in codex-rs, pi, and imp. It is not
	// a secret; PKCE protects the flow.
	clientID = "app_EMoamEEZ73f0CkXaXp7hrann"

	// originator identifies lemon-chat honestly to the backend.
	originator = "lemon-chat"

	// redirectPort is the primary loopback port the authorize redirect returns
	// to; it must match OpenAI's redirect allow-list (which cannot be changed to
	// a LAN address — hence the paste-the-code flow for remote hosting).
	redirectPort = 1455
	redirectPath = "/auth/callback"

	// scope requested during authorization.
	scope = "openid profile email offline_access"

	// refreshSkew is how long before expiry a token is considered stale and
	// eligible for a proactive refresh.
	refreshSkew = 60 * time.Second
)

// redirectURI is the loopback callback URL registered for the client. The port
// is fixed by OpenAI's allow-list, so both the local-browser flow and the
// paste-the-code flow declare the same value.
func redirectURI() string {
	return fmt.Sprintf("http://localhost:%d%s", redirectPort, redirectPath)
}

// Tokens is the set of credentials produced by a successful login and rotated
// on refresh.
type Tokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	IDToken      string    `json:"id_token"`
	AccountID    string    `json:"account_id"`
	Expiry       time.Time `json:"expiry"`
}

// Expired reports whether the access token is at or past expiry (minus skew).
func (t Tokens) Expired() bool {
	if t.Expiry.IsZero() {
		return true
	}
	return time.Now().After(t.Expiry.Add(-refreshSkew))
}

// authorizeURL builds the authorization URL the user's browser is sent to.
func authorizeURL(challenge, state string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI())
	q.Set("scope", scope)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("id_token_add_organizations", "true")
	q.Set("codex_cli_simplified_flow", "true")
	q.Set("state", state)
	q.Set("originator", originator)
	return issuer + authorizePath + "?" + q.Encode()
}

// tokenResponse is the raw token endpoint payload.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// toTokens converts a raw token response into stored Tokens, preserving the
// previous refresh token when the server omits a rotated one.
func (tr tokenResponse) toTokens(prevRefresh string) (Tokens, error) {
	if tr.Error != "" {
		return Tokens{}, fmt.Errorf("openai_auth: token endpoint error %q: %s", tr.Error, tr.ErrorDesc)
	}
	if tr.AccessToken == "" {
		return Tokens{}, fmt.Errorf("openai_auth: token response missing access_token")
	}
	refresh := tr.RefreshToken
	if refresh == "" {
		refresh = prevRefresh
	}
	expiry := time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	toks := Tokens{
		AccessToken:  tr.AccessToken,
		RefreshToken: refresh,
		IDToken:      tr.IDToken,
		Expiry:       expiry,
	}
	if acct := accountIDFromIDToken(tr.IDToken); acct != "" {
		toks.AccountID = acct
	}
	return toks, nil
}

// exchangeCode swaps an authorization code for tokens (the PKCE code exchange).
func exchangeCode(ctx context.Context, client *http.Client, code, verifier string) (Tokens, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", clientID)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI())
	form.Set("code_verifier", verifier)
	return postToken(ctx, client, form, "")
}

// refresh exchanges a refresh token for a new access token.
func refresh(ctx context.Context, client *http.Client, refreshToken string) (Tokens, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", clientID)
	form.Set("refresh_token", refreshToken)
	form.Set("scope", scope)
	return postToken(ctx, client, form, refreshToken)
}

// postToken performs a token-endpoint POST and decodes the result.
func postToken(ctx context.Context, client *http.Client, form url.Values, prevRefresh string) (Tokens, error) {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, "POST", issuer+tokenPath, strings.NewReader(form.Encode()))
	if err != nil {
		return Tokens{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return Tokens{}, err
	}
	defer resp.Body.Close()
	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return Tokens{}, fmt.Errorf("openai_auth: decoding token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK && tr.Error == "" {
		return Tokens{}, fmt.Errorf("openai_auth: token endpoint returned %d", resp.StatusCode)
	}
	return tr.toTokens(prevRefresh)
}

// accountIDFromIDToken best-effort extracts the ChatGPT account id from the
// id_token JWT claims. Returns "" if it cannot be found; the token is not
// verified here (it came directly from the token endpoint over TLS).
func accountIDFromIDToken(idToken string) string {
	claims := parseJWTClaims(idToken)
	if claims == nil {
		return ""
	}
	// The account id lives under a namespaced auth claim in Codex's id tokens.
	if authClaim, ok := claims["https://api.openai.com/auth"].(map[string]any); ok {
		if id, ok := authClaim["chatgpt_account_id"].(string); ok {
			return id
		}
	}
	if id, ok := claims["chatgpt_account_id"].(string); ok {
		return id
	}
	return ""
}

// parseJWTClaims decodes (without verifying) the payload of a JWT.
func parseJWTClaims(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil
	}
	return claims
}
