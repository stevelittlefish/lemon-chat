package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func isDuplicateUsername(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: user.username")
}

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers()
	if err != nil {
		internalError(w, err)
		return
	}
	out := make([]map[string]any, len(users))
	for i, u := range users {
		out[i] = userResponse(&u)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string  `json:"username"`
		DisplayName *string `json:"display_name"`
		Password    *string `json:"password"`
		IsAdmin     bool    `json:"is_admin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Username == "" {
		writeError(w, http.StatusBadRequest, "username required")
		return
	}
	var hash *string
	if req.Password != nil && *req.Password != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			internalError(w, err)
			return
		}
		s := string(h)
		hash = &s
	}
	user, err := s.store.CreateUser(req.Username, hash, req.IsAdmin, req.DisplayName)
	if isDuplicateUsername(err) {
		writeError(w, http.StatusConflict, "username already taken")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, userResponse(user))
}

func (s *Server) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	existing, err := s.store.UserByID(id)
	if notFoundOr500(w, err) {
		return
	}

	var req struct {
		Username    *string `json:"username"`
		DisplayName *string `json:"display_name"`
		Password    *string `json:"password"`
		IsAdmin     *bool   `json:"is_admin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	username := existing.Username
	if req.Username != nil {
		username = *req.Username
	}
	isAdmin := existing.IsAdmin
	if req.IsAdmin != nil {
		isAdmin = *req.IsAdmin
	}
	displayName := existing.DisplayName
	if req.DisplayName != nil {
		if *req.DisplayName == "" {
			displayName = nil
		} else {
			displayName = req.DisplayName
		}
	}
	passwordHash := existing.PasswordHash
	if req.Password != nil {
		if *req.Password == "" {
			passwordHash = nil
		} else {
			h, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
			if err != nil {
				internalError(w, err)
				return
			}
			s := string(h)
			passwordHash = &s
		}
	}

	if err := s.store.UpdateUser(id, username, passwordHash, isAdmin, displayName); isDuplicateUsername(err) {
		writeError(w, http.StatusConflict, "username already taken")
		return
	} else if err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminImportNotePack(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)

	var pack struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Version string `json:"version"`
		Notes   []struct {
			Key      string `json:"key"`
			Value    string `json:"value"`
			ReadOnly bool   `json:"read_only"`
		} `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&pack); err != nil {
		writeError(w, http.StatusBadRequest, "invalid note pack JSON")
		return
	}
	if pack.ID == "" || pack.Name == "" || pack.Version == "" {
		writeError(w, http.StatusBadRequest, "note pack must have id, name, and version")
		return
	}
	if len(pack.Notes) == 0 {
		writeError(w, http.StatusBadRequest, "note pack contains no notes")
		return
	}

	log.Printf("Importing note pack id=%q name=%q version=%q notes=%d user_id=%d username=%q",
		pack.ID, pack.Name, pack.Version, len(pack.Notes), user.ID, user.Username)

	imported := 0
	for _, n := range pack.Notes {
		ro := n.ReadOnly
		note, err := s.store.UpsertNoteAdmin(n.Key, n.Value, &ro, nil, nil)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("note %q: %v", n.Key, err))
			return
		}
		log.Printf("  note id=%d key=%q read_only=%v", note.ID, note.Key, ro)
		imported++
	}

	log.Printf("Note pack import complete: %d note(s) imported/updated", imported)
	writeJSON(w, http.StatusOK, map[string]any{
		"imported":     imported,
		"pack_id":      pack.ID,
		"pack_name":    pack.Name,
		"pack_version": pack.Version,
	})
}

func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if id == currentUser(r).ID {
		writeError(w, http.StatusForbidden, "cannot delete your own account")
		return
	}
	target, err := s.store.UserByID(id)
	if notFoundOr500(w, err) {
		return
	}
	if target.AvatarFilename != nil {
		s.deleteAvatarFile(*target.AvatarFilename)
	}
	if err := s.store.DeleteUser(id); err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
