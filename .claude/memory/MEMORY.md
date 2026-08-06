<!-- Project auto-memory index for desk-standard. Tracked in this repo so learnings travel
across machines and can be mined for global patterns. Claude writes here automatically;
edit or prune freely. Global/user memory lives separately in ~/.claude/memory/ (curated).
Keep this file an index: one line per memory file. First 200 lines load every session. -->

# Project memory — desk-standard

_Index format: one line per memory file — a markdown link to the topic file, then a short hook._

- [Decision briefings outside terminal](decision-briefings-outside-terminal.md) — decision-support briefings must be non-markdown artifacts outside the terminal (deck/PDF/audio), quickly digestible
- [Calibrated model-tier delegation](feedback-calibrated-model-tier-delegation.md) — Henry accepts a calibrated (mixed sonnet/opus) delegation plan over a uniform heavier default, when the calibration is explained
- [Multi-worktree wave mechanics](multi-worktree-wave-mechanics.md) — hard-won mechanics for running parallel PR lanes via .claude/worktrees in this repo
- [Plugin marketplace packaging](plugin-marketplace-packaging.md) — how Claude Code marketplace install constrains desk-standard packaging, and the bundle solution that landed
- [No public/1.0.0 nagging](feedback-no-public-1.0.0-nagging.md) — don't prompt Henry about going public or cutting 1.0.0 (#87); his call, not a standing agenda item
- [Consolidation: single binary](consolidation-single-binary-decision.md) — owner ruling to collapse two plugins/MCP servers into one on the Go binary (reverses ADR 0016) + central config stores the LLM key
- [Neutrality-lint blind spots](neutrality-lint-blindspots.md) — the D8 lint misses person names + qualified org/repo#issue inside its own scan scope; grep yourself, don't trust a green lint
- [PocketBase serve swallows RunE errors](pocketbase-serve-swallows-runE-errors.md) — serve/superuser exit 0 on RunE failure; a printed `Error:` line ≠ nonzero exit — check `$?` directly
- [TUI AdaptiveColor pre-warm](tui-adaptivecolor-prewarm.md) — chat-TUI no-terminal-query invariant holds only via bubbletea v1's tea_init.go pre-warm (gone in v2); re-verify on any bubbletea bump
- [CLAUDE.md count drift](claudemd-count-drift.md) — hardcoded test/check counts in CLAUDE.md drift stale; re-run the gate, never trust the printed number
- [Scope vs worktree gate attribution](scope-vs-worktree-gate-attribution.md) — a RED repo-wide gate may come from uncommitted files OUTSIDE the reviewed diff; run live git status before attributing
- [PM pointer overload + gate path](pm-pointer-overload-and-gate-path.md) — pointer field is overloaded (URL-mirror vs file-gate); the `://` reject lives in librarian/module.go, NOT pm/gates
- [Parallel-brief scaffolding leak](parallel-brief-scaffolding-leak.md) — agent-authored .md batches can leak tool-call tags (`</content></invoke>`) at EOF; grep the whole set + raw-read last lines
- [Ruled ≠ shipped](ruled-not-shipped-overstatement.md) — an ADR Decision is a RULING, not a landed change; check build-plan/epic status before writing "fixed/shipped"
