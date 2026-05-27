# Lemon AI — Design System

> A chat client for AI that runs **on your own machine**. Quietly capable, simple to use, no telemetry, no feature creep. Open source.

This folder is the source of truth for everything that makes Lemon AI _look_ and _read_ like Lemon AI. Foundations, components, voice, and a couple of working UI kits live in here.

---

## What we're building

**Lemon AI** is an open-source desktop/web client for locally-hosted large language models. The product opinion is:

- **Local first.** Your conversations and models live on your machine. We never see them.
- **Few options, well chosen.** The settings pane fits on one screen. We pick reasonable defaults so you don't have to.
- **Quiet UI.** Paper, ink, and a single squeeze of yellow. The chat is the product — the chrome stays out of the way.
- **Warm, hand-crafted, indie OSS.** We sound like a person who built this on a Saturday, not a startup.

### Surfaces represented here

| Surface | Folder | What it is |
|---|---|---|
| Chat app | `ui_kits/chat/` | The main thing — sidebar of conversations, message thread, composer, model picker |
| Settings | `ui_kits/settings/` | Single-screen preferences. Models, appearance, privacy, advanced |

### Sources used to build this

Brand was designed from scratch — no codebase, Figma, or external reference was provided. Everything here is original work for this project. If you have an existing repo or Figma file, drop a link and we'll pull it in and reconcile.

---

## Index

```
README.md                  ← you are here
SKILL.md                   ← portable skill prompt (Claude Code compatible)
colors_and_type.css        ← single import — color & type tokens, semantic vars
components.css             ← shared component styles (buttons, inputs, etc.)
assets/
  logo-mark.svg            ← lemon + leaf, hand-drawn
  logo-wordmark.svg        ← mark + "lemon ai" lockup
  lemon-slice.svg          ← decorative, empty states
  scribble-underline.svg   ← hand-drawn underline (also via .scribble class)
  sparkle.svg              ← AI accent
preview/                   ← cards rendered in the Design System tab
ui_kits/
  chat/                    ← the chat app — index.html + JSX components
  settings/                ← preferences pane — index.html + JSX components
```

---

## Content fundamentals

We write like a friend who knows what they're doing and isn't trying to sell you anything.

**Voice**

- **Lowercase product name in body copy** (`lemon ai`), Title Case in headings ("Lemon AI").
- **First person plural — "we"** for the project, **"you"** for the reader. Never "users."
- **Short sentences. Plain words.** A 6th-grader should follow it.
- **No exclamation points** except in error toasts where something genuinely went wrong.
- **Contractions on.** "It's", "you'll", "we've".
- **No marketing words.** No "powerful," "seamless," "revolutionary," "best-in-class," "AI-powered."
- **No em dashes as decoration.** (Use them when they earn it.)
- **No emoji in product chrome.** Emoji is fine inside chat messages because users type it; we don't put it in buttons or labels.

**Tone examples**

| ✅ Lemon | ❌ Not Lemon |
|---|---|
| "Pick a model. We'll download it in the background." | "🚀 Seamlessly onboard your AI experience" |
| "This conversation lives on your computer. Nobody else can see it." | "Industry-leading privacy and security" |
| "Couldn't reach the model. Is the server running?" | "An unexpected error occurred. Please try again later." |
| "8 conversations" | "8 Active Threads" |
| "New chat" | "+ Create New Conversation" |
| "Settings" | "Preferences & Configuration" |

**Casing**

- **Sentence case** everywhere. Buttons: `New chat`, not `New Chat`. Section headers: `Privacy`, not `PRIVACY`.
- **Eyebrows/labels** are the one exception — small mono caps with letter-spacing, used as section markers.

**Numbers and units**

- `8 messages`, not `8 Messages`. Lowercase units. Space between number and unit (`4 GB`, not `4GB`).
- Truncate with "·" rather than dashes: `llama-3.1 · 8B · Q4`.

**Errors**

- Say what failed, then what to do. "Model didn't load. Check it finished downloading." Not "Error 0x80004005."

---

## Visual foundations

The whole system is **two surfaces and one accent**.

### Colors

- **Paper** (`--paper-50` / `#FBF7EC`) — the page. Warm off-white. Never pure white anywhere.
- **Ink** (`--ink-900` / `#1B1814`) — text and most strokes. Warm near-black, not pure `#000`.
- **Lemon** (`--lemon-500` / `#F5C518`) — the single accent. Used for the logo mark, the primary button, focus rings, and the occasional scribble underline. **Limit roughly one yellow element per visible viewport.**
- **Leaf** (`--leaf-500` / `#5C7A3E`) — tiny green dot for "running locally" / "connected." Almost never bigger than 8px.
- **Brick** (`--brick-500` / `#B3361B`) — errors, destructive. Muted, never neon.
- **Sky** (`--sky-500` / `#3565A8`) — inline link color, very rare.

Yellow is loud — that's the point. If the page reads as a yellow page, you've used too much.

### Type

- **Source Sans 3** for body, buttons, labels.
- **Source Serif 4** for headings 18px and up — warm, traditional, classy without being decorative.
- **JetBrains Mono** for code, model IDs, keyboard shortcuts, and eyebrow labels.

Letter-spacing is **negative on display** sizes (-0.01 → -0.018em) and zero on body. Line-height stays generous: 1.55 for body, 1.05–1.26 for headings.

### Spacing

4px grid. Tokens go `--space-1` (4) through `--space-16` (64). Pages breathe — favor more padding than less. Sidebar gutter, panel padding: **24px** is the default; **16px** is tight; **48px** is hero.

### Backgrounds

