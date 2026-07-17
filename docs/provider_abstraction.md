# Provider abstraction — design & implementation guide

Status: **agreed, in progress.** This is a prerequisite for finishing the OpenAI OAuth feature
(see [`openai_oauth_plan.md`](openai_oauth_plan.md)) and the foundation for the target provider set:

- OpenAI (chat-completions, api_key)
- OpenAI OAuth / Codex (Responses API, oauth) — **the feature currently being built**
- Ollama native API — backlog
- Anthropic — backlog

## Why

lemon-chat currently abstracts at the **wire layer**: everything is expressed as OpenAI
chat-completions JSON, and other surfaces are translated into that shape. The Responses API is
supported via a translation shim (`BuildResponsesBody` + `ResponsesToChatSSE` in
`internal/llm/responses.go`) and dispatch is a scattered `if modelServer.UsesResponses()` at ~4
call sites (`messages.go`, `research/llm.go` ×2, `research_reports.go`), with endpoint selection
duplicated in `config.Endpoint()` and `research/llm.go`.

That works for two OpenAI-family surfaces but **breaks for Anthropic and native Ollama**, which are
not chat-completions-shaped (Anthropic: typed content blocks, top-level system, tool_result as a
user block, cache breakpoints, its own SSE grammar; Ollama native: its own request/response and
NDJSON streaming). Writing three translation shims all pretending to be chat-completions, times the
scattered dispatch, is strictly worse than a proper seam.

## Decision

Adopt a **light adaptation of imp's provider abstraction** (imp lives at `~/git/imp`, packages
under `internal/core/ai/`). Take the one part we need — a neutral request + a provider interface +
a single dispatch factory — and **skip imp's agent-runtime machinery** we don't need: the
`Stream`/`Sink` channel event runtime with `Partial` snapshots, the shared `transport` retry/idle
framework, the never-throws contract, and the `ContextWindow`/`ModelPricing` introspection methods
(lemon-chat has its own OpenRouter price worker and does no history compaction).

### Technical contrast (for reference)

| | This design (light) | imp (full) |
|---|---|---|
| Neutral model | small `Request` struct; DB keeps `chatMsg` shape | canonical `ai.Message`/`Content` union the whole app speaks |
| Streaming | synchronous `Handler` callbacks | channel event stream + per-event snapshots |
| Errors | returned normally | never-throws: errors as stream events |
| Shared HTTP loop | each provider POSTs (small helper ok) | `transport` w/ retry ladder + idle watchdog |
| Interface | `Stream` + `ListModels` | + `ContextWindow`, `ModelPricing`, `ListModelListings` |
| Blast radius | additive — providers behind a seam | invasive — rewrites messages.go, DB mapping, research |

## Target architecture

A new provider seam in `internal/llm` (keeping the package; it already holds the shared LLM code):

```go
// Neutral request. Messages stay OpenAI-chat-shaped in the DB; the provider
// lowers them to its own wire body. Prior *thinking* is never included here —
// see the Reasoning section.
type Request struct {
    Model          string
    System         string        // optional; providers that lack a system slot prepend it
    Messages       []Message     // role/content/tool_calls/tool_call_id (today's chatMsg shape)
    Tools          []Tool
    MaxTokens      int
    Temperature    *float64      // nil ⇒ omit (Codex rejects it outright)
    ReasoningEffort string       // "", "low", "medium", "high" — provider lowers to its own shape
}

// Handler receives streaming events. Synchronous callbacks map straight onto the
// chat SSE writer and research's onDelta.
type Handler struct {
    OnText     func(delta string)
    OnThinking func(delta string)              // reasoning/thinking stream (backlog UI, but wired now)
    OnToolCall func(call ToolCall)             // emitted once per completed tool call
    // usage/finish are returned in Completion
}

type Completion struct {
    Content      string
    Thinking     string        // accumulated reasoning (persisted, never replayed)
    ToolCalls    []ToolCall
    Usage        *Usage
    FinishReason string         // "stop" | "length" | "tool_calls"
}

type Provider interface {
    Stream(ctx context.Context, req Request, h Handler) (Completion, error)
    ListModels(ctx context.Context) ([]string, error)
}
```

