package server

import (
	"fmt"

	"github.com/stevelittlefish/lemon-chat/internal/config"
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
	// AdminOnly tools are withheld from the model, and hidden in the
	// character editor, unless the logged-in user is an admin.
	AdminOnly bool `json:"-"`
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
			Description: "Searches the web and returns a numbered list of results, each with a title, URL and a short snippet. Snippets are fragments, often out of date or missing context; before relying on a result, read the page with fetch_url if it is available.",
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
	"memory_list": {
		Executor: executeMemoryList,
		Type:     "function",
		Function: toolFunction{
			Name:        "memory_list",
			Description: "Lists what you remember about the user from earlier conversations, one fact per line with its id. Call it near the start of a conversation, before answering, so you can use what you already know about the user.",
			Parameters:  toolParam{Type: "object", Properties: map[string]any{}, Required: []string{}},
		},
	},
	"memory_save": {
		Executor: executeMemorySave,
		Type:     "function",
		Function: toolFunction{
			Name:        "memory_save",
			Description: "Saves one fact about the user so you remember it in future conversations. Save without being asked when the user tells you something that will still matter later: their preferences, important people and pets, ongoing projects, their work or situation, things they say about themselves. Don't save small talk, details that only matter to the current task, story or roleplay content, or sensitive personal information unless the user asks you to remember it. Write one short fact that makes sense on its own, e.g. \"Has a cat called Biscuit\". If a memory on the same subject already exists, use memory_update instead of saving a duplicate.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"text": map[string]any{
						"type":        "string",
						"description": "The fact to remember, at most 500 characters.",
					},
				},
				Required: []string{"text"},
			},
		},
	},
	"memory_update": {
		Executor: executeMemoryUpdate,
		Type:     "function",
		Function: toolFunction{
			Name:        "memory_update",
			Description: "Replaces the text of an existing memory. Use it when a fact has changed, or to add a new detail to a memory on the same subject.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"id": map[string]any{
						"type":        "integer",
						"description": "The memory's id, from memory_list.",
					},
					"text": map[string]any{
						"type":        "string",
						"description": "The new text, at most 500 characters.",
					},
				},
				Required: []string{"id", "text"},
			},
		},
	},
	"memory_forget": {
		Executor: executeMemoryForget,
		Type:     "function",
		Function: toolFunction{
			Name:        "memory_forget",
			Description: "Deletes a memory. Use it when the user asks you to forget something, or a memory is no longer true.",
			Parameters: toolParam{
				Type: "object",
				Properties: map[string]any{
					"id": map[string]any{
						"type":        "integer",
						"description": "The memory's id, from memory_list.",
					},
				},
				Required: []string{"id"},
			},
		},
	},
	"memory": {
		DisplayName: "Memory",
		Summary:     "Remembers facts about the user across conversations: list, save, update and forget. Each user has their own memories, shown on the Memories settings page.",
		Group:       "Memory",
		Order:       130,
		Members:     []string{"memory_list", "memory_save", "memory_update", "memory_forget"},
	},
	"history_list": {
		Executor: executeHistoryList,
		Type:     "function",
		Function: historyListFunction(ownHistory),
	},
	"history_search": {
		Executor: executeHistorySearch,
		Type:     "function",
		Function: historySearchFunction(ownHistory),
	},
	"history_read": {
		Executor: executeHistoryRead,
		Type:     "function",
		Function: historyReadFunction(ownHistory),
	},
	"history": {
		DisplayName: "Conversation history",
		Summary:     "Lists, searches and reads the user's own previous conversations.",
		Group:       "Memory",
		Order:       135,
		Members:     []string{"history_list", "history_search", "history_read"},
	},
	"admin_list_users": {
		Executor:  executeAdminListUsers,
		AdminOnly: true,
		Type:      "function",
		Function: toolFunction{
			Name:        "admin_list_users",
			Description: "Admin tool: lists everyone who uses this chat app, with their username, display name, number of conversations and when they were last active. Use a username from here as the user argument of admin_history_list and admin_history_search.",
			Parameters:  toolParam{Type: "object", Properties: map[string]any{}, Required: []string{}},
		},
	},
	"admin_history_list": {
		Executor:  executeAdminHistoryList,
		AdminOnly: true,
		Type:      "function",
		Function:  historyListFunction(adminHistory),
	},
	"admin_history_search": {
		Executor:  executeAdminHistorySearch,
		AdminOnly: true,
		Type:      "function",
		Function:  historySearchFunction(adminHistory),
	},
	"admin_history_read": {
		Executor:  executeAdminHistoryRead,
		AdminOnly: true,
		Type:      "function",
		Function:  historyReadFunction(adminHistory),
	},
	"admin_history": {
		DisplayName: "Conversation history (admin)",
		Summary:     "Admins only: lists the users and lists, searches and reads everyone's conversations. Withheld whenever a non-admin is chatting; every call is logged.",
		Group:       "Admin",
		Order:       145,
		AdminOnly:   true,
		Members:     []string{"admin_list_users", "admin_history_list", "admin_history_search", "admin_history_read"},
	},
	"world_state": {
		DisplayName: "World state",
		Summary:     "Session state tools: set, modify, remove, clear, and list named values scoped to this conversation.",
		Group:       "World state",
		Order:       120,
		Members:     []string{"state_set", "state_modify", "state_unset", "state_list", "state_clear"},
	},
}

