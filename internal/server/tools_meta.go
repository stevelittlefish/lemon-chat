package server

import (
	"net/http"

	"github.com/stevelittlefish/lemon-chat/internal/config"
)

// ToolMeta is the metadata shape returned to the frontend.
type ToolMeta struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Configured  bool   `json:"configured"`
	ConfigHint  string `json:"config_hint,omitempty"`
}

var allTools []ToolMeta

// InitTools wires up executors for configured image tools and builds the list
// returned by GET /api/tools. Call once at server startup.
func InitTools(cfg *config.Config) {
	allTools = []ToolMeta{
		{"get_time", "Get current time", "Returns the current local date and time.", true, ""},
		{"roll_dice", "Roll dice", "Rolls dice using standard notation (e.g. 2d6, 1d20).", true, ""},
		{"pick_random", "Pick random", "Picks one item at random from a list of options.", true, ""},
		{"random_chance", "Random chance", "Resolves a success/failure check by rolling a die against a threshold.", true, ""},
		{"fetch_url", "Fetch URL", "Fetches a URL and returns its content as markdown, or raw HTML if source is true.", true, ""},
		{"create_document", "Create document", "Saves a file (report, script, notes, etc.) the user can download.", true, ""},
		{"wikipedia_search", "Wikipedia search", "Searches Wikipedia and returns matching article titles and snippets.", true, ""},
		{"wikipedia_get_page", "Wikipedia get page", "Fetches a Wikipedia article intro + TOC, or a specific section by name.", true, ""},
		{"world_state", "World State", "Session state tools: set, modify, remove, and list named values scoped to this conversation.", true, ""},
		{"notes", "Notes", "Note store: save, load, list, delete, and append to named notes across global, user, and conversation scopes.", true, ""},
		{"note_to_self", "Note to self", toolRegistry["note_to_self"].Function.Description, true, ""},
	}

	searxngConfigured := cfg.SearXNG.URL != ""
	searxngHint := ""
	if !searxngConfigured {
		searxngHint = "Add [searxng] url = \"http://…\" to lemon.toml to enable SearXNG search."
	}
	allTools = append(allTools, ToolMeta{"searxng", "SearXNG", "Searches the web via SearXNG and returns the top results.", searxngConfigured, searxngHint})

	sdxlConfigured := cfg.ComfyUI.URL != "" && cfg.ComfyUI.SDXLWorkflow != ""
	sdxlHint := ""
	if !sdxlConfigured {
		sdxlHint = "Add [comfyui] url and sdxl_workflow to lemon.toml to enable SDXL image generation."
	} else {
		executors["generate_image_sdxl"] = makeImageExecutor(cfg.ComfyUI.URL, cfg.ComfyUI.SDXLWorkflow, 30, 7.0)
	}
	allTools = append(allTools, ToolMeta{"generate_image_sdxl", "Generate image (SDXL)", toolRegistry["generate_image_sdxl"].Function.Description, sdxlConfigured, sdxlHint})

	fluxConfigured := cfg.ComfyUI.URL != "" && cfg.ComfyUI.FluxWorkflow != ""
	fluxHint := ""
	if !fluxConfigured {
		fluxHint = "Add [comfyui] url and flux_workflow to lemon.toml to enable Flux Schnell image generation."
	} else {
		executors["generate_image_flux"] = makeImageExecutor(cfg.ComfyUI.URL, cfg.ComfyUI.FluxWorkflow, 4, 1.0)
	}
	allTools = append(allTools, ToolMeta{"generate_image_flux", "Generate image (Flux Schnell)", toolRegistry["generate_image_flux"].Function.Description, fluxConfigured, fluxHint})
}

func (s *Server) handleGetTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, allTools)
}
