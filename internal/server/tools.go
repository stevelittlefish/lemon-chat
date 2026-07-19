package server

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

// ToolContext carries per-request context to tool executors.
// There is deliberately no context.Context field: tool execution must complete
// even if the client disconnects mid-stream, so that the response is persisted
// and visible when the user returns to the conversation.
type ToolContext struct {
	ModelName       string
	ModelServer     *config.ModelServer
	ResponseTimeout time.Duration
	SearXNGURL      string
	Timezone        string
	// for tools that create attachments
	ToolCallID     string
	UserID         int64
	ConversationID int64
	Store          *store.Store
	DataDir        string
	Hub            *Hub
}

// AttachmentResult is the JSON structure returned by tools that create attachments.
// The messages layer detects this shape to emit the attachment SSE event.
type AttachmentResult struct {
	AttachmentID int64  `json:"attachment_id"`
	Title        string `json:"title"`
	Filename     string `json:"filename"`
	MimeType     string `json:"mime_type"`
	Background   bool   `json:"background,omitempty"`
	Status       string `json:"status,omitempty"` // "pending" when async; empty = ready
}

func mimeTypeForFilename(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".md":
		return "text/markdown"
	case ".py":
		return "text/x-python"
	case ".js":
		return "text/javascript"
	case ".ts":
		return "text/typescript"
	case ".go":
		return "text/x-go"
	case ".sh":
		return "text/x-sh"
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".json":
		return "application/json"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = cryptorand.Read(b)
	return hex.EncodeToString(b)
}

func executeCreateDocument(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		Title    string `json:"title"`
		Filename string `json:"filename"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.Title == "" || args.Filename == "" || args.Content == "" {
		return "", fmt.Errorf("title, filename, and content are all required and must be non-empty")
	}
	// Sanitise filename: strip any path components.
	args.Filename = filepath.Base(args.Filename)

	relDir := filepath.Join("attachments", randomID())
	absDir := filepath.Join(tctx.DataDir, relDir)
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return "", fmt.Errorf("server error: could not create storage directory (%w) — tell the user the document could not be saved due to a server storage problem", err)
	}
	if err := os.WriteFile(filepath.Join(absDir, args.Filename), []byte(args.Content), 0644); err != nil {
		return "", fmt.Errorf("server error: could not write document to disk (%w) — tell the user the document could not be saved due to a server storage problem", err)
	}
	diskPath := filepath.Join(relDir, args.Filename)

	mimeType := mimeTypeForFilename(args.Filename)
	log.Printf("Creating document attachment title=%q filename=%q conversation_id=%d", args.Title, args.Filename, tctx.ConversationID)

	att, err := tctx.Store.CreateAttachment(tctx.ToolCallID, tctx.ConversationID, args.Title, args.Filename, mimeType, diskPath)
	if err != nil {
		return "", fmt.Errorf("server error: could not record document in database (%w) — tell the user the document could not be saved due to a server error", err)
	}

	result := AttachmentResult{
		AttachmentID: att.ID,
		Title:        att.Title,
		Filename:     att.Filename,
		MimeType:     att.MimeType,
	}
	out, _ := json.Marshal(result)
	return string(out), nil
}

func executeGetTime(_ string, tctx ToolContext) (string, error) {
	now := time.Now()
	if tctx.Timezone != "" {
		loc, err := time.LoadLocation(tctx.Timezone)
		if err == nil {
			now = now.In(loc)
		} else {
			return "", fmt.Errorf("invalid timezone configuration: %w", err)
		}
	}
	return now.Weekday().String() + ", " + now.Format(time.RFC3339), nil
}

func executeRollDice(argsJSON string, _ ToolContext) (string, error) {
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
			return "", fmt.Errorf("invalid dice count %q: must be a whole number between 1 and 100", halves[0])
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
		return "", fmt.Errorf("invalid die sides %q: must be a whole number between 2 and 10000", sidesStr)
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
}

func executePickRandom(argsJSON string, _ ToolContext) (string, error) {
	var args struct {
		Options []string `json:"options"`
		Label   string   `json:"label"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if len(args.Options) < 2 {
		return "", fmt.Errorf("options must contain at least 2 items")
	}
	return args.Options[rand.Intn(len(args.Options))], nil
}

func executeRandomChance(argsJSON string, _ ToolContext) (string, error) {
	var args struct {
		Action    string `json:"action"`
		Dice      string `json:"dice"`
		Threshold int    `json:"threshold"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.Action == "" {
		return "", fmt.Errorf("action is required")
	}
	if args.Dice == "" {
		return "", fmt.Errorf("dice is required")
	}
	if args.Threshold < 1 {
		return "", fmt.Errorf("threshold must be at least 1")
	}

	diceStr := strings.ToLower(strings.TrimSpace(args.Dice))
	// Accept "d20" or "1d20" — strip leading "1d" or require plain "dN"
	if strings.HasPrefix(diceStr, "d") {
		diceStr = diceStr[1:]
	} else if strings.HasPrefix(diceStr, "1d") {
		diceStr = diceStr[2:]
	} else {
		return "", fmt.Errorf("invalid dice %q: use a single die like \"d20\" or \"d100\"", args.Dice)
	}
	sides, err := strconv.Atoi(diceStr)
	if err != nil || sides < 2 || sides > 10000 {
		return "", fmt.Errorf("invalid die %q: sides must be a whole number between 2 and 10000", args.Dice)
	}
	if args.Threshold > sides {
		return "", fmt.Errorf("threshold %d exceeds die sides %d — success would be impossible", args.Threshold, sides)
	}

	roll := rand.Intn(sides) + 1
	success := roll >= args.Threshold
	outcome := "FAILURE"
	if success {
		outcome = "SUCCESS"
	}
	return fmt.Sprintf("Check: %s | Rolled %s: %d | Threshold: %d | Result: %s",
		args.Action, args.Dice, roll, args.Threshold, outcome), nil
}

// ExecuteTool runs a tool by name and returns its result string.
func ExecuteTool(name, argsJSON string, tctx ToolContext) (string, error) {
	fn, ok := executors[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return fn(argsJSON, tctx)
}
