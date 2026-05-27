---
name: lemon-ai-design
description: Use this skill to generate well-branded interfaces and assets for Lemon AI — an open-source, locally-hosted AI chat tool with a warm, paper-and-ink, indie-OSS aesthetic. Use for production work or throwaway prototypes, mocks, slides, and marketing pages. Contains color & type tokens, brand voice rules, logo + icon assets, and full UI kits for the chat app and settings pane.
user-invocable: true
---

# Lemon AI Design Skill

Read `README.md` in this skill first — it contains the brand context, voice rules, color/type system, and visual foundations. Then explore the rest of the files.

## What's in here

- `README.md` — brand context, content fundamentals, visual foundations, iconography
- `colors_and_type.css` — single CSS file with tokens and semantic vars. Import this everywhere.
- `assets/` — logos (`logo-mark.svg`, `logo-wordmark.svg`), decorative SVGs (`lemon-slice.svg`, `scribble-underline.svg`, `sparkle.svg`), paper texture
- `preview/` — small specimen cards for each part of the system
- `ui_kits/chat/` — full chat app recreation, modular JSX components + `index.html`
- `ui_kits/settings/` — preferences pane, modular JSX components + `index.html`

## How to use this skill

**For visual artifacts** (slides, mocks, throwaway prototypes, landing pages): copy `colors_and_type.css` and the relevant `assets/` into your output, and produce a single self-contained HTML file. Reuse JSX components from the UI kits where possible; they're written as inline-React-friendly snippets.

**For production code**: copy the tokens out of `colors_and_type.css`, follow the voice rules in `README.md`, and use the UI kit components as visual references (they intentionally cut corners on real logic).

## Non-negotiables

- Paper backgrounds, never pure white.
- Warm near-black ink (`#1B1814`), never pure `#000`.
- **One lemon yellow element per viewport, max.** If the page reads as yellow, you've overused it.
- Sentence case everywhere except eyebrow labels.
- No emoji in chrome. No exclamation points outside genuine errors. No marketing words.
- Lucide icons (1.8px stroke) for UI; bespoke hand-drawn SVGs for brand decoration.
- No bluish-purple gradients, no glass/backdrop-blur, no stock imagery, no left-border-accent cards.

## If the user just invokes this skill

Ask them what they want to build — landing page, slide, internal screen, marketing asset — and a couple of clarifying questions (audience, length, surface). Then act as an expert designer producing HTML or production code, depending on the need.
