_Two manual, operator-run walkthrough harnesses for the deskkit binary. Neither runs in CI._
Status: active

# examples/

Both scripts stand up a **throwaway scratch desk** (a `mktemp` dir plus its own store home) and
assert against real command output — never against a self-report, and never against a real desk.
Each prints numbered `PASS`/`FAIL` lines and exits non-zero if any check fails, in the same style
as the repo's `verify.sh` gate.

They are **not** part of `make check`, `make verify`, `make e2e`, or CI. They exist so an operator
can watch the real thing work end to end.

| Script | Asserts | Cost |
|---|---|---|
| [`pm-walkthrough.sh`](pm-walkthrough.sh) | The document-gated PM work graph, end to end: seeds two items via the `pm` CLI, links one as blocking the other, then drives the PM MCP surface over raw JSON-RPC stdio (blocked transition refused → unblock → transition → note → claim → release), and confirms `MCP_MODULES=pm` exposes exactly the 12 PM tools and none of the librarian ride-alongs. | **Free and offline** — no LLM key, no network. |
| [`agent-loop.sh`](agent-loop.sh) | The REAL `deskkit agent` loop: an actual LLM autonomously choosing tools through the eino ReAct loop. Seeds fixtures that trip rules R1–R5, then proves the write boundary from both sides — that a run without `LIBRARIAN_AUTONOMOUS_WRITES` cannot mutate a file, and that a run with it lands a real fix that `deskkit restore` returns byte-exact (sha256-compared). Also drives a real multi-message `deskkit mcp-serve` protocol session. | **MAKES REAL, BILLED LLM CALLS.** Needs a real `ANTHROPIC_API_KEY`. |

## Running them

```
bash examples/pm-walkthrough.sh        # free; run this one freely
bash examples/agent-loop.sh            # BILLED — or: make example-agent-loop
```

Both `cd` to the repo root themselves, so they work from any directory. Both build the binary
first; set `DESKKIT_BIN=/path/to/deskkit` to reuse an existing build and skip that step.

Only `agent-loop.sh` has a Makefile target (`make example-agent-loop`, which builds first). Run
the PM one directly.

## A note on the name

These were once called `dogfood-*.sh`. That word collides with the repo's **"No in-repo
dogfooding"** rule (`CLAUDE.md`), which means something entirely different and still holds: this
repo does not register its own MCP servers on itself, and these harnesses do not point the binary
at this repo's tree — they build disposable desks instead.
