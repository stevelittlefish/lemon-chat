# Research system — work plan

Self-contained plan for reworking the research engine. Written against `main`
(the `research-algorithm-rework` branch is being discarded). When resuming, just
say "let's do the research changes" and work top-to-bottom.

**Guiding principle:** the real defects are fixable with a config split and two
prompt rewrites. Add code only where a prompt provably can't do the job, and
keep every model-facing contract **free-text** — the target is small local
models (Gemma 4B-class) that fumble JSON. Complexity is the enemy, not effort.

Do the changes in order. **Validate after Part 2 before doing anything optional.**

---

## Background — why we're doing this

A research run on an open-ended brief (home-battery cost/benefit) never
converged: it ran to the round cap re-issuing near-identical queries and emitted
a truncated report. Two root causes:

1. **Truncation misread as incompleteness.** `MaxReportTokens=8192` was shared by
   synthesis *and* the final report, so both truncated. The stop-check read the
   cut-off prose and said "not comprehensive → keep going," a runaway loop.
2. **Stop-check framed as "is the report comprehensive?"** — never satisfiable on
   a brief that partly can't be answered from the public web (live tariffs,
   installer quotes), so it never stopped on its own.

A previous rework fixed this with a ~3,400-line structured-ledger + JSON-contract
+ new-DB-tables architecture. It regressed on the target hardware (Gemma stopped
after one round because it couldn't satisfy the JSON stop contract). We're
discarding it. Most of its value was really a config value and two prompts.

---

## Part 1 — Diagnostic logging (do first; it makes the rest measurable)

**Decision: artifacts go to disk, not the DB.** This matches the existing
`<data_dir>/attachments/` precedent for bulky write-once content, keeps the
operational SQLite file lean (it also holds users/conversations/messages), and
avoids migrations, VACUUM churn, and contending on the main DB write lock from
the extraction goroutines. If cross-run benchmark queries ever genuinely need
SQL, add a small `research_event` table *later* for the tiny milestone events
only — never the bulky reports or raw LLM bodies. Don't build that speculatively.

### 1a. Run-log to disk (no engine changes)

The engine already hands the server everything needed via two existing closures:
`onProgress(Progress)` (every milestone — this is exactly the log we want) and
`onCheckpoint(State)` (full evolving state incl. `State.Report` after each round;
the DB checkpoint overwrites itself, so file snapshots are what preserve history).

Add a `runLog` helper (new file, e.g. `internal/server/research_runlog.go`) and
wire it by wrapping the two closures where the `Researcher` is constructed in
`internal/server/research.go`. No changes to `internal/research/`.

Directory layout, keyed by job id (mirror attachments):

```
<data_dir>/research/<job_id>/
  meta.json          # config + outcome — the "why did it stop" summary
  events.jsonl       # one line per milestone = the log we stream to the UI
  reports/
    round-00-plan.txt
    round-01.md      # State.Report snapshot after each stage
    round-NN.md
    final.md
  snapshots/
    round-NN.json    # full State per round (findings, queries, URLs, elapsed)
```

- [ ] `events.jsonl`: in the wrapped `onProgress`, append `{ts, phase, round,
      message}` as one JSON line **only when `p.Generated == 0`** (skips the
      250ms token-stream ticks; leaves exactly the milestone log). Guard the
      append with a `sync.Mutex` — `progress()` is called from the concurrent
      extraction goroutines.
- [ ] `snapshots/round-NN.json` + `reports/round-NN.md`: in the wrapped
      `onCheckpoint`, dump `State` as JSON and pull `State.Report` into a `.md`
      for easy reading. Write `reports/round-00-plan.txt` when the plan first
      appears; `reports/final.md` at the end.
- [ ] `meta.json`: write once at job start with the effective config (model,
      worker_model, effort, token limits, max_rounds, max_time, **git commit**,
      start time, locale). Update at end with `status`, `stop_reason`,
      `rounds_completed`, `elapsed_ms`, `findings_count`, `price_usd`. Derive
      `stop_reason` in the server from the last `deciding`/`warning` event teed
      through `onProgress` — no engine change needed.
- [ ] Delete the job's dir when the job row is deleted (one line in the delete
      handler).

### 1b. Bundle download

- [ ] `GET /api/research/{id}/bundle`: walk `<data_dir>/research/<id>/`,
      `archive/zip` it, stream as `research-<id>.zip`. Stdlib only (~30 lines).
- [ ] Add a "Download debug bundle" button to the job view in
      `static/js/research-app.js` (+ `static/js/api.js` wrapper). Add any new
      icon name to the `ICONS` array in `icons.js`.
- [ ] Log one line when a bundle is downloaded (non-hot-path handler convention).

### 1c. Raw model I/O capture — OPTIONAL, debug-gated

Tier 1 shows the *parsed* stop decision but not the prompt Gemma saw or its raw
reply — which is the smoking gun for "why did the model stop." This is the only
piece that must touch `internal/research`. Do it with **one** hook, not four.

- [ ] Add a single optional callback to the engine `Config`:
      `OnLLMExchange func(operation string, round int, prompt, response, finishReason string)`,
      invoked once after each completion in `llmCallOn`/`llmCallStream`.
