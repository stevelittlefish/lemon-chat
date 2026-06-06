package server

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
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
			Description: "Returns the current local date and time.",
			Parameters:  toolParam{Type: "object", Properties: map[string]any{}, Required: []string{}},
		},
	},
	"roll_dice": {
		Type: "function",
		Function: toolFunction{
			Name:        "roll_dice",
			Description: "Rolls dice using standard notation (e.g. 2d6, 1d20). Returns each die result and the total.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"notation": map[string]any{
						"type":        "string",
						"description": "Dice notation in NdM format, e.g. '2d6' for two six-sided dice.",
					},
				},
				Required: []string{"notation"},
			},
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
		now := time.Now()
		return now.Weekday().String() + ", " + now.Format(time.RFC3339), nil
	},
	"roll_dice": func(argsJSON string) (string, error) {
		var args struct {
			Notation string `json:"notation"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
		parts := strings.SplitN(strings.ToLower(args.Notation), "d", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid dice notation: %q", args.Notation)
		}
		count, err := strconv.Atoi(parts[0])
		if err != nil || count < 1 || count > 100 {
			return "", fmt.Errorf("invalid dice count: %q", parts[0])
		}
		sides, err := strconv.Atoi(parts[1])
		if err != nil || sides < 2 || sides > 10000 {
			return "", fmt.Errorf("invalid die sides: %q", parts[1])
		}
		rolls := make([]int, count)
		total := 0
		for i := range rolls {
			rolls[i] = rand.Intn(sides) + 1
			total += rolls[i]
		}
		if count == 1 {
			return fmt.Sprintf("Rolled %s: %d", args.Notation, rolls[0]), nil
		}
		parts2 := make([]string, count)
		for i, v := range rolls {
			parts2[i] = strconv.Itoa(v)
		}
		return fmt.Sprintf("Rolled %s: %s = %d", args.Notation, strings.Join(parts2, " + "), total), nil
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
	{"get_time", "Get current time", "Returns the current local date and time."},
	{"roll_dice", "Roll dice", "Rolls dice using standard notation (e.g. 2d6, 1d20)."},
}

func (s *Server) handleGetTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, AllTools)
}
