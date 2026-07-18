---
type: raid
status: in-review
created: 2026-05-14
updated: 2026-05-20
tags: [quillpad]
author: robin
owner: robin
---

# RAID Log: Quillpad — AI-Augmented Personal Publishing

_Living register for the Quillpad build (see the one-pager + product-spec). Reviewed at each weekly check-in._

## Risks

| ID | Risk | Probability | Impact | Mitigation | Owner | Status |
|----|------|-------------|--------|------------|-------|--------|
| R-001 | Claude metadata extraction is too unreliable to trust unattended, so every capture needs manual correction and the "removes the boring parts" promise fails. | medium | significant | Ship extraction as a *suggestion* with one-click accept/edit, never silent auto-apply. Measure correction rate on the first 50 captures; if >40%, tighten the prompt before widening. | robin | open |
| R-002 | Scope creep turns a personal publishing tool into a product build, and it stalls like the two prior attempts. | high | critical | The one-pager's "Won't Do" list is the contract. Any new feature must displace a Must-Have, not add to it. | robin | mitigated |
| R-003 | Firebase Functions cold starts make the in-editor Genkit chat feel laggy enough that Robin stops using it. | low | significant | Keep a warm path for the editor chat; show a streaming skeleton. Fall back to a lighter model for the first token. | robin | open |

## Assumptions

| ID | Assumption | Confidence | Validation plan | Status |
|----|-----------|------------|-----------------|--------|
| A-001 | A single author writing on cadence won't hit Firestore/Functions free-tier limits. | high | Watch the Firebase console through the first month of real use. | assumed |
| A-002 | CodeMirror + a few extensions is enough editor — no need for a heavier framework. | medium | Build the editor slice first; if ⌘K assist + diff view feel cramped, revisit before the polish pass. | unvalidated |
| A-003 | Magic-link auth is sufficient for an invite-only audience; no accounts/roles needed. | high | Confirm with the first 3 invited readers that email magic-link is acceptable. | assumed |

## Issues

| ID | Issue | Type | Priority | Resolution | Status |
|----|-------|------|----------|------------|--------|
| I-001 | The two source repos (`agent_quillpad`, `capture_kit`) have incompatible content models — one is post-centric, the other is item/ingestion-centric. The merge can't proceed until one model wins. | design gap | P0 | Decide the canonical content model (lean toward `capture_kit`'s item model + a `kind` field). Capture as a `decision` note. | open |
| I-002 | The bookmarklet capture path drops the page's canonical URL when the source uses a JS redirect. | tech debt | P2 | Resolve the final URL server-side on capture before storing. | open |

## Dependencies

| ID | Dependency | Type | Required by | Provider | Status | Risk if late |
|----|-----------|------|-------------|----------|--------|-------------|
| D-001 | Firebase project provisioned (Firestore, Functions, Auth, Hosting). | infra | First integration test | robin (manual setup) | not started | Blocks all end-to-end work; local dev unaffected. |
| D-002 | Decision on the canonical content model (see I-001). | decision | Editor + capture build | this project | in progress | Blocks the merge; both halves stay stalled until resolved. |
| D-003 | Anthropic API key with sufficient quota for extraction + chat. | data | Capture + editor chat | robin (Anthropic account) | met | — |
