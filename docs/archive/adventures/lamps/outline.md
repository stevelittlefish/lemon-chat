# The Lamplight League — adventure outline

A Choose Your Own (CYO) adventure for lemon-chat. Theme: **lamps**. Three acts that
deliberately change genre underfoot — it opens as a Pokémon parody and ends as a tender
sci-fi homecoming.

- **Namespace:** `g.cyo.lamps`
- **Pack id:** `cyo-lamps` → `note-packs/lamps.json`
- **Display name:** The Lamplight League
- **Tagline:** Raise your lamp. Light the road. Some journeys go further than you planned.
- **Tone:** Warm, whimsical, a little wondrous. Real stakes (health can hit 0) but never grim.

The tagline and act-1 framing should *not* telegraph the space twist. Foreshadow it only
through Flicker (the companion lamp keeps looking up at the night sky).

---

## The companion: Flicker

The spine that ties all three acts together. In act 1 the player is given a starter
**Lumen** (lamp-creature): a small, dented **brass oil lamp** creature, species *Brasswick*,
default name **Flicker** (the player may rename it; the GM tracks `companion_name`).

Flicker is warm and loyal but carries a quiet longing — it stares up at the stars at night
"as if looking for someone." This is the emotional hook that pays off in act 3, when the
player discovers Flicker came from the lamp planet and reunites it with its family.

**Mechanic — the bond lights the way.** A `flicker_bond` counter (0→3) rises by one each
time the player completes an act-1 lantern quest. In act 2 (a blacked-out building) the bond
is what lets Flicker glow brightly enough to grant **advantage** on rolls in the dark. The
warmth you build in act 1 literally lights your path through act 2. Thematic + mechanical.

---

## Act 1 — The Lamplight League (Pokémon parody)

**Setting:** the sunny region of **Lumalands**. People raise and train **Lumens** —
creatures that are all variations on lamps. The player is a brand-new trainer leaving home
town **Wickville**. **Professor Bulb** gives them Flicker and points them toward the
**Grand Lumen Gym** in **Lux City**, run by radiance-type Gym Leader **Lumina**.

**The gate to the gym:** the road to Lux City passes three great **Wayfinder Lanterns** that
have all gone dark. While they're dark the road is impassable (fog/night/wild Lumens). Each
of the three act-1 quests relights one lantern; lighting all three opens the gym road.

**Objectives (act 1):**

1. `lantern_woods` — **The Dim Woods.** The Wayfinder Lantern here is guarded/blocked by a
   spooked wild Lumen (a **Lanternjaw**). Calm or out-shine it (a Lumen encounter), then
   relight the lantern. Reward: `flicker_bond +1`.
2. `lantern_spring` — **Ember Spring.** The lantern is bone-dry. Refill it with lamp-oil from
   the spring, which means getting past a territorial **Torchcat** (Emberpaw). Reward:
   `flicker_bond +1`.
3. `lantern_ridge` — **Lookout Ridge.** Rival trainer **Spark** (cocky, neon-type Lumen) has
   "claimed" the ridge lantern and won't let the player pass without a battle. Beat Spark.
   Reward: `flicker_bond +1`.

When all three are lit → `gym_road_open = "true"`. The player travels to Lux City.

**Distinct locations / NPCs to give act 1 texture:** Wickville (home, Professor Bulb,
a **Lantern Rest** healing center), the Dim Woods, Ember Spring, Lookout Ridge, the road
into Lux City. Wild Lumens to encounter for flavor: Candlewisps, Glowbugs/Fireflits,
Streetlamp Striders, the rare Chandelier.

**Transition → act 2:** at Lux City the player enters the Grand Lumen Gym and steps onto the
battle floor to challenge Lumina. The moment the battle is about to begin, the lights die.
Set `phase = "2"`; load `g.cyo.lamps.phase2`.

---

## Act 2 — Lights Out (the gym attack & escape)

**The turn:** as the gym battle is called, the antagonist strikes. **Lord Umbra** and his
followers, **Team Eclipse**, want to snuff out every lamp in the world. Umbra blacks out the
gym, seals the doors with living shadow, and the battle is off. Lumina is cut off elsewhere
in the building. The player is trapped in the dark with Flicker.

**The secret of the gym:** the Grand Lumen Gym was built inside a converted old
**lighthouse-observatory** — the Lumen League's founders came *from the stars*. Beneath it,
sealed for generations, is a decommissioned **launch pad and rocket**. Act 2 is the escape
down to it. (This seeds act 3 and explains the rocket.)

**Objectives (act 2):** a short sequence, each a step down toward the launch pad. Flicker's
glow (driven by `flicker_bond`) gives advantage in these dark scenes.

1. `power_restored` — find the gym's backup lamp/generator and bring up emergency light so
   the player can navigate. Gate: relight it in the dark (advantage if `flicker_bond` high).
2. `door_unsealed` — the main exits are shadow-sealed. Discover the hidden stair down into
   the old observatory undercroft instead. Gate: evade Umbra's **shadow-hounds** (Gloamlings)
   and find/open the concealed way down.
3. `at_launchpad` — descend to the launch chamber and board the rocket while Umbra pursues.
   Gate: cross the launch chamber under pressure.

Inside the rocket: **Lumo**, an anthropomorphic desk-lamp robot — the ship's onboard
assistant. Warm, helpful, faintly anxious, very polite. Lumo's arrival begins act 3.

**Transition → act 3:** the player boards and the rocket launches itself. Set `phase = "3"`;
load `g.cyo.lamps.phase3`.

