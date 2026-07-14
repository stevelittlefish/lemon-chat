# lemon-chat — TODO

Status markers: `[ ]` not started · `[~]` in progress · `[x]` done

--

## Research algorithm rework

- [x] **Research engine rework — diagnostic logging + algorithm fixes.** Full plan in [`RESEARCH_TODO.md`](RESEARCH_TODO.md). Discards the oversized structured-ledger / JSON-contract / new-DB-tables approach in favour of a token-budget split, two free-text prompt rewrites, and a disk-based downloadable debug bundle — ~90% of the value at a fraction of the complexity, and no small-model (Gemma) regression. Delivered: Part 1a/1b (disk run-log + debug bundle), Part 2 (token-budget split, synthesis memory bound, stop-check rewrite), and Part 3b (fuzzy query dedup). Validated on Gemma 4B and Grok 4.5 — original runaway/truncation bug gone. Part 1c and Part 3a/3c deliberately left unbuilt (validation showed no need).

## Manually Added by User

- [x] Add a benchmark folder with the home-battery-without-solar research prompt.

- [x] Add an admin Tools page action that lists available models from every configured provider in copy-friendly tables.

- [x] Remove the admin tool for deleting orphaned messages.

- [x] When multiple messages are present from the same user, only show the avatar for the first one.

- [x] Add a default avatar for characters, models and users if they haven't uploaded one.  Use the CPU icon for models and the Drama icon for characters.  We need a silhouette of a human head for the user.

- [x] Research feature (port of docs/deep_research_spec.html to Go) — `/research` page linked from the main menu, iterative search/extract/synthesise engine in `internal/research/`, crash-resumable jobs via state checkpoints in the `research_job` table.

- [x] **Rename `sdxl_file`/`flux_file` config keys to `sdxl_workflow`/`flux_workflow`** (`internal/config/config.go:28`)

## Research

- [x] On the research list page, add a direct "HTML" link for items whose default report has a designed HTML version, opening it in a new tab (`/api/research/{id}/report/document`) without going through the report detail page. Backed by `ListResearchJobsWithDefaultHTML` → `report_html` on the list view.

- [x] Make the backend research log tell the story of a run in ~5-10 always-on stdout lines (start, plan, per-round search + synthesis, stop decision, final write, finish); the high-frequency per-page "reading" line is demoted to `debug.Log`. `logResearchProgress` in `research.go`.

- [x] Fold the debug run-log bundle into the job view's Download drop-down (shown for any terminal job, so failed/cancelled jobs can grab it too), and add an in-UI diagnostics view on the Explore page — stop reason, effective config/outcome, and a collapsible event timeline — served from `GET /api/research/{id}/debug` (`readResearchDebug`). No download needed.

- [x] Add a ZIP bundle download containing every research report and the job's displayed metadata and statistics.

- [x] Stream live generation progress for the auto HTML report's "designing the HTML report" phase. `autoGenerateHTMLReport` now passes an `onProgress` callback to `generateReportHTML` that broadcasts `research.Progress{Phase: "designing", Generated, Snippet}` over SSE, and `updateStream` labels the phase "designing HTML report", so the phase streams like the other long generation steps.

- [x] Remove the report-remix output-token cap so providers can generate up to their supported limit.

- [x] Indicate which past research results have saved remixes in the research list.

- [x] Track OpenRouter response cost in the shared LLM API layer, accumulate and store research/report-remix prices, and display them in the research UI.

- [x] Add an “open in new tab” action beside the HTML download on saved remix pages.

- [x] Stream remix generation progress over SSE and make the remix form's submitted state visibly disabled and non-interactive.

- [x] Refine the completed-report header: separate actions from metadata, reduce action-label clutter, and remove the empty remix shelf's dead space.

- [x] Add report remixes: generate and save any number of polished standalone HTML presentations from a completed report, with model selection, optional art direction, and report-page navigation.

- [x] Show the model name on past research items and make the list more compact.

- [x] Improve research citations: use stable source IDs in findings/prompts and feed raw findings into final report generation so links survive summarisation.

- [x] Normalize combined research citations like `[S13, S18]` into separate clickable source references.

- [x] Long prompts mess up the layout.  Separate into title and prompt (both optional - one must be specified)

- [x] The back button in the top left when viewing an individual piece of deep research should take you back to the list, not the menu

- [x] Brainstorming mode - alternate algorithm where we want the LLM to invent / design stuff and use web search when necessary, rather than basing the entire process around web search

- [x] In-depth report toggle: build the final report section by section from the raw findings (outline → self-critique refine → per-section write → glue) instead of one summarising pass, to stop reports losing detail. Opt-in checkbox; applies to both modes. `deepReport` pipeline in `internal/research/researcher.go`.

