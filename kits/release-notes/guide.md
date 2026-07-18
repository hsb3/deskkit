---
type: sop
status: final
created: 2026-06-30
updated: 2026-06-30
tags: [meta]
---

# SOP — release-notes

> **This is a Composite Work Doc SOP.** It owns `guide.md` + `example.md` only — **there is no `template.md`**. Release notes aren't authored from a blank scaffold; they're *assembled* from documents that already exist in the project's record. This guide tells you which sources to compose and how. See **## References** below for the inputs.

## When to write one

Write release notes whenever you cut a release — a version tag, a deploy that users will notice, a milestone you want to announce. One note per release, named by version. The output note carries `type: release-notes` and a `version` field (the universal required fields plus `version`; status family is `cadence` → `draft` while assembling, `published` once it ships).

**Red flag:** if you can't point to the previous release's notes, you don't yet know your range. Find the last published `release-notes` (or the last tag) first — everything in this note is "what changed *since then*."

## How to write one

Release notes are a **synthesis**, not a transcript. The work is reading the project's record for the period and turning it into a user-facing story of what they get.

### 1. Gather what shipped since the last release
Collect the period's record from the sources in **## References**:
- The **feature-specs** marked shipped/done in the range — these are the headline changes.
- The **meeting-notes / decisions** for the period — what was decided, what changed direction, what was deliberately cut.
- The **roadmap** items completed — confirms scope and lets you frame value in the user's language.
- The **commit history / changelog** between the two tags — the ground truth for fixes and small changes the specs don't cover.

Bound the range explicitly: `<last release tag>..HEAD` (or the dates of the two releases). Anything outside the range belongs to a different note.

### 2. Group by theme
Don't list commits in commit order. Cluster the gathered changes into a few themes a user would recognize ("Capture", "Editor", "Auth"). A theme with one line is fine; a theme that needs a paragraph probably hides two.

### 3. Write user-facing Added / Changed / Fixed
Three sections, each framed by **what the user can now do**, not what the code does:
- **Added** — new capabilities. Lead with the benefit, name the feature second.
- **Changed** — behavior that's different now (including removals and renames). Say what to expect.
- **Fixed** — bugs resolved, described as the symptom the user saw, not the internal cause.

Omit empty sections. Cite the source note for non-obvious entries (`see [[feature-spec …]]`) so a reader can go deeper.

### 4. Call out breaking changes and upgrade notes
Anything that requires the user to *act* — a migration, a config change, a removed option, a re-auth — gets its own **Upgrade notes** callout near the top, not buried in Changed. If there are none, say so; silence reads as "I forgot."

## References

This SOP **composes** these existing doc types — it does not redefine them. The note is assembled *from*:

| Source | What it contributes |
|---|---|
| `feature-spec` | The shipped features for the period — the headline Added/Changed entries. |
| `meeting-notes` | What was discussed and agreed during the cycle; context for direction changes. |
| `decision` | Ratified decisions that changed scope or behavior — the *why* behind a Changed entry. |
| `roadmap` | Which planned items landed; frames value in the user's language and confirms scope. |
| commit history / changelog | Ground-truth diff between the two release tags — fixes and small changes the specs miss. |

If one of these is missing for the period, note the gap rather than inventing the entry — the changelog is the fallback floor.

## Anti-patterns

- **Changelog dump** — pasting `git log` (or commit subjects) as the notes. Commits are an input, not the output; the reader wants value, not SHAs.
- **Internal framing** — "Refactored the extraction pipeline" tells a user nothing. Say what they can now do or what stopped breaking.
- **Buried breaking changes** — an upgrade-required item hidden in a Changed bullet. It earns its own callout up top.
- **Empty-section padding** — a "Fixed" header with "n/a" under it. Omit the section.
- **No range** — notes that don't anchor to the previous release leak last cycle's changes into this one.

## Example

See `example.md` in this folder for a worked release-notes doc — the Quillpad 0.3.0 release.
