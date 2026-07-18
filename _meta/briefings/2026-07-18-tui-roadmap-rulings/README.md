# TUI roadmap — the rulings you're signing off on (#51 / #52 / #53)

_Decision memo for Henry: what each issue proposes, the specific calls that are yours, and a
recommendation for each. Everything else in these issues is build detail the crew can decide._
Status: decided (2026-07-18) — Henry accepted all five recommendations verbatim

Background: after the chat-TUI pass (PR #48), you directed the TUI toward a full app — thread
management, context visibility, "lift and shift from Crush." Three issues carry that. One of
them (#53) is the enabler the other two build on, so it's listed first here even though it's
the highest-numbered.

---

## #53 — Migrate the TUI to the Charm v2 stack ← THE GATE RULING

**What it is.** Move the chat TUI from bubbletea/lipgloss/bubbles/glamour v1 to the v2
generation. This is the foundation Crush is built on, and it's where Charm's development goes
now. It is the enabler for lifting Crush's patterns closely: their session list, dialogs, and
status bar are all bubbles-v2 shapes.

**Ruling A — do the migration at all?**
- *Yes (recommended).* One-time mechanical cost across the TUI package; after it, #51/#52
  land cheaper and every future Charm improvement is available. v2 also gives a first-class
  safe way to ask the terminal its background color (the runtime serializes the query), which
  replaces the workaround pinning we do today for the ADR 0004 no-query rule.
- *No / not yet.* #51 and #52 are still buildable on v1 — just with more hand-rolling and a
  bigger migration later, after more code sits on v1 APIs.
- **What you're accepting if yes:** a focused migration PR with no feature changes, plus an
  ADR recording the stack decision and the updated ADR 0004 mechanism.

**Ruling B — should the TUI paint its own background (Crush-style), or keep rendering on
your terminal's background?**
- *Terminal background (recommended, at least initially).* What we have: your terminal theme
  shows through; light/dark palettes adapt to it. Keeps `--theme light|dark|auto` exactly as
  shipped.
- *Painted background (Crush's choice, only possible on v2).* The app fills its own
  background color, so it looks identical everywhere and the two-plane chrome gets full
  control — but it overrides your terminal theme, and would want a transparency toggle like
  Crush's. Can be revisited later without waste; nothing in the migration blocks it.

**Licensing note (already the working rule):** Crush itself is FSL-licensed, not MIT — we
lift *designs* and write original code on the MIT Charm libraries. No Crush code is copied.

---

## #51 — Session-management surface (threads: list, preview, rename, delete)

Baseline that already ships in PR #48: ctrl+n new thread, ctrl+o resume picker, resume
restores the transcript with full model context. #51 turns that skeleton into a real surface.

**Ruling A — launch behavior.** When you run `chat` and prior threads exist, what appears?
- *Sessions list first (recommended).* You pick up where you left off or hit `n` for new —
  the resume-first pattern. First-ever run still lands in a fresh thread.
- *Straight to a new thread (today's behavior),* with the list only behind ctrl+o.

**Ruling B — thread titles.** Lists need titles.
- *Truncated first user message (recommended for v1).* Zero cost, works offline, editable
  via rename.
- *Model-generated titles.* Nicer, but each new thread costs an LLM call; can be added later
  as an enhancement without redoing anything.

**Ruling C — v1 lifecycle scope.** Recommend **rename + delete (with confirm)** in v1;
archive/pin later if wanted. Delete is the only destructive op — it removes the thread's
messages from the store permanently.

**Layout call (crew's, unless you care):** full-screen list (recommended) vs a persistent
Crush-style sidebar. The sidebar eats ~30 columns permanently and really wants the v2 stack;
the full-screen list works on either stack and matches the picker you already know.

---

## #52 — Context-window usage + token accounting

**What it is.** Show how full the model's context window is (e.g. `41% ctx · 12.3K tok` in
the status bar), plus per-turn token counts joining the existing `model · latency` turn
footer. The accounting lives in the session layer, so the deferred webapp (#19) inherits it.

**Ruling A — show cost in dollars?**
- *Not yet (recommended).* Tokens and percent are computable from provider responses alone.
  Dollar cost needs a maintained per-model pricing table — a small ongoing upkeep commitment
  and a staleness risk. The survey already parked cost display for exactly this reason;
  confirm the deferral (or overrule and we add a pricing table with a documented update
  procedure).

**Ruling B — where the context-window sizes come from.** Percent needs each model's window
size. Proposal: a small neutral built-in table for known models **plus a profile override key**
(e.g. `models.context_window`) for anything unknown. The override key means touching
`schema/profile.schema.yaml` — the shared contract both products read — which is why it's
surfaced to you rather than just done. Recommend: yes, add the key.

**Not a ruling:** exact display format/placement — status-bar segment is the working plan;
the crew adjusts to what reads well.

---

## Sequence (recommended)

1. ✅ PR #48 + PR #54 merged (done as of this memo).
2. **Rule #53** — if accepted, the migration PR + ADR go first; it's the cheapest moment
   (least v1 code to convert).
3. **#52** token plumbing can start any time (session-layer, stack-independent); its TUI
   display lands with whichever stack is current.
4. **#51** builds after #53 so the list/dialog components come from bubbles v2 instead of
   being hand-rolled twice.

**The short version — five checkboxes (all RULED 2026-07-18, per recommendation):**
- [x] #53A: migrate to Charm v2 — **yes** (ADR 0006)
- [x] #53B: **terminal background** (painted background revisitable later on v2)
- [x] #51A: **sessions-list-first** launch when threads exist
- [x] #51B: **truncation titles** for v1, model titles later
- [x] #52A: **defer dollar-cost** display; **add** the profile context-window override key
