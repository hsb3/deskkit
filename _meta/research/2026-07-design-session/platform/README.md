_Provenance note for this folder. Status: active (2026-07-20)._

# Platform-stream design docs (migrated 2026-07-20)

These three documents were authored on `dev-tooling-desk`
(`~/Documents/EXECUTIVE_DESK/Projects/dev-tooling-desk`) by a parallel design stream running
2026-07-19/20, and migrated here on the owner's directive (2026-07-20: "work will no longer
be done out of that desk"). The copies here are the working versions; the originals carry
migration pointers and are frozen.

| File | What it is | Original home |
|---|---|---|
| `plan.md` | The desk-platform design plan — rounds R1 (requirements, closed), R2 (tech/borrow survey, closed), R3 (element model, in review). R1 ruled **"grow deskkit"** — this work lands in this repo. | `_meta/plans/desk-platform/` |
| `spec-element-model.md` | The R3 element-model proposal (three planes over a beefed spine, mined from `_headcase`) + BOTH adversarial review findings (software + research). Awaiting owner approval of 5 IA changes + 4 open questions. | `_meta/plans/desk-platform/` |
| `system-cohesion-and-datamodel.md` | The estate macro-review that settled the store (PocketBase), named the two data natures (derived census vs authored PM/KM), and framed the typed entity + typed relation primitive. | `_meta/plans/extenders-estate/` |

**How they bind here:** decision brief `../decision-book/D0-platform-frame.md` reconciles
these with the D1–D8 decision book; the platform's R1 rulings and four open questions are
now IN this design session's scope. The prior owner-gates on the dev-tooling-desk side
(the `desk-platform-progress` deck's approval ask) are superseded by this session.