### User-assisted Reddit imports

Design: `docs/feature-reddit-research-import.md`

- [x] Generate a short research slug and use `<slug>_<round>.json` for Reddit request and response filenames.
- [x] Make request copying work when the research page is served over insecure HTTP.
- [x] Define and test the versioned Reddit import/request JSON contract, URL canonicalisation, validation limits, and normalized LLM-facing text.
- [x] Add synthetic import fixtures covering nested, deleted, duplicate, oversized, partial, and prompt-injection-shaped content.
- [x] Add a debug-only single-pass test harness: SearXNG Reddit search, request export, response validation, normalized preview, extraction, and formatted finding preview.
- [x] Build a Chromium Manifest V3 "Save Reddit" extension that accepts a request bundle, visits threads sequentially, expands/scrolls within limits, and exports a response bundle with completeness warnings.
- [x] Add the per-job "Pause to import Reddit results" option, off by default, to config persistence, API views, and the research form.
- [x] Add a numbered database migration and store operations for the durable `awaiting_reddit` state and pending round/request data.
- [x] Split and canonicalize search results, checkpoint before extraction when Reddit imports are required, and exclude user wait time from the job time budget.
- [x] Add authenticated import and skip endpoints with request-ID matching, ownership checks, size limits, deduplication, and idempotent resume behavior.
- [x] Add the research-panel waiting UI with URL list, request copy/download, response paste/upload, validation feedback, skip, cancel, and reconnect behavior.
- [x] Merge imported Reddit documents through the existing untrusted-content extraction path, synthesize them with ordinary findings, and prevent repeat requests for analyzed threads.
- [x] Add restart/recovery, stale-import, duplicate-submit, cancellation, partial-capture, and end-to-end tests; document extension installation and operational limitations.
- [x] Fix the Reddit import debug harness POST requests to satisfy the authenticated-write CSRF header requirement.
- [x] Normalize root-relative Reddit comment permalinks in the browser extension before response export.
- [x] Prefix Reddit browser-extension response filenames with `reddit_`.
- [x] Move the research Reddit-import checkbox onto its own row beneath the main form controls.
- [x] Add a detailed research help page linked from the capitalized New Research form heading.
- [x] Add the supplied Reddit artwork as the Save Reddit browser extension icon.
- [x] Generate 32px, 48px, and 128px extension icons while preserving the existing 16px icon.
- [x] Clear the Reddit extension's completed-capture toolbar badge after 60 seconds.
- [x] Improve the Save Reddit extension completion state, add a load-more limit, and support exporting the active Reddit tab.
- [x] Compact the Save Reddit popup so it fits without an internal scrollbar.
- [x] Prevent current-page Reddit exports from closing when expansion links navigate the active tab.

### Research improvements — ordered queue (2026-07-13)

Fleshing out research so a finished job is an inspectable, reusable artifact — and so kicking one off is dead simple by default but deeply customisable when you want it.

