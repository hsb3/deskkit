---
type: release-notes
status: published
created: 2026-05-28
updated: 2026-05-28
tags: [quillpad]
version: "0.3.0"
---

# Quillpad 0.3.0 — "Capture that corrects itself"

_Quillpad — AI-Augmented Personal Publishing. Assembled from the feature-specs, decisions, and roadmap items closed since 0.2.0, reconciled against `v0.2.0..v0.3.0`._

This release makes capture trustworthy. Metadata extraction now arrives as a suggestion you accept or edit in one click, the canonical content model is settled, and the editor gained ⌘K assist. Invite-only readers can sign in.

## Upgrade notes

- **One-time content migration.** The merge of the two source models is live — existing captures are migrated to the unified item model on first launch. The migration runs automatically and is idempotent; no action needed, but the first launch after upgrade will take a few seconds longer.
- **No more silent auto-tagging.** If you relied on captures being tagged without review in 0.2.x, that behavior is gone by design (see Changed). New captures wait for your accept/edit.

## Added

- **Capture without the boring parts.** Saving a page now suggests title, tags, and a summary you can accept or tweak in one click — extraction is a proposal, never a silent overwrite, so you stay in control of what lands.
- **⌘K writing assist in the editor.** Select text and press ⌘K to rewrite, tighten, or expand inline, with a diff view before anything changes. Backed by the CodeMirror editor slice.
- **Magic-link sign-in for readers.** Invited readers get in with an emailed link — no account to create, no password to manage. Sufficient for the invite-only audience (decided in the 0.3 planning check-in).

## Changed

- **Captures are reviewed, not auto-applied.** Extraction moved from silent auto-tag to suggestion-with-accept. You'll see a review step on every capture; this is the fix for unreliable unattended tagging surfaced during the 0.2 trial.
- **One content model.** The post-centric and item/ingestion-centric models are unified onto a single item model with a `kind` field. Everything is an item now; "post" is just a `kind`. This unblocks the editor and capture halves that were stalled against incompatible schemas.

## Fixed

- **Bookmarklet keeps the real URL.** Capturing a page that uses a JavaScript redirect no longer stores the redirector's address — the final canonical URL is resolved server-side before saving, so your links point where you expect.
- **Editor chat first token is snappier.** The in-editor assist no longer stalls on a cold start for several seconds; a warm path plus a streaming skeleton means you see a response begin almost immediately.

---

_Next up (0.4, not in this release): scheduled publishing and a reader digest. Tracked on the roadmap._
