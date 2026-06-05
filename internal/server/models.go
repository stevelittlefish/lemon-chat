package server

import "net/http"

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	type modelResponse struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Default     bool   `json:"default"`
	}
	mode := r.URL.Query().Get("mode")
	defaultName := s.cfg.Server.DefaultModel
	var models []modelResponse
	for _, m := range s.cfg.Models {
		if mode != "" && !m.AvailableIn(mode) {
			continue
		}
		models = append(models, modelResponse{
			Name:        m.Name,
			DisplayName: m.DisplayName,
			Default:     m.Name == defaultName,
		})
	}
	if models == nil {
		models = []modelResponse{}
	}
	writeJSON(w, http.StatusOK, models)
}
