package server

import (
	"net/http"
	"sort"

	"github.com/stevelittlefish/lemon-chat/internal/config"
)

// ToolMeta is the metadata shape returned to the frontend.
type ToolMeta struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Group       string `json:"group"`
	Configured  bool   `json:"configured"`
	ConfigHint  string `json:"config_hint,omitempty"`
}

var (
	allTools  []ToolMeta
	executors = staticExecutors(toolRegistry)
)

func staticExecutors(registry map[string]toolDef) map[string]toolExecutor {
	out := make(map[string]toolExecutor)
	for id, tool := range registry {
		if tool.Executor != nil {
			out[id] = tool.Executor
		}
	}
	return out
}

func configureSearXNG(cfg *config.Config) (bool, string, toolExecutor) {
	if cfg.SearXNG.URL == "" {
		return false, "Add [searxng] url = \"http://…\" to lemon.toml to enable SearXNG search.", executeSearXNG
	}
	return true, "", executeSearXNG
}

func configureSDXL(cfg *config.Config) (bool, string, toolExecutor) {
	if cfg.ComfyUI.URL == "" || cfg.ComfyUI.SDXLWorkflow == "" {
		return false, "Add [comfyui] url and sdxl_workflow to lemon.toml to enable SDXL image generation.", nil
	}
	return true, "", makeImageExecutor(cfg.ComfyUI.URL, cfg.ComfyUI.SDXLWorkflow, 30, 7.0)
}

func configureFlux(cfg *config.Config) (bool, string, toolExecutor) {
	if cfg.ComfyUI.URL == "" || cfg.ComfyUI.FluxWorkflow == "" {
		return false, "Add [comfyui] url and flux_workflow to lemon.toml to enable Flux Schnell image generation.", nil
	}
	return true, "", makeImageExecutor(cfg.ComfyUI.URL, cfg.ComfyUI.FluxWorkflow, 4, 1.0)
}

// InitTools derives executors and frontend metadata from toolRegistry. Call once
// at server startup after configuration has loaded.
func InitTools(cfg *config.Config) {
	executors = make(map[string]toolExecutor)
	allTools = allTools[:0]

	for id, tool := range toolRegistry {
		configured := true
		hint := ""
		executor := tool.Executor
		if tool.Configure != nil {
			configured, hint, executor = tool.Configure(cfg)
		}
		if executor != nil {
			executors[id] = executor
		}
		if tool.DisplayName == "" {
			continue
		}

		description := tool.Summary
		if description == "" {
			description = tool.Function.Description
		}
		allTools = append(allTools, ToolMeta{
			ID:          id,
			DisplayName: tool.DisplayName,
			Description: description,
			Group:       tool.Group,
			Configured:  configured,
			ConfigHint:  hint,
		})
	}

	sort.Slice(allTools, func(i, j int) bool {
		return toolRegistry[allTools[i].ID].Order < toolRegistry[allTools[j].ID].Order
	})
}

func (s *Server) handleGetTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, allTools)
}
