# opencode/ — frozen spike, not shipped

Status: **descoped 2026-07-16**. v1 of this plugin targets Claude Code only.

These files are a preserved partial build of a hand-written OpenCode adapter
(Plugin-typed module + skills-copy installer + tests). The build was stopped
mid-wave when the approach was superseded: OpenCode support will come from a
common-core fan-out that *produces* both harness instances at build time, not
from a maintained adapter. See the "Scope of this build (v1)" section of the
root `README.md` for the tracking pointer.

Nothing here is wired into the package scripts, the manifest, or CI. The code
typechecks and its tests pass, so it is safe to keep in-tree as reference for
the fan-out design — treat it as an artifact of record, not a surface to extend.
