package openai_auth

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

// Login drives the interactive PKCE login flow: it starts a loopback callback
// server, opens the user's browser to the authorization page, waits for the
// redirect, validates state, and exchanges the code for tokens.
//
// It blocks until the flow completes, ctx is cancelled, or loginTimeout elapses.
func Login(ctx context.Context, client *http.Client) (Tokens, error) {
	const loginTimeout = 5 * time.Minute
	ctx, cancel := context.WithTimeout(ctx, loginTimeout)
	defer cancel()

	pk, err := newPKCE()
	if err != nil {
		return Tokens{}, err
	}
	state, err := randomURLSafe(24)
	if err != nil {
		return Tokens{}, err
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", redirectPort))
	if err != nil {
		return Tokens{}, fmt.Errorf("openai_auth: cannot bind loopback port %d for OAuth callback: %w", redirectPort, err)
	}

	type result struct {
		code string
		err  error
	}
	resultCh := make(chan result, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(redirectPath, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if e := q.Get("error"); e != "" {
			http.Error(w, "Authorization failed. You can close this tab.", http.StatusBadRequest)
			resultCh <- result{err: fmt.Errorf("openai_auth: authorization error: %s", e)}
			return
		}
		if q.Get("state") != state {
			http.Error(w, "Invalid state. You can close this tab.", http.StatusBadRequest)
			resultCh <- result{err: fmt.Errorf("openai_auth: state mismatch on callback")}
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "Missing code. You can close this tab.", http.StatusBadRequest)
			resultCh <- result{err: fmt.Errorf("openai_auth: callback missing authorization code")}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, callbackHTML)
		resultCh <- result{code: code}
	})

	srv := &http.Server{Handler: mux}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			resultCh <- result{err: err}
		}
	}()
	defer srv.Close()

	authURL := authorizeURL(pk.Challenge, state)
	if err := openBrowser(authURL); err != nil {
		log.Printf("openai_auth: could not open browser automatically (%v)", err)
	}
	log.Printf("openai_auth: open this URL to authorize:\n%s", authURL)

	select {
	case <-ctx.Done():
		return Tokens{}, ctx.Err()
	case res := <-resultCh:
		if res.err != nil {
			return Tokens{}, res.err
		}
		return exchangeCode(ctx, client, res.code, pk.Verifier)
	}
}

// openBrowser attempts to open url in the user's default browser. Best effort;
// callers should also print the URL.
func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler"}
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

const callbackHTML = `<!doctype html><html><head><meta charset="utf-8"><title>lemon-chat</title></head>` +
	`<body style="font-family:system-ui;padding:3rem;color:#1a1a1a;background:#faf8f2">` +
	`<h1 style="font-weight:600">Signed in</h1>` +
	`<p>You can close this tab and return to lemon-chat.</p></body></html>`
