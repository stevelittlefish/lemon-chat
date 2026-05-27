package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

type contextKey int

const ctxUser contextKey = 0

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		userID, err := s.store.SessionUserID(cookie.Value)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		user, err := s.store.UserByID(userID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		ctx := context.WithValue(r.Context(), ctxUser, user)
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(ctxUser).(*store.User)
		if !user.IsAdmin {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	})
}

func currentUser(r *http.Request) *store.User {
	return r.Context().Value(ctxUser).(*store.User)
}
