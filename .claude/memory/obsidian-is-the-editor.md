---
name: obsidian-is-the-editor
description: "Henry writes deskkit desk prose in Obsidian, so deskkit ships no prose editor and outside edits are the normal case"
metadata:
  node_type: memory
  type: project
  modified: 2026-08-19T00:00:00.000Z
---

Asked directly on 2026-08-19 whether a desk is a repository deskkit operates on or an app whose
data happens to be exportable, Henry answered: "primarily do my writing using obsidian."

**Consequences already folded into the SPA design:**

- **ADR 0009 (files-are-truth) stands.** Database-first was weighed and refused. The store stays
  disposable and rebuildable; only the machine-native collections (findings, runs, PM items,
  revisions, settings) are lost with it.
- **Edits from outside deskkit are the primary workflow, not an edge case.** Change detection
  needs a watcher, not sweep-on-demand, and the save path must refuse to overwrite a file that
  moved underneath it.
- **deskkit ships no prose editor.** A document's editable surface is its structured part
  (type, status, frontmatter); the body gets an "open where I write" hand-off. The editor
  command belongs in the desk's `_knowledge/profile.yaml`, never hardcoded — a hardcoded editor
  is a personalization and personalization does not ship in the binary.

**Why it matters:** two earlier design rounds drew an in-app authoring screen and an editable
prose box. Both were speculative and are now deleted. Related: [[design-shape-before-content]].
