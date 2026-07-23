---
type: readme
status: active
created: 2026-07-21
updated: 2026-07-21
tags: [model-simulations, probes]
synopsis: Reproducible scripted probes for the #126 v1-half model simulations, each building and tearing down its own throwaway scratch desk.
---

*Scripted probes backing the #126 model-simulation walkthroughs. Each builds a fresh `mktemp`
scratch desk with a hermetic `XDG_DATA_HOME` (the `librarian/verify.sh` pattern) and cleans up on
exit. None touches a real desk or the operator's real store.*

Status: active (2026-07-21)

## Run

```
go build -o /tmp/deskkit ./librarian/cmd/deskkit        # from the repo root
bash probes/<probe>.sh /tmp/deskkit
```

Captured outputs from the run of record are under `output/`.

| Probe | Backs | Output |
|---|---|---|
| `probe-s1-librarian.sh` | Scenario 1 — sweep→patrol→propose→apply→restore + disposition surface | `output/out-s1.txt` |
| `probe-s2-pm.sh` | Scenario 2 — PM gated transition + cascade, gate refusals | `output/out-s2.txt` |
| `probe-s3-greenfield.sh` | Scenario 3 — greenfield store tail (first sweep/patrol/PM item) | `output/out-s3.txt` |
| `probe-s4-brownfield.sh` | Scenario 4 — brownfield Phase 8 librarian baseline | `output/out-s4.txt` |
| `verify-frictions.sh` | F1 (cascade actor) + O4 (un-frontmattered file → orphans) | `output/out-frictions.txt` |
| `verify-d3.sh`, `verify-d3c.sh`, `verify-d3d.sh` | D2 — create-vs-update type-validation asymmetry; untyped-item gate bypass | `output/out-d3.txt`, `out-d3c.txt`, `out-d3d.txt` |

These probes live under `_meta/` and are **not** part of the shipped tree — they add no script
under `librarian/` or `scripts/`, so per #126's gate menu they trigger no product gate. They use
`--actor operator` and touch no deployment identity.
