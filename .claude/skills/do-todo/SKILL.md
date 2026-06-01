---
name: do-todo
description: Present the first three uncompleted items from TODO.md for the user to choose, then implement the chosen one and mark it done. Handles the full cycle: mark in-progress, implement, test if needed, mark complete.
user-invocable: true
---

# do-todo skill

Work through one TODO item from start to finish. Follow these steps exactly, in order.

## Step 1 — Present choices and wait for selection

Read `TODO.md`. Find the first **three** items with status `[ ]` (not started), in the order they appear in the file.

Present them as a numbered list — one line each — and ask the user to pick one. Include the section name (e.g. "Critical bugs", "High priority") as a label so the user has priority context. Example format:

```
1. [Critical bugs] Short title (`path/to/file.go:line`)
2. [High priority] Short title (`path/to/file.go:line`)
3. [Medium priority] Short title (`path/to/file.go:line`)
```

Wait for the user to reply with their choice (a number, or enough of the title to identify it) before doing anything else. Do not proceed until the user has chosen.

## Step 2 — Mark it in progress

Change its marker from `[ ]` to `[~]` in `TODO.md` now, before writing any code. This makes the current work visible if the session is interrupted.

## Step 3 — Understand the item

Read every file referenced in the TODO entry. Understand the bug or task fully before touching any code.

## Step 4 — Decide whether a test is needed

A test is warranted when:
- The item is a bug that can be exercised via an existing test harness (Go unit test or table-driven test)
- The fix has a clear pass/fail condition that a test can encode
- Adding the test won't require more scaffolding than the fix itself

A test is **not** warranted for:
- Dead-code removal (just delete the code)
- Cosmetic / naming changes
- JS-only changes with no test harness
- Items where the correct behaviour is already covered by existing tests

State your decision and reason in one sentence before proceeding.

## Step 5 — Write the failing test (if applicable)

If a test is warranted, write it now — before changing any production code. Then run it and confirm it fails. Paste the failure output as a short code block so there is a clear before record.

Use the existing test conventions in the repo:
- Go tests live alongside their packages (`*_test.go`)
- Run with `go test ./...` or a targeted `go test ./internal/...`

## Step 6 — Implement the fix

Make the minimal change that resolves the TODO item. Do not refactor surrounding code, add unrelated improvements, or expand scope beyond what the item describes. Follow all constraints in `CLAUDE.md`.

## Step 7 — Confirm the test passes (if applicable)

Re-run the test. Paste the passing output. If the test still fails, debug and fix before moving on.

If no test was written, verify the fix another way — build the project (`go build ./...`) at minimum, plus any manual verification that makes sense for the change.

## Step 8 — Mark it done

Change the marker from `[~]` to `[x]` in `TODO.md`.

## Step 9 — Report

Two to four sentences summarising: what was wrong, what you changed, and (if a test was written) that it now passes. No padding.

## Step 10 — Verification steps

List the exact steps the user should take to verify the change works and exercise the affected code paths. Be specific: name the UI action, URL, or command to run, and describe what correct behaviour looks like. Group by change if multiple items were completed. If a change is only verifiable by code inspection (e.g. dead code removal, a log line that only fires on corrupted data), say so explicitly rather than omitting the step.

---

## If the user invokes this skill with no arguments

Present the three choices as described in Step 1 and wait. Do not start implementing until the user has selected one.

## If the user names an item in the invocation (e.g. `/do-todo the scanner buffer one`)

Match it against the `[ ]` items in `TODO.md` and proceed directly to Step 2 with that item, skipping the choice prompt.