**This is an ordered work queue.** When the user says "let's do the next research improvement", pick the **lowest-numbered item still marked `[ ]`** in this list and do that one. The order encodes dependencies — the reports refactor (#2) underpins most of what follows — so don't skip ahead unless the user explicitly names a different item. Mark `[~]` when starting and `[x]` when done, so the "next" pointer stays accurate.

- [x] **1. Add an "explore" view for all saved job data** (`internal/server/research.go:530`, `static/js/research-app.js`)
  A completed job already persists every input to the final report — `findings` (with `rational`/`evidence`/`summary`), `analyzed_urls`, `queries_used`, the evolving `report`, `plan`, and `category` — but the UI only surfaces the final report. Add an explore panel that lets you browse all of it (findings grouped by source with full evidence, the query history per round, every URL attempted, the intermediate synthesis). `GetResearchJob` already returns these columns, so this is mostly a frontend build plus possibly a lighter-weight endpoint that omits nothing.

- [x] **2. Refactor: multiple reports per job** (`internal/store/research.go`, `internal/store/store.go`, `internal/store/research_remixes.go:8`)
  Introduce a `research_report` concept: each job has any number of reports, one by default. A report is a master markdown document (today's `final_report`) plus an optional HTML version (today's remix output). Add a numbered migration that creates the table and back-fills existing `final_report` values and `research_remix` rows into it, then repoint the report/remix read paths. This is the structural change the toggle and revamped remix build on — do it before #3–#6.

- [x] **3. Start-form option to auto-generate the HTML report** (`internal/server/research.go:441`, `internal/server/research.go:318`, `static/js/research-app.js:100`)
  Add a form section with a tick box (on by default) to also produce the HTML report, plus a text box for stylistic requirements. When set, after the markdown report finishes the job automatically runs what remix does today and stores the HTML on the default report. Persist the toggle and style prompt on the job so a resumed/queued job still honours them.

- [x] **4. Optional separate model for the HTML report step** (`internal/server/research.go:441`, `internal/config/config.go:23`)
  Default to using the job's model for the whole process, but allow overriding just the HTML-generation model in the start form (and via config default). Thread it through to the auto-remix step from #3.

- [x] **5. SPIKE: per-phase model configuration** (`internal/research/researcher.go`, `internal/research/llm.go:21`, `internal/config/config.go:23`)
  Investigate whether letting a job use different models for different phases is worth it — e.g. a cheap local model for search/extract, a strong model for the final report, another for the HTML. Every phase already routes through `llmCall`/`llmCallStream`, so plumbing a per-phase model is tractable; the question is the config/UI complexity vs. benefit. Write up a recommendation before committing to an implementation.
  **Recommendation:** `docs/spike-per-phase-model-config.md` — do a two-tier worker/writer split (one optional `worker_model` for the high-volume extraction + mechanical phases, job model for synthesis/report), not a full per-phase matrix.

- [x] **6. Implement the two-tier worker/writer model split** (`internal/research/llm.go:21`, `internal/research/researcher.go:90`, `internal/server/research.go:222`, `internal/config/config.go:23`)
  Act on the #5 spike (`docs/spike-per-phase-model-config.md`). Add one optional `worker_model` that handles the high-volume extraction phase plus the mechanical calls (slug, classify, query generation, decide), keeping the job model for synthesis and the final/deep report. Steps: (1) introduce a `modelEndpoint{Model, APIBase, APIKey}` and give the `Researcher` `writer`/`worker` endpoints (worker defaults to writer), parameterising `llmCall`/`llmCallStream` by endpoint and updating the worker-tier call sites; (2) add `[research] worker_model` config + a `research_job.worker_model` column via numbered migration + persistence, mirroring #4; (3) resolve and validate both endpoints via `ServerForModel` at launch; (4) add the optional "worker model" control to the start form and surface it in the job detail/explore view. Do not build the full per-phase matrix — the spike rejected it.

- [x] **7. Revamp remix into report regeneration** (`internal/server/research_reports.go`)
  Today remix only reskins the existing `final_report` into HTML and can't recover detail the writer dropped. Extend it to also (a) regenerate a fresh markdown report from the raw `findings`/`report`/`plan` (reusing `finalReport`/`deepReport`), or (b) do both, saving results as new reports on the job. Model is configurable per #4. Depends on the reports refactor (#2).

- [x] **8. Holistic progressive-disclosure UI revamp** (`static/js/research-app.js:100`, `static/research.html`)
  Gave the start form two-tier progressive disclosure: title, prompt, a compact model/mode/effort row, and the start button stay visible, with worker model, time limit, in-depth report, HTML report (+ nested style/model), and Reddit import behind a "More options" toggle. Consolidated the detail view's Markdown/HTML downloads into a single "Download" popover (reusing the `.menu` component), and added an `HTML: <model>` meta chip shown only when the HTML-report model differs from the main model. The job list was already compact; the remix form and report view already reveal controls on demand, so they were left as-is.

- [x] **9. Preserve all sources and citations in the HTML report** (`internal/server/research_reports.go`, `internal/server/research.go`, `static/js/research-app.js`)
  Made inline citations clickable straight through to the source. Strengthened `reportHTMLSystemPrompt` to require every `[S1]` marker be rendered as a direct external `<a href>` (resolved from the report's source list / reference definitions, `target="_blank"`), never an in-page `#fragment` anchor. Also relaxed the report-document sandbox so those links can actually open: added `allow-popups allow-popups-to-escape-sandbox` to both the CSP `sandbox` directive (`research_reports.go`, `research.go`) and the two viewer iframes (`research-app.js`), while keeping scripts disabled. Applies to old reports on regeneration.

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

- [ ] Look into ways to follow Facebook links, if possible

- [ ] **Add "set as background" button to inline images** (`static/js/thread.js:187`)
  `buildInlineImage` shows only a download button on hover. Add a second hover button (image/wallpaper icon) that calls `setBackground(att.id)` so users can set any generated image as the conversation background without asking the model to use `background: true`.

- [x] **Size image placeholder to match final image dimensions** (`static/js/thread.js:64`)
  `buildImagePlaceholder` used a fixed 320×160 box, but a loaded image can be taller (inline images are capped at 320×240). When an async image resolved it grew past the placeholder, pushing text down and forcing a manual scroll while the content jumped. Fixed by sizing the placeholder to the exact render box from the tool-call `args` width/height (defaulting 1024×1024) against the same max constraints, so the swap is seamless. Also fixed the reload-during-pending path to render a sized placeholder the WS event can resolve in place.
