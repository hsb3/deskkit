_The SOP kit library — the desk's authoring templates, one kit per doc type._
Status: active

# kits/

**What this is.** 23 SOP "kits" — a `guide.md` (when to use it, how to fill it), a `template.md`
(the blank scaffold), and an `example.md` (a worked instance) per doc type. Ported from the
origin vault per desk decision 0013 S4(a) (the SOP-kit port work order) and indexed by
the root [`kits.yaml`](../kits.yaml) manifest. The frontmatter contract each kit emits lives in
[`schema/doctypes.yaml`](../schema/doctypes.yaml) (schema v1's doc-type dimension); the port and
its gap dispositions are recorded in ADR 0006.

**Structure.** Every kit has the guide/template/example (G/T/E) trio, except:

- **Composite kits** — `postmortem`, `release-notes` — ship guide + example only. They are
  assembled from other kits / the project record, not authored from a blank template, so the
  template is omitted **by design** (`composite: true` in the manifest).
- **`user-defined`** — nonstandard loose project-flavored type stubs (`decision`/`meeting`/`note`/
  `task`), no G/T/E trio. Its `meeting`/`note` types deliberately sit outside the canonical schema
  (see ADR 0006).

**Identity-neutral.** These are shipped artifacts: no personal names, orgs, repos, issues, or
private paths (0013 item 9). The worked examples use a neutral persona (Robin) and a neutral sample
product (Quillpad). `kits/` is inside the `scripts/check-neutrality.mjs` scan surface, so edits are
held to that bar.

**Drift-guarded.** `node scripts/check-kits.mjs` (in `make check` + CI) fails if `kits.yaml` and
this tree disagree — a missing/added file or an untracked/absent kit dir.

## Vault freeze (D4)

The originating vault copies (the origin vault's `shared/sops/` kit set) are the **historical source**
and are now **frozen**: kit and schema authoring for these doc types happens **here**, not in the
vault, which reverts to a read-only work journal (desk decision 0013 S4(a) precondition). To
change a kit or its frontmatter contract, edit `kits/` + `schema/doctypes.yaml` in this repo and
let the drift guard + neutrality lint hold the line — do not re-fork from the vault.
