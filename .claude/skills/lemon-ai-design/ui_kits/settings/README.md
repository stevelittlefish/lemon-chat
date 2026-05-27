# Settings UI Kit

A single-screen preferences pane. Models, Appearance, Privacy, Advanced, About. The whole point of Lemon AI is that this screen should fit on a laptop without scrolling much — every option earns its slot.

## Files

- `index.html` — entry point
- `App.jsx` — root, holds nav state + setting values
- `SettingsNav.jsx` — left rail of sections
- `Section.jsx` — wraps a panel with title + group of `Field` rows
- `Field.jsx` — label / description / control row
- `controls/` — `Toggle.jsx`, `Select.jsx`, `Slider.jsx`, `Segmented.jsx`, `Stepper.jsx`
- `panels/` — `ModelsPanel.jsx`, `AppearancePanel.jsx`, `PrivacyPanel.jsx`, `AdvancedPanel.jsx`, `AboutPanel.jsx`
- `Icon.jsx` — copied from the chat kit

## What's interactive

- Click a section in the left nav → switches the right panel
- Flip toggles → state persists per-session
- Change theme → "Match system" / "Light" radio updates visually
- Change temperature → slider updates the live value display
- Each model in the list has Set as default / Remove actions

## What's faked

- Nothing actually saves
- "Download model" doesn't run a download
- The system-prompt textarea accepts text but it's not wired to anything
