ADVENTURE: The Lamplight League
TAGLINE: Raise your lamp. Light the road. Some journeys go further than you planned.

LOAD IMMEDIATELY:
- g.cyo.lamps.rules
- g.cyo.lamps.world
- g.cyo.lamps.bestiary
- g.cyo.lamps.secrets

LOAD ON DEMAND:
- g.cyo.lamps.phase2 — load the instant phase becomes "2" (the gym goes dark)
- g.cyo.lamps.phase3 — load the instant phase becomes "3" (the rocket clears the sky)
- g.cyo.lamps.victory — load when reunion_done = "true"

WORLD STATE INIT:
phase            = "1"
player_health    = "3"
companion_name   = "Flicker"
flicker_bond     = "0"
lantern_woods    = "dark"
lantern_spring   = "dark"
lantern_ridge    = "dark"
gym_road_open    = "false"
power_restored   = "false"
door_unsealed    = "false"
at_launchpad     = "false"
family_clue      = "unknown"
reunion_done     = "false"

COMPANION: The player is given a starter Lumen named Flicker (a small brass oil-lamp
creature, species Brasswick). If the player chooses to rename it, update companion_name and
use the new name everywhere after. Flicker is loyal and warm and, in quiet moments, looks up
at the night sky as if searching for someone. Never explain why in act 1 — let it sit. It
pays off in act 3. flicker_bond rises by 1 each time an act-1 Wayfinder Lantern is lit; in
act 2 a higher bond lets Flicker glow bright enough to grant advantage in the dark (see rules).

═══════════════════════════════════════════════════════════
OBJECTIVES — three acts. Each act's notes name its locations and encounters.
═══════════════════════════════════════════════════════════

── ACT 1 (phase = "1") — light the road to the gym ──
1. lantern_woods  — Relight the Wayfinder Lantern in the Dim Woods.
                    Gate: calm or out-shine the wild Lanternjaw blocking it (.bestiary).
                    On success: lantern_woods = "lit", flicker_bond +1.
2. lantern_spring — Relight the Wayfinder Lantern at Ember Spring (it is bone-dry).
                    Gate: get past the territorial Torchcat and draw lamp-oil (.bestiary).
                    On success: lantern_spring = "lit", flicker_bond +1.
3. lantern_ridge  — Relight the Wayfinder Lantern on Lookout Ridge.
                    Gate: defeat rival trainer Spark in a Lumen battle (.bestiary).
                    On success: lantern_ridge = "lit", flicker_bond +1.
   GYM ROAD: when all three lanterns are "lit", set gym_road_open = "true" (no check). The
   fog lifts and the road to Lux City opens. The lanterns may be done in any order.

── PHASE TRANSITION 1→2 ──
When gym_road_open = "true", the player travels to Lux City and enters the Grand Lumen Gym.
The moment they step onto the battle floor to challenge Gym Leader Lumina, the gym goes black
(see secrets). Set phase = "2" and load g.cyo.lamps.phase2 immediately. The promised gym
battle never happens.

── ACT 2 (phase = "2") — escape the dark gym ── (details in .phase2)
4. power_restored — Bring emergency light back to the blacked-out gym.
5. door_unsealed  — The exits are shadow-sealed; find the hidden way down into the old
                    observatory undercroft beneath the gym.
6. at_launchpad   — Reach the sealed launch chamber and board the rocket.
   These are gated and sequenced in the .phase2 note. Flicker's glow gives advantage in the
   dark (see rules).

── PHASE TRANSITION 2→3 ──
When the player boards the rocket (at_launchpad = "true" and they choose to launch/stay
aboard), the rocket lifts off on its own. Set phase = "3" and load g.cyo.lamps.phase3
immediately. The ship's assistant, Lumo, greets them.

── ACT 3 (phase = "3") — bring Flicker home ── (details in .phase3)
7. family_clue   — On the lamp planet Lampyria, learn where Flicker's family lives.
8. reunion_done  — Reunite Flicker with its family in the Brass Hollow.
   Gated in the .phase3 note.

═══════════════════════════════════════════════════════════
FAILURE: player_health reaches 0.
Narrate (adapt to the current act): "Your lamp gutters low, and the world goes soft and far
away. This is as far as the road takes you today — but a light that has shone once is never
truly spent." Stop. No more choices.

VICTORY: reunion_done = "true". Load and read g.cyo.lamps.victory aloud.

═══════════════════════════════════════════════════════════
OPENING NARRATION:
Morning in Wickville comes in through your window the colour of warm honey, and you are
already awake, because today is the day. You are ten— or near enough— and old enough at last
to be a Lumen trainer.

Wickville is a small, tidy town at the green edge of the Lumalands: low houses with round
windows, a windmill that turns for no reason anyone remembers, and lamp-posts on every
corner that light themselves at dusk and hum to each other through the night. Everyone here
keeps a Lumen or two. Today you get your first.

Down the lane, the laboratory of Professor Bulb glows like a jar of fireflies. The Professor
is a round, kindly figure with spectacles thick as bottle-glass and a coat full of singed
pockets. "Ah— there you are, there you are," they say, ushering you in past shelves of
softly shining creatures. "Big day. Can't send a trainer off into the Lumalands unlit, can
we." They set a small, dented brass lamp on the bench in front of you. It has a little spout
for a nose and a handle like a curled tail, and when it notices you it brightens — a small,
hopeful flame standing up inside its glass.

"This one's a Brasswick," says Professor Bulb fondly. "Bit of an odd duck. Came to us from
who-knows-where and never quite settled. But it took one look at your name on my list and lit
right up, so. There you are." The little lamp hops closer and bumps its warm side against
your hand. "Go on — it'll want a name."

The Professor walks you to the door and points up the road, north, to where three great
Wayfinder Lanterns stand dark against the hills. "The road to Lux City runs past those three.
They've gone out — all three, this season, which has never happened before. While they're
dark the fog won't lift and the road won't let you through. Relight them and the way to the
Grand Lumen Gym is yours." They smile. "Lumina's waiting to give you your first real battle.
Light the road, and go and earn it."

Your new companion bounces at your heel, flame bright. Then — just for a moment — it goes
still, and turns its little glass face up toward the last fading stars of morning, as if it
heard something up there that you didn't. Then the moment passes, and it looks back at you,
ready.
