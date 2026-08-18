_The reusable shape behind the container build: one Go binary, one embedded store, one config
ladder, one auth model. Written for a sibling application to copy the shape, not this repo's
history of building it._
Status: active

# The single-binary pattern

## The shape

**cobra CLI + embedded PocketBase + an MCP module registry + an embedded web surface + a thin
plugin face — all compiled into one Go binary.** No shared library is extracted from this repo;
a sibling application copies the *shape* (the same five ingredients, wired the same way in its
own `main.go`), it does not import a framework or vendor this binary's packages.

- **cobra** gives the CLI its command tree (`init`, `sweep`, `patrol`, `serve`, `mcp-serve`,
  `chat`, `superuser`, …).
- **Embedded PocketBase** (`CGO_ENABLED=0`, pure-Go SQLite driver) is the store: no separate
  database process, no network hop between the binary and its data. `serve` is a PocketBase
  app with the binary's own routes and migrations layered on top.
- **An MCP module registry** exposes the same tool core over `mcp-serve` (stdio) that the CLI
  and the chat surface call directly — one implementation, three callers.
- **An embedded web surface** (`go:embed`) ships a browser page inside the binary, so serving it
  needs no separate frontend build or toolchain at runtime.
- **A thin plugin face** (a Claude Code plugin bundle) wires an agent harness to the same MCP
  server; it carries no runtime of its own.

Reusable takeaway: put the domain logic behind one tool core, then grow CLI / MCP / web / plugin
as thin callers over it, instead of maintaining parallel implementations per surface.

## The config ladder

Two separate resolution chains; do not conflate them.

**Value precedence** (an individual setting — LLM provider, model, superuser email, …):

```
env > per-desk profile (_knowledge/profile.yaml) > central config ($XDG_CONFIG_HOME/<app>/config.yaml) > default
```

The central config is the one place a secret (an LLM API key) may be stored at rest on the
operator's own machine; it sits below the per-desk profile so a desk can still override it.

