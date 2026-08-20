import { describe, expect, it } from 'vitest'
// Vite's `?raw` rather than node:fs: this repo's tsconfig types are vite/client only, and the
// bundler resolving the path is one less thing to keep in step with a file move.
import browseSource from '../pages/Browse.svelte?raw'
import {
  adoptInto,
  browsePages,
  checksumOf,
  childFilter,
  currentDoctype,
  draftOf,
  editedKeys,
  isEditable,
  pathOf,
  previewOf,
  realtimeCollection,
  recordPatch,
  sourceKeys,
  titleOf,
  writeSet,
  type BrowseConfig,
} from './collections'

// The falsifiable test of the whole CRUD template: a second writable collection, invented here,
// that no component has ever heard of. Every question the finder asks about a record — is this
// editable, what do I subscribe to, what titles it, what goes in the write payload, where is the
// path and the checksum — is answered from this config alone. If making these pass ever needs a
// component edit, the generalization is not done.
const notes: BrowseConfig = {
  collection: 'notes',
  title: 'Notes',
  sort: 'slug',
  columns: [{ key: 'slug', label: 'Slug' }],
  detailBlocks: ['body'],
  search: ['slug'],
  titleField: 'slug',
  preview: 'summary',
  realtime: true,
  edit: {
    fields: [
      { key: 'phase', label: 'Phase', kind: 'text' },
      { key: 'kind', label: 'Kind', kind: 'doctype' },
      { key: 'state', label: 'State', kind: 'status' },
    ],
    pathField: 'file_path',
    checksumField: 'sha',
    deletable: true,
    bodyHandoff: true,
  },
  children: {
    collection: 'comments',
    filter: 'note = "{id}"',
    sort: 'seq',
    columns: [{ key: 'text', label: 'Text' }],
  },
}

const note = {
  id: 'n1',
  slug: 'weekly',
  summary: '  a running note  ',
  phase: 'open',
  kind: 'journal',
  state: 'draft',
  file_path: 'notes/weekly.md',
  sha: 'abc123',
  body: 'long text',
}

describe('a second writable collection, derived from config alone', () => {
  it('is editable because it declares an edit block, not because of its name', () => {
    expect(isEditable(notes)).toBe(true)
    expect(isEditable(browsePages.findings)).toBe(false)
    expect(isEditable(browsePages.pm)).toBe(false)
  })

  it('subscribes to its own collection, and only when it asks to', () => {
    expect(realtimeCollection(notes)).toBe('notes')
    expect(realtimeCollection({ ...notes, realtime: false })).toBeNull()
    expect(realtimeCollection(browsePages.findings)).toBeNull()
  })

  it('titles the instance from titleField, falling back to the id', () => {
    expect(titleOf(notes, note)).toBe('weekly')
    expect(titleOf(notes, { id: 'n2' })).toBe('n2')
    expect(titleOf(notes, { id: 'n3', slug: '' })).toBe('n3')
  })

  it('builds the draft and the write payload from the declared fields only', () => {
    expect(editedKeys(notes)).toEqual(['phase', 'kind', 'state'])
    expect(draftOf(notes, note)).toEqual({ phase: 'open', kind: 'journal', state: 'draft' })
    // A key left over from another collection's draft never rides along into frontmatter.
    expect(writeSet(notes, { phase: 'shut', kind: 'journal', state: 'draft', stray: 'x' })).toEqual({
      phase: 'shut',
      kind: 'journal',
      state: 'draft',
    })
    // A field the record simply has no value for still goes as the empty string, not undefined.
    expect(draftOf(notes, { id: 'n4' })).toEqual({ phase: '', kind: '', state: '' })
  })

  it('addresses the write route by the configured path and checksum fields', () => {
    expect(pathOf(notes, note)).toBe('notes/weekly.md')
    expect(checksumOf(notes, note)).toBe('abc123')
    expect(pathOf(browsePages.findings, note)).toBe('')
  })

  it('shows a preview line per row, trimmed, and none when none is configured', () => {
    expect(previewOf(notes, note)).toBe('a running note')
    expect(previewOf({ ...notes, preview: undefined }, note)).toBe('')
    expect(previewOf(notes, { id: 'n5' })).toBe('')
  })

  it('binds a child list to its parent, escaping the id into the expression', () => {
    expect(childFilter(notes.children!, 'n1')).toBe('note = "n1"')
    expect(childFilter(notes.children!, 'a"b')).toBe('note = "a\\"b"')
  })

  it('reads the DRAFTED doctype, so changing the type re-derives the status family at once', () => {
    expect(currentDoctype(notes, note, { kind: 'journal' })).toBe('journal')
    expect(currentDoctype(notes, note, { kind: 'decision' })).toBe('decision')
    // No draft entry (not editing the type) falls back to what is on the record.
    expect(currentDoctype(notes, note, {})).toBe('journal')
    expect(currentDoctype(browsePages.pm, note, {})).toBe('')
  })
})

