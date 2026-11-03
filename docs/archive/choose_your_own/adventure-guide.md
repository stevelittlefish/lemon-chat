# Building text adventures for lemon-chat

This guide explains how to build a choose-your-own-adventure / RPG experience using lemon-chat's notes and world state tools. The system is designed so that:

- You write the lore and rules as notes before play begins.
- A game master character reads those notes and runs the adventure.
- Objectives and dice rolls enforce real pacing — the GM cannot skip or shortcut the story.

---

## How it works

```
You (author)           lemon-chat                       Player
──────────────         ─────────────────────────        ──────────
Write lore notes  →    g.eldoria (manifest)
                        g.eldoria.rules
                        g.eldoria.world                 Starts conversation
                        g.eldoria.bestiary
                                                   ←    "I want to play Eldoria"
                        GM loads manifest
                        GM loads lore notes
                        GM inits world state
                                                   ←    Player makes choices
                        GM rolls dice
                        GM updates world state
                        GM narrates next scene      →
```

The GM character is a standard lemon-chat character with a carefully written system prompt and access to the `notes` and `world_state` tool groups, plus `roll_dice` and `random_chance`.

---

## Note naming conventions

Notes use a scope prefix followed by the adventure name and an optional sub-path:

```
g.eldoria              ← manifest (top-level, no sub-path after adventure name)
g.eldoria.rules        ← game mechanics the GM must follow
g.eldoria.world        ← geography, factions, general lore
g.eldoria.bestiary     ← monsters and encounters
g.eldoria.act2         ← lore for a later act (loaded on demand)
g.eldoria.secrets      ← GM-only information; never revealed to the player
```

**Scope:**
- `g.` — global, visible to all users and conversations. Use this for all adventure lore. It persists until you delete it.
- `u.` — user-scoped, visible only to the logged-in user across all conversations.
- `c.` — conversation-scoped, deleted when the conversation ends. Use this for notes the GM writes *during* play (session journal, discovered clues).

**Mark all lore notes read-only** when you create them (via the Notes settings page). The GM cannot modify or delete read-only notes, so your lore is safe from accidental overwrite.

**Bare-term search:** `note_list("eldoria")` finds `g.eldoria*`, `u.eldoria*`, and `c.eldoria*` in one call. The system prompt instructs the GM to use bare terms so it searches all scopes. You should never instruct the GM to include the `g.` prefix in search calls.

---

## The manifest note

The manifest (`g.<adventure>`) is the entry point for an adventure. The GM loads it first, then follows its instructions. It contains five sections.

### Format

```
ADVENTURE: The Secrets of Eldoria
TAGLINE: An ancient forest holds a curse only you can break.

LOAD IMMEDIATELY:
- g.eldoria.rules
- g.eldoria.world

LOAD ON DEMAND:
- g.eldoria.bestiary  — load when the player first enters combat
- g.eldoria.act2      — load when world state act = "2"
- g.eldoria.secrets   — load immediately (GM eyes only; never quote to player)

WORLD STATE INIT:
act           = "1"
player_health = "3"
pendant_fire  = "not_found"
pendant_water = "not_found"
pendant_storm = "not_found"
gate_opened   = "false"

OBJECTIVES:
1. pendant_fire   — Find the Fire Pendant. Gate: defeat the Fire Guardian. Location: The Ember Caves.
2. pendant_water  — Find the Water Pendant. Gate: navigate the Flooded Vault. Location: The Drowned Keep.
3. pendant_storm  — Find the Storm Pendant. Gate: solve the Sky Temple puzzle. Location: The Sky Temple.
4. gate_opened    — Open the Grand Gate. Gate: all three pendants found. Location: The Grand Gate.

FAILURE: player_health reaches 0 → "The darkness takes you. Eldoria remains cursed."
VICTORY: gate_opened = "true" → read the victory text from g.eldoria.victory

OPENING NARRATION:
The village of Millhaven has been quiet for three days — too quiet. The elder meets you at the
gate, her face drawn. "The Whispering Forest woke last night," she says. "We need someone
brave enough to find out why. Are you that person?"
```

### OBJECTIVES detail

Each objective line has three parts:
- **Key** — the world state key that tracks completion (e.g. `pendant_fire`)
- **Label** — a human-readable description the GM can use in narration
- **Gate** — the challenge the player must face (and the dice must favour) before the objective can be marked complete
- **Location** — where in the world this objective is found (helps the GM construct scenes)

Objectives are ordered. The GM must complete them in sequence unless the manifest explicitly says they can be tackled in any order.

---

## Lore note types

### `.rules` — the mechanics the GM must follow

This is the most important note. It defines how dice work, what success means, and what the GM is and is not allowed to do. The system prompt defers to this note for anything mechanical.

