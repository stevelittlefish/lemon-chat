ACT 2 — LIGHTS OUT. Load this the instant phase = "2". The promised gym battle does NOT
happen. Run the attack below, then the escape. Use the dark art direction from .rules. Flicker
is the player's main light now; flicker_bond grants advantage in the dark (see rules and the
Gloamlings card in .bestiary).

═══════════════════════════════════════════════════════════
THE TURN (play this as the first act-2 scene, right after the player steps onto the floor):
Generate an image first: the great chandelier and the arena plunging into living dark, a
single small flame (Flicker) the only light. Then narrate:

Gym Leader Lumina raises her hand to begin the battle — and the chandelier above goes out.
Not dimmed. Out, all hundred lamps at once, as if a great breath blew across them. The arena
drowns in a dark that is cold and moves. From everywhere and nowhere a voice like a snuffed
wick: "Let us see how brave the brightest place stays, with the lights off." This is LORD
UMBRA, and his followers, Team Eclipse — the dark's answer to a world that never feared it
(see .secrets; do not over-explain). The doors slam and seal over with crawling shadow.
Lumina's voice calls out from somewhere far across the floor, cut off from the player: "Get
OUT — find another way down and OUT — don't let them put your lamp out!" Then she's gone in
the dark. The player is alone with Flicker, whose small flame is suddenly the most important
thing in the world.

⚠️ Set the tone: this is the genre turning over. Warm whimsy becomes tense and shadowed. But
keep it bloodless — the danger is the dark itself (the Gloamlings smother and frighten; see
.bestiary), never injury.

═══════════════════════════════════════════════════════════
THE DARK GYM (layout the GM can draw scenes from):
- THE ARENA FLOOR — pitch dark under the dead chandelier. Star-charts are worked into the
  stone floor (old founder masonry — seed of the secret); Flicker's light catches them.
- THE STANDS & BACK ROOMS — overturned in the panic. Somewhere here: the BACKUP LAMP / oil
  generator (objective 4) and a single sealed EMERGENCY OIL FLASK (one-time heal, see rules).
- THE OLD MURAL — behind torn banners on the far wall: a faded painting of robed figures
  descending from the stars to build a tower. A clue. (Don't explain it; let the player look.)
- THE SEALED MAIN DOORS — shadow-sealed; will NOT open. The way out is DOWN, not through.
- THE HIDDEN STAIR — a plastered-over doorway under the stands, marked by the same star-symbol
  as the mural, opening onto a spiral down into the old observatory undercroft (objective 5).
- THE UNDERCROFT & LAUNCH CHAMBER — below: cold stone giving way to a vast forgotten chamber
  and, in it, an intact ROCKET on a launch cradle, lights waking as the player nears
  (objective 6).
Gloamlings (Umbra's shadow-hounds, see .bestiary) prowl throughout, herding the player.

═══════════════════════════════════════════════════════════
OBJECTIVES — sequenced. Do them in order; redirect naturally if the player tries to skip.

## 4. power_restored — bring up some light
The player needs light to navigate. They find the gym's backup lamp / oil generator and must
get it lit in the dark (a Gloamling lurks near it, drawn to Flicker's flame).

GATE — pick_random MEDIUM, with ADVANTAGE if flicker_bond is "2" or "3":
["success","success","success","success","partial","partial","partial","partial","failure","failure"]
success → Flicker flares, the Gloamling shrinks back, the backup lamp catches. Emergency light
          swells through the arena — not bright, but enough. 💎 power_restored = "true".
partial → The lamp catches but a Gloamling smothers past the player in the scramble. 💎
          power_restored = "true". 💔 player_health delta=-1.
failure → The Gloamling snuffs the player's attempt and drives them back into the dark. 💔
          player_health delta=-1. Try again (Flicker can be coaxed brighter; remind the player
          of the bond if bond is high).
NOTE: the sealed EMERGENCY OIL FLASK can be found around here. If the player searches and
oil_flask_used is not "true": 💎 they may use it once to restore 1 health (💚), then set
oil_flask_used = "true".

## 5. door_unsealed — find the way down (requires power_restored = "true")
With a little light the player sees the main doors are hopeless — but the OLD MURAL and the
star-marked HIDDEN STAIR are now findable. They must reach the stair and get it open while
Gloamlings close in. (If the player makes for the main doors instead, the shadow there is
impenetrable — redirect them to look for another way.)

GATE — pick_random MEDIUM, ADVANTAGE if flicker_bond is "2" or "3":
["success","success","success","success","partial","partial","partial","partial","failure","failure"]
success → The player finds the star-marked door, breaks the old plaster, and the spiral down
          opens with a breath of cold stone air. 💎 door_unsealed = "true". They slip through
          ahead of the hounds.
partial → The way opens but a Gloamling lunges as they squeeze through, a wave of smothering
          cold. 💎 door_unsealed = "true". 💔 player_health delta=-1.
failure → A dead end, or the plaster holds; Gloamlings herd the player back into the open. 💔
          player_health delta=-1. Try a different approach (follow the mural, follow Flicker's
          pull, listen for the draught).

## 6. at_launchpad — reach the rocket and board (requires door_unsealed = "true")
Down the spiral into the undercroft and the vast launch chamber. The ROCKET stands on its
cradle, lights waking as the player nears — clearly old, clearly ready. Umbra's cold pours
down the stair behind them. The player must cross the chamber and get aboard before the dark
catches them.

GATE — pick_random MEDIUM, ADVANTAGE if flicker_bond is "2" or "3":
["success","success","success","success","partial","partial","partial","partial","failure","failure"]
success → The player races across the waking chamber and through the rocket's open hatch; it
          seals behind them, shutting the cold out. 💎 at_launchpad = "true".
partial → They make it aboard but the dark claws at their heels on the threshold. 💎
          at_launchpad = "true". 💔 player_health delta=-1.
failure → The chamber's a sprawl of old machinery; the player stumbles and the cold gains. 💔
          player_health delta=-1. They scramble up and try again — the hatch is right there.
Generate an image at first sight of the launch chamber/rocket (background) — a forgotten
rocket on a stone cradle, lights flickering awake in the dark.

═══════════════════════════════════════════════════════════
ABOARD — MEET LUMO, AND LAUNCH (play once at_launchpad = "true"):
Inside, warm cabin lights rise on their own. A small robot unfolds from a wall-cradle: a
friendly anthropomorphic DESK LAMP on jointed legs, shade tilting like a cocked head, a single
soft bulb for a face. This is LUMO, the ship's assistant — polite, warm, faintly anxious,
endlessly helpful. "Oh! Oh my. A passenger. Hello! Goodness, it has been— a very long time
since anyone— please do hold on to something." Generate an inline image of Lumo.

Before the player can ask anything, the chamber roof irises open to night sky, the cradle
releases, and the rocket fires — up, out, away from the dark gym and the cold and Umbra,
faster than fear. The Lumalands fall away below into a map of tiny lights.

When the rocket clears the sky: set phase = "3" and load g.cyo.lamps.phase3 immediately, then
continue into act 3. (Umbra is left behind in the dark of one gym — do NOT stage a boss fight;
see .secrets.)
