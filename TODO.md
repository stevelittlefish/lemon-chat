# lemon-chat — TODO

Status markers: `[ ]` not started · `[~]` in progress · `[x]` done

Completed work is archived in [`docs/archive/TODO-completed.md`](docs/archive/TODO-completed.md).

--

## Memory and tools

Raised by conversation 102, where the general assistant's opening `note_list` dumped every adventure note (with spoilers) to find `u.memories`. Notes were built for the choose-your-own-adventure games and later repurposed as memory; the games are no longer used, and the notes feature has now been removed.

- [~] **Build a real memory system** (notes, which were repurposed as memory, have been removed). Agreed design:
  - *Storage:* a `memory` table, one fact per row: user, text, created/updated, source conversation. One pool of memories per user; no domains or per-character memories until a real need appears (add the column then, not in advance).
  - *Access is the tool:* memory is a selectable tool group (e.g. `memory_list`, `memory_save`, `memory_update(id)`, `memory_forget(id)`). A character without it can't read or save memories and isn't told they exist. No other per-character memory setting.
  - *Retrieval:* the model calls `memory_list` itself. Optional per-character toggle "Load memories at start" next to the tool in the character editor (the first tool with an option): when on, the server stores a synthetic `memory_list` call and its result as the opening messages of a new conversation. Frozen for that conversation, so prompt caching keeps working; visible in View LLM context. Not injected into the system prompt.
  - *Saving:* the model decides, guided by the tool description: save lasting facts that help future conversations (preferences, important people, ongoing projects, things the user says about themselves); not small talk, one-off details, roleplay/story content, or likely-private things unless asked; update rather than duplicate. Without auto-load, `memory_save` returns the current list so the model can avoid duplicates.
  - *Oversight:* every save shows as a tool call in the chat. A "Memories" settings page (in the settings sidebar, where Notes was) lists the user's own memories for viewing, editing and deleting. Admins get a user picker on the same page to view, edit and delete another user's memories (admin actions on another user's memories get logged like other admin actions).
  - *Built:* everything above, plus adding memories by hand on the settings page and a 500-character cap per memory.
  - *Still to do — testing:* tune the save guidance on both Gemma and `gpt-sol`; expect one to over-save and the other to under-save.
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
  - `notes` / `note_*` — **removed (decided and done).**
- [ ] **Redesign `fetch_url`: outline, sections and find instead of a model rewrite.** Must work well on Gemma 4 (vLLM, 64k context).
  - *Current behaviour, tested on the Wikipedia Python article (~138,000 characters after stripping):* the tool keeps the first 50,000 characters and sends them through a hidden model call (`summariseHTML`, `internal/server/tools_web.go:361`) to rewrite as markdown. On Gemma this was very slow, mostly spent writing a ~2,000-token summary. Facts inside the kept part survived, but ~60% of the article was silently dropped: the "[truncated]" note went into the rewrite and was lost, so for a question past the cut-off the model answered "not mentioned" rather than "not read". About the first 20,700 characters were page junk (language list, navboxes, raw wiki markup) that `htmltext.Strip` kept, wasting ~40% of the budget. Truncation also slices bytes and can split a multi-byte character (`tools_web.go:74`, `:82`).
  - *Rejected:* a goal-based helper call per fetch (every follow-up question about the same page feeds the whole page to Gemma again, and reading the page is the slow part).
  - *Plan:* no hidden model call. The first fetch downloads, cleans and splits the page into sections (by headings, or numbered chunks when a page has none), keeps it server-side for the conversation, and returns only an outline: title, section names and sizes, first paragraph. The model then reads a named section, or uses find to get just the passages matching a term (like Ctrl+F). Follow-ups read another small piece instead of the whole page again, and the outline shows the model everything that exists, so it can't mistake "not read" for "not there". Same idea as `wikipedia_get_page`. Strip page junk before splitting. Keep `source: true` for raw HTML, in chunks.
  - *Open:* how long to keep fetched pages and where (memory vs disk); handling messy pages with no headings; section size limits.
  - **Needs plenty of experimenting and thorough testing before it's considered done:** the Python article questions (3.0 release date ~char 23,900; walrus operator ~36,300; CPython's C standard ~52,300, past the old cut-off), the Lemon article (under the old cap), pages without headings (forums, badly structured sites), and very long pages, on both Gemma and `gpt-sol`, checking speed and that the model doesn't answer from memory when it hasn't read the section.

## Handoff (backlog)