- **No full-bleed photography.** No stock imagery.
- **No bluish-purple gradients.** No glass / blur effects.
- **No repeating geometric patterns.** Decoration, when used, is a single hand-drawn SVG accent (a lemon-slice mark, a scribble underline).

### Animation

- **Easing**: `cubic-bezier(0.2, 0.7, 0.2, 1)` for almost everything. `cubic-bezier(0.16, 1, 0.3, 1)` for "things arriving."
- **Durations**: 120ms (micro), 200ms (default), 320ms (entrances).
- **No bounces.** No spring physics. Things settle, they don't oscillate.
- **No skeleton-shimmer.** Loading uses a soft, breathing "pulse" of opacity 0.5 → 0.85, or a typing-dot trio.
- **Reduced motion**: respect it. Animations become instant transitions, except for the AI typing indicator which is the one piece of motion that must remain.

### Interaction states

- **Hover**: surfaces lift one step on the tonal scale (`bg → bg-lifted`), or a 1px border darkens one step. Buttons darken slightly (`accent → accent-hover`). No transforms.
- **Press**: a slight inward shadow (`--shadow-inset`) and **scale 0.98**. No bigger.
- **Focus**: a soft yellow ring (`--ring-accent` — 3px, 30% lemon). For non-accent inputs, use `--ring-focus` (blue 25%). Never remove outlines.
- **Disabled**: opacity 0.45, no pointer events, never gray-on-gray-on-gray (keep ink color, drop opacity).

### Borders

- 1px is the rule. 2px only for the logo mark itself and form-input focus.
- Color: `--border` (`#D6CDB7`) for default, `--border-soft` (`#E7DFC9`) for hairlines, `--ink-900` for the rare "stamped" outline (logo, chips on press).

### Shadows

- Shadows are **warm** (slight brown cast), not gray. See `--shadow-sm/md/lg`.
- We layer paper-on-paper: a card sits on the page with `--shadow-sm`; a dialog gets `--shadow-lg`. No more than 3 elevation steps in a view.
- Inset shadow on pressed buttons and inset surfaces.

### Corner radii

- Buttons: **6–8px** (small/medium).
- Inputs: **8px**.
- Cards: **12px**.
- Dialogs / sheets: **18px**.
- Pills (status, model chips): **999px**.

Nothing is fully sharp. Nothing is overly rounded.

### Cards

A card is `--bg-lifted` (warm cream-white) on the page, with `--border` 1px, `--radius-lg` (12px), and `--shadow-sm`. Interactive cards lift to `--shadow-md` on hover.

### Transparency and blur

- We don't use backdrop-filter blur. Modals get a `rgba(27, 24, 20, 0.35)` warm-ink scrim with no blur.
- Transparency is reserved for the message-thread fade at the top (a "protection gradient" mask, 24px high, from paper to transparent so the scroll edge feels soft).

### Imagery vibe

If real imagery is added later: warm-toned, soft contrast, possibly a slight grain. Never cool or clinical. Black-and-white photography is OK. No stock-photo people.

---

## Iconography

- **Set**: [Lucide icons](https://lucide.dev), 1.8px stroke, rounded line caps and joins, 24×24 viewBox. Loaded via CDN (`unpkg.com/lucide@latest`) so we don't bundle the whole set.
- **Why Lucide?** It's open source, stroke-based, warm enough to sit alongside Funnel Sans, and broad enough to cover every chat/settings need.
- **Size**: 16px (inline, dense UI), 18px (buttons), 20px (sidebar nav), 24px (empty states). Always color: `currentColor`.
- **Brand icons** (logo mark, lemon slice, scribble) live as SVGs in `assets/` and are bespoke — they're hand-drawn-feeling with a heavier 2.4px stroke, deliberately different from Lucide so they read as illustrations, not UI icons.
- **Emoji**: we don't ship emoji in product chrome. Users type whatever they want into chat — that's their canvas.
- **Unicode glyphs**: we use `·` (middle dot) as a separator, `→` for "next," and `↵` for "send." That's it.
- **Substitution note**: Lucide is a stand-in for the eventual hand-drawn icon set we'd like to commission — flag this as TBD on a real launch.

### Brand assets in `assets/`

- `logo-mark.svg` — the lemon, standalone. Use at ≥ 24px.
- `logo-wordmark.svg` — lemon + "lemon ai" lockup. Use at ≥ 120px wide.
- `lemon-slice.svg` — decorative, for empty states and the "running locally" badge.
- `scribble-underline.svg` — hand-drawn underline for emphasis (also available via the `.scribble` CSS class).
- `sparkle.svg` — AI accent, used at the model picker.

---

## How to use this in a project

1. `<link rel="stylesheet" href="colors_and_type.css">` once at the top of your page.
2. Set `<body>` to use `body { font-family: var(--font-sans); background: var(--bg); color: var(--fg); }` (the CSS already does this on `html`).
3. Compose with semantic vars (`--fg`, `--bg-lifted`, `--accent`, `--border`), not raw palette tokens.
4. Pull headings from the type classes (`.h1`, `.h2`, `.eyebrow`, etc.).
5. For icons, drop `<script src="https://unpkg.com/lucide@latest"></script>` and use `<i data-lucide="message-square"></i>`.

For working examples, open `ui_kits/chat/index.html` and `ui_kits/settings/index.html`.

---

## Caveats & known TBDs

- **Source Serif 4 + Source Sans 3** is the long-term pairing. Free, widely available, and intentionally "normal" — they get out of the way.
- **Lucide icons are a stand-in** for an eventual bespoke set.
- **No dark mode yet** — the system was designed paper-first. Dark variant should invert ink/paper but keep the same lemon accent.
- **No motion in this static set.** The chat-typing dots and entrance fades are described but only lightly implemented in the kits.
