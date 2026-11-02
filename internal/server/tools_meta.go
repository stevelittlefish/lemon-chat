package server

import (
	"log"
	"net/http"
	"sort"

	"github.com/stevelittlefish/lemon-chat/internal/comfyui"
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
	// staticToolRegistry keeps the compiled-in definitions so InitTools can be
	// re-run (tests do) without inheriting a previous config's schema overrides.
	staticToolRegistry = cloneToolRegistry(toolRegistry)
)

func cloneToolRegistry(registry map[string]toolDef) map[string]toolDef {
	out := make(map[string]toolDef, len(registry))
	for id, tool := range registry {
		out[id] = tool
	}
	return out
}

func staticExecutors(registry map[string]toolDef) map[string]toolExecutor {
	out := make(map[string]toolExecutor)
	for id, tool := range registry {
		if tool.Executor != nil {
			out[id] = tool.Executor
		}
	}
	return out
}

func configureSearXNG(cfg *config.Config) toolConfig {
	if cfg.SearXNG.URL == "" {
		return toolConfig{Hint: "Add [searxng] url = \"http://…\" to lemon.toml to enable SearXNG search.", Executor: executeSearXNG}
	}
	return toolConfig{Configured: true, Executor: executeSearXNG}
}

func configureImageGen(cfg *config.Config) toolConfig {
	c := cfg.ComfyUI
	if c.URL == "" || len(c.Profiles) == 0 {
		return toolConfig{Hint: "Add [comfyui] url, default_profile and at least one [comfyui.profile.<name>] to lemon.toml to enable image generation."}
	}
	reg, err := comfyui.Load(c.Workflows, c.Profiles, c.DefaultProfile)
	if err != nil {
		log.Printf("Image generation disabled: %v", err)
		return toolConfig{Hint: "Image generation is misconfigured: " + err.Error()}
	}
	reg.LogStatus()
	fn := imageGenFunction(reg)
	return toolConfig{Configured: true, Executor: makeImageExecutor(c.URL, reg), Function: &fn}
}

// InitTools derives executors and frontend metadata from toolRegistry. Call once
// at server startup after configuration has loaded.
func InitTools(cfg *config.Config) {
	executors = make(map[string]toolExecutor)
	allTools = allTools[:0]

	for id, tool := range staticToolRegistry {
		toolRegistry[id] = tool
		configured := true
		hint := ""
		executor := tool.Executor
		if tool.Configure != nil {
			tc := tool.Configure(cfg)
			configured, hint, executor = tc.Configured, tc.Hint, tc.Executor
			if tc.Function != nil {
				tool.Function = *tc.Function
				toolRegistry[id] = tool
			}
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