- [ ] Server writes `llm.jsonl` (`{ts, operation, round, model, prompt, response,
      finish_reason}`) **only when a per-job capture flag or global `debug` is
      set** — off by default, same philosophy as `--token-log`. Never always-on.
- [ ] Include `llm.jsonl` in the bundle when present.

**Do NOT** add `research_event`/`research_llm_calls` tables, migrations, a
four-hook trace system, or disposition state machines. That was the old rework.

---

## Part 2 — Research algorithm fixes (the actual bug)

### 2a. Split the token budget (config — fixes the truncated report)

`main` shares `MaxReportTokens=8192` between synthesis and the final report, so
the report is hard-capped at 8k. This is the single biggest user-visible symptom.

- [ ] In `internal/config/config.go`, add `synthesis_tokens` (default ~8192) and
      `final_report_tokens` (default ~32768). Keep `max_report_tokens` as a
      backward-compat alias that maps to `synthesis_tokens` if the new key is
      unset. Document in `lemon.toml.example`.
- [ ] In `internal/research/researcher.go`, use `synthesis_tokens` for
      `synthesize()` and `final_report_tokens` for `finalReport()` (and section
      writes if deep-report is on) instead of the single `MaxReportTokens`.

### 2b. Bound the synthesis working memory (prompt only — stops truncation at source)

The evolving prose report grows every round and eventually truncates regardless
of budget. Fix it in the prompt, not with a structured ledger.

- [ ] In `internal/research/prompts.go`, add to `synthesizePrompt`: *"Keep this
      working summary under ~1,500 words. Compress or drop older background
      before dropping anything with a citation [S#]."* A report that stays
      ~1,500 words never approaches even the 8k ceiling.

### 2c. Rewrite the stop-check prompt (prompt + one wiring line — fixes non-convergence AND false NOs)

Keep it **free-text `YES/NO — reason`**. No JSON. This is the contract that
already works on small models and is exactly what the old rework broke.

- [ ] In `prompts.go`, rewrite `stopPrompt` to:
  - Explicitly say: **ignore prose polish, formatting, citation completeness, and
    length** (kills the "cuts off mid-conclusion → NO" misread).
  - Anchor completion on the plan we **already generate**: `state.Plan` already
    holds sub-questions + success criteria. Ask: *"Is any sub-question or success
    criterion still missing evidence?"* (This is the "deliverable checklist" idea
    for free — the data already exists; just feed `state.Plan` into the prompt.)
  - To justify continuing, require: a *specific* unanswered question that public
    web search could *plausibly* answer, AND a *materially different* search not
    already tried. State that private facts, future outcomes, unverifiable live
    data, and already-tried searches do **not** justify another round.
  - Keep output `YES/NO — one-sentence reason`.
- [ ] In `researcher.go`, pass `state.Plan` into the stop-check call.
- [ ] **Keep `MinRounds` as a floor.** The old rework deleted it and immediately
      got a 1-round premature stop. A minimum-rounds floor is a free guardrail.

---

## Part 3 — Cheap quality wins — OPTIONAL, only if a real run shows the need

Do NOT do these preemptively. Run the validation first.

- [ ] **Truncation detection, used minimally (~10 lines).** Plumb `finish_reason`
      through `internal/llm/llm.go`. Use it for one thing: if `synthesize()`
      comes back truncated, discard it and keep the previous round's summary
      instead of overwriting memory with a mangled version. No continuation-merge,
      no recovery loop.
- [ ] **Fuzzy query dedup (~15 lines).** `main` dedups queries by exact lowercase
      match, so reordered `site:X OR site:Y` slips through. Add a token-set
      equality check in query generation to catch near-identical thrash.
- [ ] **Per-domain source cap (~10 lines).** In `searchAll`, cap results per
      domain per round to stop SEO-page spam crowding out primary sources.

---

## Explicitly NOT doing (scope guards)

These were in the discarded rework or a follow-up list. They solve problems we
don't have yet or add complexity out of proportion to value:

- Structured evidence ledger with stable claim/gap/assumption/calc IDs.
- JSON stop-decision contract, gap/strategy extraction, novelty scoring.
- `research_event` / `research_llm_calls` DB tables + migrations, four-hook trace
  system, disposition state machines.
- Continuation-merge, budget-reserve math, auto-section promotion, deterministic
  arithmetic validation, source-quality metadata scoring.

If Part 2's prompt-bounded memory genuinely can't hold on real runs, *then*
consider lightweight structured memory — validated first, as a last resort.

---

## Validation (the step the old rework skipped)

After Parts 1 + 2, before any of Part 3:

- [ ] Run the home-battery brief (the original failing query) on **Gemma 4B-class
      AND a strong model**.
- [ ] Download the debug bundle for each and check two numbers: **rounds-to-stop**
      and **whether the final report is complete** (not truncated).
- [ ] Confirm from `events.jsonl` that it stops for a *sensible* reason (gaps
      genuinely unanswerable / plan satisfied), not the round cap.
- [ ] Only reach for Part 3 items that a real run proves necessary.

Target: ~90% of the old rework's value at a fraction of the complexity and none
of the small-model regression.