describe('the shipped pages', () => {
  it('gives every page a titleField, since it replaces the old hardcoded chain', () => {
    for (const [name, cfg] of Object.entries(browsePages)) {
      expect(cfg.titleField, name).toBeTruthy()
    }
  })

  it('makes documents the writable one, on status and type, addressed by path + checksum', () => {
    const docs = browsePages.documents
    expect(isEditable(docs)).toBe(true)
    expect(realtimeCollection(docs)).toBe(docs.collection)
    expect(editedKeys(docs)).toEqual(['status', 'type'])
    // `tags` is a YAML block array; the write path takes single-line scalars, so it is not here.
    expect(editedKeys(docs)).not.toContain('tags')
    expect(docs.edit?.pathField).toBe('path')
    expect(docs.edit?.checksumField).toBe('checksum')
    expect(docs.edit?.deletable).toBe(true)
    expect(docs.edit?.bodyHandoff).toBe(true)
    expect(docs.edit?.fields.find((f) => f.key === 'status')?.kind).toBe('status')
    expect(docs.edit?.fields.find((f) => f.key === 'type')?.kind).toBe('doctype')
    // The frontmatter key is `type`; the indexed column is `doctype` (migration 0019).
    expect(docs.edit?.fields.find((f) => f.key === 'type')?.from).toBe('doctype')
    expect(sourceKeys(docs)).toEqual(['status', 'doctype'])
  })

  it('expresses the agent runs side-fetch as a child list rather than a special case', () => {
    const kids = browsePages.runs.children
    expect(kids?.sort).toBe('seq')
    expect(kids?.columns.length).toBeGreaterThan(0)
    expect(childFilter(kids!, 'r7')).toContain('r7')
  })

  it('leaves the read-only pages read-only', () => {
    for (const name of ['findings', 'runs', 'pm']) {
      expect(isEditable(browsePages[name]), name).toBe(false)
      expect(editedKeys(browsePages[name]), name).toEqual([])
    }
  })
})

// The other half of "adding a writable entity is a config entry and nothing else": the finder
// must not learn a collection's name again. This scans the component rather than trusting the
// review that removed them.
describe('Browse.svelte knows no collection by name', () => {
  const source = browseSource
  /** Asserts on a boolean, not on the string: `.not.toContain` on a 600-line component prints
   * the whole file into the failure. */
  const absent = (needle: string) => expect(source.includes(needle), `found ${needle}`).toBe(false)

  it('mentions no collection name and no per-collection branch', () => {
    for (const cfg of Object.values(browsePages)) {
      absent(`'${cfg.collection}'`)
      absent(`"${cfg.collection}"`)
    }
    absent("'messages'")
    absent('.collection ===')
  })

  it('reaches no record field by a literal name the config is supposed to own', () => {
    for (const holder of ['rec', 'selected', 'row', 'fresh']) {
      for (const field of ['path', 'checksum', 'status', 'doctype', 'type', 'tags']) {
        absent(`${holder}.${field}`)
      }
    }
    absent('statusDraft')
    absent('runMessages')
  })

  it('opens no browser modal — one would freeze the harness that drives this app', () => {
    for (const bad of ['confirm(', 'alert(', 'prompt(']) absent(bad)
  })

  // Every key has a button: the two verbs that gained keys have to be reachable by mouse too,
  // and their buttons have to say which key. Delete's key must drive the SAME armed state the
  // button toggles rather than a second path.
  it('gives the keyed verbs an on-screen control that names its key', () => {
    expect(source).toContain('Open body — o')
    expect(source).toContain('Delete — ${MOD_LABEL}⌫')
    expect(source).toContain("case 'body':")
    expect(source).toContain("case 'delete':")
    // One delete path: the key case calls the same function the button's onclick does.
    expect(source).toContain('onclick={remove}')
    expect(source.match(/deleteArmed = true/g)?.length).toBe(1)
  })
})

