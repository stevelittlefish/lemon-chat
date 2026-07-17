package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/openai_auth"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

type Server struct {
	cfg         *config.Config
	store       *store.Store
	hub         *Hub
	modelClient *http.Client
	research    *researchManager
	oauth       *openai_auth.Provider
}

func New(cfg *config.Config, st *store.Store, hub *Hub) *Server {
	dialTimeout := time.Duration(cfg.Server.DialTimeoutSeconds) * time.Second
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: dialTimeout,
			}).DialContext,
		},
	}
	InitTools(cfg)
	oauth := openai_auth.NewProvider(openai_auth.NewStoreAdapter(st), client)
	return &Server{cfg: cfg, store: st, hub: hub, modelClient: client, research: newResearchManager(), oauth: oauth}
}

// tokenSource returns a TokenSource that yields the correct bearer token for
// srv: the shared OAuth token when the server uses oauth, otherwise its static
// api_key. Resolving lazily (rather than baking in a string) lets long-running
// and detached work pick up refreshed OAuth tokens.
func (s *Server) tokenSource(srv *config.ModelServer) config.TokenSource {
	if srv.UsesOAuth() {
		return s.oauth.Token
	}
	return config.StaticToken(srv.APIKey)
}

// bearerToken resolves the current bearer token for srv.
func (s *Server) bearerToken(ctx context.Context, srv *config.ModelServer) (string, error) {
	return s.tokenSource(srv)(ctx)
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Auth
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.requireAuth(s.handleLogout))
	mux.HandleFunc("GET /api/auth/me", s.requireAuth(s.handleMe))
	mux.HandleFunc("PATCH /api/auth/profile", s.requireAuth(s.handleUpdateProfile))
	mux.HandleFunc("PATCH /api/auth/password", s.requireAuth(s.handleChangePassword))
	mux.HandleFunc("PUT /api/auth/avatar", s.requireAuth(s.handleUploadUserAvatar))
	mux.HandleFunc("DELETE /api/auth/avatar", s.requireAuth(s.handleDeleteUserAvatar))

	// User avatars (read by any authenticated user)
	mux.HandleFunc("GET /api/users/{id}/avatar", s.requireAuth(s.handleServeUserAvatar))

	// Models
	mux.HandleFunc("GET /api/models", s.requireAuth(s.handleModels))

	// Attachments
	mux.HandleFunc("GET /api/attachments/{id}", s.requireAuth(s.handleGetAttachment))

	// Tools
	mux.HandleFunc("GET /api/tools", s.requireAuth(s.handleGetTools))

	// Notes
	mux.HandleFunc("GET /api/notes", s.requireAuth(s.handleListNotes))
	mux.HandleFunc("GET /api/notes/{id}", s.requireAuth(s.handleGetNote))
	mux.HandleFunc("PUT /api/notes", s.requireAuth(s.handleUpsertNote))
	mux.HandleFunc("DELETE /api/notes/{id}", s.requireAuth(s.handleDeleteNote))
	mux.HandleFunc("PATCH /api/notes/{id}/read-only", s.requireAuth(s.handleSetNoteReadOnly))

	// Conversations
	mux.HandleFunc("GET /api/conversations", s.requireAuth(s.handleListConversations))
	mux.HandleFunc("POST /api/conversations", s.requireAuth(s.handleCreateConversation))
	mux.HandleFunc("PATCH /api/conversations/{id}", s.requireAuth(s.handleUpdateConversation))
	mux.HandleFunc("DELETE /api/conversations/{id}", s.requireAuth(s.handleDeleteConversation))
	mux.HandleFunc("POST /api/conversations/import_chat", s.requireAuth(s.handleImportConversation))
	mux.HandleFunc("POST /api/conversations/{id}/fork", s.requireAuth(s.handleForkConversation))
	mux.HandleFunc("POST /api/conversations/{id}/regenerate-title", s.requireAuth(s.handleRegenerateTitle))

	// Messages
	mux.HandleFunc("GET /api/conversations/{id}/messages", s.requireAuth(s.handleListMessages))
	mux.HandleFunc("POST /api/conversations/{id}/messages", s.requireAuth(s.handleSendMessage))
	mux.HandleFunc("GET /api/conversations/{id}/messages/{msgId}/context", s.requireAuth(s.handleGetMessageContext))
	mux.HandleFunc("POST /api/conversations/{id}/first-message", s.requireAuth(s.handleFirstMessage))

	// Characters
	mux.HandleFunc("GET /api/characters", s.requireAuth(s.handleListCharacters))
	mux.HandleFunc("GET /api/characters/{id}", s.requireAuth(s.handleGetCharacter))
	mux.HandleFunc("POST /api/characters", s.requireAuth(s.handleCreateCharacter))
	mux.HandleFunc("PATCH /api/characters/{id}", s.requireAuth(s.handleUpdateCharacter))
	mux.HandleFunc("DELETE /api/characters/{id}", s.requireAuth(s.handleDeleteCharacter))
	mux.HandleFunc("GET /api/characters/{id}/avatar", s.requireAuth(s.handleServeCharacterAvatar))
	mux.HandleFunc("PUT /api/characters/{id}/avatar", s.requireAuth(s.handleUploadCharacterAvatar))
	mux.HandleFunc("DELETE /api/characters/{id}/avatar", s.requireAuth(s.handleDeleteCharacterAvatar))
	mux.HandleFunc("POST /api/characters/{id}/set-default", s.requireAdmin(s.handleSetDefaultCharacter))
	mux.HandleFunc("DELETE /api/characters/default", s.requireAdmin(s.handleClearDefaultCharacter))

	// Admin
	mux.HandleFunc("GET /api/admin/users", s.requireAdmin(s.handleAdminListUsers))
	mux.HandleFunc("POST /api/admin/users", s.requireAdmin(s.handleAdminCreateUser))
	mux.HandleFunc("PATCH /api/admin/users/{id}", s.requireAdmin(s.handleAdminUpdateUser))
	mux.HandleFunc("DELETE /api/admin/users/{id}", s.requireAdmin(s.handleAdminDeleteUser))
	mux.HandleFunc("GET /api/admin/tools/models", s.requireAdmin(s.handleAdminListModels))
	mux.HandleFunc("POST /api/admin/note-packs/import", s.requireAdmin(s.handleAdminImportNotePack))

	// Research
	mux.HandleFunc("GET /api/research", s.requireAuth(s.handleListResearch))
	mux.HandleFunc("GET /api/research/defaults", s.requireAuth(s.handleResearchDefaults))
	mux.HandleFunc("POST /api/research", s.requireAuth(s.handleStartResearch))
	mux.HandleFunc("GET /api/research/{id}", s.requireAuth(s.handleGetResearch))
	mux.HandleFunc("GET /api/research/{id}/bundle", s.requireAuth(s.handleDownloadResearchBundle))
	mux.HandleFunc("GET /api/research/{id}/debug-bundle", s.requireAuth(s.handleDownloadResearchDebugBundle))
	mux.HandleFunc("GET /api/research/{id}/debug", s.requireAuth(s.handleGetResearchDebug))
	mux.HandleFunc("GET /api/research/{id}/report/document", s.requireAuth(s.handleGetResearchReportDocument))
	mux.HandleFunc("POST /api/research/{id}/reports", s.requireAuth(s.handleRegenerateResearchReport))
	mux.HandleFunc("GET /api/research/{id}/reports/{reportId}", s.requireAuth(s.handleGetResearchReportVariant))
	mux.HandleFunc("GET /api/research/{id}/reports/{reportId}/document", s.requireAuth(s.handleGetResearchReportVariantDocument))
	mux.HandleFunc("GET /api/research/{id}/events", s.requireAuth(s.handleResearchEvents))
	mux.HandleFunc("POST /api/research/{id}/cancel", s.requireAuth(s.handleCancelResearch))
	mux.HandleFunc("POST /api/research/{id}/reddit-import", s.requireAuth(s.handleResearchRedditImport))
	mux.HandleFunc("POST /api/research/{id}/reddit-skip", s.requireAuth(s.handleResearchRedditSkip))
	mux.HandleFunc("DELETE /api/research/{id}", s.requireAuth(s.handleDeleteResearch))
	mux.HandleFunc("POST /api/debug/reddit-import", s.debugOnly(s.requireAuth(s.handleRedditImportHarness)))

	// WebSocket
	mux.HandleFunc("GET /ws", s.handleWS)

	// Completions
	mux.HandleFunc("GET /api/completions", s.requireAuth(s.handleListCompletions))
	mux.HandleFunc("POST /api/completions", s.requireAuth(s.handleCreateCompletion))
	mux.HandleFunc("GET /api/completions/{id}", s.requireAuth(s.handleGetCompletion))
	mux.HandleFunc("PATCH /api/completions/{id}", s.requireAuth(s.handleUpdateCompletion))
	mux.HandleFunc("DELETE /api/completions/{id}", s.requireAuth(s.handleDeleteCompletion))
	mux.HandleFunc("POST /api/completions/{id}/run", s.requireAuth(s.handleRunCompletion))
	mux.HandleFunc("POST /api/completions/{id}/undo", s.requireAuth(s.handleUndoCompletion))
	mux.HandleFunc("POST /api/completions/{id}/redo", s.requireAuth(s.handleRedoCompletion))
	mux.HandleFunc("POST /api/completions/{id}/regenerate-title", s.requireAuth(s.handleRegenerateCompletionTitle))

	// Root — no-cache so deploys are picked up immediately
	mux.HandleFunc("GET /{$}", serveFile("static/index.html"))

	// Menu and feature pages
	mux.HandleFunc("GET /menu", serveFile("static/menu.html"))
	mux.HandleFunc("GET /complete", serveFile("static/complete.html"))
	mux.HandleFunc("GET /research", serveFile("static/research.html"))
	mux.HandleFunc("GET /research/help", serveFile("static/research-help.html"))
	mux.HandleFunc("GET /debug/reddit-import", s.debugOnly(serveFile("static/reddit-import-debug.html")))

	// Settings pages
	mux.HandleFunc("GET /settings", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/settings/account", http.StatusFound)
	})
	mux.HandleFunc("GET /settings.html", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/settings/account", http.StatusMovedPermanently)
	})
	mux.HandleFunc("GET /settings/account", serveFile("static/settings/account.html"))
	mux.HandleFunc("GET /settings/characters", serveFile("static/settings/characters.html"))
	mux.HandleFunc("GET /settings/characters/new", serveFile("static/settings/character-edit.html"))
	mux.HandleFunc("GET /settings/characters/{id}/edit", serveFile("static/settings/character-edit.html"))
	mux.HandleFunc("GET /settings/users", serveFile("static/settings/users.html"))
	mux.HandleFunc("GET /settings/tools", serveFile("static/settings/tools.html"))
	mux.HandleFunc("GET /settings/notes", serveFile("static/settings/notes.html"))
	mux.HandleFunc("GET /settings/import_chat", serveFile("static/settings/import_chat.html"))

	// Static files — no forced no-cache; ETags handle revalidation naturally
	mux.Handle("/", http.FileServer(http.Dir("static")))

	return mux
}

func (s *Server) debugOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.cfg.Server.Debug {
			http.NotFound(w, r)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func internalError(w http.ResponseWriter, err error) {
	log.Printf("error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

// pathID parses the "id" path parameter as an int64. On failure it writes a
// 400 response and returns ok=false; callers should return immediately.
func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

// notFoundOr500 maps a store error to an HTTP response: store.ErrNotFound
// becomes 404, any other non-nil error becomes 500. It returns true when it
// wrote a response (err was non-nil), so callers can write
// `if notFoundOr500(w, err) { return }`.
func notFoundOr500(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return true
	}
	internalError(w, err)
	return true
}

func serveFile(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, path)
	}
}
