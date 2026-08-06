---
name: neutrality-lint-blindspots
description: desk-standard D8 identity-neutrality lint (scripts/check-neutrality.mjs) misses real identity tokens inside its own scan scope
metadata:
  type: project
---

The desk-standard D8 neutrality lint (`scripts/check-neutrality.mjs`, scans `plugin/` + `librarian/`)
has two structural blind spots that let real identities pass inside its stated scope:

1. **Person names** — the profile-value denylist is EMPTY in-repo (only `profile.example.yaml`
   placeholders exist), and there is no structural name detector. So a bare name like "Henry"
   in a scanned file passes. Found shipped in `plugin/README.md` ("(2026-07-16, Henry)").
2. **Qualified `org/repo#issue`** — bare hostless slugs are deliberately not matched (Go
   import-path collision), and the `#\d+` detector rejects a `#N` preceded by a word char
   (for `wb#37` shorthand). Combined, `hsb3/dotfiles-agents-workbench#50` passes (the `#50` is
   preceded by 'h'). Found shipped in `plugin/README.md` and root `README.md`.

**Why:** AC5's grep clause promises "no name, no hsb3/org, no owner/repo in the shipped plugin,"
but the automated lint only enforces URL/SSH-github, bare-`#N`, and `project N` structural forms
plus a (usually empty) profile denylist. The grep promise is stronger than the lint.

**How to apply:** when auditing neutrality, grep the scan scope yourself for
`hsb3|henry|burden|dotfiles-agents` — do not trust a green lint. Sanctioned cross-refs (the
wb#50 pointer) pass by coincidence, not via `schema/neutrality-lint.allow` — so the "sanctioned
escape" mechanism is bypassed.

**Exact scan scope (verified 2026-08-05, `check-neutrality.mjs:54`):
`SCAN_DIRS = ["plugin", "plugins", "librarian", "kits"]`, recursive, `.md`/`.txt` included.**
Inverse trap seen in issue bodies: a draft claimed `librarian/README.md` is neutrality-exempt as
"not shipped binary/plugin content" — FALSE, it is under `librarian/` so it IS scanned. Only
`docs/`, `scripts/`, root files, CHANGELOG are exempt. Do not accept "this librarian/ or plugin/
file is exempt" — verify against SCAN_DIRS, not prose.

**Update 2026-08-05:** the two README leaks found above have since been fixed (grep of
`plugin/README.md` is clean); the structural blind spots (empty denylist, word-char-prefixed
`#N` bypass) are the durable part.
