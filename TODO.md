# lemon-chat — TODO

Status markers: `[ ]` not started · `[~]` in progress · `[x]` done

Completed work is archived in [`docs/archive/TODO-completed.md`](docs/archive/TODO-completed.md).

--

## Notes, memory and tools (to review one by one)

Raised by conversation 102, where the general assistant's opening `note_list` dumped every adventure note (with spoilers) to find `u.memories`. Notes were built for the choose-your-own-adventure games and later repurposed as memory; the games are no longer used. Each item below needs a decision before any work.

- [ ] **Replace notes-as-memory with a real memory system.** Sketch: a `memory` table (one fact per row: user, text, created/updated, source conversation); all of a user's memories put into the system prompt automatically, so no lookup is needed (search only if it grows to hundreds of facts); `memory_save` / `memory_update(id)` / `memory_forget(id)` tools; a settings page to view, edit and delete memories. Starts empty — existing `u.memories` content isn't worth keeping. Open questions: (1) per-character opt-in, or available to every character? (2) save only on "remember…", or let the model decide too?
- [ ] **Remove the notes feature.** The `note_*` tools and `notes` group (`internal/server/tools_registry.go`, `tools_state_notes.go`), the notes settings page (`static/settings/notes.html`, `static/js/settings-notes.js`), note-pack import (`handleAdminImportNotePack`, `internal/server/admin.go:196`), `internal/store/notes.go`, and a migration dropping the `note` table (destructive — confirm first). Existing notes don't need preserving.
- [x] **Review every tool: keep, change or remove.** Reviewed one by one; outcome per tool:
  - `get_time` — **keep as is (decided).** Not moving the date into the system prompt: it breaks prompt caching, and models ignore it in long conversations.
  - `searxng` — **keep (decided); description improved** to say results are snippets and to read pages with `fetch_url` before relying on them. Name and ID unchanged.
  - `fetch_url` — **keep, redesign (decided).** See the fetch_url item below.
  - `wikipedia_search` / `wikipedia_get_page` — **keep as is (decided).** Outline-then-section reading works well; measured sections were a sensible size for Gemma (largest seen ~20,700 characters).
  - `create_document` — **keep as is (decided).**
  - `generate_image` / `view_image` — **keep as is (decided).**
  - `roll_dice`, `pick_random`, `random_chance` — **keep as is (decided).** Only enabled on characters that want them.
  - `world_state` / `state_*` — **keep as is (decided).** Opt-in; useful for any game or tracker within a conversation, not only the adventures.
  - `note_to_self` — **keep as is (decided).** Essential for games: the model commits to hidden outcomes before narrating, so it can't cheat, and the user can check afterwards via View LLM context. Not part of the notes feature (no-op executor, `tools_state_notes.go:71`; doesn't use the `note` table).
  - `notes` / `note_*` — see the two items above.
- [ ] **Redesign `fetch_url`: outline, sections and find instead of a model rewrite.** Must work well on Gemma 4 (vLLM, 64k context).
  - *Current behaviour, tested on the Wikipedia Python article (~138,000 characters after stripping):* the tool keeps the first 50,000 characters and sends them through a hidden model call (`summariseHTML`, `internal/server/tools_web.go:361`) to rewrite as markdown. On Gemma this was very slow, mostly spent writing a ~2,000-token summary. Facts inside the kept part survived, but ~60% of the article was silently dropped: the "[truncated]" note went into the rewrite and was lost, so for a question past the cut-off the model answered "not mentioned" rather than "not read". About the first 20,700 characters were page junk (language list, navboxes, raw wiki markup) that `htmltext.Strip` kept, wasting ~40% of the budget. Truncation also slices bytes and can split a multi-byte character (`tools_web.go:74`, `:82`).
  - *Rejected:* a goal-based helper call per fetch (every follow-up question about the same page feeds the whole page to Gemma again, and reading the page is the slow part).
  - *Plan:* no hidden model call. The first fetch downloads, cleans and splits the page into sections (by headings, or numbered chunks when a page has none), keeps it server-side for the conversation, and returns only an outline: title, section names and sizes, first paragraph. The model then reads a named section, or uses find to get just the passages matching a term (like Ctrl+F). Follow-ups read another small piece instead of the whole page again, and the outline shows the model everything that exists, so it can't mistake "not read" for "not there". Same idea as `wikipedia_get_page`. Strip page junk before splitting. Keep `source: true` for raw HTML, in chunks.
  - *Open:* how long to keep fetched pages and where (memory vs disk); handling messy pages with no headings; section size limits.
  - **Needs plenty of experimenting and thorough testing before it's considered done:** the Python article questions (3.0 release date ~char 23,900; walrus operator ~36,300; CPython's C standard ~52,300, past the old cut-off), the Lemon article (under the old cap), pages without headings (forums, badly structured sites), and very long pages, on both Gemma and `gpt-sol`, checking speed and that the model doesn't answer from memory when it hasn't read the section.

## User attachments (backlog)

- [ ] **Text file attachments.** Allow users to upload and send text files in chat.

## Image generation (backlog)

- [ ] **Advanced image generation tools.** A second, opt-in tool set alongside the general-purpose `generate_image` (`internal/server/tools_image.go`), for experimenting with image generation from chat: list every registered workflow in `comfyui/workflows.toml` (not just `exposed` ones, with descriptions and availability), and generate with extra parameters such as CFG, steps, sampler/scheduler and explicit width/height (e.g. the same prompt at a fixed seed across several CFG values). Needs new optional bindings in the registry (`internal/comfyui/registry.go`) for those inputs, validated per workflow like the existing ones.

## Provider abstraction & OAuth (backlog)

Design docs: [`docs/provider_abstraction.md`](docs/provider_abstraction.md),
[`docs/openai_oauth_plan.md`](docs/openai_oauth_plan.md). Work is on the `openai-oauth` branch.

- [ ] **Streaming thinking: persist reasoning to the chat log, show on reload, exclude from model input.** Add a `reasoning TEXT` column to the `message` table (`internal/store/messages.go`, new migration), populate on assistant rows via the provider `Handler.OnThinking`/`Completion.Thinking`, return it from `handleListMessages`, render a collapsible thinking block in the frontend, and stream it live over a new SSE event. The outbound lowering must never include it. Also drop `include:["reasoning.encrypted_content"]` from `BuildResponsesBody` since reasoning is not replayed. See `docs/provider_abstraction.md` → Reasoning.

- [ ] **Ollama native-API provider.** New provider package implementing the `Provider` seam against Ollama's native `/api/chat` (its own request/response + NDJSON streaming), selected by `api = "ollama"`. Additive once the provider seam exists.

- [ ] **Anthropic provider.** New provider package implementing the `Provider` seam against the Anthropic Messages API (typed content blocks, top-level system, tool_result as user block, cache breakpoints, its own SSE grammar), selected by `api = "anthropic"`. Additive once the provider seam exists.
