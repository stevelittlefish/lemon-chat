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
			Description: "Rolls dice using standard notation (e.g. 2d6, d20, 2d6+4, d20-5). Returns each die result and the total.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"notation": map[string]any{
						"type":        "string",
						"description": "Dice notation, e.g. '2d6', 'd20', '2d6+4', 'd20-5'. Count is optional (omit for 1 die).",
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
		halves := strings.SplitN(strings.ToLower(args.Notation), "d", 2)
		if len(halves) != 2 {
			return "", fmt.Errorf("invalid dice notation: %q", args.Notation)
		}

		// Empty prefix means 1 die (e.g. "d20")
		count := 1
		if halves[0] != "" {
			var err error
			count, err = strconv.Atoi(halves[0])
			if err != nil || count < 1 || count > 100 {
				return "", fmt.Errorf("invalid dice count: %q", halves[0])
			}
		}

		// Right part may include a modifier: "6", "6+4", "20-5"
		right := halves[1]
		modifier := 0
		sidesStr := right
		if idx := strings.IndexAny(right, "+-"); idx != -1 {
			var err error
			modifier, err = strconv.Atoi(right[idx:])
			if err != nil {
				return "", fmt.Errorf("invalid modifier: %q", right[idx:])
			}
			sidesStr = right[:idx]
		}
		sides, err := strconv.Atoi(sidesStr)
		if err != nil || sides < 2 || sides > 10000 {
			return "", fmt.Errorf("invalid die sides: %q", sidesStr)
		}

		rolls := make([]int, count)
		diceTotal := 0
		for i := range rolls {
			rolls[i] = rand.Intn(sides) + 1
			diceTotal += rolls[i]
		}
		total := diceTotal + modifier

		rollStrs := make([]string, count)
		for i, v := range rolls {
			rollStrs[i] = strconv.Itoa(v)
		}
		rollExpr := strings.Join(rollStrs, " + ")

		if modifier > 0 {
			return fmt.Sprintf("Rolled %s: %s + %d = %d", args.Notation, rollExpr, modifier, total), nil
		} else if modifier < 0 {
			return fmt.Sprintf("Rolled %s: %s - %d = %d", args.Notation, rollExpr, -modifier, total), nil
		} else if count == 1 {
			return fmt.Sprintf("Rolled %s: %d", args.Notation, rolls[0]), nil
		}
		return fmt.Sprintf("Rolled %s: %s = %d", args.Notation, rollExpr, total), nil
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
