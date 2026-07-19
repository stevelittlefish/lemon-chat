# Async image generation

## Overview

Currently `generate_image_sdxl` and `generate_image_flux` block the entire SSE stream for up to 120 seconds while ComfyUI generates the image. The LLM cannot continue, the frontend is frozen, and if the generation fails the model receives an error it must narrate to the user.

This spec describes making image generation asynchronous:

- The tool call returns immediately with a "pending" attachment ID
- The LLM continues writing its response while the image generates in the background
- When generation completes the frontend receives a WebSocket push and swaps the placeholder for the real image
- If generation fails the frontend shows an error state; the LLM is unaffected

---

## User-visible behaviour

**Before:** Text streams, then freezes for 10–30 seconds, then the image appears and text resumes.

**After:** Text streams immediately. A "Generating image…" placeholder appears inline. The image fades in when ready. If it fails, the placeholder shows an error message.

---

## Current architecture

```
LLM calls tool
  → ExecuteTool() in messages.go
    → makeImageExecutor() in tools.go
      → POST /prompt to ComfyUI          (up to 15s)
      → poll /history/{id} every 1s      (up to 120s)  ← SSE stream blocked here
      → GET /view to download image
      → CreateAttachment() in store
      → return AttachmentResult JSON
  → messages.go emits SSE `attachment` event
  → LLM continues
```

---

## Proposed architecture

```
LLM calls tool
  → ExecuteTool() in messages.go
    → makeImageExecutor() in tools.go
      → POST /prompt to ComfyUI          (up to 15s, still sync — fast)
      → CreatePendingAttachment() in store
      → launch goroutine (poll + download + finalise)
      → return AttachmentResult{..., Status: "pending"}  ← returns immediately
  → messages.go emits SSE `attachment` event with status="pending"
  → frontend renders placeholder card
  → LLM continues writing

[background goroutine]
  → poll /history/{id} every 1s          (up to 120s)
  → GET /view to download image
  → write file to disk
  → FinaliseAttachment() in store
  → hub.BroadcastAttachmentReady()       → WS push to all clients
  → [on error] hub.BroadcastAttachmentError()

[frontend WS handler]
  → receives attachment_ready or attachment_error
  → finds placeholder card by attachment ID
  → swaps in image or error state
```

The initial POST to ComfyUI stays synchronous. It's fast (< 1s) and its failure should be reported to the LLM immediately so it can tell the user ComfyUI is unreachable. Only the polling and download move to the background.

---

## Data model changes

### Migration v21 → v22

Add `status` and `error` columns to the `attachment` table.