**Store/desk location** (where the embedded database and the desk's files live), first match
wins:

1. explicit `--dir` (overrides everything)
2. `DESK_ROOT` + `DESK_NAME` env vars
3. a profile discovered by walking up from the current directory
4. an unresolvable identity falls back to `$XDG_DATA_HOME/<app>/<name>/`
5. still unresolvable and no `--dir` → refuse to start (exit 1), never guess

**A container configures entirely through env plus `--dir`/an explicit store path — no XDG
assumptions.** There is no home directory to discover a walked-up profile from, and no
`$XDG_DATA_HOME` an operator has set up by hand; the entrypoint must name `DESK_ROOT` /
`DESK_NAME` / the store location explicitly rather than relying on any leg of the ladder that
depends on a real user environment.

## The auth model

The load-bearing section.

### Public mode is derived, never opted into

Whether the binary hardens itself is **derived from every resolved listen address**, not from a
separate flag — a server that opens two listeners (an HTTP one and an HTTPS one) is classified
on both, because the framework only surfaces one of them on its own startup event and a naive
read of that single value can miss an exposed second listener. Loopback (`127.0.0.1`,
`localhost`, `::1`, or empty meaning the dependency's own default) is today's local behavior,
unchanged. Anything else — a wildcard bind, a bare `:PORT` with no host, a routable IP, or a
hostname that cannot be proven loopback — makes the *whole process* **public mode**, because the
hardening (auth prerequisites, route auth, CORS) is a per-process posture that cannot apply to
half a server. An address that cannot be proven loopback classifies as public: fail closed.

Why derive it instead of a `--public` opt-in flag: a flag can be forgotten while the process
still binds a wildcard address, which fails **open**. A mode computed from the bind addresses
themselves cannot be forgotten, because it isn't a second thing to remember — it's read off the
values that already determine exposure.

### Public mode refuses to start without a superuser — and verifies it, doesn't predict it

In public mode, `serve` checks its auth prerequisites **before opening the listener** — a
refusal means nothing binds. This gate went through two real bypasses during hardening; both are
fixed now, and both are worth carrying into a sibling verbatim because the class of bug recurs.

**Bypass 1 — a security check counted a row that is not an account.** The naive gate was "does
the store hold at least one superuser record?" The embedded store's own first-run installer
writes a placeholder superuser row purely to mint a one-time setup link — it carries no password
anyone holds. That row satisfies a plain count, so one ordinary local `serve` (which triggers the
installer) permanently disarmed the public gate for that store: it would later boot publicly with
zero *administrable* accounts. The fix excludes that specific placeholder identity from the
count, mirroring the check the framework's own installer uses internally for the same reason.
**Reusable lesson: the framework can seed rows that satisfy your security check without being a
real account. Count administrable accounts, not rows.**

**Bypass 2 — the gate predicted an end state instead of verifying it.** The first fix still
passed the gate whenever the superuser email/password env vars were merely *non-empty*, trusting
a later, non-fatal bootstrap call to turn them into a real account. That bootstrap call could
fail (a password the store's validation rejects, say) without stopping the boot — the failure
landed as a log line in a database table, not on stderr, and the public listener came up anyway
with no administrable account at all. The fix: on the public path, `serve` now provisions the
superuser account **fatally, right there**, and then **re-counts** administrable superusers
before proceeding — only a count greater than zero lets the listener open. **Reusable lesson: a
fail-closed gate must verify the end state, not predict it** — checking the inputs that are
*supposed to* produce a safe state is not the same as checking that the state was produced.

Neither prerequisite met → the process exits nonzero and prints why, before any port is open.

Setting **exactly one** of the superuser email/password env vars is a loud fatal error, in
**every** mode, loopback included — an auth environment that is almost right must fail loudly,
never degrade quietly. Both left unset stays a silent no-op on a loopback bind; that is normal
local UX and unchanged.

Superuser bootstrap is **idempotent and never upserts**: if an account with that email already
exists, its password is left alone. An upsert would rotate the account's token-signing key and
invalidate every live session token — a real incident in a sibling project (2026-08). Rotate a
superuser password through the admin console or a dedicated command, never by re-setting the
boot env var and restarting.

### Nil collection rules are fail-closed — confirm it, don't assume it

Every domain collection in this codebase's store had, going in, **nil** API rules on every verb.
In this store's engine, a nil rule means superuser-only, so nothing needed a lockdown migration
before serving publicly — this was **measured** (every domain collection returned 403
unauthenticated), not taken on faith. Reusable lesson: check what "no rule set" means in your
store's engine, and verify it live before trusting it — don't just read the docs and assume your
collections inherited the safe default.

### The framework's own stock auth collection is not safe by default

The embedded store ships its own stock end-user auth collection out of the box, already present
in every store before any application migration runs. Its stock posture is **open signup with
immediate login** — measured on the unmodified binary: a brand-new, unverified signup got a 200
on signup and a 200 on login. That is a live hole the moment the process binds a public
interface, and it comes from the dependency, not from application code that's easy to spot in
review. A migration on that collection adds an operator-held approval flag (a new field,
defaulting to unset/false) and makes both "email verified" and "approved" a precondition of
authenticating; the create rule blocks a client from setting the approval field on itself, even
to `false`; delete is superuser-only (the stock rule let a user delete their own record, which
under an approval gate would let a rejected account churn straight back into a fresh signup).
**Audit every auth collection your framework ships you before exposing it — the stock defaults
are tuned for a trusted network, not the open internet.**

Two operational corollaries worth stating plainly, not discovering in an incident:

- The approval field's default means the migration **locks out every account that existed before
  it ran**, not just future signups — an account added under the old open-signup posture cannot
  authenticate again until an operator approves it. Warn whoever runs this migration on a store
  with real accounts already in it; it reads as an outage if nobody said so first.
- De-approving a member (`approved = false`) is **not an immediate kill switch**. The auth rule
  is evaluated when a token is minted, not on every request that uses it, so a token issued
  before de-approval keeps working until it naturally expires. Revoking access immediately needs
  a separate mechanism (e.g. rotating that record's own credential); flipping the flag alone
  only stops *future* logins.

Because no outbound mail is configured, "email verified" cannot be satisfied by the user alone;
in practice an operator with superuser rights flips both the verified and approved flags by hand
in the admin console. That is the real approval workflow until outbound mail is wired up, which
would let self-service verification work.

### What public mode changes on the embedded web surface

The browser session surface's three routes go from unauthenticated (loopback default) to
requiring a valid token from the end-user or superuser auth collection, and a same-origin guard
on the state-changing routes switches from a loopback-origin allowlist to strict same-origin
(the request's `Origin` must match its own `Host`, checked in application code — a separate
mechanism from CORS headers below).

**CORS is a two-mode split, and it caught a real trap.** The embedded store's own default
middleware answers every route with a wildcard `Access-Control-Allow-Origin` unless something
unbinds it — that default shipped on the public path too, until this was caught and fixed. The
landed behavior: on a public bind, that default middleware is left unbound only when it is still
carrying the framework's wildcard default — an operator-supplied explicit allowlist flag is
detected and preserved instead of being thrown away along with the wildcard, so a deployment that
adds a separate frontend origin sets that allowlist and it is honored, never silently dropped. A
self-contradictory allowlist (a bare wildcard mixed with explicit origins in the same value) is
refused outright on a public bind rather than resolved either way — a security instruction that
contradicts itself is not a preference to guess at, and discarding half of it silently is the same
defect class as everything else in this section; loopback stays exempt since local development is
not a trust boundary. With no explicit allowlist, no `Access-Control-Allow-Origin` header is
emitted at all (the browser surface is served same-origin and needs no cross-origin allowlist).
Side effect worth knowing before you copy this: unbinding that middleware also removes its
`OPTIONS` handler, so a cross-origin preflight on a public bind with no allowlist answers 404
instead of 204 — which blocks the cross-origin request, the intended outcome, but is a status
change and not a bug report. On a loopback bind, the framework's stock wildcard is deliberately
left as-is, unchanged, matching the rule that local `serve` behavior stays byte-for-byte what it
always was. The reusable lesson for a sibling application: **a framework's permissive default can
survive your own auth hardening silently**, because the header comes from the framework's own
middleware, not from a line you wrote or a test you can see failing. Verify security headers
against a live response, not against your own diff.

Accuracy trap: the admin console's SPA shell is static HTML and returns 200 to a stranger — that
is expected, not an opening. Every API call behind that shell is still superuser-gated and
returns 403 unauthenticated. Describe the exposure precisely (shell loads, data doesn't), not as
one status code.

### Two accepted limitations

1. Because the browser session page requires an auth token and a plain browser navigation sends
   no `Authorization` header, **a bare browser visit to the hosted page returns 401.** The page
   is reachable only by a client that sets the header itself. Making it directly browsable needs
   a cookie- or token-bootstrap flow — a separate design decision, not part of this shape.
2. Self-service email verification does not work without outbound mail configured (see above).
   Approval is a manual admin-console step until SMTP is wired up.

## The deploy recipe

**Multi-stage image**: a full Go build stage compiles the static (`CGO_ENABLED=0`) binary,
version-stamped via an `-ldflags -X` from the repo's version file; the runtime stage is a small
general-purpose base image carrying CA certificates (for outbound calls), a shell (for the
entrypoint script), and a healthcheck client — not a distroless/scratch base, which has none of
the three.

**The entrypoint never bakes a desk into the image.** It runs the binary's own `init` against
the desk root only if that desk's profile is absent, then `exec`s `serve` with an explicit
`--http 0.0.0.0:<port>` and an explicit store directory — so a redeploy on the same volume finds
the existing desk and store rather than re-scaffolding.

**One volume, two things on it**: the desk's file tree and the embedded store's database both
live under a single mounted volume, so one volume is the entire unit of persistence. A container
restart against the same volume — same data, same accounts, nothing re-initialized.

**Known, accepted limitations of this image** (state them, don't paper over them):

- Runs as **root** inside the container. A fresh named volume is root-owned and the image has no
  init system to chown-then-drop-privileges without adding a privilege-drop helper — accepted as
  a stated trade-off, not a recommendation.
- Built and smoke-tested on one CPU architecture locally; the Dockerfile itself is
  architecture-neutral (no hardcoded arch tag), but treat an untested architecture as unverified
  until it's actually built there.
- Base images are referenced by **mutable tags**, not a pinned digest. This repo pins every CI
  action by commit SHA but has no equivalent rule for container base images yet — a known gap,
  stated as one, not as a pattern to copy.

**Env list for a hosted service:**

Names below are this binary's; a sibling substitutes its own. None of them carries a deployment
identity, so they are safe to copy verbatim.

| Var | Required? | Purpose |
|---|---|---|
| `PORT` | optional (default `8090`) | bind port; most hosts inject this |
| `PB_SUPERUSER_EMAIL` + `PB_SUPERUSER_PASSWORD` | **required together** for a public bind, unless the store already holds a superuser | first-run superuser account; `serve` refuses to start without one or the other. Setting exactly one is a fatal error |
| `DESK_ROOT` | optional (image default `/data/desk`) | desk file tree location |
| `DESK_NAME` | optional (image default `desk`) | desk identity for the config resolver |
| `STORE_DIR` | optional (entrypoint default `/data/store`) | embedded store's database location |
| `LLM_PROVIDER`, `LLM_MODEL`, `LLM_API_KEY_ENV` | optional | consulted only when the chat surface is actually used |
| `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / `GEMINI_API_KEY` | optional (required for a working chat) | provider API key, read at chat-session construction, not at boot |
| `PM_ENABLED` | optional (default on) | turns the work-graph module on or off |

## Verifying the shape on a new host

Before trusting a deploy of this shape anywhere: build the image, run it against a real named
volume, and assert by HTTP status code — health endpoint 200, an unauthenticated request to a
protected collection 401/403, a superuser-authenticated request 200, an unapproved self-signup
refused, approval flipping it to allowed, and a restart against the same volume keeping the
data. Script it once and keep the transcript; a security posture asserted in prose without a
runnable check against it is not verified, it's hoped.
