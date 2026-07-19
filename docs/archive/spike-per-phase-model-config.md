# SPIKE: per-phase model configuration for research

**Status:** recommendation — not yet implemented
**Queue item:** Research improvements #5
**Author:** investigation only; no behaviour change in this branch

## Question

Is it worth letting a single research job use different models for different
phases (e.g. a cheap local model for search/extract, a strong model for the
final report)? This spike surveys the phases, the plumbing cost, and the
value, and lands on a recommendation.

## How model selection works today

Every LLM call in the engine goes through `llmCall` / `llmCallStream`
(`internal/research/llm.go`), and both hardcode a single endpoint:

```go
llm.ChatCompleteWithUsage(ctx, r.client, r.cfg.APIBase+"/chat/completions",
    r.cfg.APIKey, r.cfg.Model, msgs, …)
```

`r.cfg.Model`, `r.cfg.APIBase`, and `r.cfg.APIKey` are set once when the job
launches (`internal/server/research.go`), resolved from
`ServerForModel(job.Model)`. So there is exactly one model per job, and the
model is inseparable from its server's base URL + key.

**Key gotcha:** a per-phase model is *not* just a model-name swap. A different
model may live on a different `[[model_server]]`, so each alternate model needs
its own `(Model, APIBase, APIKey)` triple resolved up front. Any implementation
must thread an *endpoint*, not a string.

Note: the HTML-report step (queue item #4) already has its own optional model,
but it lives in the server layer (`generateReportHTML`), not inside the engine,
so it does not touch this plumbing.

## The phases and their LLM calls

| Phase | Call | max_tokens | Frequency | Quality sensitivity |
|---|---|---|---|---|
| Slug | `llmCall` | 64 | once/job | none |
| Classify | `llmCall` | 20 | once/job | low |
| Plan | `llmCall` | 1024 | once/job | medium |
| Query generation | `llmCall` | 4096 | once/round | medium |
| **Extraction** | `llmCall` | 2048 | **per URL, per round** | medium |
| Synthesis | `llmCallStream` | `MaxReportTokens` (8192) | once/round | high |
| Decide / stop | `llmCall` | 128 | once/round | low |
| Final report | `llmCallStream` | `MaxReportTokens` | once/job | **highest** |
| Deep-report (outline/refine/section/glue) | both | 1024–`MaxReportTokens` | several/job | high |

Brainstorm mode mirrors these with the `brainstorm*` prompts; the shape is the
same.

Two facts dominate the analysis:

1. **Extraction is the volume hotspot.** `limit := MaxURLsPerRound * len(queries)`
   (default 3 × several queries) llmCalls *per round*, over up to 8 rounds — so
   extraction is easily 80–90% of all LLM calls in a job. It is also
   embarrassingly parallel and only needs "pull the goal-relevant facts out of
   this page", which a small model does acceptably.

2. **Synthesis and the final/deep report define output quality.** These are the
   token-heavy, reasoning-heavy calls, and there are only a handful of them per
   job.

The mechanical calls (slug, classify, decide, query-gen) are cheap *and* few, so
they barely move cost either way — but they're also zero-risk to downgrade.

## Natural tiers

Rather than a per-phase matrix, the phases cluster into two tiers that map
cleanly onto "what kind of model do I want here":

- **Worker tier** — extraction + the mechanical calls (slug, classify,
  query-gen, decide). High volume or trivial output; a fast/cheap/local model is
  a good fit. This is where nearly all the cost and latency live.
- **Writer tier** — plan, synthesis, final report, deep-report pipeline. Few
  calls, but they determine the quality of the artifact. Keep these on the
  strong job model.

## Options

### Option 0 — do nothing
Leave one model per job (plus the #4 HTML override). Zero cost, zero benefit.

### Option 1 — two-tier "worker model" (recommended)
Add **one** optional `worker_model` alongside the job model. When set, the
worker tier (extraction + mechanical phases) uses it; the writer tier stays on
the job model. Mirrors the #4 pattern exactly:

- Config default: `[research] worker_model = ""` (falls back to the job model).
- Start form: an optional "worker model" dropdown, default "same as research
  model" — revealed under the advanced/customisation controls.
- Persist `worker_model` on the job so a resumed job still honours it.

**Engine change (small):** introduce a `modelEndpoint{Model, APIBase, APIKey}`
and give the `Researcher` two of them (`writer`, `worker`, worker defaulting to
writer). Have `llmCall`/`llmCallStream` take an endpoint argument (or add
`llmCallWorker` thin wrappers), and update the ~5 worker-tier call sites. The
server resolves both endpoints via `ServerForModel` at launch and validates
them (as `handleStartResearch` already does for the model and HTML model).

**Captures** the bulk of the achievable cost/latency win (extraction volume)
while keeping the quality-defining calls on the strong model. Config surface is
one field. Leaves the door open to Option 2 if ever wanted.

### Option 2 — full per-phase map
A `map[phase]model` in config + UI. Maximum flexibility, but:
- Config/UI complexity balloons (9+ phases × mode variants), and most phases are
  not worth tuning independently.
- Same underlying `modelEndpoint` plumbing as Option 1, but many more call sites
  and a much larger validation/persistence surface.
- Hard to explain to a user what each phase does or why they'd change it.

The marginal benefit over Option 1 is small: once extraction and the mechanical
calls are on the worker model and the report is on the writer model, there's
little left to independently tune.

## Recommendation

**Do Option 1 (two-tier worker/writer split).** It captures the concentrated
win — extraction is the overwhelming majority of calls and the natural place for
a cheap/local model — at a config and UI cost that mirrors the #4 HTML-model
override we just shipped, and with a small, reusable engine refactor
(`modelEndpoint` + endpoint-parameterised `llmCall`). Skip the full per-phase
matrix (Option 2): it multiplies complexity for benefit that the two-tier split
already captures.

Suggested follow-up work items if we proceed:
1. Add `modelEndpoint` and thread `writer`/`worker` endpoints through the engine;
   default worker → writer.
2. Add `[research] worker_model` config + `research_job.worker_model` column
   (numbered migration) + persistence, mirroring #4.
3. Add the optional "worker model" control to the start form (progressive
   disclosure — see queue item #7) and validate it server-side.
4. Surface the worker model in the job's explore/detail view where the model is
   already shown.
