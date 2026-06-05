package server

import (
	"fmt"
	"net/http"
	"time"
)

type toolParam struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
	Required   []string       `json:"required"`
}

type toolFunction struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Parameters  toolParam `json:"parameters"`
}

type toolDef struct {
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

var toolRegistry = map[string]toolDef{
	"get_time": {
		Type: "function",
		Function: toolFunction{
			Name:        "get_time",
			Description: "Returns the current UTC date and time.",
			Parameters:  toolParam{Type: "object", Properties: map[string]any{}, Required: []string{}},
		},
	},
}

// ToolDefsForCharacter returns tool definitions for the given tool IDs.
func ToolDefsForCharacter(toolIDs []string) []toolDef {
	var out []toolDef
	for _, id := range toolIDs {
		if def, ok := toolRegistry[id]; ok {
			out = append(out, def)
		}
	}
	return out
}

var executors = map[string]func(string) (string, error){
	"get_time": func(_ string) (string, error) {
		now := time.Now().UTC()
		return now.Weekday().String() + ", " + now.Format(time.RFC3339), nil
	},
}

// ExecuteTool runs a tool by name and returns its result string.
func ExecuteTool(name, argsJSON string) (string, error) {
	fn, ok := executors[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return fn(argsJSON)
}

// ToolMeta is the metadata shape returned to the frontend.
type ToolMeta struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

// AllTools is the full list of available tools exposed to the frontend.
var AllTools = []ToolMeta{
	{"get_time", "Get current time", "Returns the current UTC date and time."},
}

func (s *Server) handleGetTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, AllTools)
}
