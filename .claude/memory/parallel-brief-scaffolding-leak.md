---
name: parallel-brief-scaffolding-leak
description: Parallel-authored markdown deliverables (decision briefs, dossiers) can leak the authoring agent's tool-call scaffolding (</content></invoke>) at EOF; grep every file end before accepting
metadata:
  type: feedback
---

When verifying a batch of markdown deliverables written by parallel independent agents (e.g. the
2026-07 design-session decision book), do not assume clean file boundaries. One builder emitted
its final content INSIDE a Write/tool call and the closing `</content>` + `</invoke>` XML tags
leaked into the committed file as literal trailing lines.

**Why:** an agent that constructs a file via a tool invocation can accidentally serialize the tool
envelope into the payload; a reviewer reading only the "analysis" body never sees it because it
sits after the last prose line. It passes a heading-structure check and an evidence check — it only
shows on a raw EOF read or a tag grep.

**Recurred 2026-07-20** in the staged issue-body batch: two files not formally under review leaked
`</content>`/`</invoke>` at EOF; the whole-set grep caught them where a targets-only review would
not have. Always scan the whole sibling set, not just the named targets.

**How to apply:** for any batch of agent-authored files, run
`grep -rnE '</?(invoke|content|parameter|antml)' *.md` across the whole set as a standalone gate,
AND raw-read the last ~5 lines of each file. Classify as a content-integrity defect
(blocking-cosmetic: trivial to fix, but ships broken markdown). Also watch for inconsistent
section-numbering conventions across parallel briefs — cosmetic, but a tell that no cross-file
template lint ran. See [[claudemd-count-drift]] for the general "re-run the gate, never trust the
self-report" posture. (The `_meta/plans` corpus these examples came from was removed 2026-08-05;
the lesson applies to any future agent-authored batch.)
