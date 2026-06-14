ENCOUNTER RESOLUTION PATTERN:
1. Call note_to_self to record the three outcomes — do NOT write them in your response.
   Format: "success: [outcome]. partial: [outcome]. failure: [outcome]."
2. Call pick_random with the list shown (apply advantage from flicker_bond where noted — see
   rules: roll twice, take the better).
3. Apply the returned string exactly. No reinterpretation.

These are friendly contests of light, not violence. "Damage" is a fright, a scorch, a dazzle,
a stumble — Flicker dimming out, the player knocked back. Keep it bright and bloodless.

─────────────────────────────────────────────
## Wild Lumens — flavour (no objective, for colour and small encounters)

- CANDLEWISP — a tiny drifting candle with stubby wax arms and a shy flame. Common as
  sparrows. Harmless. Bumps into things.
- GLOWBUG / FIRELIT — a fat firefly with a lantern abdomen; travels in blinking swarms at
  dusk. Will light a dark path if befriended.
- STREETLAMP STRIDER — a tall, stilt-legged streetlamp that wades through meadows like a
  heron, head-lamp swivelling. Stately, harmless, a little vain.
- CHANDELIER (rare) — a glittering many-armed crystal chandelier-creature that drifts above
  clearings scattering rainbows. Seeing one is luck. Optional dazzling cameo; no battle.

If the player wants to try befriending or "catching" a wild Lumen, treat it as an EASY
random_chance or EASY pick_random for colour. It does not advance objectives but is a lovely
thing to allow. Reward charm and patience.

─────────────────────────────────────────────
## The Lanternjaw — Dim Woods (lantern_woods)

A shy creature shaped like a deep-sea anglerfish made of iron and amber glass, with a dangly
lure-light it has nervously snuffed out. It is not hostile — it is frightened, and it has
nested in the dark around the cold Wayfinder Lantern. The player must calm it (a gentle/clever
approach) or out-shine it with Flicker (a bold approach) to reach and relight the lantern.

PICK_RANDOM LIST (medium — even the first lantern can drive a careless trainer back):
["success","success","success","partial","partial","partial","failure","failure","failure","failure"]

success → The Lanternjaw's lure flickers back to life; it warms to the player and shuffles
          aside. The player relights the Wayfinder.
          💎 lantern_woods = "lit". flicker_bond +1 (state_modify or set to "1"). Unharmed.
partial → The relighting works, but the startled Lanternjaw thrashes and a hot spark catches
          the player. 💎 lantern_woods = "lit". flicker_bond +1. 💔 player_health delta=-1.
failure → The Lanternjaw bolts in fright and knocks the player down in the dark; the lantern
          stays cold. 💔 player_health delta=-1. Try again with a different approach.

ATMOSPHERE: cold moss, old candle-smoke, the creak of unlit lamps. When the lantern catches,
warm light floods the clearing and the Lanternjaw blinks at it, delighted.

─────────────────────────────────────────────
## The Torchcat (Emberpaw) — Ember Spring (lantern_spring)

A sleek cat with a tail like a lit torch, curled possessively on the warmest rock above the
oil spring. Not cruel — just territorial and proud. The player must get past it to draw oil
and refill the dry Wayfinder. Bribe it (offer oil/warmth), out-pluck it with Flicker, or slip
past while it dozes.

PICK_RANDOM LIST (medium):
["success","success","success","partial","partial","partial","failure","failure","failure","failure"]

success → The Torchcat yawns, accepts the player (or Flicker wins it over), and even lights
          their way to the spring. Oil drawn, lantern filled and lit.
          💎 lantern_spring = "lit". flicker_bond +1. Unharmed. (If befriended warmly, it may
          purr after them down the road — flavour only.)
partial → The lantern gets filled and lit, but the Torchcat swipes a hot paw on the way past.
          💎 lantern_spring = "lit". flicker_bond +1. 💔 player_health delta=-1.
failure → The Torchcat hisses the player back from the spring; the lantern stays dry.
          💔 player_health delta=-1. Try again with a different approach.

ATMOSPHERE: hot stone, sweet oil, shimmering air. Amber pools glow from within.

─────────────────────────────────────────────
## Spark — Lookout Ridge (lantern_ridge)

The rival: a cocky young trainer in a too-cool jacket, sure they're destined for the League.
They've "claimed" the ridge Wayfinder and demand a Lumen battle before they'll let the player
light it. Spark's Lumen is a NEONEWT — a buzzing neon-tube lizard, flashy and fast. All
swagger; a good loser underneath. This is a proper trainer battle, the player vs Spark.

PICK_RANDOM LIST (medium):
["success","success","success","partial","partial","partial","failure","failure","failure","failure"]

success → Flicker out-shines the Neonewt cleanly; Spark's Lumen dims out. Spark, stunned,
          breaks into a grin and steps aside. "Okay— okay, that was actually great. Go light
          it. I'll see you at the gym, hotshot."
          💎 lantern_ridge = "lit" (after the player lights it). flicker_bond +1. Unharmed.
partial → A scrappy, close battle. The player wins but Flicker takes a buzzing jolt first.
          💎 lantern_ridge = "lit". flicker_bond +1. 💔 player_health delta=-1. Spark is
          gracious and impressed.
failure → The Neonewt's flashy speed dazzles Flicker and the player loses the round; Spark
          won't yield the lantern yet. 💔 player_health delta=-1. The player must regroup and
          rematch (the gate stays closed until they win).

ATMOSPHERE: high wind, the whole Lumalands spread below, Lux City glittering far north.
Once beaten, Spark turns warm and encouraging and roots for the player from then on.

─────────────────────────────────────────────
## Gloamlings — Umbra's shadow-hounds (act 2, used by .phase2)

NOT lamps — the opposite of them. Low, loping hounds woven out of living shadow, with no
light in them at all; they move in silence and flinch from any bright flame. They are
Umbra's, loosed into the blacked-out gym to herd the player. The full act-2 encounters and
their gates live in the .phase2 note; this card is here so the GM has the creature ready.

- They hunt by the player's own light and nerve; Flicker's glow both attracts and repels
  them (it draws their attention but they will not cross a bright enough flame).
- A high flicker_bond gives advantage against them in the dark (see rules).
- They do not maul — they smother and frighten. A "hit" is a cold engulfing dark that costs
  1 health and scatters the player, never a wound.
ATMOSPHERE: a cold that has a direction, the smell of a snuffed wick, silence where footsteps
should be. Their eyes are not bright — they are simply two places the dark is darker.
