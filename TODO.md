# lemon-chat — TODO

Status markers: `[ ]` not started · `[~]` in progress · `[x]` done

--

## Manually Added by User

- [x] When multiple messages are present from the same user, only show the avatar for the first one.

- [x] Add a default avatar for characters, models and users if they haven't uploaded one.  Use the CPU icon for models and the Drama icon for characters.  We need a silhouette of a human head for the user.

- [x] Research feature (port of docs/deep_research_spec.html to Go) — `/research` page linked from the main menu, iterative search/extract/synthesise engine in `internal/research/`, crash-resumable jobs via state checkpoints in the `research_job` table.

- [x] **Rename `sdxl_file`/`flux_file` config keys to `sdxl_workflow`/`flux_workflow`** (`internal/config/config.go:28`)

## Research

- [x] Improve research citations: use stable source IDs in findings/prompts and feed raw findings into final report generation so links survive summarisation.

- [x] Normalize combined research citations like `[S13, S18]` into separate clickable source references.

- [x] Long prompts mess up the layout.  Separate into title and prompt (both optional - one must be specified)

- [x] The back button in the top left when viewing an individual piece of deep research should take you back to the list, not the menu

- [x] Brainstorming mode - alternate algorithm where we want the LLM to invent / design stuff and use web search when necessary, rather than basing the entire process around web search

- [x] In-depth report toggle: build the final report section by section from the raw findings (outline → self-critique refine → per-section write → glue) instead of one summarising pass, to stop reports losing detail. Opt-in checkbox; applies to both modes. `deepReport` pipeline in `internal/research/researcher.go`.



## Code review (2026-06-13)

Findings from `code_review_2026-06-13.md`, in suggested priority order.

### Security

- [x] **Stored XSS — sanitise rendered markdown** (`static/js/markdown.js:37`)
  `marked.parse(text)` is assigned to `innerHTML` with no sanitiser; assistant output (which includes fetched web-page content and shared character prompts) executes as live HTML. Vendor DOMPurify (no build step — drop in `static/js/vendor/`) and sanitise the output of `render()` before assigning. Highest priority.

