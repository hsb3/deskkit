---
name: app-family-domain-split
description: "Owner's app family and domain split (2026-08-18): deskkit = documents + agents; task-manager = tasks (and the auth donor); Kaneo is a stopgap — don't frame deskkit features as Kaneo replacements"
metadata: 
  node_type: memory
  type: project
  originSessionId: 137c2321-7686-4ba8-90fb-1bd3a6a45a36
  modified: 2026-08-18T06:46:15.670Z
---

Henry is building a family of small-footprint apps on one pattern (single Go binary: cobra CLI +
embedded PocketBase + MCP module registry + embedded SPA; deploys to Railway). Kaneo is an explicit
**stopgap** while that tooling is built. Domain split, his framing verbatim (2026-08-18): deskkit is
"general documents with built in ai agents and access points for external ai agents"; **task-manager
= tasks**; dbagent and keel are siblings. Corrected me once for casting deskkit's SPA as a Kaneo/board
replacement — don't repeat that. task-manager is also the **auth donor**: its PB auth work is the
family template (DESK-51/52). Ruled on the board: DESK-45 (scope + PM freeze), DESK-46 (embedded
SPA), DESK-47 (family-wide frontend stack, proposed). Related: [[consolidation-single-binary-decision]].
