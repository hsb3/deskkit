---
name: claudemd-count-drift
description: desk-standard CLAUDE.md hardcodes test/check counts (plugin bun test, go test, verify.sh) that drift stale — always re-run, never trust the number
metadata:
  type: project
---

CLAUDE.md bakes literal counts into prose that go stale because nothing regen-guards them.
As of 2026-07-19 every audited count was wrong:

- **plugin `bun test` "(45)"** — actual was **59**; the test scope grew (opencode + desk-pm
  added per [[frozen-spike-ci-containment]]) but the number wasn't updated.
- **librarian `go test ./...` "(315)"** — actual was **314** run cases; 315 counted `func Test*`
  symbols including a `TestMain` harness. Off-by-one.
- **verify.sh "47 checks"** — the run printed **48 total**; static `check "` grep undercounts.
  Runtime N is authoritative.

**Why:** these counts are hand-written prose with no drift guard — unlike VERSION/manifests
(check-version-sync) or kits (check-kits). The file's own "Keeping this file current" rule
(update in the same change as the code) is exactly what's being violated.

**How to apply:** never confirm a numeric count in CLAUDE.md from the file alone. Re-run the
actual gate: `cd plugin && bun test <current glob>` (count "Ran N"),
`cd librarian && go test ./... -v | grep -c '^--- PASS'` (top-level),
`bash librarian/verify.sh | tail -1`. Everything else audited in CLAUDE.md (make targets, tree
paths, script names, MCP tool names, skill names, port 8090, env vars, docs paths) was accurate
— only the counts drift.

**Update 2026-08-05:** the counts have all churned again since the audit (CLAUDE.md now says
verify.sh has 61 checks). The specific numbers above are historical; the lesson — re-run the
gate, never trust the printed number — is the durable part.
