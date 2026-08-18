_The SOP kit library — the desk's authoring templates, one kit per doc type._
Status: active

# kits/

**What this is.** One SOP "kit" per doc type — a `guide.md` (when to use it, how to fill it), a
`template.md` (the blank scaffold), and an `example.md` (a worked instance) — indexed by the root
[`kits.yaml`](../kits.yaml) manifest. Kits were ported into this repo per ADR 0006; kit and schema
authoring happens here now, not anywhere upstream — don't fork these files from elsewhere.

**Structure.** Every kit has the guide/template/example (G/T/E) trio, except:

- **Composite kits** ship guide + example only, no template — they're assembled from other kits
  or the project record rather than authored from a blank scaffold, so the template is omitted
  **by design**. Flagged `composite: true` in the manifest.
- **`user-defined`** — nonstandard, loose project-flavored type stubs, no G/T/E trio. Flagged
  `nonstandard: true` in the manifest.

**Frontmatter contract.** Each kit emits frontmatter conforming to
[`schema/doctypes.yaml`](../schema/doctypes.yaml) — the doc-type dimension of schema v1: universal
fields, per-type required/optional fields, and the enums and status families a filled-in doc must
satisfy. A kit's `type:` (or `emits:`, for a kit whose output type differs from its dir name) must
name an entry defined there.

**Identity-neutral.** `kits/` — this file included — is inside the
[`scripts/check-neutrality.mjs`](../scripts/check-neutrality.mjs) scan surface, so any prose or
example you add here is held to the same no-hardcoded-identity bar as the rest of the shipped
tree: no person, org, or repo name, and no bare issue reference. Cite a decision as a bare
`ADR NNNN` only.

## The manifest and the drift guard

[`kits.yaml`](../kits.yaml) is the single ordered index of `kits/`: one `- dir: <name>` block per
kit dir, each with `type:` and a one-line `files: [a.md, b.md, ...]` list. `node
scripts/check-kits.mjs` (wired into `make check` + CI) fails if the manifest and the on-disk tree
disagree in either direction: a manifest entry whose dir or listed file is missing on disk, or a
kit dir / `.md` file on disk that no manifest entry names.

The guard's parser only reads three keys per entry — `dir:`, `type:`, and `files:` — and all three
are required (a missing `type` or an empty `files` list is reported as drift, same as a
tree/manifest mismatch). `emits:`, `composite: true`, and `nonstandard: true` are documentation
for humans and any future manifest consumer; the parser skips lines it doesn't recognize, so it
doesn't enforce them — get one wrong and the gate stays green, but a reader (or a future tool)
relying on the manifest will be misled. Set them correctly anyway.

**Re-run it:** `node scripts/check-kits.mjs` — bare, not piped. Clean tree prints one OK line with
the live kit/file counts (trust that number over anything written in this file — it's the guard's
own count, not a copy). A failing run prints one of:

```
check-kits: FAIL — N manifest/tree drift(s):
  <dir>: manifest kit dir is missing on disk (kits/<dir>/)
  <dir>/<file>: listed in manifest but missing on disk
  <dir>/<file>: present on disk but not listed in the manifest
  kits/<dir>/: kit dir on disk not named in the manifest
```

— search for `check-kits: FAIL` to land back here.

### The blocklist check

The same run also scans every text file (`.md`, `.yaml`, `.yml`, `.json`, `.txt`) under `kits/`
and `schema/`, recursively — this README included — for a fixed list of literal strings tied to
this kit set's pre-repo origin (personal-name fragments, an absolute home-directory path prefix, a
couple of legacy project/directory names). It's a backstop for exactly what the identity-neutrality
lint's profile-derived denylist can't see on its own, since that denylist only knows placeholder
values, not a leftover string from before the port. The live list lives in `scripts/check-kits.mjs`
(`BLOCKLIST_CI` / `BLOCKLIST_CS`) — deliberately not reproduced here, so this file doesn't trip the
check it's documenting. A hit prints:

```
check-kits: FAIL — N origin-vault blocklist hit(s) in kits + schema/:
  <path>:<line>: blocked term "<term>"
```

## Adding a kit

1. Create `kits/<name>/` with its files — normally `guide.md`, `template.md`, `example.md`; omit
   `template.md` only for a composite kit.
2. Add a matching `- dir: <name>` block to `kits.yaml`: `type:` (the `schema/doctypes.yaml` entry
   it emits — add `emits:` too if that differs from `<name>`), `composite: true` or
   `nonstandard: true` if applicable, and `files:` listing exactly what you created.
3. If the kit emits a doc `type` not already defined in `schema/doctypes.yaml`, that file needs
   the new type first — the guard doesn't check this, so a kit can point at an undefined type
   without the gate noticing.
4. Land the new dir and the new manifest block in the same change (order between them doesn't
   matter to the guard, only that both exist by the time it runs), then run `node
   scripts/check-kits.mjs` and `node scripts/check-neutrality.mjs`, both bare.

## Editing a kit

- Content-only edits to an existing `guide.md` / `template.md` / `example.md` (filename unchanged)
  need no manifest change.
- Renaming, adding, or removing a file inside a kit dir requires updating that kit's `files:` list
  to match, in the same change.
- Renaming the kit dir itself requires updating `dir:` and moving the directory together.
- Changing what a kit emits requires updating `type:`/`emits:` in `kits.yaml` and confirming the
  target type exists in `schema/doctypes.yaml`.

## Removing a kit

Delete `kits/<name>/` and its whole `- dir: <name>` block in `kits.yaml` in the same change, then
run `node scripts/check-kits.mjs` — deleting only one side leaves the other flagged as drift.

## Status

`kits/` is frozen: the ported set is settled, no further additions/removals/renames planned short
of a post-1.0 expansion (run the guard for the live kit/file count). This file documents the
mechanics for whenever that reopens.
