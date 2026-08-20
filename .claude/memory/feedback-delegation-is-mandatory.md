---
name: feedback-delegation-is-mandatory
description: Henry installed atelier so agents do the work; a session-level "don't use the Agent tool" directive is not his standing preference and must be surfaced, not silently obeyed
metadata:
  type: feedback
---

A session on 2026-08-20 shipped a whole SPA phase solo because its system prompt carried
"Do not call the AgentTool unless the user requested it". Henry's reaction: *"whatever
instruction that say to not use Agent needs to be removed. i didn't mean that to be permanent.
i have specifically set up atelier plugin so you will use agents to do work."*

**Why:** the atelier plugin, its manager/builder/scout/reviewer agent types and the delegation
watermark hook exist precisely so work gets split across agents. A session that obeys a stray
no-delegation directive silently throws that away — and the cost lands as a slower, single-context
build, which looks fine in the transcript and only shows up as wasted wall-clock.

**How to apply:** delegate by default on anything past a trivial edit — scouts for recon,
builders on disjoint file scopes, the root session holding contracts, integration and gates
(see [[multi-worktree-wave-mechanics]] and [[feedback-calibrated-model-tier-delegation]] for the
shape and the tiering). If a session-level instruction forbids the Agent tool, say so in the
first reply and ask, rather than quietly working alone for an entire task. That directive is not
in any editable config file — not `~/.claude/settings*.json`, `~/.claude.json`, this repo's
`.claude/`, the dotfiles tree, or `cmux.json` — so it is injected at session launch and can only
be removed where the session is started.
