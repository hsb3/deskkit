---
# This repo's handoff is not a file. It is Kaneo task DESK-21 (`HANDOFF` label, Documents
# lane). External mode means atelier's freshness hooks stat `stamp` instead of searching
# for a HANDOFF.md that deliberately does not exist here any more.
handoff:
  mode: external
  stamp: .claude/handoff.stamp
  location: Kaneo board task DESK-21 (Documents lane, label HANDOFF)
---

`/handoff` rewrites DESK-21's description, then touches `.claude/handoff.stamp`.
**Board first, stamp last** — the guard reads the stamp's age, never the board's content,
so touching it early certifies a handoff that has not happened yet.

This file is tracked on purpose: it is project policy, not machine state, and it has to
survive a fresh clone or a move to another machine. The stamp is gitignored, because an
mtime is exactly the thing that should not travel. A fresh clone therefore has no stamp,
and the first manual `/compact` is refused until `/handoff` runs — the intended answer,
not a bug.

Nothing else is armed. `enforce`, `protected`, `isolate`, and `effort` are absent on
purpose. Confirm what is live rather than what is written, from the project root:

    python3 "${CLAUDE_PLUGIN_ROOT}/skills/activation/scripts/activation.py" check
