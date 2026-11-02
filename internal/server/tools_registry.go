package server

import "github.com/stevelittlefish/lemon-chat/internal/config"

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

type toolExecutor func(string, ToolContext) (string, error)

// toolConfigurer reports whether a tool's external dependencies are configured
// and returns its executor. Function, when non-nil, replaces the tool's static
// model-facing definition (for schemas that depend on config).
type toolConfigurer func(*config.Config) toolConfig

type toolConfig struct {
	Configured bool
	Hint       string
	Executor   toolExecutor
	Function   *toolFunction
}

type toolDef struct {
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`

	DisplayName string         `json:"-"`
	Summary     string         `json:"-"`
	Group       string         `json:"-"`
	Order       int            `json:"-"`
	Executor    toolExecutor   `json:"-"`
	Members     []string       `json:"-"`
	Configure   toolConfigurer `json:"-"`
}

var toolRegistry = map[string]toolDef{
	"create_document": {
		DisplayName: "Create document",
		Summary:     "Saves a file (report, script, notes, etc.) the user can download.",
		Group:       "General",
		Order:       60,
		Executor:    executeCreateDocument,
		Type:        "function",
		Function: toolFunction{
			Name:        "create_document",
			Description: "Creates a downloadable file. Use for reports, plans, scripts, code, or any content the user will want to save. Choose the filename extension to match the content type (e.g. report.md, script.py, notes.txt).",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"title": map[string]any{
						"type":        "string",
						"description": "Human-readable title shown in the chat.",
					},
					"filename": map[string]any{
						"type":        "string",
						"description": "Suggested filename including extension, e.g. report.md or analysis.py.",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Full text content of the document.",
					},
				},
				Required: []string{"title", "filename", "content"},
			},
		},
	},
	"searxng": {
		DisplayName: "SearXNG",
		Summary:     "Searches the web via SearXNG and returns the top results.",
		Group:       "Search & knowledge",
		Order:       90,
		Executor:    executeSearXNG,
		Configure:   configureSearXNG,
		Type:        "function",
		Function: toolFunction{
			Name:        "searxng",
			Description: "Search the web via SearXNG and return the top results.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The search query.",
					},
					"max_results": map[string]any{
						"type":        "integer",
						"description": "Number of results to return (1–40, default 10).",
					},
					"page": map[string]any{
						"type":        "integer",
						"description": "Page number of results (default 1). Use higher values to retrieve more results beyond the first page.",
					},
				},
				Required: []string{"query"},
			},
		},
	},
	"get_time": {
		DisplayName: "Get current time",
		Summary:     "Returns the current local date and time.",
		Group:       "General",
		Order:       10,
		Executor:    executeGetTime,
		Type:        "function",
		Function: toolFunction{
			Name:        "get_time",
			Description: "Returns the current local date and time.",
			Parameters:  toolParam{Type: "object", Properties: map[string]any{}, Required: []string{}},
		},
	},
	"roll_dice": {
		DisplayName: "Roll dice",
		Summary:     "Rolls dice using standard notation (e.g. 2d6, 1d20).",
		Group:       "General",
		Order:       20,
		Executor:    executeRollDice,
		Type:        "function",
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
	"pick_random": {
		DisplayName: "Pick random",
		Summary:     "Picks one item at random from a list of options.",
		Group:       "General",
		Order:       30,
		Executor:    executePickRandom,
		Type:        "function",
		Function: toolFunction{
			Name:        "pick_random",
			Description: "Picks one item at random from a list of options. Use when you want a random outcome from a defined set — encounter type, weather, NPC mood, loot, etc. Define all options before calling; the result is server-side random and cannot be influenced.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"options": map[string]any{
						"type":        "array",
						"description": "The options to pick from. Must contain at least 2 items.",
						"items":       map[string]any{"type": "string"},
					},
				},
				Required: []string{"options"},
			},
		},
	},
	"random_chance": {
		DisplayName: "Random chance",
		Summary:     "Resolves a success/failure check by rolling a die against a threshold.",
		Group:       "General",
		Order:       40,
		Executor:    executeRandomChance,
		Type:        "function",
		Function: toolFunction{
			Name: "random_chance",
			Description: `Resolves a binary success/failure check by rolling a die and comparing the result to a threshold. Use this whenever an action has an uncertain outcome — picking a lock, landing a hit, resisting a spell, sneaking past a guard.

CRITICAL: You must call this tool BEFORE narrating the outcome of any uncertain action. The server rolls the die; the result is binding. You must narrate exactly what the result says — do not reinterpret, soften, or override a failure as a success (or vice versa).

How to use:
  - action:    Describe exactly what is being determined, e.g. "player picks the lock" or "enemy notices the player".
  - dice:      Which die to roll, e.g. "d20", "d6", "d100". A single die only — no count prefix, no modifier.
  - threshold: The minimum roll needed for success (roll >= threshold = success).

Examples:
  - Hard d20 check (need 18+): dice="d20", threshold=18 → succeeds on rolls 18, 19, or 20 (15% chance)
  - Medium d20 check (need 10+): dice="d20", threshold=10 → succeeds on rolls 10–20 (55% chance)
  - Percentage-based check: use dice="d100". For a 70% success rate set threshold=31 (rolls 31–100 succeed, 70 of 100 outcomes). Formula: threshold = 101 - desired_percentage.`,
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"action": map[string]any{
						"type":        "string",
						"description": "What is being determined, e.g. \"player picks the lock\" or \"arrow hits the target\". Used in the result message.",
					},
					"dice": map[string]any{
						"type":        "string",
						"description": "The die to roll, e.g. \"d20\", \"d6\", \"d100\". Single die only — no count prefix, no modifier.",
					},
					"threshold": map[string]any{
						"type":        "integer",
						"description": "Minimum roll needed to succeed. Roll >= threshold = success. For a percentage-based check on d100, use threshold = 101 - desired_percentage (e.g. 70% chance → threshold 31).",
					},
				},
				Required: []string{"action", "dice", "threshold"},
			},
		},
	},
	"fetch_url": {
		DisplayName: "Fetch URL",
		Summary:     "Fetches a URL and returns its content as markdown, or raw HTML if source is true.",
		Group:       "General",
		Order:       50,
		Executor:    executeFetchURL,
		Type:        "function",
		Function: toolFunction{
			Name:        "fetch_url",
			Description: "Fetches the content of a URL. Returns a clean markdown summary by default. Set source to true to get the raw HTML source instead.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "The URL to fetch.",
					},
					"source": map[string]any{
						"type":        "boolean",
						"description": "If true, return the raw HTML source instead of a markdown summary.",
					},
				},
				Required: []string{"url"},
			},
		},
	},
	"generate_image": {
		DisplayName: "Generate image",
		Summary:     "Generates an image with ComfyUI from a text prompt, using a configured profile or workflow.",
		Group:       "Image generation",
		Order:       100,
		Configure:   configureImageGen,
		Type:        "function",
		// Function is replaced at startup by imageGenFunction, which adds the
		// configured profiles and workflow IDs as enums.
		Function: imageGenFunction(nil),
	},
	"view_image": {
		DisplayName: "View image",
		Summary:     "Lets the model look at an image generated in this conversation. Only offered to vision models.",
		Group:       "Image generation",
		Order:       110,
		Executor:    executeViewImage,
		Type:        "function",
		Function: toolFunction{
			Name:        "view_image",
			Description: "Look at an image generated earlier in this conversation, e.g. to check it matches what the user asked for or to answer questions about it. The image is shown to you in the next message. Waits for the image if it is still generating.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"image_id": map[string]any{
						"type":        []any{"integer", "null"},
						"description": "The attachment_id returned by generate_image. Null or omit for the most recently generated image.",
					},
				},
				Required: []string{},
			},
		},
	},
	"wikipedia_search": {
		DisplayName: "Wikipedia search",
		Summary:     "Searches Wikipedia and returns matching article titles and snippets.",
		Group:       "Search & knowledge",
		Order:       70,
		Executor:    executeWikipediaSearch,
		Type:        "function",
		Function: toolFunction{
			Name:        "wikipedia_search",
			Description: "Searches Wikipedia and returns matching article titles and snippets. Use this to find the correct article title before calling wikipedia_get_page.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The search query.",
					},
					"max_results": map[string]any{
						"type":        "integer",
						"description": "Number of results to return (1–10, default 5).",
					},
					"lang": map[string]any{
						"type":        "string",
						"description": "Wikipedia language code (e.g. \"en\", \"fr\", \"de\"). Defaults to \"en\".",
					},
				},
				Required: []string{"query"},
			},
		},
	},
	"wikipedia_get_page": {
		DisplayName: "Wikipedia get page",
		Summary:     "Fetches a Wikipedia article intro + TOC, or a specific section by name.",
		Group:       "Search & knowledge",
		Order:       80,
		Executor:    executeWikipediaGetPage,
		Type:        "function",
		Function: toolFunction{
			Name:        "wikipedia_get_page",
			Description: "Fetches a Wikipedia article. Without a section argument, returns the intro paragraph and a table of contents listing all sections. With a section argument, returns the full text of that section. Use iteratively to read an article section by section.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"title": map[string]any{
						"type":        "string",
						"description": "The Wikipedia article title, exactly as returned by wikipedia_search.",
					},
					"section": map[string]any{
						"type":        "string",
						"description": "Section name to retrieve. Omit to get the intro and table of contents.",
					},
					"lang": map[string]any{
						"type":        "string",
						"description": "Wikipedia language code (e.g. \"en\", \"fr\", \"de\"). Defaults to \"en\".",
					},
				},
				Required: []string{"title"},
			},
		},
	},
	"state_set": {
		Executor: executeStateSet,
		Type:     "function",
		Function: toolFunction{
			Name:        "state_set",
			Description: "Sets one or more named values in the conversation's persistent state. Use this when something is first established: a stat, a condition, an inventory item, or any named fact. Replaces the value if the key already exists.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"items": map[string]any{
						"type":        "array",
						"description": "List of key/value pairs to set.",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"key": map[string]any{
									"type":        "string",
									"description": "The name of the value to set. Use short, descriptive keys (e.g. \"hp\", \"max_hp\", \"poisoned\", \"torch\").",
								},
								"value": map[string]any{
									"type":        "string",
									"description": "The value to store. Numbers, text, and flags (\"yes\"/\"no\") are all valid.",
								},
							},
							"required": []string{"key", "value"},
						},
					},
				},
				Required: []string{"items"},
			},
		},
	},
	"state_modify": {
		Executor: executeStateModify,
		Type:     "function",
		Function: toolFunction{
			Name:        "state_modify",
			Description: "Adds a numeric delta to a stored value. Use this for any numeric change — damage, healing, spending gold, item counts. Never compute the new value yourself; always call this tool so the result is reliable. Errors if the key does not exist or the stored value is not numeric.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"key": map[string]any{
						"type":        "string",
						"description": "The key whose value to modify.",
					},
					"delta": map[string]any{
						"type":        "number",
						"description": "Amount to add. Use negative values to subtract (e.g. -3 for 3 points of damage).",
					},
				},
				Required: []string{"key", "delta"},
			},
		},
	},
	"state_unset": {
		Executor: executeStateUnset,
		Type:     "function",
		Function: toolFunction{
			Name:        "state_unset",
			Description: "Removes a named key from state. Use when a condition ends, an item is fully consumed, or a fact is no longer relevant. Errors if the key is not set.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"key": map[string]any{
						"type":        "string",
						"description": "The key to remove.",
					},
				},
				Required: []string{"key"},
			},
		},
	},
	"state_list": {
		Executor: executeStateList,
		Type:     "function",
		Function: toolFunction{
			Name:        "state_list",
			Description: "Returns all currently stored key/value pairs for this conversation. Call this at the start of a session or before any check that depends on current state.",
			Parameters:  toolParam{Type: "object", Properties: map[string]any{}, Required: []string{}},
		},
	},
	"state_clear": {
		Executor: executeStateClear,
		Type:     "function",
		Function: toolFunction{
			Name:        "state_clear",
			Description: "Deletes all state for this conversation. Use when starting a fresh game or resetting a session. Returns how many keys were removed.",
			Parameters:  toolParam{Type: "object", Properties: map[string]any{}, Required: []string{}},
		},
	},
	"note_to_self": {
		DisplayName: "Note to self",
		Group:       "Reasoning",
		Order:       140,
		Executor:    executeNoteToSelf,
		Type:        "function",
		Function: toolFunction{
			Name:        "note_to_self",
			Description: "Records a private thought, plan, or reminder visible only in the model's context — not shown in chat. Use to track intentions, reasoning steps, or reminders without surfacing them to the user.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"text": map[string]any{
						"type":        "string",
						"description": "The thought or note to record.",
					},
				},
				Required: []string{"text"},
			},
		},
	},
	"note_save": {
		Executor: executeNoteSave,
		Type:     "function",
		Function: toolFunction{
			Name: "note_save",
			Description: `Saves a note. Creates or replaces the note at the given key. Fails if the note is marked read-only.

Keys must start with a scope prefix:
  g.  global — visible to all users and sessions, persists forever
  u.  user   — visible only to you, persists across all your conversations
  c.  conversation — scoped to this conversation, deleted when the conversation is deleted

After the prefix, use lowercase letters, digits, underscores, hyphens, and dots as segment separators (e.g. g.eldoria.bestiary). No consecutive dots, no trailing dot.

Use notes for long-form prose content: lore entries, NPC descriptions, session briefs, memories. For short numeric or boolean values use state_set instead.`,
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"key": map[string]any{
						"type":        "string",
						"description": "Full key including scope prefix, e.g. \"g.eldoria\", \"u.quest_log\", \"c.session_brief\".",
					},
					"value": map[string]any{
						"type":        "string",
						"description": "Content to store. Leading and trailing whitespace is stripped automatically.",
					},
				},
				Required: []string{"key", "value"},
			},
		},
	},
	"note_load": {
		Executor: executeNoteLoad,
		Type:     "function",
		Function: toolFunction{
			Name:        "note_load",
			Description: "Loads a single note by its exact key. Returns the full value, read-only status, and last-updated timestamp. Returns an error if the note does not exist or is not accessible in the current scope.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"key": map[string]any{
						"type":        "string",
						"description": "Exact key including scope prefix, e.g. \"g.eldoria\", \"u.quest_log\".",
					},
				},
				Required: []string{"key"},
			},
		},
	},
	"note_list": {
		Executor: executeNoteList,
		Type:     "function",
		Function: toolFunction{
			Name: "note_list",
			Description: `Lists accessible notes. Returns {"notes": [...], "message": "..."} where each note has key, excerpt, read_only, and updated_at. Does not return full values — use note_load to retrieve content. At most 50 notes are returned; if the list is truncated a message field explains this — use a more specific prefix to narrow results.

The prefix parameter controls what is searched:
  - Omit or pass "" to list all accessible notes.
  - Pass a scope letter ("g", "u", or "c") to list all notes in that scope.
  - Pass a scoped path like "g.eldoria" to list global notes at that key or under it.
  - Pass a bare term like "test" to find keys starting with g.test, u.test, or c.test.

Call note_list at the start of a session to discover what notes are available, then note_load specific keys you need.`,
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"prefix": map[string]any{
						"type":        "string",
						"description": "Optional filter. Scope letter (\"g\", \"u\", \"c\") lists all in that scope; scoped path (\"g.eldoria\") finds global notes starting with that key; bare term (\"test\") finds keys starting with g.test, u.test, or c.test.",
					},
				},
				Required: []string{},
			},
		},
	},
	"note_delete": {
		Executor: executeNoteDelete,
		Type:     "function",
		Function: toolFunction{
			Name:        "note_delete",
			Description: "Deletes a note by its exact key. Returns an error if the note does not exist, is not accessible, or is marked read-only.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"key": map[string]any{
						"type":        "string",
						"description": "Exact key including scope prefix.",
					},
				},
				Required: []string{"key"},
			},
		},
	},
	"note_append": {
		Executor: executeNoteAppend,
		Type:     "function",
		Function: toolFunction{
			Name:        "note_append",
			Description: "Appends text to an existing note without overwriting it. A blank line is inserted between the existing content and the new text. If the note does not exist it is created. Fails if the note is marked read-only. Use this to accumulate information incrementally — NPC discoveries, session events, growing lorebook entries — without risking overwriting earlier content.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"key": map[string]any{
						"type":        "string",
						"description": "Exact key including scope prefix.",
					},
					"text": map[string]any{
						"type":        "string",
						"description": "Text to append. Leading and trailing whitespace is stripped automatically.",
					},
				},
				Required: []string{"key", "text"},
			},
		},
	},
	"world_state": {
		DisplayName: "World state",
		Summary:     "Session state tools: set, modify, remove, clear, and list named values scoped to this conversation.",
		Group:       "World state",
		Order:       120,
		Members:     []string{"state_set", "state_modify", "state_unset", "state_list", "state_clear"},
	},
	"notes": {
		DisplayName: "Notes",
		Summary:     "Note store: save, load, list, delete, and append to named notes across global, user, and conversation scopes.",
		Group:       "Notes",
		Order:       130,
		Members:     []string{"note_save", "note_load", "note_list", "note_delete", "note_append"},
	},
}

// withoutTool returns defs minus the tool named name.
func withoutTool(defs []toolDef, name string) []toolDef {
	out := defs[:0:0]
	for _, d := range defs {
		if d.Function.Name != name {
			out = append(out, d)
		}
	}
	return out
}

// ToolDefsForCharacter returns model-facing definitions for the selected tools.
// Compound tools expand to the member definitions declared in toolRegistry.
func ToolDefsForCharacter(toolIDs []string) []toolDef {
	var out []toolDef
	for _, id := range toolIDs {
		tool, ok := toolRegistry[id]
		if !ok {
			continue
		}
		if len(tool.Members) == 0 {
			out = append(out, tool)
			continue
		}
		for _, memberID := range tool.Members {
			if member, ok := toolRegistry[memberID]; ok {
				out = append(out, member)
			}
		}
	}
	return out
}
