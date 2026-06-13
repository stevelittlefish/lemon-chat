# lemon-chat — TODO

Status markers: `[ ]` not started · `[~]` in progress · `[x]` done

--

## Manually Added by User

- [x] When multiple messages are present from the same user, only show the avatar for the first one.

- [x] Add a default avatar for characters, models and users if they haven't uploaded one.  Use the CPU icon for models and the Drama icon for characters.  We need a silhouette of a human head for the user.

- [x] Research feature (port of docs/deep_research_spec.html to Go) — `/research` page linked from the main menu, iterative search/extract/synthesise engine in `internal/research/`, crash-resumable jobs via state checkpoints in the `research_job` table.

- [ ] **Rename `sdxl_file`/`flux_file` config keys to `sdxl_workflow`/`flux_workflow`** (`internal/config/config.go:28`)
  The current names suggest a generic file path; `_workflow` better conveys that these are ComfyUI workflow JSON files. Update the struct tags, references in `tools.go`, hint strings, and `lemon.toml.example`.

## Research

- [x] Long prompts mess up the layout.  Separate into title and prompt (both optional - one must be specified)

- [ ] The back button in the top left when viewing an individual piece of deep research should take you back to the list, not the menu

- [x] Brainstorming mode - alternate algorithm where we want the LLM to invent / design stuff and use web search when necessary, rather than basing the entire process around web search

- [ ] Look into ways to follow Reddit and facebook links, if possible

- [ ] Question: are there any more useful parameters to expose on the research UI?