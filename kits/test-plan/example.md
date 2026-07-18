---
type: test-plan
status: approved
created: 2026-05-13
updated: 2026-05-19
tags: [quillpad]
author: robin
priority: P1
parent_engineering_spec: "[[capture-auto-classify-engineering-spec]]"
---

# Test Plan: Capture + auto-classify flow

_How we prove the bookmarklet capture path and Claude auto-classification work before trusting them in real use (parent spec: [[capture-auto-classify-engineering-spec]]). Coverage maps 1:1 to that spec's acceptance criteria. Reviewed at the weekly check-in alongside the RAID log (R-001 tracks extraction reliability)._

## Scope

**In scope** — what this plan verifies:

- The bookmarklet POSTs a captured URL + selection to the capture Function and gets a stored item back
- Claude extracts title, summary, and a suggested `kind` from the page, returned as a *suggestion* (never silently applied — RAID R-001)
- The canonical URL is resolved server-side even when the source uses a JS redirect (RAID I-002)
- The accept/edit UI lets the author confirm or correct the suggestion in one action
- Captures degrade gracefully when extraction fails (item still stored, marked needs-review)

**Out of scope** — explicitly not tested here, and why:

- Editor / ⌘K assist — separate spec and test plan ([[editor-assist-test-plan]])
- Firestore free-tier limits — operational concern tracked as RAID A-001, watched in the console, not a test
- Magic-link auth — covered by [[auth-magic-link-test-plan]]
- Bulk re-classification of historical items — not in the capture flow's v1

## Approach

| Level | Tool | What it covers |
|-------|------|----------------|
| Unit | Vitest | URL canonicalization (redirect resolution), the classify-prompt builder, suggestion-vs-apply guard |
| Integration | Vitest + Firebase emulator | Capture Function end to end: POST → resolve URL → call Claude (mocked) → write item to Firestore |
| E2E | Playwright | Bookmarklet → capture → accept/edit panel → item appears in the feed, against the emulator |
| Manual | — | Extraction *quality* on 50 real pages: is the suggested `kind` right, is the summary usable (RAID R-001 correction-rate gate) |

## Test cases

| ID | Scenario | Steps | Expected |
|----|----------|-------|----------|
| TC-001 | Happy-path capture | 1. Trigger bookmarklet on an article. 2. POST reaches the Function. 3. Claude (mocked) returns title/summary/kind. | Item stored with `status: needs-review`; suggestion fields populated; 200 returned. |
| TC-002 | Suggestion is never auto-applied | 1. Capture an item. 2. Inspect the stored record before any user action. | `kind` sits in a `suggested_kind` field, not `kind`; `kind` stays null until the author accepts (RAID R-001). |
| TC-003 | JS-redirect URL resolved | 1. Capture a page whose source URL 302/JS-redirects. 2. Read the stored `canonical_url`. | Stored URL is the final resolved destination, not the redirector (RAID I-002). |
| TC-004 | Accept suggestion | 1. Open the accept/edit panel. 2. Click Accept. | `suggested_kind` → `kind`; item flips to `status: published`; one write, no refetch (trust the mutation response). |
| TC-005 | Edit before accept | 1. Open the panel. 2. Change the kind + summary. 3. Save. | Edited values persist; `kind` set from the edit, not the suggestion. |
| TC-006 | Extraction failure degrades | 1. Capture with Claude returning an error/timeout. | Item still stored with the raw URL + selection; marked `needs-review`; no 5xx to the bookmarklet; error surfaced in the panel. |
| TC-007 | Duplicate capture | 1. Capture the same canonical URL twice. | Second capture updates the existing item rather than creating a duplicate row. |

## Coverage matrix

| Requirement (from spec) | Test case(s) | Level |
|-------------------------|--------------|-------|
| Bookmarklet stores a captured item | TC-001 | integration |
| Claude suggestion is surfaced, never silently applied | TC-002, TC-004 | unit + integration |
| Canonical URL resolved through JS redirects | TC-003 | unit |
| Author can accept or edit in one action | TC-004, TC-005 | e2e |
| Capture survives extraction failure | TC-006 | integration |
| No duplicate items on re-capture | TC-007 | integration |
| Extraction is good enough to trust (correction rate < 40%) | Manual 50-page run | manual |

## Environments

| Environment | Used for | Data / setup |
|-------------|----------|--------------|
| local | unit + integration | Firebase emulator suite; Claude API mocked with recorded fixtures; no live key |
| CI | unit + integration on every PR | Same emulator; mocked Claude; `firebase emulators:exec` |
| staging | e2e + manual quality run | Real Firebase project; live Anthropic key (RAID D-003, met); 50-page seed list for the correction-rate measurement |

## Exit / acceptance criteria

This plan passes — and the capture flow is provable — when:

- [ ] Every coverage-matrix row has a passing test case (TC-001 … TC-007)
- [ ] Unit + integration suites green in CI
- [ ] Playwright e2e green against the emulator (TC-004, TC-005)
- [ ] Manual 50-page run complete with measured `suggested_kind` **correction rate below 40%** (the RAID R-001 gate; if higher, tighten the prompt before widening)
- [ ] No open P0/P1 defects against capture or classification
