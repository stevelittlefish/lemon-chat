# Feature: ComfyUI Image Generation

## Summary

Add a `generate_image` tool that sends a prompt to a locally-running ComfyUI instance, waits for the SDXL image to be generated, saves the result to disk, and renders it **inline** in the chat thread. Designed for storytelling characters that illustrate scenes as the story unfolds.

## Goals

- Model calls `generate_image(prompt)` and an image appears inline in the message thread
- Uses a single hardcoded SDXL workflow — no per-conversation configuration
- ComfyUI server URL and workflow path are configured in `lemon.toml`
- Images stored on disk, linked to their message via the `attachment` table (same infrastructure as document artifacts)
- Tool is opt-in per character

## Non-goals

- Multiple workflows or per-character workflow selection (future)
- ControlNet, image-to-image, or other advanced modes
- Image editing after generation
- Progress streaming (just a loading indicator while waiting)

## Prerequisites

This feature builds on the attachment infrastructure from `feature-document-artifacts.md`, which must be implemented first. By the time this feature is built, the following already exist:

- `attachment` table in SQLite with `tool_call_id`, `conversation_id`, `filename`, `mime_type`, `disk_path`
- Files stored on disk at `data/attachments/{uuid}/{filename}`
- `GET /api/attachments/:id` endpoint serving files
- `GET /api/conversations/:id/attachments` endpoint for loading attachments from history
- `internal/store/attachments.go` with `CreateAttachment`, `GetAttachment`, `ListAttachmentsByConversation`
- `internal/server/attachments.go` with the route handlers
- `tool_call` SSE event shape with optional `attachment` field
- Executor signature: `func executor(toolCallID, argsJSON string) (string, error)`

---

## Technical design

### Config (`lemon.toml`)

```toml
[comfyui]
url      = "http://localhost:8188"   # ComfyUI server base URL
workflow = "data/comfyui/sdxl.json"  # path to workflow JSON file
```

New config struct in `internal/config/config.go`:

```go
type ComfyUIConfig struct {
    URL      string `toml:"url"`
    Workflow string `toml:"workflow"`
}
```

Added to the top-level `Config` struct:

```go
ComfyUI ComfyUIConfig `toml:"comfyui"`
```

If `ComfyUI.URL` is empty, the `generate_image` tool is unavailable at runtime (executor returns an error). The tool still appears in character editor toggles so it can be pre-configured.

### Workflow JSON and prompt injection

ComfyUI workflows are arbitrary JSON graphs. To avoid requiring users to annotate their workflow, we use a convention: the workflow JSON file has two special string values that the executor replaces at runtime:

- `"__PROMPT__"` — replaced with the positive prompt
- `"__NEGATIVE_PROMPT__"` — replaced with the negative prompt (or an empty string if none provided)

The executor does a simple JSON marshal/unmarshal with string replacement before submitting. Users creating a workflow for use with lemon-chat should set those placeholder strings on their KSampler's connected text nodes.

### ComfyUI API sequence

1. **POST `/prompt`** — submit the modified workflow:
   ```json
   {"prompt": <workflow object>, "client_id": "<random uuid>"}
   ```
   Response: `{"prompt_id": "abc123", "number": 1}`

2. **Poll GET `/history/{prompt_id}`** — every 1 second until the job appears. When complete, the response contains output image filenames:
   ```json
   {"abc123": {"outputs": {"9": {"images": [{"filename": "ComfyUI_00001_.png", "subfolder": "", "type": "output"}]}}}}
   ```

3. **GET `/view?filename=...&subfolder=...&type=output`** — download the image bytes.

4. Save to `data/attachments/{uuid}/image.png`, insert `attachment` record, return attachment metadata.

Timeout: 120 seconds total polling time. If the job doesn't complete, return an error string as the tool result.

### Tool definition

```json
{
  "name": "generate_image",
  "description": "Generates an image using Stable Diffusion. Use to illustrate scenes, characters, or objects described in the story. Be descriptive — include art style, lighting, and mood.",
  "parameters": {
    "type": "object",
    "properties": {
      "prompt": {
        "type": "string",
        "description": "Detailed visual description of the image. Include subject, setting, art style, lighting, mood."
      },
      "negative_prompt": {
        "type": "string",
        "description": "Things to exclude from the image (optional)."
      }
    },
    "required": ["prompt"]
  }
}
```

