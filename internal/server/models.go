package server

import "net/http"

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	type modelResponse struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Default     bool   `json:"default"`
	}
	defaultName := s.cfg.ModelServer.Default
	models := make([]modelResponse, len(s.cfg.Models))
	for i, m := range s.cfg.Models {
		models[i] = modelResponse{
			Name:        m.Name,
			DisplayName: m.DisplayName,
			Default:     m.Name == defaultName,
		}
	}
	writeJSON(w, http.StatusOK, models)
}
