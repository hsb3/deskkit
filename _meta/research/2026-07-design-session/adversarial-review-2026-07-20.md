_Adversarial review of the design + planning corpus (ADRs 0009-0018 + the decision book,
dossiers, platform stream, plans, and issue bodies), run before the v1 build lanes open.
Companion to [`design-planning-package.md`](design-planning-package.md) (the corpus map).
Status: active (2026-07-20)._

# Adversarial review — design + planning corpus

**Verdict: GO-WITH-FIXES — 0 blockers.** The build lanes are safe to open. Ten findings were
raised; all ten survived independent adversarial verification (0 refuted), and four
originally-MAJOR raises were de-escalated by the verifiers. Two MAJOR planning-coverage edits
should land in the design-session epic (#129) before its #114/#118/#120/#123 wave dispatches
in parallel; the remaining eight fold into their owning slices.

## Process & provenance

- **Corpus:** the seven clusters indexed in [`design-planning-package.md`](design-planning-package.md).
- **Method:** 6 skeptical dimension reviewers (traceability, cross-doc consistency,
  spec-vs-reality, buildability, neutrality/artifacts, unverified-claims) -> per-finding
  adversarial verification (each finding re-derived from source, defaulting to refuted) ->
  opus-xhigh synthesis.
- **Agents:** 17 total (6 finders + 10 verifiers + 1 synthesis), 0 errors. Reviewers/verifiers
  ran opus/high; synthesis opus/xhigh. ~1.5M subagent tokens, ~12.5 min.
- **Result:** 10 raised, 10 CONFIRMED, 0 refuted; 4 severity de-escalations applied. The
  survivor list is fully source-grounded — every finding carries a path:line citation.

---

## Verdict

**GO-WITH-FIXES.** No blocker survived verification — 10 of 10 findings were re-derived from source and CONFIRMED, 0 refuted, and verification *de-escalated* four "MAJOR" originals to MINOR/NIT. The two remaining MAJORs are both planning-coverage defects in the design-session epic (#129) and its children: each is a one-to-few-line edit, and both fail loudly (unsatisfiable close-when / red CI) rather than shipping a corrupt artifact. The other eight are documentation- or planning-completeness gaps. The lanes can open; land the two epic-level edits before the affected `#114/#118/#120/#123` wave dispatches in parallel, and fold the rest into their owning slices.

## Blockers — clear before building

None. No finding rose to blocker: the strongest defects are gate-caught at CI or are close-condition coverage gaps, not code-correctness breaks. The verifier explicitly capped both MAJORs below BLOCKER (#1 is "trivially fixable by adding a deliverable"; #7 "fails loudly at CI rather than shipping a corrupt artifact").

## Fix during the affected slice (MAJOR — design-session epic #129 lane)

Both MAJORs live in epic #129's wave and should be resolved before its children dispatch in parallel.

**1. Epic close-when is unsatisfiable: 3 shipped TS plugin tools are claimed by no child.** (traceability, CONFIRMED / MAJOR)
- What: #129's close-when requires the ADR 0014 audit to find "zero unclaimed tools ... TS plugin tools" (`_meta/plans/epic-design-session-v1/issue-body.md:41-42`), but `profile_get`, `profile_validate`, and `knowledge_index` are claimed by no shipped skill/agent (only `template_render` is, via desk-setup + brownfield-adoption). ADR 0014's four Decision calls resolve only mount/eino/packaging (`docs/decisions/0014-agent-integration-contract.md:26-40`); #114's deliverables A–E name owners for `import` + admin console but never the 3 TS tools (`_meta/plans/agent-integration-contract/issue-body.md:42-95`); #122 only notes the gap; #119 claims only librarian tools. #114-E pointedly resolves the two *analogous* unclaimed-surface findings (import, admin console) while omitting this one — a scoping omission, not a deliberate deferral.
- Fix: add a deliverable to #114 (the contract-audit owner) that either claims the three tools via a skill/persona or documents them as no-persona-by-design, mirroring the import/admin-console treatment in deliverable E. Otherwise #129 cannot legitimately close and 1.0.0 inherits an unowned hole.

**2. The epic's shared-file coordination rule covers only the migration chain — two other shared surfaces collide under parallel dispatch.** (buildability + consistency; **merges** the #114-C/#120 prompt-embed finding [MAJOR] and the #118/#123 `query.go` finding [MINOR — same root, same fix location])
- What: #129 declares exactly one coordination rule, the #118/#123 migration-number / SchemaVersion chain (`epic-design-session-v1/issue-body.md:17-23`). Two other shared files are unmanaged:
  - **Prompt embed (MAJOR):** #114-C edits `librarian/templates/librarian-system-prompt.txt` (drops phantom apply_fix/restore, 7→5 tools) while #120's new `scripts/check-prompt-drift.mjs` pins that `.txt` byte-identical to the spec's fenced block at `docs/development/specs/pocket-librarian-v1-spec.md:1435-1464`. #114-C's scope is the `.txt` + a Go test only — not the spec block — and neither issue declares a dependency edge (#114 "Depends on" names only ADR 0015; grep found zero cross-references). If #120 lands first, #114's PR fails `check-prompt-drift.mjs`, and the librarian-scoped builder has no in-scope docs fix. The `.txt` is a three-way source: #114 edits, #120 guards, #119-D generates the bundle persona from it.
  - **`query.go` / `patrol.go` / spec (MINOR):** #118 (slice B `tools/query.go`, slice C `tools/patrol.go`) and #123 (slice B lists `sweep.go, query.go, propose_fix.go, patrol.go`) both edit these Go files plus `docs/development/specs/pocket-librarian-v1-spec.md`; only the migration chain is serialized (`findings-lifecycle-completion/plan.md:100,122-124`; `document-identity-hygiene/plan.md:40`). Verification capped this at MINOR: the migration rule already forbids concurrent landing, and today's edit regions are disjoint hunks git auto-merges — but the coordination note is silent on it, giving false "only the migration numbers need serializing" confidence.
- Fix: extend the epic coordination rule (and the affected plans' Cross-issue sections) to name both extra shared surfaces — either declare #114 must merge before #120 and add the `:1435-1464` block (+ the `:2316` Decisions bullet) to #114-C's scope so the embed edit and its guarded copy move together; and name `query.go` / `patrol.go` / the spec as serialized shared surfaces between #118/#123.

## Cleanup / nits (MINOR + NIT — fold into the owning slice)

**Traceability — ADR Affects surfaces owned by no child (same pattern as MAJOR #1, lower stakes):**
- **README librarian tool list is stale and unowned** (MINOR). ADR 0016 names "`librarian/README.md` stale counts" as Affects (`docs/decisions/0016-ts-boundary-deskkit-proxy.md:42-44`); README `:214-216` omits `record_feedback` from the default MCP tools (authoritative list: `docs/development/specs/tool-surface.md:70`). prompt-governance touches the README only at `:191-210`; #121 pins `docs/development/specs/tool-surface.md`, not the README. → fold the `:214` correction into #114 or #121, or record it explicitly re-scoped.
- **ADR 0009 Affects cites CHARTER.md, but CHARTER carries none of the truth-regime vocabulary** (MINOR, ~NIT). `0009-platform-frame.md:52-55` names "constraint-wall statements in `docs/development/CHARTER.md` and the specs"; CHARTER has zero hits for the staged/files-are-truth vocabulary and no issue updates it (#124 touches only the two specs + schema/README). Verification: the specs *do* carry the constraint wall, so only the CHARTER half of the citation is inaccurate (a stale non-negotiable also missing from CHARTER's invariant list). → add a CHARTER staged-truth paragraph to #124, or correct ADR 0009's Affects line.

**Consistency — spec/ADR wording defects (both already partly self-caught by the corpus):**
- **pm-spec pointer field-table row still advertises "issue URL"** (MINOR — verifier downgraded from MAJOR). `docs/development/specs/pm-system-v1-spec.md:396` still reads `| pointer | text | doc path / issue URL / other locus |` while the corrected R6.1 at `:744-745` and ADR 0010 say an issue URL is not a gate pointer. **Already scoped for fix in #115** (`pointer-grammar-spec/issue-body.md:31-39`) — verify #115 lands the `:396` edit and keeps #115/#116 sequencing. Code already fails `://` closed and is test-pinned, so this is doc-only.
- **ADR 0013 mislabels the dismissed-retirement remap as a "down-remap"** (MINOR). `0013-...:23,40-41` — retiring an enum makes the SHRINK the forward step, so the data-first remap must live in the forward migration; the "down" label is direction-inverted vs. the 0010/0012 precedent it cites. The build plan's Q1 already untangles it for anyone building from the plan (`findings-lifecycle-completion/plan.md:216-223`); the ADR text alone remains a data-safety hazard for a builder working from the ADR. → add a dated erratum to ADR 0013 (forward = remap-then-shrink; down merely re-adds the value).

**Neutrality-artifact planning errors:**
- **agent-integration-contract tells builders to allowlist a docs-only spec** (MINOR — verifier downgraded from MAJOR). Deliverable A / its AC require adding `docs/development/specs/agent-integration-contract-v1-spec.md` to the neutrality allowlist "in `scripts/check-neutrality.mjs`" (`plan.md:51`, `issue-body.md:48-50,75-77`) — but `check-neutrality.mjs:52` scans only `plugin/librarian/kits` (docs/ is never scanned, so the entry is a dead no-op) and the allowlist is the data file `schema/neutrality-lint.allow`, not the script. Every sibling docs-only issue states the correct opposite. → delete the allowlist clause from deliverable A and its acceptance criterion.
- **typed-reference-contract allowlists only the librarian copy of references.yaml, not the new bundle copy** (MINOR). The plan creates two neutrality-scanned copies — `librarian/internal/core/schema/references.yaml` and (via the extended `package.json` cp step) `plugin/claude-plugin/schema/references.yaml` — but plans an exemption for only the first (`issue-body.md:69-74,119-122`). If the builder mirrors the cited `doctypes.yaml` precedent (bare `#49/#55`), the bundle copy trips neutrality with no planned exemption. → require ADR refs written as "ADR 0011" (not bare `#N`, per CLAUDE.md) so neither copy needs an entry, or allowlist both scanned copies.

**NIT:**
- **#114-B acceptance criterion's "no other flag" parenthetical is stale** (NIT — verifier downgraded from MINOR). The AC asserts the desk-pm mount "(PM_ENABLED=true, no other flag)" returns exactly 12 tools, but the issue's own default rule (unset module set = all modules = 17) and deliverable B (add `MCP_MODULES=pm` to drop 17→12) mean the 12-tool mount must carry the added flag. → reword to "(PM_ENABLED=true and MCP_MODULES=pm declared)". Recoverable from adjacent text; affects no shipped code.

## What is solid

The corpus is fundamentally sound and the lanes are safe to open. Eight of ten findings are MINOR/NIT completeness gaps; the two MAJORs are single-edit coverage fixes, not architecture or code-correctness breaks. Notably strong:
- **The enforcement infrastructure works as designed.** The neutrality scanner, the new prompt-drift guard, the kit/version drift guards, and the migration-chain serialization all fail loudly — every MAJOR here is a defect the repo's own gates would catch at CI, not something that ships silently (#7 is red-CI-on-second-PR, not a corrupt artifact).
- **The corpus is self-aware.** The pm-spec pointer defect (#5) is already queued for correction in #115/#116; the ADR 0013 down-remap direction (#6) is already untangled in the build plan's Q1. The planning process is catching its own edges.
- **ADR provenance and code-citation discipline hold.** Every finding could be pinned to an exact ADR Affects line and spec line number *because* the ADRs enumerate their surfaces and are cited from code — the defects are omissions *within* a working traceability system, not the absence of one.
- **The one coordination rule that exists is correct** — the gap in MAJOR #2 is that it must name two more shared files, not that its migration-serialization mechanism is wrong.

## Refuted / non-issues

**Zero of the ten raised findings were refuted.** Every claim was independently re-derived from source and returned CONFIRMED, so the survivor list above is fully source-grounded and can be trusted as-is. Verification did, however, *de-escalate severity* on four findings where the original raise over-stated impact — and those corrected severities, not the original raises, drive this decision:
- #118/#123 `query.go` overlap: MAJOR → MINOR (git surfaces conflicts; migration rule already forces sequential landing; today's hunks are disjoint).
- pm-spec pointer field-table row: MAJOR → MINOR (authoritative R6.1 + ADR 0010 already correct; code fails `://` closed and is test-pinned; fix already queued in #115).
- docs-only neutrality allowlist entry: MAJOR → MINOR (worst realistic outcome is dead config + a wrong-file pointer a reviewer catches; no shipped bug).
- #114-B acceptance parenthetical: MINOR → NIT (fully recoverable from adjacent deliverable-B text).

No finding was inflated in the other direction, and none was dropped — the reader can treat all ten as real, with the severities above.

---

_Recommended follow-up: apply the two MAJOR edits to epic #129 + #114 (and sync the staged
issue bodies to the live issues via `sync-bodies.py --push`), then fold the eight MINOR/NIT
items into their owning slices as they build. None blocks opening lane #114._