- [x] **Notes IDOR — add ownership checks** (`internal/server/notes.go:24,82,112`)
  `handleGetNote`, `handleDeleteNote`, and `handleSetNoteReadOnly` operate on a note id with no scope/ownership check, defeating the `g.`/`u.`/`c.` visibility model. After `GetNoteByID`, verify the note is visible to `currentUser(r)` (global, owned by the user, or in one of the user's conversations) before returning/mutating.

- [x] **Attachment IDOR — add ownership check** (`internal/server/attachments.go:11`)
  `handleGetAttachment` serves any attachment by id. Join through `conversation` and require `conversation.user_id == currentUser.ID` (or admin) in the query.



- [x] **Escape filename in Content-Disposition** (`internal/server/attachments.go:29`)
  `att.Filename` is interpolated into the header inside quotes without escaping; `filepath.Base` doesn't strip `"`/newlines. Use `mime.FormatMediaType`.

- [x] **Add WebSocket Origin check** (`internal/server/ws.go:80`)
  `handleWS` never validates `Origin` (CSWSH). Mitigated by `SameSite=Strict` cookie, but add the check as defence in depth.

### Bugs

- [x] **`title_prompt` is dead — wire it up or remove it** (`internal/tasks/titles.go:299`)
  The column is stored, editable in the character editor, and exposed via `GetConversationTitlePrompt`, but the title worker uses a hardcoded prompt and never reads it. Either use the per-character title prompt in `generateTitle`, or remove the field, `GetConversationTitlePrompt`, and the UI control.

- [x] **Research SSE subscribe-after-finish race** (`internal/server/research.go:499`)
  If `run.finish()` runs between `get(id)` and `subscribe()`, the new channel is never closed and the client never receives `[DONE]` (hung SSE connection). Have `subscribe()` detect the finished state and return a channel that's closed after replaying `last`.

- [x] **WebSocket broadcast has no write deadline** (`internal/server/ws.go:43`)
  One slow/half-dead client blocks the broadcast loop, stalling notifications for everyone. Set a write deadline per `conn.Write`, or push to a per-client buffered channel with a drop policy (as the research hub does).

- [x] **User message persisted before model reachability is confirmed** (`internal/server/messages.go:274`)
  On model-unreachable the user turn is left stored with no assistant reply. Persist after a successful response start, or clean up on early failure.


- [x] **`note_list` prefix semantics differ between model and UI** (`internal/store/notes.go:198`)
  Model-facing `ListNotes` treats a bare term as `g.foo%` (matches `foobar`); the settings `ListUserVisibleNotes` uses `g.foo.%` (segment boundary). Pick one rule and share it.

- [x] **Migration version-label gap** (`internal/store/store.go:228`)
  The v8 block is followed by `if version < 10` labelled "v9 → v10" with no v8→v9 block. Works, but relabel/comment to avoid confusion.

- [x] **`parseJSONObject` shadows builtin `close`** (`internal/research/llm.go:266`)
  Rename the local to `closeIdx`.

### Duplication

- [x] **Extract a shared streaming chat-completion helper** (`internal/server/messages.go:311`, `internal/server/completions.go:246`, `internal/research/llm.go:78`)
  The SSE scanner loop (`1<<20` buffer, `"data: "` prefix, `[DONE]`) and request plumbing are reimplemented three times.

- [x] **Extract a shared non-streaming chat-completion helper** (`internal/server/tools.go:1320`, `internal/research/llm.go:23`, `internal/tasks/titles.go:162,299`)
  Build payload → set auth → decode `choices[0].Message.Content` is duplicated four times.

- [x] **De-duplicate SearXNG search** (`internal/server/tools.go:862`, `internal/research/web.go:23`)
  Implemented twice with separate response structs.

- [x] **De-duplicate HTML stripping** (`internal/server/tools.go:1305`, `internal/research/web.go:67`)
  Two separate regex-based strippers.

- [x] **Unify conversation/completion title generation** (`internal/tasks/titles.go`)
  Near-duplicate function pairs for the two entity types.

- [x] **Add a handler helper for the parse-id → Get → 404/500 pattern** (`internal/server/`)
  Repeated dozens of times across the conversation/character/completion/research/notes handlers.

### Organisation & maintainability

- [ ] **Split `tools.go` by concern** (`internal/server/tools.go`)
  1637 lines mixing schemas, executors, three external-service clients, HTML stripping, and image generation. Split into registry / web / image / notes-state files.

- [ ] **Reduce the four hand-synced tool lists** (`internal/server/tools.go:58,627,1595`, `static/js/settings-character-edit.js`)
  `toolRegistry`, `executors`, `allTools`, and `TOOL_GROUPS` must be kept in sync manually. CLAUDE.md has already drifted (missing `note_to_self`, `state_clear`; wrong `world_state` expansion). Drive them from one declaration and fix the CLAUDE.md tool table.

- [ ] **Break up `handleSendMessage`** (`internal/server/messages.go:195`)
  ~430 lines doing model resolution, tool-loop orchestration, SSE framing, persistence, and title triggers. Extract the tool loop and title-trigger block at minimum.

- [ ] **`gofmt` six files and add gofmt/vet to CI** (`internal/server/{admin,avatars,messages,notes,research}.go`, `internal/store/research.go`)
  Flagged by `gofmt -l` (vet and build are clean). CI should fail on a gofmt diff.

- [ ] **Periodic expired-session cleanup** (`internal/store/sessions.go:41`)
  Expired sessions are only deleted lazily on access; never-revisited sessions linger forever. Add a sweeper (an index on `expires_at` already exists).

- [ ] **Make tool timeouts named/configurable** (`internal/server/tools.go`, `internal/research/web.go`)
  Magic literals (120s ComfyUI poll, 30s/15s/5s fetch timeouts) scattered inline; name them or move to config as the research engine already does.

- [ ] **Resolve static asset paths absolutely** (`internal/server/server.go:149,169`)
  `serveFile` and the `FileServer` use relative `static/` paths that depend on the process CWD; resolve against an absolute asset dir. Also disable directory listings on the catch-all `FileServer`.

- [x] **Image generation `background` parameter** (`internal/server/tools.go:210,251`)
  Add an optional `background` boolean parameter (default false) to both image generation tools. When true and the image loads, set it as the chat background (`static/js/thread.js:186`). Apply a legibility treatment (e.g. semi-transparent overlay on the message column) so text remains readable over the image.

- [x] **Async image generation** (`internal/server/tools.go:1400`)
  `makeImageExecutor` blocks the SSE stream for up to 120s while ComfyUI polls. Per `docs/feature-async-image-generation.md`: return a pending attachment immediately after the ComfyUI submit, poll/download in a background goroutine, and push `attachment_ready`/`attachment_error` to the frontend via WebSocket when done.

- [ ] Look into ways to follow Reddit and facebook links, if possible

- [ ] **Add "set as background" button to inline images** (`static/js/thread.js:187`)
  `buildInlineImage` shows only a download button on hover. Add a second hover button (image/wallpaper icon) that calls `setBackground(att.id)` so users can set any generated image as the conversation background without asking the model to use `background: true`.

- [x] **Size image placeholder to match final image dimensions** (`static/js/thread.js:64`)
  `buildImagePlaceholder` used a fixed 320×160 box, but a loaded image can be taller (inline images are capped at 320×240). When an async image resolved it grew past the placeholder, pushing text down and forcing a manual scroll while the content jumped. Fixed by sizing the placeholder to the exact render box from the tool-call `args` width/height (defaulting 1024×1024) against the same max constraints, so the swap is seamless. Also fixed the reload-during-pending path to render a sized placeholder the WS event can resolve in place.
