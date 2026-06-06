package server

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

type Server struct {
	cfg         *config.Config
	store       *store.Store
	hub         *Hub
	modelClient *http.Client
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
	return &Server{cfg: cfg, store: st, hub: hub, modelClient: client}
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

	// Conversations
	mux.HandleFunc("GET /api/conversations", s.requireAuth(s.handleListConversations))
	mux.HandleFunc("POST /api/conversations", s.requireAuth(s.handleCreateConversation))
	mux.HandleFunc("PATCH /api/conversations/{id}", s.requireAuth(s.handleUpdateConversation))
	mux.HandleFunc("DELETE /api/conversations/{id}", s.requireAuth(s.handleDeleteConversation))
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

	// Admin
	mux.HandleFunc("GET /api/admin/users", s.requireAdmin(s.handleAdminListUsers))
	mux.HandleFunc("POST /api/admin/users", s.requireAdmin(s.handleAdminCreateUser))
	mux.HandleFunc("PATCH /api/admin/users/{id}", s.requireAdmin(s.handleAdminUpdateUser))
	mux.HandleFunc("DELETE /api/admin/users/{id}", s.requireAdmin(s.handleAdminDeleteUser))
	mux.HandleFunc("POST /api/admin/tools/delete-orphaned-messages", s.requireAdmin(s.handleAdminDeleteOrphanedMessages))

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

	// Static files — no forced no-cache; ETags handle revalidation naturally
	mux.Handle("/", http.FileServer(http.Dir("static")))

	return mux
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

func serveFile(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, path)
	}
}
