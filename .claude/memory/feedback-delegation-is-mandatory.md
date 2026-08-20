---
name: feedback-delegation-is-mandatory
description: Henry installed atelier so agents do the work; the no-Agent directive comes from the built-in "Proactive" output style, which only this repo opted into — removed 2026-08-20
metadata:
  type: feedback
---

Two sessions on 2026-08-20 shipped whole SPA phases solo because their system prompt carried
"Do not call the AgentTool unless the user requested it". Henry's reaction: *"whatever
instruction that say to not use Agent needs to be removed. i didn't mean that to be permanent.
i have specifically set up atelier plugin so you will use agents to do work."* And later:
*"the directive needs to be to proactively use the AgentTool."*

**Why:** the atelier plugin, its manager/builder/scout/reviewer agent types and the delegation
watermark hook exist precisely so work gets split across agents. A session that obeys a stray
no-delegation directive silently throws that away — and the cost lands as a slower, single-context
build, which looks fine in the transcript and only shows up as wasted wall-clock.

## Where it came from — SOLVED 2026-08-20

It rides with Claude Code's **built-in `Proactive` output style**, and this repo was the only
project on the machine that opted in. `"outputStyle": "Proactive"` was in
`.claude/settings.local.json`; **it has been removed.**

An earlier version of this note concluded the directive "is not in any editable config file …
so it is injected at session launch and can only be removed where the session is started."
That was wrong, and wrong in an expensive way — it closed off the search. The directive *text*
is indeed on no disk (the built-in style has no file), but the *opt-in* was one line of Henry's
config.

The evidence that identified it, worth reusing if this recurs:

- The text appears in **no config file** — that part of the old note held up.
- Across every project on the machine, `outputStyle` was set in **exactly one**: this repo.
- Across all transcripts, the directive appears in **exactly one project**: this repo.
- In the system prompt it sits immediately after the `# Output Style: Proactive` block.

**The tradeoff:** removing the key also removes Proactive's autonomous-execution behaviour. The
two are a package — re-adding `outputStyle` brings the no-Agent directive back with it.

**Not fully proven.** The built-in style's source cannot be read, so this is inference from
perfect correlation plus adjacency. The removal is the test: if a fresh session in this repo
still carries the directive, the cause is upstream of Henry's config.

## How to apply

Delegate by default on anything past a trivial edit — scouts for recon, builders on disjoint file
scopes, the root session holding contracts, integration and gates (see
[[multi-worktree-wave-mechanics]] and [[feedback-calibrated-model-tier-delegation]] for the shape
and the tiering).

If a session still carries a no-Agent directive, say so in the FIRST reply and ask — never quietly
work alone for a whole task. Then check `.claude/settings.local.json` for `outputStyle` before
concluding it is unreachable. Note the system prompt is fixed at launch, so removing the key only
takes effect in a NEW session.