```
DICE RULES:
- Roll 2d6 for all challenges.
- 10+  : Full success. The player achieves their goal cleanly.
- 7–9  : Partial success. The player succeeds but at a cost (1 health, lose an item, raise an alarm).
- 6–   : Failure. The player does not achieve the goal. Costs 1 health.

HEALTH:
- Player starts with 3 health.
- At 0 health: narrate the failure ending and stop. No more choices.
- Healing is only available at: the village inn (once per adventure), a healing herb (found in the forest, one-time).

PACING:
- Each response is one scene. One scene = one choice resolved.
- Never combine two encounters into a single response.
- Each response ends with exactly 3 choices.
- Never narrate the player taking an action they did not choose.

SECRETS:
- Notes ending in .secrets contain information the GM knows but the player must discover.
- Never quote or paraphrase .secrets content directly. Use it only to inform what the GM hints at.
```

### `.world` — geography and general lore

Describe the world in sections the GM can reference quickly. Use headings so the GM can load the whole note and skim to the relevant part.

```
## The Village of Millhaven
A small farming community at the forest edge. Pop. ~200. Notable NPCs: Elder Wren (wise, worried),
Bram the blacksmith (gruff but helpful), young Pip (knows a secret path into the forest).

## The Whispering Forest
Ancient, semi-sentient. Trees creak in patterns that sound like speech.
Three main paths: the Ember Trail (south-east, leads to caves), the Wet Road (north, floods in rain),
the High Road (east, steep, leads to Sky Temple).
...
```

### `.bestiary` — monsters and encounters

Load this on demand (when combat starts) to keep context lean.

```
## Fire Guardian
A spirit of living flame that protects the Fire Pendant.
DIFFICULTY: Hard — player needs 10+ on 2d6 to defeat.
ON PARTIAL (7–9): The guardian is weakened but scorches the player (1 health). Pendant obtainable.
ON FAILURE (6-): The guardian drives the player back. Pendant not obtainable this attempt. Player loses 1 health.
DESCRIPTION: Twelve feet tall, roars in a language of crackling embers.
```

### `.secrets` — GM-only lore

Information that should inform the narrative but never be given to the player directly:

```
The curse on Eldoria was placed by Elder Wren herself, fifty years ago, to protect the village
from a worse fate — an invading army that never came. She does not know how to lift it and is
too ashamed to admit she caused it. If the player confronts her with all three pendants, she
will break down and confess.
```

The system prompt instructs the GM: *"notes ending in `.secrets` are for your use as narrator only — do not quote, summarise, or directly reveal their contents to the player."*

---

## World state design

World state is conversation-scoped (it lives only in the current conversation) and is best used for short, enumerable values. The manifest defines the initial values; the GM sets them at adventure start and updates them as play progresses.

### Good candidates for world state

| Key | Example values | Purpose |
|---|---|---|
| `act` | `"1"`, `"2"`, `"3"` | Controls which lore loads on demand |
| `player_health` | `"3"`, `"2"`, `"1"`, `"0"` | Tracks vitality; 0 = game over |
| `<objective_key>` | `"not_found"`, `"found"` | Objective completion tracking |
| `inventory` | `"torch,rope,key"` | Comma-separated item list |
| `scene_flag` | `"false"`, `"true"` | One-time event flags |

### Things that don't belong in world state

- Long prose descriptions (use `c.` notes for those)
- Derived values (if health < 1 = dead, just check health)
- Things already in the lore notes

### Gate checking

The GM checks world state before every scene. A "gate" is a prerequisite that must be met before an objective can be resolved. The manifest defines gates; the `.rules` note defines how checks work.

Example: the manifest says objective 4 (`gate_opened`) requires all three pendants. The GM must verify `pendant_fire`, `pendant_water`, and `pendant_storm` are all `"found"` before allowing the gate to open — regardless of what the player tries to do.

---

## The GM character

Create a character in lemon-chat with:

- **Tools:** `world_state`, `notes`, `roll_dice`, `random_chance`
- **No** `note_save`, `note_delete`, or `note_append` in the expanded tool list — the GM should only read lore, not write it. (Give it the `notes` group only if you want it to keep a session journal in `c.` scope; otherwise restrict to `note_load` and `note_list`.)

> **Note:** The `notes` group expands to all five note tools. If you want the GM to be read-only, do not use the `notes` group shorthand — instead list `note_load` and `note_list` explicitly in the character's tools array once that feature is supported, or accept that it has write access and rely on lore notes being marked read-only.

---

## System prompt template

Copy this and adapt it for your adventure setting. Replace `[bracketed]` sections.

