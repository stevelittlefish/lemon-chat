package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	user, err := s.store.UserByUsername(req.Username)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	if user.PasswordHash != nil {
		if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
			log.Printf("auth: failed login for user %q", req.Username)
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
	}

	log.Printf("auth: user %q logged in", req.Username)
	sessionID, err := s.store.CreateSession(user.ID)
	if err != nil {
		internalError(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})

	writeJSON(w, http.StatusOK, userResponse(user))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	log.Printf("Logout user_id=%d username=%q", user.ID, user.Username)
	cookie, err := r.Cookie("session")
	if err == nil {
		_ = s.store.DeleteSession(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "session",
		Value:   "",
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Unix(0, 0),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, userResponse(currentUser(r)))
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DisplayName *string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	user := currentUser(r)
	displayName := user.DisplayName
	if req.DisplayName != nil {
		if *req.DisplayName == "" {
			displayName = nil
		} else {
			displayName = req.DisplayName
		}
	}
	log.Printf("Updating profile user_id=%d", user.ID)
	if err := s.store.UpdateUser(user.ID, user.Username, user.PasswordHash, user.IsAdmin, displayName); err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if len(req.NewPassword) < 6 {
		writeError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	user := currentUser(r)

	if user.PasswordHash != nil {
		if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
			writeError(w, http.StatusUnauthorized, "current password is incorrect")
			return
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		internalError(w, err)
		return
	}
	hashStr := string(hash)
	log.Printf("Changing password user_id=%d", user.ID)
	if err := s.store.UpdateUser(user.ID, user.Username, &hashStr, user.IsAdmin, user.DisplayName); err != nil {
		internalError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func userResponse(u *store.User) map[string]any {
	return map[string]any{
		"id":           u.ID,
		"username":     u.Username,
		"display_name": u.DisplayName,
		"is_admin":     u.IsAdmin,
		"has_password": u.PasswordHash != nil,
		"has_avatar":   u.AvatarFilename != nil,
	}
}

func (s *Server) handleUploadUserAvatar(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	data, ext, err := receiveAvatar(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	prefix := fmt.Sprintf("user-%d", user.ID)
	filename, err := s.writeAvatar(prefix, data, ext)
	if err != nil {
		internalError(w, err)
		return
	}
	if err := s.store.SetUserAvatar(user.ID, filename); err != nil {
		internalError(w, err)
		return
	}
	log.Printf("Uploading user avatar user_id=%d", user.ID)
	writeJSON(w, http.StatusOK, map[string]any{"has_avatar": true})
}

func (s *Server) handleDeleteUserAvatar(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	log.Printf("Deleting user avatar user_id=%d", user.ID)
	if user.AvatarFilename != nil {
		s.deleteAvatarFile(*user.AvatarFilename)
	}
	if err := s.store.ClearUserAvatar(user.ID); err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleServeUserAvatar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	u, err := s.store.UserByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	if u.AvatarFilename == nil {
		writeError(w, http.StatusNotFound, "no avatar")
		return
	}
	s.serveAvatarFile(w, r, *u.AvatarFilename)
}
