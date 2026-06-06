# Feature: Document Artifacts

## Summary

Add a `create_document` tool that allows the model to produce downloadable files (markdown reports, Python scripts, plain text, etc.) as first-class artifacts. Documents appear as attachment cards in the chat thread; clicking one opens a side panel with rendered content and a download button.

## Goals

- Model can proactively call `create_document` to produce a file the user can download
- Files are stored on disk and linked to the message that created them
- Side panel renders document content (markdown rendered, code syntax-highlighted)
- Works with weak open-source models — the model only has to call the tool correctly; it never has to embed attachment references in its prose
- Tool is opt-in per character via the existing character editor toggle system

## Non-goals

- User-uploaded attachments (separate feature)
- Editing documents after creation
- Image generation (separate feature — see `feature-comfyui-image.md`)

---

## Technical design

### Attachment storage

Files live on disk at `data/attachments/{uuid}/{filename}`. The directory is created per-attachment so filenames don't collide. `data/attachments/` should be added to `.gitignore`.

New `attachment` table (migration v16):

```sql
CREATE TABLE attachment (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    tool_call_id    TEXT NOT NULL,
    conversation_id INTEGER NOT NULL REFERENCES conversation(id),
    title           TEXT NOT NULL,
    filename        TEXT NOT NULL,
    mime_type       TEXT NOT NULL,
    disk_path       TEXT NOT NULL,
    created_at      DATETIME NOT NULL
);
```

`tool_call_id` is the OpenAI-style call ID (e.g. `"call_abc123"`) from the model response. It is known before the tool executes and is already stored on the tool-result message (`message.tool_call_id`), so attachments can be joined to their messages without any post-execution patching.

New endpoint: `GET /api/attachments/:id` — reads `disk_path`, serves with `Content-Disposition: attachment; filename="<filename>"` and correct `Content-Type`. Auth required (same session check as other API endpoints).

### Tool definition

```json
{
  "name": "create_document",
  "description": "Creates a downloadable file. Use for reports, plans, scripts, code, or any content the user will want to save. Choose the filename extension to match the content type (e.g. report.md, script.py, notes.txt).",
  "parameters": {
    "type": "object",
    "properties": {
      "title":    {"type": "string", "description": "Human-readable title shown in the chat"},
      "filename": {"type": "string", "description": "Suggested filename including extension, e.g. report.md or analysis.py"},
      "content":  {"type": "string", "description": "Full text content of the document"}
    },
    "required": ["title", "filename", "content"]
  }
}
```

### Tool executor (`internal/server/tools.go`)

The executor signature is extended to accept the `tool_call_id` alongside the args JSON:

```go
func executorCreateDocument(toolCallID, argsJSON string) (string, error)
```

`executorCreateDocument` does the following:

1. Parse args JSON → `{title, filename, content}`
2. Generate a UUID, create directory `data/attachments/{uuid}/`
3. Write `content` to `data/attachments/{uuid}/{filename}`
4. Infer MIME type from extension (`.md` → `text/markdown`, `.py` → `text/x-python`, `.txt` → `text/plain`, everything else → `application/octet-stream`)
5. Insert row into `attachment` table with `tool_call_id` — no patching needed

The tool executor returns a result JSON string:

```json
{"attachment_id": 42, "title": "My Report", "filename": "report.md", "mime_type": "text/markdown"}
```

### SSE event

The existing `tool_call` SSE event is extended. When a tool result contains attachment JSON, the server emits:

```json
{"tool_call": {"name": "create_document", "attachment": {"id": 42, "title": "My Report", "filename": "report.md", "mime_type": "text/markdown"}}}
```

The `attachment` field is only present when the tool produced one. The frontend uses this to render an attachment card immediately during streaming, without waiting for a page reload.

### `AllTools` metadata entry

```go
{"create_document", "Create document", "Saves a file (report, script, notes, etc.) the user can download."}
```

---

## Files touched

| File | Change |
|---|---|
| `internal/store/store.go` | Migration v16: create `attachment` table |
| `internal/store/attachments.go` | New file — `CreateAttachment`, `GetAttachment`, `ListAttachmentsByConversation` |
| `internal/server/tools.go` | Add `create_document` def, executor, and `AllTools` entry |
| `internal/server/messages.go` | Emit `tool_call` SSE event with attachment metadata after tool executes |
| `internal/server/server.go` | Register `GET /api/attachments/:id` and `GET /api/conversations/:id/attachments` routes |
| `internal/server/attachments.go` | New file — `handleGetAttachment`, `handleListConversationAttachments` |
| `static/js/thread.js` | Render attachment cards; open/close artifact panel on click; load attachment cards from history |
| `static/js/app.js` | Wire up artifact panel open/close events |
| `static/index.html` | Add artifact panel DOM structure |
| `static/css/app.css` | Artifact panel layout styles |

---

## Frontend: artifact panel

The main chat area gains a third column (right panel) that is hidden by default and slides in when an artifact is opened.

```
┌─ sidebar ─┬─── thread ────────────────┬─── artifact panel ──────┐
│           │                           │  report.md          [x] │
│           │  ┌─ attachment card ────┐ │  ───────────────────── │
│           │  │ [doc] My Report     │ │  # My Report            │
│           │  │ report.md  [Open]   │ │                         │
│           │  └─────────────────────┘ │  Lorem ipsum...         │
│           │                           │                         │
│           │                           │  [Download]             │
└───────────┴───────────────────────────┴─────────────────────────┘
```

- Attachment card in thread: file type icon (Lucide `file-text` for markdown, `file-code` for code, `file` for generic), title, filename as muted subtitle, "Open" button
- Panel header: title, filename, close button (`x`), download button (links to `GET /api/attachments/:id` with `Content-Disposition: attachment`)
- Panel body: for `.md` files, render with `marked`; for code files (`.py`, `.js`, `.go`, etc.), wrap in `<pre><code>`; for `.txt`, wrap in `<pre>`
- Panel is per-conversation state (closes when navigating to a different conversation)

### Loading attachment cards from history

Add `GET /api/conversations/:id/attachments` which returns all attachments for the conversation joined to their tool-result messages via `tool_call_id`. Fetch this alongside messages on load. For each attachment, `thread.js` renders an attachment card after the assistant message that precedes the matching tool-result message.

---

## Configuration

No new config required. The `data/` directory is already used for the SQLite database.

---

## Acceptance criteria

- [ ] Model calls `create_document`, file appears on disk and in `attachment` table
- [ ] Attachment card renders in the chat thread during streaming
- [ ] Clicking "Open" opens the artifact side panel with rendered content
- [ ] Download button downloads the file with correct filename and MIME type
- [ ] Attachment cards reload correctly after page refresh
- [ ] Tool appears in character editor toggles
- [ ] Conversations without the tool enabled follow the existing code path unchanged
