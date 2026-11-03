package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/stevelittlefish/lemon-chat/internal/llm"
	"github.com/stevelittlefish/lemon-chat/internal/store"
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
		// CanUseRestricted grants access to restricted (typically paid) models.
		CanUseRestricted bool `json:"can_use_restricted"`
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
	log.Printf("Creating user username=%q is_admin=%t can_use_restricted=%t admin_id=%d", req.Username, req.IsAdmin, req.CanUseRestricted, currentUser(r).ID)
	user, err := s.store.CreateUser(req.Username, hash, req.IsAdmin, req.DisplayName)
	if isDuplicateUsername(err) {
		writeError(w, http.StatusConflict, "username already taken")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	if req.CanUseRestricted {
		if err := s.store.SetUserCanUseRestricted(user.ID, true); err != nil {
			internalError(w, err)
			return
		}
		user.CanUseRestricted = true
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
		// CanUseRestricted grants access to restricted (typically paid) models.
		CanUseRestricted *bool `json:"can_use_restricted"`
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

	canUseRestricted := existing.CanUseRestricted
	if req.CanUseRestricted != nil {
		canUseRestricted = *req.CanUseRestricted
	}

	log.Printf("Updating user id=%d username=%q is_admin=%t can_use_restricted=%t password_changed=%t admin_id=%d", id, username, isAdmin, canUseRestricted, req.Password != nil, currentUser(r).ID)
	if err := s.store.UpdateUser(id, username, passwordHash, isAdmin, displayName); isDuplicateUsername(err) {
		writeError(w, http.StatusConflict, "username already taken")
		return
	} else if err != nil {
		internalError(w, err)
		return
	}
	if canUseRestricted != existing.CanUseRestricted {
		if err := s.store.SetUserCanUseRestricted(id, canUseRestricted); err != nil {
			internalError(w, err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminListModels(w http.ResponseWriter, r *http.Request) {
	type providerResponse struct {
		Name    string   `json:"name"`
		APIBase string   `json:"api_base"`
		Models  []string `json:"models"`
		Error   string   `json:"error,omitempty"`
	}

	user := currentUser(r)
	log.Printf("server: handleAdminListModels — querying %d provider(s) user_id=%d username=%q", len(s.cfg.ModelServers), user.ID, user.Username)
	providers := make([]providerResponse, len(s.cfg.ModelServers))
	var wg sync.WaitGroup
	for i := range s.cfg.ModelServers {
		srv := &s.cfg.ModelServers[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("server: handleAdminListModels — querying provider=%q api_base=%q", srv.Name, srv.APIBase)
			prov := llm.NewProvider(s.modelClient, srv, s.tokenSource(srv), s.oauthAccountID(srv))
			models, err := prov.ListModels(r.Context())
			result := providerResponse{
				Name:    srv.Name,
				APIBase: srv.APIBase,
				Models:  models,
			}
			if err != nil {
				result.Models = []string{}
				result.Error = err.Error()
				log.Printf("server: handleAdminListModels — provider=%q failed: %v", srv.Name, err)
			} else {
				log.Printf("server: handleAdminListModels — provider=%q returned %d model(s)", srv.Name, len(models))
			}
			providers[i] = result
		}()
	}
	wg.Wait()
	log.Printf("server: handleAdminListModels — completed %d provider(s) user_id=%d", len(providers), user.ID)
	writeJSON(w, http.StatusOK, providers)
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
	// Deleting a user wipes all their data, so the caller must echo the
	// username back — the UI makes the admin type it.
	if r.URL.Query().Get("confirm") != target.Username {
		writeError(w, http.StatusBadRequest, "confirm must match the username")
		return
	}
	log.Printf("Deleting user id=%d username=%q admin_id=%d", id, target.Username, currentUser(r).ID)
	// Stop any in-flight research for this user; its rows are about to go.
	if jobs, err := s.store.ListResearchJobs(id, -1, 0); err == nil {
		for _, j := range jobs {
			if run := s.research.get(j.ID); run != nil {
				log.Printf("Cancelling research job id=%d for deleted user id=%d", j.ID, id)
				run.requestCancel()
			}
		}
	}
	attPaths, avatars, err := s.store.DeleteUser(id)
	if notFoundOr500(w, err) {
		return
	}
	store.RemoveAttachmentFiles(s.cfg.Server.DataDir, attPaths)
	for _, f := range avatars {
		s.deleteAvatarFile(f)
	}
	w.WriteHeader(http.StatusNoContent)
}
