---
name: pocketbase-serve-swallows-runE-errors
description: librarian serve/superuser RunE errors exit 0 (PocketBase discards them); only main() os.Exit paths + the cmdErr wrapper give nonzero — serve is registered inside Start() so it escapes the wrapper
metadata:
  type: project
---

In `librarian/cmd/pocket-librarian/main.go`, PocketBase's `Execute()` runs `RootCmd.Execute()`
in a goroutine and **discards the RunE error** (upstream: "leave to the commands to decide").
The code compensates two ways, and each has a hole:

1. **cmdErr wrapper** (main.go ~160) wraps every `app.RootCmd.Commands()` RunE to record the
   first error and `os.Exit(1)` after Start(). BUT it runs *before* `app.Start()`, and
   **serve/superuser are registered by PocketBase inside Start()** — so they are NOT wrapped.
   The tool commands (sweep/query/restore/mcp-serve/gui) ARE wrapped and correctly exit 1.
2. **main() os.Exit(1)** for the store-LOCATION guard fires before app.Start(), covering serve.

**Consequence found on branch feat/23-xdg-store-home-desk-guard:** the ADR-0002 desk
open-guard on serve (an `OnServe().BindFunc` that returns the guard error) correctly refuses
to serve — the HTTP server never binds, e.Next() not called — but the process **exits 0**,
while the same guard via `requireConfig` on every tool command exits 1. Asymmetry: serve's
LOCATION guard exits 1 (main os.Exit), serve's OPEN guard exits 0 (OnServe error swallowed).
gui re-execs serve as a child and returns `child.Run()`, so it inherits the exit-0.

**How to apply:** when verifying any "serve refuses / fails closed" claim here, do NOT trust
that a printed `Error:` line means nonzero exit. Run serve in the foreground with a timeout
and check `$?` directly. An OnServe-returned error blocks serving but is invisible to exit-code
supervisors. A fix would route serve's block through os.Exit (or wrap serve's RunE after
Start registers it).