```go
if version < 22 {
    log.Println("store: migrating v21 → v22 (add status and error to attachment)")
    if _, err := s.db.Exec(`ALTER TABLE attachment ADD COLUMN status TEXT NOT NULL DEFAULT 'ready'`); err != nil {
        return err
    }
    if _, err := s.db.Exec(`ALTER TABLE attachment ADD COLUMN error TEXT`); err != nil {
        return err
    }
    if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (22, ?)`, now()); err != nil {
        return err
    }
    version = 22
    log.Println("store: migration v21 → v22 complete")
}
```

Valid `status` values: `"pending"`, `"ready"`, `"error"`.

Existing rows default to `"ready"` (correct — they were created synchronously and already have a disk path).

---

## Backend changes

### 1. `internal/store/attachments.go`

**Update `Attachment` struct:**

```go
type Attachment struct {
    ID             int64  `json:"id"`
    ToolCallID     string `json:"tool_call_id"`
    ConversationID int64  `json:"conversation_id"`
    Title          string `json:"title"`
    Filename       string `json:"filename"`
    MimeType       string `json:"mime_type"`
    DiskPath       string `json:"disk_path"`
    Status         string `json:"status"`  // "pending" | "ready" | "error"
    Error          string `json:"error,omitempty"`
    CreatedAt      string `json:"created_at"`
}
```

Update `CreateAttachment`, `GetAttachment`, and `ListAttachmentsByConversation` to include the two new columns in their SELECT and INSERT statements. `CreateAttachment` should always write `status = 'ready'` (it is used by the sync `create_document` tool which is unaffected by this change).

**Add `CreatePendingAttachment`:**

Creates an attachment row with `status = 'pending'` and no disk path. Returns the attachment with its new ID so the tool can return it immediately.

```go
func (s *Store) CreatePendingAttachment(toolCallID string, convID int64, title, filename, mimeType string) (*Attachment, error) {
    ts := now()
    res, err := s.db.Exec(
        `INSERT INTO attachment (tool_call_id, conversation_id, title, filename, mime_type, disk_path, status, created_at)
         VALUES (?, ?, ?, ?, ?, '', 'pending', ?)`,
        toolCallID, convID, title, filename, mimeType, ts,
    )
    if err != nil {
        return nil, err
    }
    id, err := res.LastInsertId()
    if err != nil {
        return nil, err
    }
    return &Attachment{
        ID: id, ToolCallID: toolCallID, ConversationID: convID,
        Title: title, Filename: filename, MimeType: mimeType,
        Status: "pending", CreatedAt: ts,
    }, nil
}
```

**Add `FinaliseAttachment`:**

Called by the background goroutine when the image is ready.

```go
func (s *Store) FinaliseAttachment(id int64, diskPath string) error {
    _, err := s.db.Exec(
        `UPDATE attachment SET status = 'ready', disk_path = ? WHERE id = ?`,
        diskPath, id,
    )
    return err
}
```

**Add `FailAttachment`:**

Called by the background goroutine on error.

```go
func (s *Store) FailAttachment(id int64, errMsg string) error {
    _, err := s.db.Exec(
        `UPDATE attachment SET status = 'error', error = ? WHERE id = ?`,
        errMsg, id,
    )
    return err
}
```

---

### 2. `internal/server/ws.go`

Add two new broadcast methods.

**`BroadcastAttachmentReady`** — sent when the background goroutine completes successfully:

```go
func (h *Hub) BroadcastAttachmentReady(att *store.Attachment) {
    data, _ := json.Marshal(map[string]any{
        "type":            "attachment_ready",
        "id":              att.ID,
        "conversation_id": att.ConversationID,
        "title":           att.Title,
        "filename":        att.Filename,
        "mime_type":       att.MimeType,
    })
    h.broadcast(data)
}
```

**`BroadcastAttachmentError`** — sent when the background goroutine fails:

```go
func (h *Hub) BroadcastAttachmentError(convID, attID int64, errMsg string) {
    data, _ := json.Marshal(map[string]any{
        "type":            "attachment_error",
        "id":              attID,
        "conversation_id": convID,
        "error":           errMsg,
    })
    h.broadcast(data)
}
```

---

### 3. `internal/server/tools.go`

**Update `ToolContext`:**

Add the Hub so the background goroutine can broadcast when done.

```go
type ToolContext struct {
    // ... existing fields ...
    Hub *Hub  // for async tool completion broadcasts
}
```

**Update `AttachmentResult`:**

Add a `Status` field so `messages.go` can distinguish pending from ready results.

```go
type AttachmentResult struct {
    AttachmentID int64  `json:"attachment_id"`
    Title        string `json:"title"`
    Filename     string `json:"filename"`
    MimeType     string `json:"mime_type"`
    Status       string `json:"status,omitempty"` // "pending" when async; omitted/empty = ready
}
```

**Update `makeImageExecutor`:**

Split the current function into two phases:

*Phase 1 (synchronous, stays in the tool call):*
- Parse args
- Read workflow file
- POST to ComfyUI `/prompt` — if this fails, return an error immediately (LLM hears about it)
- `CreatePendingAttachment` in the store
- Launch the background goroutine (Phase 2)
- Return `AttachmentResult{..., Status: "pending"}` immediately

*Phase 2 (background goroutine):*
- Poll `/history/{prompt_id}` every second, up to 120s
- On timeout: `FailAttachment`, `hub.BroadcastAttachmentError`
- On success: download image from `/view`, write to disk
- On download error: `FailAttachment`, `hub.BroadcastAttachmentError`
- On success: `FinaliseAttachment`, `hub.BroadcastAttachmentReady`

The goroutine captures `tctx.Store`, `tctx.Hub`, `tctx.DataDir`, `tctx.ConversationID`, and the pending attachment ID by value before launching.

```go
func makeImageExecutor(comfyURL, workflowFile string, defaultSteps int, defaultCFG float64) func(string, ToolContext) (string, error) {
    return func(argsJSON string, tctx ToolContext) (string, error) {
        // ... parse args, build workflow, same as now ...

        // Phase 1: submit to ComfyUI
        promptID, err := submitToComfyUI(comfyURL, promptPayload)
        if err != nil {
            return "", err  // still synchronous — LLM hears about unreachable ComfyUI
        }

        // Create pending attachment immediately
        att, err := tctx.Store.CreatePendingAttachment(tctx.ToolCallID, tctx.ConversationID, "Generated image", "image.png", "image/png")
        if err != nil {
            return "", fmt.Errorf("server error: could not create pending attachment: %w", err)
        }

        log.Printf("Generating image async attachment_id=%d prompt_id=%q conversation_id=%d", att.ID, promptID, tctx.ConversationID)

        // Phase 2: background goroutine
        go func(attID, convID int64) {
            img, err := pollComfyUI(comfyURL, promptID)
            if err != nil {
                log.Printf("Image generation failed attachment_id=%d: %v", attID, err)
                tctx.Store.FailAttachment(attID, err.Error())
                tctx.Hub.BroadcastAttachmentError(convID, attID, err.Error())
                return
            }

            diskPath, err := downloadAndSave(tctx.DataDir, img, comfyURL)
            if err != nil {
                log.Printf("Image download failed attachment_id=%d: %v", attID, err)
                tctx.Store.FailAttachment(attID, err.Error())
                tctx.Hub.BroadcastAttachmentError(convID, attID, err.Error())
                return
            }

            if err := tctx.Store.FinaliseAttachment(attID, diskPath); err != nil {
                log.Printf("Image finalise failed attachment_id=%d: %v", attID, err)
                tctx.Hub.BroadcastAttachmentError(convID, attID, err.Error())
                return
            }

            finalAtt, _ := tctx.Store.GetAttachment(attID)
            log.Printf("Image generation complete attachment_id=%d", attID)
            tctx.Hub.BroadcastAttachmentReady(finalAtt)
        }(att.ID, tctx.ConversationID)

        result := AttachmentResult{
            AttachmentID: att.ID,
            Title:        "Generated image",
            Filename:     "image.png",
            MimeType:     "image/png",
            Status:       "pending",
        }
        out, _ := json.Marshal(result)
        return string(out), nil
    }
}
```

Extract the polling and download logic into private helper functions (`submitToComfyUI`, `pollComfyUI`, `downloadAndSave`) to keep the executor readable. These already exist implicitly in the current function body — just give them names.

---

### 4. `internal/server/messages.go`

**Thread the hub into `ToolContext`:**

```go
tctx := ToolContext{
    // ... existing fields ...
    Hub: s.hub,
}
```

**Handle pending attachments in the SSE emit block:**

The existing check (`json.Unmarshal` into `AttachmentResult`) already fires for both pending and ready results because both shapes have `attachment_id`. No structural change needed here.

However, the `attachment` SSE event should include the `status` field so the frontend knows whether to render a placeholder or a real card:

```go
if jsonErr := json.Unmarshal([]byte(result), &attResult); jsonErr == nil && attResult.AttachmentID != 0 {
    attEvt, _ := json.Marshal(map[string]any{"attachment": map[string]any{
        "id":           attResult.AttachmentID,
        "tool_call_id": tc.id,
        "title":        attResult.Title,
        "filename":     attResult.Filename,
        "mime_type":    attResult.MimeType,
        "status":       attResult.Status, // "pending" or empty/omitted = ready
    }})
    fmt.Fprintf(w, "data: %s\n\n", attEvt)
    flusher.Flush()
}
```

---

### 5. Startup cleanup

When the server restarts, any goroutines that were running are gone. Attachments stuck in `status = 'pending'` will never resolve.

Add a call in `store.go` after migrations complete:

```go
func (s *Store) ClearPendingAttachments() (int, error) {
    res, err := s.db.Exec(
        `UPDATE attachment SET status = 'error', error = 'Server restarted while image was generating'
         WHERE status = 'pending'`,
    )
    if err != nil {
        return 0, err
    }
    n, _ := res.RowsAffected()
    return int(n), nil
}
```

Call this from `server.go` or wherever `store.Open` is called at startup:

```go
if n, err := store.ClearPendingAttachments(); err != nil {
    log.Printf("Warning: could not clear pending attachments: %v", err)
} else if n > 0 {
    log.Printf("Cleared %d pending attachment(s) left over from previous run", n)
}
```

---

## Frontend changes

### Attachment card states

The existing attachment card (rendered on the `attachment` SSE event) needs three visual states:

| State | Trigger | Appearance |
|---|---|---|
| `pending` | SSE `attachment` event with `status: "pending"` | Placeholder card with a spinner and "Generating image…" |
| `ready` | WS `attachment_ready` event | Normal image card (same as current sync behaviour) |
| `error` | WS `attachment_error` event | Error card with message |

The card should carry a `data-attachment-id` attribute so the WS handler can find it.

### SSE handler (thread.js or wherever attachments are currently rendered)

When a `attachment` SSE event arrives:

```js
if (data.attachment) {
    const att = data.attachment
    const card = renderAttachmentCard(att)  // returns a DOM element
    card.dataset.attachmentId = att.id
    messageEl.appendChild(card)
}
```

`renderAttachmentCard` checks `att.status`:
- `"pending"` → render spinner placeholder (no download link, no image)
- anything else / absent → render the existing ready card

### WS handler (ws.js)

Add handling for the two new message types:

```js
case 'attachment_ready': {
    const card = document.querySelector(`[data-attachment-id="${msg.id}"]`)
    if (card) {
        card.replaceWith(renderReadyAttachmentCard(msg))
    }
    break
}
case 'attachment_error': {
    const card = document.querySelector(`[data-attachment-id="${msg.id}"]`)
    if (card) {
        card.replaceWith(renderErrorAttachmentCard(msg))
    }
    break
}
```

`renderReadyAttachmentCard` and `renderErrorAttachmentCard` produce the same DOM structure as the existing card but in ready and error states respectively.

### Attachment card when loading a conversation

When a conversation is loaded (e.g. after a page refresh), attachment cards are reconstructed from the stored messages. The `GetAttachment` lookup already returns the `status` column after the migration, so the card renders in its final state (ready or error) correctly — no placeholder needed on load.

---

## Error handling summary

| Failure point | Behaviour |
|---|---|
| ComfyUI unreachable (Phase 1, sync) | Tool returns error → LLM narrates it → no pending attachment created |
| ComfyUI returns non-200 (Phase 1, sync) | Same as above |
| ComfyUI times out during polling (Phase 2, async) | `FailAttachment` + `BroadcastAttachmentError` → frontend shows error card |
| Image download fails (Phase 2, async) | Same as above |
| File write fails (Phase 2, async) | Same as above |
| Server restart with pending attachment | `ClearPendingAttachments` at startup → `status = 'error'`; card shows error on next load |

---

## What is not changing

- `create_document` remains synchronous — it writes to disk locally and is fast.
- `generate_image_flux` uses the same `makeImageExecutor` function and gets async behaviour for free once the function is updated.
- The `/api/attachments/:id` serve handler is unchanged — it checks `disk_path` which is empty for pending attachments, but pending attachments are never fetched directly (the frontend doesn't show a link until the card is in `ready` state).

---

## Open questions

1. **Should the pending card show a cancel button?** ComfyUI does support DELETE `/queue` to cancel a job, but that is out of scope for this feature.
2. **Conversation_id filtering in WS broadcasts.** Currently all clients receive all `attachment_ready` events. Only the client viewing that conversation will find a matching card and act on it; others ignore it harmlessly. If broadcast volume becomes a concern, add per-conversation filtering to the hub.
3. **Timeout value.** The current 120-second timeout is inherited. Consider making it configurable in `lemon.toml` under `[comfyui]`.
