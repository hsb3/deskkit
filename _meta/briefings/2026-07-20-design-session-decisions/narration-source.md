# Audio overview source — design session decision package (2026-07-20)

Audience: Henry, the owner. Companion to the fifteen-slide deck. Speak plainly; this walks
the eight decisions he approved this morning. Say "decision one" not "D1". Say "issue
eighty-three" not "hash eighty-three".

## Where we stand

This is the design session package for desk standard. The bug floor merged this morning —
all nine milestone issues closed. You then approved the decision list: eight decisions,
covering the two concerns you raised — agent symmetry and the document data model — plus
everything the evidence phase surfaced. Every decision has a written brief with options,
line-cited evidence, and blast radius. Two independent reviews passed: every citation
checks out, and five load-bearing claims were re-proven directly from source code. Nothing
in this deck is a ruling yet — the session's job is to turn each decision into a ruling or
an explicit deferral, and each ruling becomes an architecture decision record.

## The order

Decide model first, then workflow, then surfaces. The later decisions consume the earlier
rulings — the agent parity question gets easier once the model rulings say what every
surface must expose.

## Decision one — pointer grammar

What is a pointer allowed to be? Work items point at documents, and the accepted forms are
implemented but written down nowhere. The options run from simply ratifying today's
behavior in the spec, through real inside-the-file addressing with stable anchors, to a
tolerant middle where a moved heading falls back to the whole file. The review sharpened
the stakes: the spec promises pointers to GitHub issue URLs, and the shipped default gates
reject them today — that contradiction is live, not theoretical.

## Decision two — typed cross-references

Does "graduated to: forty-two" become a real reference? Today it's a plain string with no
record of which repo it means. The qualifier can only come from the per-desk profile, so
resolution is desk-relative by construction — and shipped code may bake in no default. The
options: keep it field-local, adopt one shared reference type with the qualifier resolved
at read time, freeze the qualifier at write time, or specify the contract now and migrate
later. The hard test is that the store must rebuild from disk.

## Decision three — item types

May a typo'd item type skip every gate? Creating a work item never checks its type, and an
unknown type matches no gate rules, so it advances through every phase ungated — re-proven
from source. Validate at creation, warn only, or rule the behavior deliberate and document
it. Small decision, one code point.

## Decision four — the findings lifecycle and the adoption log

Dispositions shipped in the bug floor, but the machine is half-finished: a dismissed state
no code can set, counts that disagree between commands, no record of who dispositioned a
finding or why. And the adoption log has five of its six event kinds never written. The
options run from closing the lies and building nothing, to wiring the log as the real
activity ledger, to killing it entirely. The review found the kill option is not free —
the log has live readers on three surfaces and a role in the desk-collision guard. One
requirement only you can state: must who-disposed-and-why survive a store rebuild?

## Decision five — the agent contract, your headline

One integration contract, two agents. The frame: persona instructions, tool mount, wake
layer, and write-gate policy are the four contract parameters; the librarian and the PM
agent each instantiate them. Deliberate differences survive as documented parameters;
accidents become debt. Four calls inside it: whether the librarian gets a Claude Code
bundle; per-module mounts versus one shared mount with tool-level gating; whether the
in-binary loop keeps PM tools it has no instructions for; and whether the contract names
one instruction source per surface. The evidence widened this beyond the original concern —
unclaimed tools exist on every surface, including the librarian's own stale system prompt.

## Decision six — prompt governance

Which copy of the agent's instructions wins? They live in three places: compiled into the
binary, editable rows in the database, and markdown in the plugin. The review proved a
database edit silently vanishes if the store is rebuilt. Git as truth, database as truth, a
documented split by prompt class, or the status quo documented — and every option has to
say what reset-to-shipped means.

## Decision seven — spec versus reality

What does the TypeScript boundary promise? The spec reads as if the plugin's TypeScript
server can call librarian tools; it ships exactly four profile tools. The review judged the
spec sentence ambiguous rather than flatly broken. Your note at sign-off leans this one:
you want the TypeScript seam carrying server-backed capabilities — a significant
convenience. That points at extending the boundary rather than amending the spec down, with
the proxy design becoming a work item.

## Decision eight — identity and hygiene

You promoted this from backlog to a full decision this morning. Three independent calls:
rename identity — today renaming a document discards its history; the confusingly shared
entity-type name between a column and an unrelated schema enum; and the silent
five-thousand-character text caps. Each has a leave-it option; whatever wins on identity
must survive a store rebuild.

## After the rulings

Each ruling lands as an architecture decision record, cited where it binds. Spec deltas
follow, then the build plan — epics and issues with acceptance criteria and gate labels,
deliverables and parallelism, never timelines. Every model change is a forward migration,
so your live desks migrate; nothing is edited in place. The PM default-on lane — issue
eighty-three — slots wherever the rulings put it.

## The ask

Rule the eight. An explicit deferral is a valid ruling; an open question is not. Answer by
the sign-off form or live — either way, every decision leaves the session with an owner and
a record.
