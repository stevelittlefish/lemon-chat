# lemon-chat — TODO

Status markers: `[ ]` not started · `[~]` in progress · `[x]` done

Completed work is archived in [`docs/archive/TODO-completed.md`](docs/archive/TODO-completed.md).

--

## User attachments (backlog)

- [ ] **Image attachments.** Allow users to upload and send image files in chat.
- [ ] **Text file attachments.** Allow users to upload and send text files in chat.
- [ ] **Paste images into the chat composer.** Accept image data pasted from the clipboard and attach it to the pending message.

## Provider abstraction & OAuth (backlog)

Design docs: [`docs/provider_abstraction.md`](docs/provider_abstraction.md),
[`docs/openai_oauth_plan.md`](docs/openai_oauth_plan.md). Work is on the `openai-oauth` branch.

- [ ] **Streaming thinking: persist reasoning to the chat log, show on reload, exclude from model input.** Add a `reasoning TEXT` column to the `message` table (`internal/store/messages.go`, new migration), populate on assistant rows via the provider `Handler.OnThinking`/`Completion.Thinking`, return it from `handleListMessages`, render a collapsible thinking block in the frontend, and stream it live over a new SSE event. The outbound lowering must never include it. Also drop `include:["reasoning.encrypted_content"]` from `BuildResponsesBody` since reasoning is not replayed. See `docs/provider_abstraction.md` → Reasoning.

- [ ] **Ollama native-API provider.** New provider package implementing the `Provider` seam against Ollama's native `/api/chat` (its own request/response + NDJSON streaming), selected by `api = "ollama"`. Additive once the provider seam exists.

- [ ] **Anthropic provider.** New provider package implementing the `Provider` seam against the Anthropic Messages API (typed content blocks, top-level system, tool_result as user block, cache breakpoints, its own SSE grammar), selected by `api = "anthropic"`. Additive once the provider seam exists.
