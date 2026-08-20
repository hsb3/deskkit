// Browse configuration per collection, and the pure derivations the finder reads off it.
//
// This file is the falsifiable half of the CRUD template: adding a second writable entity has
// to be an entry here and NOTHING else. Every behaviour that used to be an `=== 'files'` branch
// inside the component is a function below, so it can be asserted against a config the component
// has never seen.
//
// Browse never writes PocketBase rows directly: the edits it offers go through the server's
// write-through routes (POST /desk/doc/write, POST /desk/doc/delete — record-original-first,
// reversible via `restore`), which own the disk write and the row update together. Field lists
// mirror the Go migrations — unknown fields are simply not rendered, so a schema drift degrades,
// not breaks.

export interface Column {
  key: string
  label: string
  /** Render a related record's field instead (requires `expand` on the config). */
  expandKey?: string
}

export interface EditableField {
  /** The FRONTMATTER key the write route sets. */
  key: string
  label: string
  /** 'text' = free input. 'status' = family-aware picker driven by the record's doctype.
   * 'doctype' = picker over the served doctype vocabulary. Both pickers degrade to a text
   * input when the vocabulary is unavailable — never an empty select with no way to type. */
  kind: 'text' | 'status' | 'doctype'
  /** The record COLUMN that seeds this field's draft, when it differs from the frontmatter key.
   * Documents are the live case: frontmatter says `type`, the indexed column is `doctype`
   * (renamed in migration 0019). Getting this wrong is not a cosmetic bug — a draft seeded from
   * a column that does not exist reads as '' and saves as '', erasing the value on disk. */
  from?: string
}

/** The record column a field's value actually lives in. The one place `from` is resolved, so
 * nothing outside this file ever repeats the expression that caused that erasure. */
function sourceOf(f: EditableField): string {
  return f.from ?? f.key
}

export interface ChildList {
  collection: string
  /** `{id}` is substituted with the parent record's id. */
  filter: string
  sort?: string
  columns: Column[]
}

export interface EditConfig {
  /** Frontmatter keys the browser may set, through the write-through route. */
  fields: EditableField[]
  /** Record field carrying the desk-relative path the write route addresses. */
  pathField: string
  /** Record field carrying the CAS checksum. */
  checksumField: string
  /** Offer Delete (reversible via `restore`). */
  deletable?: boolean
  /** Offer "open the body where I write it" — only renders if the desk declares editor_url. */
  bodyHandoff?: boolean
}

export interface BrowseConfig {
  collection: string
  title: string
  columns: Column[]
  sort?: string
  filter?: string
  expand?: string
  /** Fields worth a full-width block in the detail pane (long prose/content). */
  detailBlocks: string[]
  /** Fields the finder's search box matches, OR'd together. Omit and the box is not rendered:
   * a search that silently matches nothing is worse than no search. */
  search?: string[]
  /** Which record field titles the instance pane. Replaces the hardcoded path/title/id chain. */
  titleField: string
  /** Field shown under each finder row — why rows are fat enough to recognise a doc by. */
  preview?: string
  /** Absent = a read-only collection. Present = the CRUD verbs, on these fields. */
  edit?: EditConfig
  /** Live-update open records from this collection's realtime feed. */
  realtime?: boolean
  /** A related list rendered under the detail blocks. */
  children?: ChildList
}

/** A record as this module reads one: a bag of unknown fields with an id. Declared structurally
 * so the derivations are testable without PocketBase's RecordModel. */
export type Rec = Record<string, unknown>

// --- derivations: everything the component used to hardcode --------------------------------

/** Writable at all? The one question that used to be `config.collection === 'files'`. */
export function isEditable(cfg: BrowseConfig): boolean {
  return cfg.edit != null
}

/** Which collection's realtime feed to follow for an open record, or null for none. */
export function realtimeCollection(cfg: BrowseConfig): string | null {
  return cfg.realtime ? cfg.collection : null
}

/** What titles the instance pane. Falls back to the id — every record has one, so the header is
 * never blank no matter how wrong the config's `titleField` is. */
export function titleOf(cfg: BrowseConfig, rec: Rec): string {
  const v = rec[cfg.titleField]
  const s = v === null || v === undefined ? '' : String(v)
  return s || String(rec.id ?? '')
}

/** The one-line preview under a finder row, or '' when this collection declares none. */
export function previewOf(cfg: BrowseConfig, rec: Rec): string {
  if (!cfg.preview) return ''
  const v = rec[cfg.preview]
  if (v === null || v === undefined) return ''
  return (typeof v === 'object' ? JSON.stringify(v) : String(v)).trim()
}

/** Frontmatter keys this collection lets the browser set. Empty for a read-only collection. */
export function editedKeys(cfg: BrowseConfig): string[] {
  return cfg.edit ? cfg.edit.fields.map((f) => f.key) : []
}

/** The edit form's starting state: every configured field, as a string, from the record. */
export function draftOf(cfg: BrowseConfig, rec: Rec): Record<string, string> {
  const d: Record<string, string> = {}
  for (const f of cfg.edit?.fields ?? []) d[f.key] = String(rec[sourceOf(f)] ?? '')
  return d
}

/** The record columns the edit fields read from — what the detail list has to hide, so a value
 * never appears twice under two names. */
export function sourceKeys(cfg: BrowseConfig): string[] {
  return (cfg.edit?.fields ?? []).map(sourceOf)
}

/** The same draft keyed by RECORD columns, for the optimistic local update after a save. The
 * wire wants frontmatter keys (`writeSet`); the loaded record wants its own columns. Keeping
 * the two apart is the whole point — folding a frontmatter-keyed patch onto a record invents a
 * phantom column and leaves the real one stale. */