// A `files` row exactly as PocketBase hands one over: the doctype lives in the `doctype`
// COLUMN and there is no `type` key on the record at all. Seeding the Type field from `type`
// reads '' and then SAVES '', erasing the document's type in its frontmatter on disk — on the
// app's most common action, a status change. These are the tests that catch that.
const fileRow = {
  id: 'f1',
  path: '_knowledge/decisions/0007.md',
  doctype: 'decision',
  status: 'proposed',
  checksum: 'sha-1',
  synopsis: 'why we did it',
}

describe('a field whose frontmatter key differs from its record column', () => {
  const docs = browsePages.documents

  it('seeds the draft from the COLUMN, so the Type picker opens on the real value', () => {
    expect('type' in fileRow).toBe(false)
    expect(draftOf(docs, fileRow)).toEqual({ status: 'proposed', type: 'decision' })
  })

  it('never sends an empty value for a field the record actually had one for', () => {
    // The data-loss case: open a document, change only the status, save.
    const draft = { ...draftOf(docs, fileRow), status: 'accepted' }
    const set = writeSet(docs, draft)
    expect(set).toEqual({ status: 'accepted', type: 'decision' })
    for (const [k, v] of Object.entries(set)) expect(v, `${k} sent empty`).not.toBe('')
  })

  it('writes the wire under the frontmatter key and the record under its column', () => {
    const draft = draftOf(docs, fileRow)
    expect(Object.keys(writeSet(docs, draft))).toEqual(['status', 'type'])
    expect(recordPatch(docs, draft)).toEqual({ status: 'proposed', doctype: 'decision' })
    // No phantom `type` on the record, and the real column is the one that moves.
    expect(recordPatch(docs, { ...draft, type: 'analysis' }).doctype).toBe('analysis')
    expect(recordPatch(docs, draft)).not.toHaveProperty('type')
  })

  it('reads the DRAFTED type through the field, with no separate doctypeField to disagree', () => {
    expect(currentDoctype(docs, fileRow, draftOf(docs, fileRow))).toBe('decision')
    expect(currentDoctype(docs, fileRow, { type: 'analysis' })).toBe('analysis')
    // No draft yet (nothing opened): fall back to the column, never to a missing key.
    expect(currentDoctype(docs, fileRow, {})).toBe('decision')
  })

  it('adopts an incoming column into an untouched field and leaves a typed one alone', () => {
    const draft = draftOf(docs, fileRow)
    const fresh = { ...fileRow, doctype: 'analysis', status: 'accepted' }
    expect(adoptInto(docs, draft, fileRow, fresh)).toEqual({ status: 'accepted', type: 'analysis' })
    // Typed into: the operator's edit survives the realtime event.
    const typed = { ...draft, type: 'journal' }
    expect(adoptInto(docs, typed, fileRow, fresh)).toEqual({ status: 'accepted', type: 'journal' })
  })

  it('hides the column as well as the frontmatter key, so no value shows up twice', () => {
    expect(sourceKeys(docs)).toContain('doctype')
    expect(editedKeys(docs)).toContain('type')
  })
})
