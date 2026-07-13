package server

import (
	"log"
	"net/http"
)

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	type modelResponse struct {
		Name          string   `json:"name"`
		DisplayName   string   `json:"display_name"`
		Default       bool     `json:"default"`
		PromptUSD     *float64 `json:"prompt_usd,omitempty"`     // USD per prompt token
		CompletionUSD *float64 `json:"completion_usd,omitempty"` // USD per completion token
	}
	mode := r.URL.Query().Get("mode")
	defaultName := s.cfg.Server.DefaultModel
	// Prices are best-effort chrome; a lookup failure just omits them.
	prices, err := s.store.ModelPrices()
	if err != nil {
		log.Printf("Loading model prices: %v", err)
		prices = nil
	}
	var models []modelResponse
	for _, m := range s.cfg.Models {
		if mode != "" && !m.AvailableIn(mode) {
			continue
		}
		resp := modelResponse{
			Name:        m.Name,
			DisplayName: m.DisplayName,
			Default:     m.Name == defaultName,
		}
		if p, ok := prices[m.Name]; ok {
			prompt, completion := p.PromptUSD, p.CompletionUSD
			resp.PromptUSD, resp.CompletionUSD = &prompt, &completion
		}
		models = append(models, resp)
	}
	if models == nil {
		models = []modelResponse{}
	}
	writeJSON(w, http.StatusOK, models)
}