---

## Act 3 — Lampyria (the lamp planet, homecoming)

**The reveal:** once clear of the planet, Lumo sheepishly explains the autopilot is locked
to a destination it cannot override — coordinates set long ago. Some time passes (narrated
time-skip). They arrive at **Lampyria**: a whole world made of lamps. Lamps everywhere —
lamp-trees, lantern-cities, streetlight-forests, a sky full of floating paper lanterns.

And Flicker *lights up* — this is its home. The longing finally makes sense.

**Objectives (act 3):** short and heartfelt.

1. `family_clue` — explore the lamp city and learn where Flicker's kin live: the
   **Filament Quarter / Brass Hollow**, the old district of brass oil-lamps. Gate: a gentle
   social/navigation challenge (asking the lamp-folk, following Flicker's pull).
2. `reunion_done` — bring Flicker to the Brass Hollow and reunite it with its family. Gate:
   a final warm beat — Flicker lights the family's long-cold **homecoming flame** and is
   recognized. No combat; a heartfelt check.

When `reunion_done = "true"` → **VICTORY.** Read `g.cyo.lamps.victory`. Everyone lives happily
ever after. Umbra and the dark are left far behind; the lamp world's combined light is the
kind of thing his shadow could never hold. (We do not "defeat" Umbra on screen — escape and
homecoming are the resolution, matching the brief.)

---

## Cast & creatures (quick reference)

- **Flicker** — starter Brasswick (brass oil-lamp Lumen), the companion. Longs for the stars.
- **Professor Bulb** — the Professor Oak parody in Wickville; gives the player Flicker.
- **Spark** — the rival trainer; neon-type Lumen; act-1 lantern 3 battle.
- **Gym Leader Lumina** — radiance-type leader of the Grand Lumen Gym, Lux City. The battle
  that never happens.
- **Lord Umbra / Team Eclipse** — the antagonist; living shadow; wants to extinguish all
  lamps. Attacks the gym.
- **Lumo** — anthropomorphic desk-lamp robot; the rocket's onboard assistant; act-3 guide.
- **Wild Lumens:** Candlewisp, Glowbug/Fireflit, Lanternjaw, Torchcat (Emberpaw),
  Streetlamp Strider, Chandelier (rare), Gloamlings (Umbra's shadow-hounds, act 2).
- **Lampyrians:** the lamp-folk of the planet; Flicker's brass-lamp family in the Brass Hollow.

---

## Mechanics summary

- **Resolution:** `pick_random` difficulty tables (EASY/MEDIUM/HARD), failure-weighted
  (40/30/30, 30/30/40, 20/30/50). `random_chance` for binary checks. Never `roll_dice`.
  Act-1 lantern gates are MEDIUM; act-2 escape gates are HARD (the Flicker-bond advantage is
  what keeps them fair at ~25% failure rather than a brutal 50%); act-3 family clue is EASY and
  the reunion climax is a deliberate no-failure exception (happily-ever-after).
- **Health:** start 3. Healing sources: **Lantern Rest** centers in act-1 towns (limited);
  one emergency **oil flask** in act 2; a **hearth-lamp** moment in act 3. No invented healing.
- **Advantage:** high `flicker_bond` → roll twice, take the better result, in dark act-2
  scenes (and any scene where Flicker's glow plainly helps).
- **Images (SDXL):** generous. Style leans bright storybook in act 1, dark/dramatic in act 2,
  luminous wondrous in act 3. Backgrounds on every new location; inline for creatures/items/
  reveals. The `.rules` note carries per-act art direction.

## World state keys

```
phase            = "1"            # drives LOAD ON DEMAND for phase2 / phase3 notes
player_health    = "3"
companion_name   = "Flicker"      # GM updates if the player renames it
flicker_bond     = "0"            # 0..3, +1 per act-1 lantern
# act 1
lantern_woods    = "dark"         # -> "lit"
lantern_spring   = "dark"
lantern_ridge    = "dark"
gym_road_open    = "false"
# act 2
power_restored   = "false"
door_unsealed    = "false"
at_launchpad     = "false"
# act 3
family_clue      = "unknown"      # -> "known"
reunion_done     = "false"
```

## Note files (each authored as its own markdown file, then combined into the pack)

| File | Note key | Load | Purpose |
|---|---|---|---|
| `manifest.md` | `g.cyo.lamps` | entry | manifest: load lists, world-state init, objectives, opening |
| `rules.md` | `g.cyo.lamps.rules` | immediately | dice tables, health, healing, advantage, art direction, pacing |
| `world.md` | `g.cyo.lamps.world` | immediately | act-1 Lumalands geography + NPCs |
| `bestiary.md` | `g.cyo.lamps.bestiary` | immediately | Lumen encounter cards (act-1 creatures + act-2 Gloamlings) |
| `phase2.md` | `g.cyo.lamps.phase2` | on demand (phase="2") | the gym attack + escape locations/objectives |
| `phase3.md` | `g.cyo.lamps.phase3` | on demand (phase="3") | Lampyria + the reunion quest |
| `secrets.md` | `g.cyo.lamps.secrets` | immediately (GM eyes only) | Umbra's nature, the founders-from-the-stars truth, Flicker's origin |
| `victory.md` | `g.cyo.lamps.victory` | at victory | the homecoming ending |

Build step: a small script reads these eight files and emits `note-packs/lamps.json` with the
shared CYO character block (system prompt + tools, copied from `eldoria.json`) and the eight
notes (`read_only: true`).