### Tool executor (`internal/server/tools.go`)

`executorGenerateImage` needs access to the `ComfyUIConfig`. Pass the config to the executor at registration time (closure or struct method — use a closure on the `Server` struct, consistent with how other server dependencies are handled).

Steps:
1. Parse args → `{prompt, negative_prompt}`
2. Read workflow JSON from `cfg.ComfyUI.Workflow`
3. Marshal workflow to string, replace `__PROMPT__` and `__NEGATIVE_PROMPT__`
4. Unmarshal back to `map[string]any`
5. POST to `cfg.ComfyUI.URL + "/prompt"` with a random `client_id`
6. Poll `/history/{prompt_id}` until complete or timeout
7. Download image via `/view?...`
8. Save to `data/attachments/{uuid}/image.png`, call `store.CreateAttachment` with `toolCallID` and `mime_type = "image/png"`, return JSON result

Return value (same shape as `create_document`):
```json
{"attachment_id": 43, "title": "Generated image", "filename": "image.png", "mime_type": "image/png"}
```

### SSE event

Same extension as documents — the `tool_call` SSE event includes an `attachment` field when an image is produced:

```json
{"tool_call": {"name": "generate_image", "attachment": {"id": 43, "title": "Generated image", "filename": "image.png", "mime_type": "image/png"}}}
```

### Loading indicator

Image generation can take 10–60 seconds. When the frontend receives a `tool_call` event for `generate_image` with no attachment yet (i.e., the tool is still running), it shows a loading placeholder in the thread — a muted box with a spinner or animated pulse. When the final tool_call event arrives with the attachment ID, replace the placeholder with the rendered image.

The server emits two events:
1. `{"tool_call": {"name": "generate_image"}}` — when the tool starts (before ComfyUI finishes)
2. `{"tool_call": {"name": "generate_image", "attachment": {...}}}` — when the image is ready

This requires the server to stream the "started" event immediately when the tool is called, then stream the "completed" event after the executor returns.

---

## Frontend: inline image rendering

Images do **not** use the artifact side panel. They render directly in the thread.

When `thread.js` encounters a `tool_call` SSE event or loads an image attachment from history:

- **Loading state:** a `div.image-placeholder` with a subtle pulse animation and a small label ("Generating image…")
- **Loaded state:** an `<img>` element pointing to `/api/attachments/:id` (served inline, not as download), with a small download button overlaid in the corner

```
┌──────────────────────────────────┐
│                                  │
│   [generated image]       [↓]   │
│                                  │
└──────────────────────────────────┘
```

The `↓` download button links to `GET /api/attachments/:id?download=1`. The endpoint checks a `download` query param: if present, adds `Content-Disposition: attachment`; otherwise serves inline (`Content-Disposition: inline`).

Images are capped at `max-width: 100%` and `max-height: 512px` in CSS to avoid overwhelming the thread.

---

## Files touched

| File | Change |
|---|---|
| `internal/config/config.go` | Add `ComfyUIConfig` struct and field on `Config` |
| `internal/server/tools.go` | Add `generate_image` def, executor (ComfyUI client logic), `AllTools` entry |
| `internal/server/messages.go` | Emit "tool started" SSE event before executor runs (for loading state) |
| `static/js/thread.js` | Loading placeholder; inline image rendering; download button overlay |
| `static/css/app.css` | Image container and loading pulse styles |
| `lemon.toml.example` | Document `[comfyui]` config block |

No DB migration needed — the `attachment` table already exists.

---

## Acceptance criteria

- [ ] `[comfyui]` section in `lemon.toml` with `url` and `workflow` configures the tool
- [ ] Model calls `generate_image`, a loading placeholder appears immediately in the thread
- [ ] When generation completes, the placeholder is replaced with the rendered image
- [ ] Image persists after page reload (loads from `attachment` table)
- [ ] Download button downloads the image file
- [ ] If ComfyUI is unreachable or times out, the model receives an error result and responds gracefully
- [ ] Tool appears in character editor toggles
- [ ] Characters without the tool enabled are unaffected
