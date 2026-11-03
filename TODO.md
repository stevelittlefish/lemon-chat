# lemon-chat — TODO

Status markers: `[ ]` not started · `[~]` in progress · `[x]` done

Completed work is archived in [`docs/archive/TODO-completed.md`](docs/archive/TODO-completed.md).

--

## Bugs

- [ ] **Parallel tool calls lose their arguments on Responses-API servers** (`internal/llm/responses.go:328`). In conversation 102 (`gpt-sol` on the Codex server) the model made six parallel note calls; all six arrived with `"arguments": ""` and failed with `invalid args: unexpected end of JSON input`, then worked when retried one at a time. The converter only forwards arguments from `response.function_call_arguments.delta` and ignores `response.function_call_arguments.done` / `response.output_item.done` (the `Arguments` field on `responsesEvent`, `responses.go:213`, is never read). Likely cause: for parallel calls the server sends arguments only in the final event. Not yet confirmed against a wire capture (successful requests aren't in `model_errors.log`; needs a `--token-log` run). Fix: track whether each tool index got any deltas and, if not, emit the full arguments from the `.done` event.

## Notes, memory and tools (to review one by one)

Raised by conversation 102, where the general assistant's opening `note_list` dumped every adventure note (with spoilers) to find `u.memories`. Notes were built for the choose-your-own-adventure games and later repurposed as memory; the games are no longer used. Each item below needs a decision before any work.

- [ ] **Replace notes-as-memory with a real memory system.** Sketch: a `memory` table (one fact per row: user, text, created/updated, source conversation); all of a user's memories put into the system prompt automatically, so no lookup is needed (search only if it grows to hundreds of facts); `memory_save` / `memory_update(id)` / `memory_forget(id)` tools; a settings page to view, edit and delete memories. Starts empty — existing `u.memories` content isn't worth keeping. Open questions: (1) per-character opt-in, or available to every character? (2) save only on "remember…", or let the model decide too?
- [ ] **Remove the notes feature.** The `note_*` tools and `notes` group (`internal/server/tools_registry.go`, `tools_state_notes.go`), the notes settings page (`static/settings/notes.html`, `static/js/settings-notes.js`), note-pack import (`handleAdminImportNotePack`, `internal/server/admin.go:196`), `internal/store/notes.go`, and a migration dropping the `note` table (destructive — confirm first). Existing notes don't need preserving.
- [ ] **Review every tool: keep, change or remove.** Audit notes so far:
  - `get_time` — **keep as is (decided).** Not moving the date into the system prompt: it breaks prompt caching, and models ignore it in long conversations.
  - `searxng` — **keep (decided); description improved** to say results are snippets and to read pages with `fetch_url` before relying on them. Name and ID unchanged.
  - `fetch_url` — sends up to 50,000 characters of page text through a second full model call (`summariseHTML`, `internal/server/tools_web.go:361`) just to rewrite it as markdown: slow, uses quota, and may drop the detail the model wanted. Could return `htmltext`'s structure-preserving output directly. Truncation slices bytes, which can split a multi-byte character (`tools_web.go:74`, `:82`).
  - `wikipedia_search` / `wikipedia_get_page` — overlaps with SearXNG but cleaner for factual lookups; section-by-section reading works well.
  - `create_document` — produces the download/artifact cards.
  - `generate_image` / `view_image` — ComfyUI registry-backed; the most carefully built tools.
  - `roll_dice`, `pick_random` — written for games; harmless, but enabled on the general assistant.
  - `random_chance` — game-only; its description is a long game-master instruction that adds clutter wherever it's enabled.
  - `world_state` / `state_*` — conversation key/value state (HP, gold, inventory); adventure-only. Backed by the `conversation_state` table.
  - `note_to_self` — no-op executor (`tools_state_notes.go:71`); adventure-only, and reasoning models don't need it.
  - `notes` / `note_*` — see the two items above.

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
