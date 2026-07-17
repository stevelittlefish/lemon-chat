// Package openai_auth implements a Codex-style OAuth 2.0 (PKCE) login flow
// against OpenAI's auth server, producing refreshable tokens that lemon-chat
// stores once and shares across all users on the home network.
package openai_auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// pkce holds a generated PKCE verifier/challenge pair.
type pkce struct {
	Verifier  string
	Challenge string
}

// newPKCE generates a fresh PKCE pair using the S256 challenge method.
func newPKCE() (pkce, error) {
	verifier, err := randomURLSafe(32)
	if err != nil {
		return pkce{}, err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return pkce{Verifier: verifier, Challenge: challenge}, nil
}

// randomURLSafe returns nBytes of cryptographic randomness as a base64url
// (no padding) string, suitable for PKCE verifiers and state values.
func randomURLSafe(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
