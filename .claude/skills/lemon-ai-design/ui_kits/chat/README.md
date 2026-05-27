# Chat UI Kit

The main Lemon AI surface: a left rail of conversations, a message thread in the middle, a composer at the bottom, and a model picker tucked into the header.

This is a **visual + interaction recreation** — fake state, fake replies, no actual model running. It exists so designers can pull pieces out and recompose them in real product work or in mocks.

## Files

- `index.html` — entry point, opens a working chat
- `App.jsx` — root; holds conversations + active chat state
- `Sidebar.jsx` — left rail (logo, new chat button, conversation list, settings link)
- `ChatHeader.jsx` — top bar with title + model picker
- `Thread.jsx` — message list + empty state
- `Message.jsx` — single user/assistant bubble
- `Composer.jsx` — input area + send + attach
- `ModelPicker.jsx` — dropdown of locally-installed models
- `Icon.jsx` — tiny inline Lucide icons

## What's interactive

- Click any conversation in the sidebar → loads its thread
- Click **New chat** → creates an empty conversation, focuses the composer
- Type & press Enter → adds a user message, simulates a streamed AI reply
- Click the model name in the header → opens the model picker
- Hover a conversation row → reveals the kebab; click → menu (rename, delete)

## What's faked

- Replies are canned per-conversation; "streaming" is just a setTimeout drip
- No real persistence (refresh resets)
- "Download model" doesn't actually download anything
