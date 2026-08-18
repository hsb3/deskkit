_OS-level isolation for deskkit's initial and supervised runs._
Status: active

## What this is

deskkit's initial and supervised runs execute inside an OS-level sandbox (spec
§10.5). This is **belt-and-suspenders** isolation around the record-original-first write
boundary (decision 0014) during the early autonomous-write period: it constrains what the
*process* can touch regardless of a tool bug. It is isolation only — it does **not** change
the agent harness, the tool logic, or the §5.4 write gate; the same binary runs, just
fenced in.

Two profiles are provided. Both are **owner-overridable**: run the binary directly (no
sandbox, no container) whenever you don't need the extra fence.

## When it's required

- **Local supervised runs** (an operator watching `apply-fix` or an agent run against a real
  desk) — use the macOS default, `deskkit.sb` via `run-sandboxed.sh`.
- **CI / portable runs** — use the Docker alternative, `docker-run.sh`.
- Routine read-only commands (`sweep`, `patrol`, `query`, `restore`) don't need the fence —
  they're the safe, always-run-anywhere path. The fence matters for anything that can write
  (`apply-fix`, an autonomous agent run with `LIBRARIAN_AUTONOMOUS_WRITES=true`).

## Default — macOS `sandbox-exec`

`deskkit.sb` (verbatim from the spec) default-denies everything, then allows only:

- filesystem read/write inside three subtrees: `DESK_ROOT` (the desk being stewarded),
  `PB_DATA` (the PocketBase SQLite data dir), and `BIN_DIR` (the binary's own directory);
- outbound network to the configured LLM provider host only (`api.anthropic.com` by
  default), plus DNS.

The provider host is **derived from the provider's base URL**, not hardcoded — it
substitutes when the provider swaps (`api.openai.com` for OpenAI,
`generativelanguage.googleapis.com` for Gemini). Pass the right value via
`PROVIDER_HOSTPORT` when it changes.

Run it:

```bash
DESK_ROOT=/path/to/a/desk DESK_NAME=my-desk \
  ./run-sandboxed.sh apply-fix --run <run_id>
# or: ./run-sandboxed.sh serve --http=127.0.0.1:8090
```

**Adaptation note:** the spec's illustrative invocation nests `PB_DATA` under `DESK_ROOT`
(`$DESK_ROOT/pb_data`). This build's actual default is PocketBase's own convention — a
`pb_data/` directory relative to wherever the binary is invoked from (see
`../.gitignore`), i.e. normally the repo root, not under `DESK_ROOT`.
`run-sandboxed.sh` defaults `PB_DATA` to that real location; override it (or pass `--dir`
to the binary) if your layout differs.

## Portable / CI alternative — Docker

`docker-run.sh` bind-mounts `DESK_ROOT` read-write at the same path inside the container
(so config/paths stay identity-neutral) and joins an egress-restricted user-defined network
(`deskkit-egress` by default) whose firewall policy permits only the provider host
+ DNS — Docker doesn't filter egress per-host natively, so the network (or an allow-list
`HTTPS_PROXY`, noted inline in the script) is what does the restricting. Create the network
once before first use:

```bash
docker network create deskkit-egress
# then configure that network's firewall/egress policy to allow only
# $PROVIDER_HOST:443 + DNS
```

Run it:

```bash
DESK_ROOT=/path/to/a/desk DESK_NAME=my-desk \
  ./docker-run.sh apply-fix --run <run_id>
```

`PROVIDER_HOST` in the Docker egress policy is the same base-URL-derived host as the macOS
profile's `PROVIDER_HOSTPORT` — a provider swap updates one allow-list entry in both places.
