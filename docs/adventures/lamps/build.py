#!/usr/bin/env python3
"""Combine the per-note markdown files in this folder into note-packs/lamps.json.

Each markdown file maps to one note key in the g.cyo.lamps namespace. The shared
"Choose Your Own" game-master character (system prompt + tools) is reused verbatim from an
existing pack so importing this pack does not regress the character. Re-run after editing any
of the .md files: python3 docs/adventures/lamps/build.py
"""
import json
import pathlib

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parents[2]  # repo root (docs/adventures/lamps -> repo)

# File -> note key. Order here is the order notes appear in the pack.
NOTES = [
    ("manifest.md", "g.cyo.lamps"),
    ("rules.md",    "g.cyo.lamps.rules"),
    ("world.md",    "g.cyo.lamps.world"),
    ("bestiary.md", "g.cyo.lamps.bestiary"),
    ("phase2.md",   "g.cyo.lamps.phase2"),
    ("phase3.md",   "g.cyo.lamps.phase3"),
    ("secrets.md",  "g.cyo.lamps.secrets"),
    ("victory.md",  "g.cyo.lamps.victory"),
]

# Reuse the shared GM character (name, tools, system prompt) from the latest pack.
character = json.loads((ROOT / "note-packs" / "discombobulation.json").read_text())["character"]

notes = []
for filename, key in NOTES:
    value = (HERE / filename).read_text().rstrip("\n")
    notes.append({"key": key, "read_only": True, "value": value})

pack = {
    "id": "cyo-lamps",
    "name": "The Lamplight League",
    "version": "1.0.0",
    "description": (
        "Notes for the Choose Your Own text adventure game master character. Import this pack "
        "to create all eight game notes at once. A three-act adventure that begins as a "
        "lamp-themed monster-taming romp and becomes a journey to the stars. The character "
        "system prompt is included below for reference only — it is not imported."
    ),
    "character": character,
    "notes": notes,
}

out = ROOT / "note-packs" / "lamps.json"
out.write_text(json.dumps(pack, indent=2, ensure_ascii=False) + "\n")
print(f"Wrote {out} ({out.stat().st_size} bytes, {len(notes)} notes)")