// The history tool schemas. The regular history_* tools and the admin
// admin_history_* tools take the same arguments, except that the admin list
// and search also take user, and their descriptions say whose conversations
// they cover.
var (
	historySinceSchema = map[string]any{
		"type":        "string",
		"description": "Only messages sent on or after this date, written YYYY-MM-DD.",
	}
	historyUntilSchema = map[string]any{
		"type":        "string",
		"description": "Only messages sent on or before this date, written YYYY-MM-DD.",
	}
	historyCharacterSchema = map[string]any{
		"type":        "string",
		"description": "Only conversations with a character whose name contains this, e.g. \"tutor\".",
	}
	adminHistoryUserSchema = map[string]any{
		"type":        "string",
		"description": "Only this user's conversations: a username or display name from admin_list_users. Leave it out to cover every user.",
	}
)

func historyListFunction(sc historyScope) toolFunction {
	props := map[string]any{
		"since":     historySinceSchema,
		"until":     historyUntilSchema,
		"character": historyCharacterSchema,
		"limit": map[string]any{
			"type":        "integer",
			"description": "How many conversations to list (default 20, at most 50).",
		},
		"offset": map[string]any{
			"type":        "integer",
			"description": "How many conversations to skip, to see older ones (default 0).",
		},
	}
	desc := "Lists the user's previous conversations, most recent first, with each one's id, title, character, number of messages and dates. Use since and until for a period (e.g. last week) and character for conversations with one character. To find a topic, use history_search instead."
	if sc.admin {
		props["user"] = adminHistoryUserSchema
		desc = "Admin tool: lists previous conversations of every user, or of one user if you give user, most recent first, with whose each one is, its id, title, character, number of messages and dates. Use since and until for a period (e.g. last week) and character for conversations with one character. To find a topic, use admin_history_search instead."
	}
	return toolFunction{
		Name:        sc.tool("list"),
		Description: desc,
		Parameters:  toolParam{Type: "object", Properties: props, Required: []string{}},
	}
}

func historySearchFunction(sc historyScope) toolFunction {
	props := map[string]any{
		"query": map[string]any{
			"type":        "string",
			"description": "Words to look for, separated by spaces. Common words like \"the\" are ignored.",
		},
		"conversation_id": map[string]any{
			"type":        "integer",
			"description": "Step 2 only: search inside this one conversation and list its matching messages.",
		},
		"since":     historySinceSchema,
		"until":     historyUntilSchema,
		"character": historyCharacterSchema,
	}
	whose := "the user's previous conversations"
	if sc.admin {
		props["user"] = adminHistoryUserSchema
		whose = "every user's previous conversations (or one user's, if you give user)"
	}
	desc := fmt.Sprintf("Searches %s for words. Use it in two steps. Step 1: search with a query (optionally since, until, character); you get the best-matching conversations, each with an extract. Step 2: search again with the same query and the conversation_id of the one you want; you get its matching messages with their message numbers. Then read around a match with %s. Put several words in the query, including other words for the same thing, since you don't know the exact wording (e.g. \"guitar chords tuning\"). Any of the words can match, word endings don't matter, and conversations matching more words come first. If nothing useful comes back, try different words.", whose, sc.tool("read"))
	if sc.admin {
		desc = "Admin tool: " + desc
	}
	return toolFunction{
		Name:        sc.tool("search"),
		Description: desc,
		Parameters:  toolParam{Type: "object", Properties: props, Required: []string{"query"}},
	}
}

func historyReadFunction(sc historyScope) toolFunction {
	whose := "a previous conversation"
	if sc.admin {
		whose = "any user's previous conversation"
	}
	desc := fmt.Sprintf("Reads messages from %s: who said what and when, numbered from 1 (tool calls are left out). To read around a match from %s, give start a few messages before it and count, e.g. start=40 count=8 for message 43. Without start and count it reads from the beginning, as much as fits, and says where to continue. Don't say something wasn't discussed unless you have read the whole conversation.", whose, sc.tool("search"))
	if sc.admin {
		desc = "Admin tool: " + desc
	}
	return toolFunction{
		Name:        sc.tool("read"),
		Description: desc,
		Parameters: toolParam{
			Type: "object",
			Properties: map[string]any{
				"conversation_id": map[string]any{
					"type":        "integer",
					"description": fmt.Sprintf("The conversation's id, from %s or %s.", sc.tool("list"), sc.tool("search")),
				},
				"start": map[string]any{
					"type":        "integer",
					"description": "The message number to start at (default 1, the beginning).",
				},
				"count": map[string]any{
					"type":        "integer",
					"description": "How many messages to read (default: as many as fit).",
				},
			},
			Required: []string{"conversation_id"},
		},
	}
}

// withoutAdminTools drops the admin-only tools, for a non-admin's request.
func withoutAdminTools(defs []toolDef) []toolDef {
	out := defs[:0:0]
	for _, d := range defs {
		if !d.AdminOnly {
			out = append(out, d)
		}
	}
	return out
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