```
You are [The Chronicler / a name appropriate to the setting], a game master running a
text-based adventure for one player. Your job is to narrate, track state, and create a
fair, engaging experience with real stakes and real possibility of failure.

## Session start

When a new conversation begins:
1. Call note_list with no prefix to see available notes.
2. Identify adventure manifests: notes whose key matches g.<name> with no further dots
   (e.g. g.eldoria, not g.eldoria.world). These are the playable adventures.
3. Present the adventure list to the player. Ask which they want to play.
4. When the player chooses, call note_load on the manifest key (e.g. note_load g.eldoria).
5. Follow the manifest's LOAD IMMEDIATELY list — call note_load for each note listed.
6. Follow the manifest's LOAD ON DEMAND list — remember these for later, do not load yet.
7. Execute the manifest's WORLD STATE INIT — call state_set for each key/value pair.
8. Read the OPENING NARRATION from the manifest.
9. End your first message with exactly 3 choices.

## Every turn

At the start of every turn:
1. Call state_list to read the current game state.
2. Write the scene narration: 3–5 paragraphs. Show, don't tell.
3. Before resolving any encounter or objective attempt: roll dice. Do not pre-determine
   the result. Let the dice decide success or partial success or failure.
4. Apply the outcome as defined in g.<adventure>.rules.
5. Update world state to reflect what happened (objectives found, health lost, flags set).
6. Check whether any LOAD ON DEMAND notes should now be loaded (e.g. act changed).
7. End your response with exactly 3 numbered choices. No exceptions.

## Objectives and gates

The manifest lists OBJECTIVES in order. Before any scene that could resolve an objective:
1. Check state_list to confirm all prerequisite objectives are complete.
2. If prerequisites are not met, the objective cannot be reached yet — redirect the scene.
3. If prerequisites are met, run the encounter. Roll dice. Apply the outcome from the rules note.
4. On success or partial success: set the objective key to "found" in world state.
5. On failure: do not set the objective. Apply health cost. Narrate the setback.
6. Never mark an objective complete without a dice roll.
7. Never skip an objective because the player asks to.

## Health and failure

- Health starts at the value from WORLD STATE INIT (check state_list at turn start).
- Each failure on a dice roll costs 1 health: state_modify player_health delta=-1.
- When health reaches 0: narrate the FAILURE ending from the manifest. Stop. No more choices.
- Never restore health except at lore-defined sources (check g.<adventure>.rules).
- Never invent healing that isn't in the lore.

## Hard rules — these override everything else

1. Every encounter resolves via dice. No exceptions.
2. Objectives cannot be completed without meeting their gate condition.
3. Every response ends with exactly 3 numbered choices.
4. Never narrate an action the player did not choose.
5. Never combine two scenes into one response.
6. Never write a victory or failure ending while objectives remain incomplete.
7. Notes ending in .secrets are for your use as narrator only. Never quote, paraphrase,
   or directly reveal their contents. Use them only to inform your hints and foreshadowing.
8. When the player tries to do something outside the choices, interpret it as the closest
   available choice or ask them to pick from the three.
```

---

## Writing tips

### Objective design

- 3–5 objectives is the right range for a single session. Fewer feels thin; more is exhausting.
- Each objective should be in a distinct location with a distinct feel. Variety keeps the adventure interesting.
- Make some objectives harder than others. If the Fire Guardian requires 10+ but a puzzle only needs 7+, skilled players can route around difficulty.
- Give each location one interesting NPC or detail the player can interact with before the gate encounter. It builds attachment.

### Dice gate design

- Hard gates (10+ on 2d6) should be rare — reserved for the climax. Statistically a player rolls 10+ about 1 in 6 times on 2d6.
- Most objectives should be gated at 7+ (moderate). That's roughly 50/50 with a partial-success option.
- Partial success is your best tool: the player gets the objective but at a cost. It feels earned rather than free, and it erodes health in a way that creates tension without killing momentum.
- Always define what happens on partial and on failure. Don't leave it to the GM to improvise — that's how the story gets too easy.

### Lore notes

- Write lore notes like a DM guide, not a novel. Short paragraphs, clear headings, scannable.
- Include things the GM can drop into narration as sensory details: smells, sounds, textures.
- Put secrets where they belong: in `.secrets` notes. Don't hide secret information in the middle of `.world` — the GM might accidentally surface it.

### Health design

- 3 health is tight enough to create tension without being brutal.
- 5 health is more forgiving and better for younger players or longer adventures.
- Consider the number of dice-gated encounters: with 4 objectives and 3 health, any two failures = game over. Test the math.

---

## Checklist for a new adventure

```
[ ] Decide the setting and theme
[ ] Outline 3–5 objectives in sequence, each in a distinct location
[ ] Write the difficulty and outcome for each objective's gate encounter
[ ] Create the notes:
    [ ] g.<name>              manifest (all sections complete)
    [ ] g.<name>.rules        dice rules, health rules, pacing rules
    [ ] g.<name>.world        geography, key NPCs, atmosphere
    [ ] g.<name>.bestiary     monster/encounter cards (if combat-heavy)
    [ ] g.<name>.secrets      GM-only twists and reveals
    [ ] g.<name>.victory      victory narration (optional, linked from manifest)
[ ] Mark all notes read-only in the Notes settings page
[ ] Create or update the GM character with the system prompt
[ ] Test: play the first two scenes and check the GM follows the rules
```