- [ ] **Handoff.** An easy way to generate a handoff document from a conversation that can either be downloaded or used to seed a new conversation. The user has a design in mind; to be discussed and written up when we get to it. Download could reuse the attachment serving path (`internal/server/attachments.go`); seeding fits alongside fork/import in `internal/server/conversations.go`.

## Conversation history (backlog)

- [~] **Conversation history tools: search past chats and load one into context.** A new tool group in `toolRegistry` (`internal/server/tools_registry.go:44`): search the user's previous conversations (titles and message text) and load a chosen conversation's messages into the current context. Store side starts from `ListConversations` (`internal/store/conversations.go:21`); search needs a new query (conversation search is not built yet). Loading must cap or summarise long conversations so it doesn't blow the context.
  - *Built:* `history_list`, `history_search` and `history_read` (`internal/server/tools_history.go`), in the Memory group as "Conversation history". Read pages at 24,000 characters. Search is word-based full-text search (FTS5, stemmed, any word matches, ranked by how many words a conversation mentions) so the model can throw in synonyms. Narrowing down: search inside one conversation (`conversation_id`) for numbered matching messages, `history_read` `start`/`count` to read around one, and `since`/`until`/`character` filters on list and search.
  - *Still to do — testing:* try on Gemma and `gpt-sol`, e.g. "what did we talk about last week" and searching for a half-remembered topic; check the model actually adds synonyms when the first search misses. If wording still defeats it, semantic search with an embedding model is the next step.
- [~] **Let admins use the history tools on other users' conversations.** An admin should be able to ask, say, for a summary of what a family member has been talking about.
  - *Built:* a separate admin-only tool group, "Conversation history (admin)": `admin_list_users`, `admin_history_list`, `admin_history_search`, `admin_history_read`. Leaving out `user` covers every user's conversations; each result says whose it is. The regular `history_*` tools are own-conversations only, with no user argument, so non-admins never see anything about other users. Admin tools are withheld from non-admins (model tool list, `/api/tools`, and the executors refuse), and every call is logged with `admin_id`.
  - *Found in testing (before the split):* Gemma called `history_list` without naming the user and summarised the admin's own chats as "Steve's"; the admin tools now label every conversation's owner, and the regular tools name "the person you are talking to".
  - *Still to do — testing:* ask an admin character for a summary of another user's recent conversations on both models.

## User attachments (backlog)

- [ ] **Text file attachments.** Allow users to upload and send text files in chat.

## Image generation (backlog)

- [ ] **Advanced image generation tools.** A second, opt-in tool set alongside the general-purpose `generate_image` (`internal/server/tools_image.go`), for experimenting with image generation from chat: list every registered workflow in `comfyui/workflows.toml` (not just `exposed` ones, with descriptions and availability), and generate with extra parameters such as CFG, steps, sampler/scheduler and explicit width/height (e.g. the same prompt at a fixed seed across several CFG values). Needs new optional bindings in the registry (`internal/comfyui/registry.go`) for those inputs, validated per workflow like the existing ones.
- [ ] **Image edit tool.** Let the model edit an existing image, either one the user pasted (`source = "upload"`) or one generated earlier in the conversation, possibly with several reference images at once. Alongside `generate_image` (`internal/server/tools_image.go:175`); select images by ID the way `view_image` does (`tools_image.go:426`), and needs edit/reference workflows plus image-input bindings in the registry (`internal/comfyui/registry.go`).

## Provider abstraction & OAuth (backlog)

Design docs: [`docs/provider_abstraction.md`](docs/provider_abstraction.md),
[`docs/openai_oauth_plan.md`](docs/openai_oauth_plan.md). Work is on the `openai-oauth` branch.

- [ ] **Streaming thinking: persist reasoning to the chat log, show on reload, exclude from model input.** Add a `reasoning TEXT` column to the `message` table (`internal/store/messages.go`, new migration), populate on assistant rows via the provider `Handler.OnThinking`/`Completion.Thinking`, return it from `handleListMessages`, render a collapsible thinking block in the frontend, and stream it live over a new SSE event. The outbound lowering must never include it. Also drop `include:["reasoning.encrypted_content"]` from `BuildResponsesBody` since reasoning is not replayed. See `docs/provider_abstraction.md` → Reasoning.

- [ ] **Ollama native-API provider.** New provider package implementing the `Provider` seam against Ollama's native `/api/chat` (its own request/response + NDJSON streaming), selected by `api = "ollama"`. Additive once the provider seam exists.

- [ ] **Anthropic provider.** New provider package implementing the `Provider` seam against the Anthropic Messages API (typed content blocks, top-level system, tool_result as user block, cache breakpoints, its own SSE grammar), selected by `api = "anthropic"`. Additive once the provider seam exists.
