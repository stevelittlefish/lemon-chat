# lemon-chat — TODO

Status markers: `[ ]` not started · `[~]` in progress · `[x]` done

--

## Manually Added by User

[x] When multiple messages are present from the same user, only show the avatar for the first one.

[x] Add a default avatar for characters, models and users if they haven't uploaded one.  Use the CPU icon for models and the Drama icon for characters.  We need a silhouette of a human head for the user.

[x] Research feature (port of docs/deep_research_spec.html to Go) — `/research` page linked from the main menu, iterative search/extract/synthesise engine in `internal/research/`, crash-resumable jobs via state checkpoints in the `research_job` table.

- [ ] **Rename `sdxl_file`/`flux_file` config keys to `sdxl_workflow`/`flux_workflow`** (`internal/config/config.go:28`)
  The current names suggest a generic file path; `_workflow` better conveys that these are ComfyUI workflow JSON files. Update the struct tags, references in `tools.go`, hint strings, and `lemon.toml.example`.