export function recordPatch(cfg: BrowseConfig, draft: Record<string, string>): Record<string, string> {
  const p: Record<string, string> = {}
  for (const f of cfg.edit?.fields ?? []) p[sourceOf(f)] = draft[f.key] ?? ''
  return p
}

/** Fold a newer version of the record into a draft, leaving alone any field the operator has
 * already typed into. `before` is the record the draft was seeded from — a field still equal to
 * what it was seeded with is one nobody has touched. */
export function adoptInto(
  cfg: BrowseConfig,
  draft: Record<string, string>,
  before: Rec,
  after: Rec,
): Record<string, string> {
  const next = { ...draft }
  for (const f of cfg.edit?.fields ?? []) {
    const src = sourceOf(f)
    if (next[f.key] === String(before[src] ?? '')) next[f.key] = String(after[src] ?? '')
  }
  return next
}

/** The `set` map POSTed to the write route: only the configured keys, so a stale draft entry
 * from a previously-selected collection can never ride along into someone's frontmatter. */
export function writeSet(cfg: BrowseConfig, draft: Record<string, string>): Record<string, string> {
  const set: Record<string, string> = {}
  for (const k of editedKeys(cfg)) set[k] = draft[k] ?? ''
  return set
}

/** The record field the write route addresses, and the checksum the save is staked on. */
export function pathOf(cfg: BrowseConfig, rec: Rec): string {
  return cfg.edit ? String(rec[cfg.edit.pathField] ?? '') : ''
}

export function checksumOf(cfg: BrowseConfig, rec: Rec): string {
  return cfg.edit ? String(rec[cfg.edit.checksumField] ?? '') : ''
}

/** Which doctype the status picker is answering to. The DRAFTED type wins over the saved one:
 * changing a document's type changes its status family, and the legal statuses have to follow
 * in the same interaction rather than after a save. */
export function currentDoctype(cfg: BrowseConfig, rec: Rec, draft: Record<string, string>): string {
  const f = cfg.edit?.fields.find((x) => x.kind === 'doctype')
  if (!f) return ''
  return String(draft[f.key] ?? rec[sourceOf(f)] ?? '')
}

/** A child list's filter with `{id}` bound to the parent. The id is quoted into the expression,
 * so quote and backslash are escaped rather than interpreted — PocketBase ids are alphanumeric,
 * but nothing here should depend on that staying true. */
export function childFilter(child: ChildList, id: string): string {
  return child.filter.replaceAll('{id}', id.replaceAll('\\', '\\\\').replaceAll('"', '\\"'))
}

export const browsePages: Record<string, BrowseConfig> = {
  documents: {
    collection: 'files',
    title: 'Documents',
    sort: 'path',
    filter: 'deleted = false',
    columns: [
      { key: 'path', label: 'Path' },
      { key: 'doctype', label: 'Doctype' },
      { key: 'dir_kind', label: 'Dir' },
      { key: 'status', label: 'Status' },
    ],
    detailBlocks: ['synopsis', 'content'],
    search: ['path', 'synopsis'],
    titleField: 'path',
    preview: 'synopsis',
    realtime: true,
    edit: {
      // `tags` is deliberately absent: it is a YAML block array on most docs and the write path
      // takes single-line scalars, so the field would be a box asking for raw YAML.
      fields: [
        { key: 'status', label: 'Status', kind: 'status' },
        // Frontmatter key `type`, indexed column `doctype` (migration 0019 renamed it).
        { key: 'type', label: 'Type', kind: 'doctype', from: 'doctype' },
      ],
      pathField: 'path',
      checksumField: 'checksum',
      deletable: true,
      bodyHandoff: true,
    },
  },
  findings: {
    collection: 'patrol_findings',
    title: 'Findings',
    expand: 'file',
    columns: [
      { key: 'rule', label: 'Rule' },
      { key: 'severity', label: 'Severity' },
      { key: 'state', label: 'State' },
      { key: 'disposition', label: 'Disposition' },
      { key: 'file', label: 'File', expandKey: 'path' },
    ],
    detailBlocks: ['detail', 'proposed_fix', 'reason'],
    search: ['rule', 'detail'],
    titleField: 'rule',
    preview: 'detail',
  },
  runs: {
    collection: 'agent_runs',
    title: 'Agent runs',
    sort: '-created',
    columns: [
      { key: 'created', label: 'When' },
      { key: 'trigger', label: 'Trigger' },
      { key: 'status', label: 'Status' },
      { key: 'run_label', label: 'Label' },
      { key: 'model', label: 'Model' },
      { key: 'step_count', label: 'Steps' },
    ],
    detailBlocks: ['input_summary', 'output_summary', 'error'],
    search: ['run_label', 'input_summary'],
    titleField: 'run_label',
    preview: 'input_summary',
    children: {
      collection: 'messages',
      filter: 'run = "{id}"',
      sort: 'seq',
      columns: [
        { key: 'seq', label: '#' },
        { key: 'role', label: 'Role' },
        { key: 'tool_name', label: 'Tool' },
        { key: 'content', label: 'Content' },
        { key: 'tool_calls', label: 'Tool calls' },
      ],
    },
  },
  pm: {
    collection: 'items',
    title: 'PM items',
    sort: '-updated',
    columns: [
      { key: 'title', label: 'Title' },
      { key: 'type', label: 'Type' },
      { key: 'phase', label: 'Phase' },
      { key: 'blocked', label: 'Blocked' },
      { key: 'court', label: 'Court' },
      { key: 'priority', label: 'Priority' },
      { key: 'updated', label: 'Updated' },
    ],
    detailBlocks: ['body'],
    search: ['title', 'body'],
    titleField: 'title',
    preview: 'body',
  },
}
