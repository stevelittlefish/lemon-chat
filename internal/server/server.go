package server

import (
	"encoding/json"
	"net/http"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

type Server struct {
	cfg   *config.Config
	store *store.Store
	hub   *Hub
}

func New(cfg *config.Config, st *store.Store, hub *Hub) *Server {
	return &Server{cfg: cfg, store: st, hub: hub}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Auth
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.requireAuth(s.handleLogout))
	mux.HandleFunc("GET /api/auth/me", s.requireAuth(s.handleMe))
	mux.HandleFunc("PATCH /api/auth/password", s.requireAuth(s.handleChangePassword))

	// Models
	mux.HandleFunc("GET /api/models", s.requireAuth(s.handleModels))

	// Conversations
	mux.HandleFunc("GET /api/conversations", s.requireAuth(s.handleListConversations))
	mux.HandleFunc("POST /api/conversations", s.requireAuth(s.handleCreateConversation))
	mux.HandleFunc("DELETE /api/conversations/{id}", s.requireAuth(s.handleDeleteConversation))

	// Messages
	mux.HandleFunc("GET /api/conversations/{id}/messages", s.requireAuth(s.handleListMessages))
	mux.HandleFunc("POST /api/conversations/{id}/messages", s.requireAuth(s.handleSendMessage))

	// Characters
	mux.HandleFunc("GET /api/characters", s.requireAuth(s.handleListCharacters))
	mux.HandleFunc("POST /api/characters", s.requireAuth(s.handleCreateCharacter))
	mux.HandleFunc("PATCH /api/characters/{id}", s.requireAuth(s.handleUpdateCharacter))
	mux.HandleFunc("DELETE /api/characters/{id}", s.requireAuth(s.handleDeleteCharacter))

	// Admin
	mux.HandleFunc("GET /api/admin/users", s.requireAdmin(s.handleAdminListUsers))
	mux.HandleFunc("POST /api/admin/users", s.requireAdmin(s.handleAdminCreateUser))
	mux.HandleFunc("PATCH /api/admin/users/{id}", s.requireAdmin(s.handleAdminUpdateUser))
	mux.HandleFunc("DELETE /api/admin/users/{id}", s.requireAdmin(s.handleAdminDeleteUser))

	// WebSocket
	mux.HandleFunc("GET /ws", s.handleWS)

	// Static files
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
