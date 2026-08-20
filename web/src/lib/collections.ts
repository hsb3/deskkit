// Browse configuration per collection. Browse never writes PocketBase rows directly:
// the one edit it offers goes through the server's write-through route (POST
// /desk/doc/write — record-original-first, reversible via `restore`), which owns the
// disk write and the row update together. Field lists mirror the Go migrations —
// unknown fields are simply not rendered, so a schema drift degrades, not breaks.

export interface Column {
  key: string
  label: string
  /** Render a related record's field instead (requires `expand` on the config). */
  expandKey?: string
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
      { key: 'synopsis', label: 'Synopsis' },
    ],
    detailBlocks: ['synopsis', 'content'],
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
      { key: 'detail', label: 'Detail' },
    ],
    detailBlocks: ['detail', 'proposed_fix', 'reason'],
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
      { key: 'input_summary', label: 'Input' },
    ],
    detailBlocks: ['input_summary', 'output_summary', 'error'],
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
  },
}
