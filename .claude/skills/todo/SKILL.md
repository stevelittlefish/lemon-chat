---
name: todo
description: Add a new item to TODO.md. The user describes the task or bug; you locate the relevant code, write a precise entry with a file path and line number, and insert it into the right priority section.
user-invocable: true
---

# todo skill

Add one item to `TODO.md`. Follow these steps in order.

## Step 1 — Understand the request

The user will describe a bug, task, or improvement in natural language. If the description is ambiguous or you are missing information needed to locate the relevant code, ask one clarifying question before proceeding. Do not ask multiple questions; ask the most important one.

## Step 2 — Locate the relevant code

Find the file(s) and line number(s) the item refers to. Use grep, find, or read relevant files as needed. Every TODO entry must include a concrete code location (`path/to/file.go:line` or a line range) — a vague entry with no location is not useful.

If the item is purely organisational (e.g. "add a section to the README") and has no single code location, omit the location.

## Step 3 — Write the entry

Format the entry as a single bullet matching the existing style in `TODO.md`:

```
- [ ] **Short imperative title** (`path/to/file.go:line`)
  One or two sentences explaining what is wrong or what needs to change, and why it matters. Be specific — describe the observable problem or the risk, not just "fix X".
```

- Title: sentence case, imperative verb, ≤ 10 words
- Location: in backticks, immediately after the title, in parentheses
- Body: optional second line with detail; only add it if the fix is non-obvious or the risk needs explanation

## Step 4 — Choose the right section and position

`TODO.md` has these sections in priority order:
- **Critical bugs** — security issues, data loss, silent corruption, broken invariants
- **High priority** — reliability issues, race conditions, resource leaks, missing error handling that can cause visible failures
- **Medium priority** — correctness gaps, missing validation, dead code that is misleading, performance issues that affect normal use
- **Low priority / polish** — style inconsistencies, minor inefficiencies, UX papercuts, developer-experience improvements

Pick the section that matches the severity. Within a section, add the new item at the **end** of that section, after all existing items.

## Step 5 — Insert the item

Edit `TODO.md` to add the item in the chosen location. Do not reorder, reformat, or touch any other part of the file.

## Step 6 — Confirm

One sentence: name the section you added it to and quote the title you used. Nothing else.

---

## If the user invokes this skill with no arguments

Ask them to describe the item they want to add. One sentence prompt, no preamble.