- **Factory:** `NewProvider(srv *config.ModelServer, tok config.TokenSource, accountID string) Provider`
  — the single place that maps `srv.API` → concrete provider. Call sites become protocol-blind.
- **Provider packages** (or files within `internal/llm` to start — one file each): `openaichat`,
  `openaicodex` (Responses+OAuth), later `ollama`, `anthropic`. Each lowers `Request` to its wire
  body and decodes its stream into `Handler`/`Completion`.
- **Reuse:** the existing `BuildResponsesBody` + Responses SSE decoding become the guts of the
  `openaicodex` provider, retargeted to emit `Handler` events directly instead of re-emitting
  chat-completions frames. `ChatComplete*` become the guts of `openaichat`.

### Call-site migration (no behaviour change)

Replace the branching in these with `provider.Stream(...)`:

- `internal/server/messages.go` — chat send loop (the tool loop stays; it consumes `OnToolCall`).
- `internal/research/llm.go` — `llmCallOn`, `llmCallStreamFinish`.
- `internal/server/research_reports.go` — HTML report writer.
- `internal/tasks/titles.go`, `internal/server/tools.go` (`summariseHTML`) — currently still on
  static api_key; fold into the factory too so oauth works there.

Delete the now-dead scattered `UsesResponses()` branches and the duplicate endpoint builders once
migrated.

## Reasoning / streaming thinking (BACKLOG — but shape it into the seam now)

Requirement: stream thinking live, **persist it to the chat log on disk**, show it on session
reload, but **never send it back to the model**.

The seam already carries it: `Request.ReasoningEffort` in, `Handler.OnThinking` +
`Completion.Thinking` out. The "persist but don't replay" asymmetry lands in exactly one place —
the outbound lowering (`buildChatMsgs` today, the provider's `lower(history)→Request` in the seam)
reads content + tool_calls and simply never reads the stored reasoning field. That guarantee is
free *because* dispatch is centralised; it would have to be re-remembered per provider under the
old scattered scheme.

When implemented (later):

- **Storage:** migration adds `reasoning TEXT NOT NULL DEFAULT ''` to the `message` table
  (`internal/store/messages.go`); populated on assistant rows only; returned by `handleListMessages`
  for reload. Keep it a **separate column** (not folded into the tool_calls JSON) so the
  "never replayed" boundary stays obvious. Each assistant row in a tool loop carries its own.
- **Provider decode:** codex `response.reasoning_*` deltas → `OnThinking`; anthropic `thinking`
  content-block deltas → `OnThinking`; chat/ollama models that inline `<think>…</think>` are split
  in the decoder (replaces research's ad-hoc `stripThinking`).
- **Codex simplification:** since we do **not** replay reasoning, drop the
  `include:["reasoning.encrypted_content"]` currently in `BuildResponsesBody` — that request field
  only matters if you round-trip the encrypted blob (imp does; we won't). Keep only the
  human-readable summary for display.
- **Frontend:** new SSE event (e.g. `{"thinking":"…"}`) → collapsible thinking block; render the
  stored `reasoning` on reload.

Add `OnThinking`/`Thinking`/`ReasoningEffort` to the interface in the **first** refactor commit
even though nothing populates them yet — retrofitting after four providers exist means touching
every decoder.

## Backlog (tracked in `TODO.md`)

- Streaming thinking: persist reasoning to the chat log, show on reload, exclude from model input.
- Ollama native-API provider.
- Anthropic provider.

## Sequencing

1. **(this step)** Provider seam + factory; wrap current chat-completions as `openaichat`; recast
   Responses as `openaicodex`; migrate call sites. `OnThinking`/`Thinking`/`ReasoningEffort` present
   but unpopulated. **No behaviour change.** → commit, push, **user tests on local server** before
   proceeding.
2. Resume OpenAI OAuth: Phase 6 login UI/CLI (paste-the-code primary) on the clean base; live
   end-to-end verification of Codex chat + research.
3. Backlog items (reasoning, ollama, anthropic) as separate efforts.
