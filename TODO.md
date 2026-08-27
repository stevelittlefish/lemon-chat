# lemon-chat — TODO

Status markers: `[ ]` not started · `[~]` in progress · `[x]` done

Completed work is archived in [`docs/archive/TODO-completed.md`](docs/archive/TODO-completed.md).

--

## User attachments (backlog)

- [x] **Image attachments (vision input).** Users upload an image and send it with text to vision-capable models. Reused the `attachment` table (added `message_id` + `source` columns, migration v43); `POST /api/conversations/{id}/attachments` upload handler; `buildChatMsgs`/`chatMsg.MarshalJSON` lower to chat-completions `image_url` parts and `internal/llm/responses.go` to `input_image`; gated on a `vision` flag on `[[model]]`; composer attach button + paste + thumbnail strip, thread renders sent images. Orphan cleanup for never-sent uploads is still outstanding — see below.
- [ ] **Clean up orphaned image uploads.** Uploads whose message is never sent (`source='upload'` and `message_id IS NULL`) accumulate on disk and in the DB. Add a background sweep (mirror `internal/tasks/cleanup.go`) that deletes upload attachments older than some threshold with a NULL `message_id`, removing both the row and the `<data_dir>/attachments/<id>/` directory.
- [ ] **Text file attachments.** Allow users to upload and send text files in chat.
- [x] **Paste images into the chat composer.** Clipboard image paste is handled by the composer's paste listener (`static/js/composer.js`), uploading each pasted image as an attachment.

## Provider abstraction & OAuth (backlog)

Design docs: [`docs/provider_abstraction.md`](docs/provider_abstraction.md),
[`docs/openai_oauth_plan.md`](docs/openai_oauth_plan.md). Work is on the `openai-oauth` branch.

- [ ] **Streaming thinking: persist reasoning to the chat log, show on reload, exclude from model input.** Add a `reasoning TEXT` column to the `message` table (`internal/store/messages.go`, new migration), populate on assistant rows via the provider `Handler.OnThinking`/`Completion.Thinking`, return it from `handleListMessages`, render a collapsible thinking block in the frontend, and stream it live over a new SSE event. The outbound lowering must never include it. Also drop `include:["reasoning.encrypted_content"]` from `BuildResponsesBody` since reasoning is not replayed. See `docs/provider_abstraction.md` → Reasoning.

- [ ] **Ollama native-API provider.** New provider package implementing the `Provider` seam against Ollama's native `/api/chat` (its own request/response + NDJSON streaming), selected by `api = "ollama"`. Additive once the provider seam exists.

- [ ] **Anthropic provider.** New provider package implementing the `Provider` seam against the Anthropic Messages API (typed content blocks, top-level system, tool_result as user block, cache breakpoints, its own SSE grammar), selected by `api = "anthropic"`. Additive once the provider seam exists.
