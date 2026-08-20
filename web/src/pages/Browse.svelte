<script lang="ts">
  // The finder, and the instance it opens into. One component, three allocations of the screen:
  // while you are looking, the list IS the screen and no detail pane is reserved; once a row is
  // open the instance takes the canvas and the list minimises into the rail's lit F button;
  // editing is that same allocation with different verbs and never navigates.
  //
  // NOTHING in here knows which collection it is showing. Every question that used to be a
  // named-collection branch is a derivation in lib/collections.ts, so a second writable entity
  // is a config entry and no edit to this file — which is the whole point, and is asserted by
  // lib/collections.test.ts against a config this component has never seen, plus a scan of this
  // file for any collection name creeping back in.
  //
  // Every mouse target here has a key and vice versa — the keys arrive as intents on the shell's
  // action bus (App owns the keyboard; this component owns what rows and edits mean).
  import { onMount, tick } from 'svelte'
  import type { RecordModel } from 'pocketbase'
  import { pb } from '../lib/pb'
  import {
    adoptInto,
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
    type Column,
  } from '../lib/collections'
  import {
    doctypeNames,
    fetchDoctypes,
    isStrandedStatus,
    legalStatuses,
    statusChoices,
    type Doctypes,
  } from '../lib/doctypes'
  import { deleteDoc, writeDocField, type DocWriteResult } from '../lib/docwrite'
  import { editorHref, fetchResolvedSettings } from '../lib/settings'
  import Icon from '../lib/Icon.svelte'
  import type { Action } from '../lib/keys'
  import { MOD_LABEL, level, onAction, outOf, setLevel, stickyFinder } from '../lib/shell'

  let { config }: { config: BrowseConfig } = $props()

  let records = $state<RecordModel[]>([])
  let page = $state(1)
  let totalPages = $state(1)
  let error = $state('')
  let loading = $state(false)
  let selected = $state<RecordModel | null>(null)
  let childRows = $state<RecordModel[]>([])

  // Finder state: which row the keyboard is on, and what is being searched for.
  let cursor = $state(0)
  let query = $state('')
  let searchEl = $state<HTMLInputElement | null>(null)
  let formEl = $state<HTMLFormElement | null>(null)
  let bodyEl = $state<HTMLAnchorElement | null>(null)
  let listEl = $state<HTMLElement | null>(null)
  let searchTimer: ReturnType<typeof setTimeout> | undefined

  // Edit state. The draft is a map keyed by the config's editable fields — never one named
  // field — so the form is whatever the collection declares.
  let draft = $state<Record<string, string>>({})
  let saving = $state(false)
  let saveNote = $state('')
  let saveError = $state('')
  let conflict = $state<DocWriteResult | null>(null)
  /** Delete is two clicks in place, never a browser modal: a modal freezes the automation
   * harness this app is driven by, and it steals the screen for a question the button can ask
   * itself. Armed state is dropped by anything that changes what "this" means. */
  let deleteArmed = $state(false)

  // The document vocabulary and the desk's editor hand-off. Both are fetched once and both
  // degrade to nothing: a picker with no vocabulary becomes a text input, and a desk with no
  // editor_url simply has no Open-body verb.
  let vocab = $state<Doctypes | null>(null)
  let editorTemplate = $state('')
  let deskRoot = $state('')

  const PAGE_SIZE = 50
  const ed = $derived(config.edit ?? null)
  const editable = $derived(isEditable(config))
  const watchTarget = $derived(realtimeCollection(config))

  /** Combine the config's standing filter with the search box. Every term goes through
   * `pb.filter` with its own placeholder, so a typed quote is escaped rather than interpreted —
   * the field names are ours, the value never is. */
  function searchFilter(cfg: BrowseConfig, q: string): string | undefined {
    const term = q.trim()
    if (!term || !cfg.search?.length) return cfg.filter
    const params: Record<string, string> = {}
    const or = cfg.search
      .map((f, i) => {
        params[`q${i}`] = term
        return `${f} ~ {:q${i}}`
      })
      .join(' || ')
    const matched = pb.filter(`(${or})`, params)
    return cfg.filter ? `(${cfg.filter}) && ${matched}` : matched
  }

  async function load(cfg: BrowseConfig, p: number, q: string) {
    loading = true
    error = ''
    selected = null
    try {
      const res = await pb.collection(cfg.collection).getList(p, PAGE_SIZE, {
        sort: cfg.sort,
        filter: searchFilter(cfg, q),
        expand: cfg.expand,
      })
      records = res.items
      totalPages = Math.max(res.totalPages, 1)
      page = p
      cursor = 0
    } catch (e) {
      records = []
      error = e instanceof Error ? e.message : String(e)
    } finally {
      loading = false
    }
  }

  // A new collection is a new search: carrying the last one over would show an empty list that
  // looks like an empty collection.
  $effect(() => {
    query = ''
    load(config, 1, '')
  })

  /** Typing narrows the list as you go; the debounce is what keeps that from being one request
   * per keystroke. */
  function onQuery() {
    clearTimeout(searchTimer)
    searchTimer = setTimeout(() => load(config, 1, query), 200)
  }

  async function select(rec: RecordModel) {
    selected = rec
    childRows = []
    draft = draftOf(config, rec)
    saveNote = ''
    saveError = ''
    conflict = null
    deleteArmed = false
    const kids = config.children
    if (!kids) return
    try {
      const res = await pb.collection(kids.collection).getList(1, 200, {
        filter: childFilter(kids, rec.id),
        sort: kids.sort,
      })
      childRows = res.items
    } catch {
      // the related list is unavailable: the parent record still renders
    }
  }

  function scrollCursorIntoView() {
    tick().then(() => {
      listEl?.querySelectorAll('tbody')[cursor]?.scrollIntoView({ block: 'nearest' })
    })
  }

  /** j / k. From an open instance these walk the collection WITHOUT reopening the list — unless
   * the sticky-finder preference is off, in which case moving on hands the screen back to the
   * finder. That one setting is the whole difference between the two reading styles. */
  function move(step: number) {
    if (!records.length) return
    cursor = Math.min(Math.max(cursor + step, 0), records.length - 1)
    scrollCursorIntoView()
    if ($level === 'finder') return
    if ($stickyFinder) select(records[cursor])
    else setLevel('finder')
  }

  function openCursor() {
    const rec = records[cursor]
    if (!rec) return
    select(rec)
    setLevel('reading')
  }

  function openRow(rec: RecordModel, index: number) {
    cursor = index
    select(rec)
    setLevel('reading')
  }

  /** `e`. Editing is a level, not a screen: the same instance, different verbs. */
  async function startEdit() {
    if (!editable) return
    if (!selected) openCursor()
    if (!selected) return
    setLevel('editing')
    await tick()
    // The first control in the form, whatever kind the config made it. Found rather than bound,
    // because a `bind:this` inside the field loop would leave us holding the LAST one.
    const first = formEl?.querySelector<HTMLElement>('input, select')
    first?.focus()
    if (first instanceof HTMLInputElement) first.select()
  }

  /** ESC: exactly one level out, and the abandoned draft goes with the level it belonged to.
   * Leaving `editing` this way IS Revert — the same function backs the button, so there is one
   * behaviour rather than two that have to be kept agreeing. */
  function back() {
    const next = outOf($level)
    if ($level === 'editing' && selected) {
      draft = draftOf(config, selected)
      conflict = null
      saveError = ''
    }
    deleteArmed = false
    setLevel(next)
  }

  function handle(a: Action) {
    switch (a.kind) {
      case 'search':
        searchEl?.focus()
        searchEl?.select()
        break
      case 'next':
        move(1)
        break
      case 'prev':
        move(-1)
        break
      case 'open':
        if ($level === 'finder') openCursor()
        break
      case 'modify':
        startEdit()
        break
      case 'body':
        // Clicks the button rather than re-implementing it: the key and the control are then
        // literally the same path, which is the promise the rail makes about the whole map.
        bodyEl?.click()
        break
      case 'delete':
        // Arms on the first press, commits on the second — the SAME state the button toggles,
        // so there is one delete path and it is never a modal.
        remove()
        break
      case 'save':
        if ($level === 'editing' && selected) save(checksumOf(config, selected))
        break
      case 'back':
        back()
        break
    }
  }

  onMount(() => {
    fetchDoctypes().then((v) => (vocab = v))
    fetchResolvedSettings().then((r) => {
      editorTemplate = r?.editor_url?.value ?? ''
      deskRoot = r?.desk_root?.value ?? ''
    })
    return onAction(handle)
  })

  // ⌘B can bring the screen back to an instance before one has been chosen. Rather than showing
  // an empty pane, take the row the cursor is already on.
  $effect(() => {
    if ($level !== 'finder' && !selected && records.length) openCursor()
  })

  /** Fold a server-side version of a record into the pane and its list row. Fields are
   * assigned onto the existing proxies (never reassigned) so the realtime effect below
   * doesn't see a new identity and resubscribe on every event. */
  function applyIncoming(rec: RecordModel) {
    if (selected?.id === rec.id) {
      // Only adopt an incoming value into a field the operator has not typed into. Keyed by
      // RECORD columns, not frontmatter keys — see collections.adoptInto.
      draft = adoptInto(config, draft, selected, rec)
      Object.assign(selected, rec)
    }
    const row = records.find((r) => r.id === rec.id)
    if (row) Object.assign(row, rec)
  }

  // Live-follow the selected record where the config asks for it: the binary watches the desk
  // tree, so an edit made in another editor lands here. Older servers without realtime, and
  // collections that do not ask for it, degrade silently to a static pane.
  const watchId = $derived(watchTarget ? (selected?.id ?? '') : '')
  $effect(() => {
    const id = watchId
    const coll = watchTarget
    if (!id || !coll) return
    let stop: (() => void) | null = null
    let dropped = false
    pb.collection(coll)
      .subscribe(id, (e) => {
        if (e.record) applyIncoming(e.record)
      })
      .then((unsub) => {
        if (dropped) unsub()
        else stop = unsub
      })
      .catch(() => {
        /* no realtime on this server: the pane just stays as loaded */
      })
    return () => {
      dropped = true
      stop?.()
    }
  })

  /** Save the draft through the server's write-through route. `base` is the checksum the save
   * is staked on — the loaded one normally, the 409's fresh one when the operator explicitly
   * chooses to overwrite.
   *
   * A refusal (a write-protected path, a field the frontmatter editor will not take) arrives as
   * a thrown message and is shown as-is, with the record open and the draft untouched: the
   * server's reason is the answer, and there is no client-side guard here pre-empting it. */
  async function save(base: string) {
    const rec = selected
    const cfg = config.edit
    if (!rec || !cfg || saving) return
    saving = true
    saveNote = ''
    saveError = ''
    const set = writeSet(config, draft)
    try {
      const res = await writeDocField(pathOf(config, rec), base, set, pb.authStore.token || null)
      if (res.outcome === 'conflict') {
        conflict = res
        return
      }
      conflict = null
      applyIncoming({
        ...rec,
        ...recordPatch(config, draft),
        [cfg.checksumField]: res.checksum ?? rec[cfg.checksumField],
      } as RecordModel)
      saveNote = res.outcome === 'noop' ? 'Already saved — no change on disk.' : 'Saved to disk.'
      // A finished save is a finished edit: back to reading, the level ⌘↵ was invoked from.
      setLevel('reading')
    } catch (e) {
      saveError = e instanceof Error ? e.message : String(e)
    } finally {
      saving = false
    }
  }

  /** Delete: two clicks in place. The first arms the button, the second commits. Reversible on
   * the server — the original is recorded before the file goes — so the confirm is a guard
   * against a slip, not against loss. */
  async function remove() {
    const rec = selected
    const cfg = config.edit
    if (!rec || !cfg?.deletable || saving) return
    if (!deleteArmed) {
      deleteArmed = true
      return
    }
    saving = true
    saveNote = ''
    saveError = ''
    try {
      const res = await deleteDoc(
        pathOf(config, rec),
        checksumOf(config, rec),
        pb.authStore.token || null,
      )
      if (res.outcome === 'conflict') {
        conflict = res
        deleteArmed = false
        return
      }
      records = records.filter((r) => r.id !== rec.id)
      cursor = Math.min(cursor, Math.max(records.length - 1, 0))
      selected = null
      deleteArmed = false
      setLevel('finder')
    } catch (e) {
      saveError = e instanceof Error ? e.message : String(e)
      deleteArmed = false
    } finally {
      saving = false
    }
  }

  /** Conflict resolution "Reload": adopt what the server has now and drop the edit. */
  async function reload() {
    const rec = selected
    if (!rec) return
    try {
      const fresh = await pb.collection(config.collection).getOne(rec.id)
      draft = draftOf(config, fresh)
      applyIncoming(fresh)
      conflict = null
      saveNote = ''
      saveError = ''
    } catch (e) {
      saveError = e instanceof Error ? e.message : String(e)
    }
  }

  function cell(rec: RecordModel, col: Column): string {
    if (col.expandKey) {
      const rel = (rec.expand as Record<string, RecordModel> | undefined)?.[col.key]
      return rel ? String(rel[col.expandKey] ?? '') : ''
    }
    const v = rec[col.key]
    if (v === null || v === undefined) return ''
    return typeof v === 'object' ? JSON.stringify(v) : String(v)
  }

  function shortFields(rec: RecordModel): [string, string][] {
    const edited = [...editedKeys(config), ...sourceKeys(config)]
    return Object.entries(rec)
      .filter(([k]) => !['expand', 'collectionId', 'collectionName'].includes(k))
      .filter(([k]) => !config.detailBlocks.includes(k))
      .filter(([k]) => !edited.includes(k))
      .map(([k, v]) => [k, typeof v === 'object' && v !== null ? JSON.stringify(v) : String(v ?? '')])
  }

  // The status picker answers to the DRAFTED doctype, not the saved one: changing a document's
  // type changes its status family, and the legal list has to follow in the same interaction.
  const doctype = $derived(selected ? currentDoctype(config, selected, draft) : '')
  const legal = $derived(legalStatuses(vocab, doctype))
  const types = $derived(doctypeNames(vocab))
  const bodyHref = $derived(
    ed?.bodyHandoff && selected ? editorHref(editorTemplate, deskRoot, pathOf(config, selected)) : '',
  )
</script>

<div class="browse">
  {#if $level === 'finder'}
    <div class="list" bind:this={listEl}>
      <div class="head">
        <h2>{config.title}</h2>
        {#if config.search?.length}
          <input
            class="search"
            type="search"
            bind:this={searchEl}
            bind:value={query}
            oninput={onQuery}
            placeholder={`Search — ${MOD_LABEL}K`}
            aria-label={`Search ${config.title}`}
          />
        {/if}
        <div class="spacer"></div>
        {#if totalPages > 1}
          <div class="pager">
            <button disabled={page <= 1 || loading} onclick={() => load(config, page - 1, query)}>‹</button>
            <span>{page} / {totalPages}</span>
            <button disabled={page >= totalPages || loading} onclick={() => load(config, page + 1, query)}>›</button>
          </div>
        {/if}
      </div>
      {#if error}
        <p class="error">{error}</p>
      {:else if !loading && records.length === 0}
        <p class="empty">{query ? 'Nothing matches that.' : 'Nothing here yet.'}</p>
      {:else}
        <!-- One <tbody> per record so the preview line can sit full-width UNDER its row and
             still highlight, hover and scroll as one thing. Rows are fat on purpose: this
             allocation reserves no detail pane, so the list itself has to be enough to
             recognise a document by. -->
        <table>
          <thead>
            <tr>
              {#each config.columns as col}<th>{col.label}</th>{/each}
            </tr>
          </thead>
          {#each records as rec, i (rec.id)}
            {@const preview = previewOf(config, rec)}
            <tbody class:on={i === cursor}>
              <tr aria-selected={i === cursor} onclick={() => openRow(rec, i)}>
                {#each config.columns as col}
                  <td class="cell">{cell(rec, col)}</td>
                {/each}
              </tr>
              {#if preview}
                <tr class="preview-row" onclick={() => openRow(rec, i)}>
                  <td colspan={config.columns.length}><p class="preview">{preview}</p></td>
                </tr>
              {/if}
            </tbody>
          {/each}
        </table>
        <p class="hint">j / k walk the rows · ↵ opens · {MOD_LABEL}K searches</p>
      {/if}
    </div>
  {:else if selected}
    <div class="instance">
      <div class="instance-head">
        <!-- The level-out control. In editing it IS Revert: same function, same key, one
             behaviour rather than two that have to be kept agreeing. -->
        <button class="verb" onclick={back}>
          <Icon name={$level === 'editing' ? 'revert' : 'back'} />
          {$level === 'editing' ? 'Revert' : config.title} — esc
        </button>
        <h3>{titleOf(config, selected)}</h3>
        <div class="spacer"></div>
        {#if editable && $level === 'reading'}
          <button class="verb" onclick={startEdit}><Icon name="modify" /> Modify — e</button>
        {/if}
        {#if bodyHref && $level === 'reading'}
          <!-- The body is never editable in the browser. This hands it to where the operator
               actually writes, via the desk's own editor_url template — an anchor the OS
               resolves, nothing shelled out. A desk that declares no editor gets no button. -->
          <a class="verb" bind:this={bodyEl} href={bodyHref}><Icon name="body" /> Open body — o</a>
        {/if}
        {#if ed?.deletable}
          <button class="verb danger" class:armed={deleteArmed} disabled={saving} onclick={remove}>
            <Icon name="remove" />
            {deleteArmed ? 'Really delete? — again to confirm' : `Delete — ${MOD_LABEL}⌫`}
          </button>
        {/if}
        <span class="walk">j / k for the next one</span>
      </div>

      {#if ed}
        <form
          class="edit"
          bind:this={formEl}
          onsubmit={(e) => {
            e.preventDefault()
            if (selected) save(checksumOf(config, selected))
          }}
        >
          {#each ed.fields as f (f.key)}
            {@const cur = draft[f.key] ?? ''}
            {@const stranded = f.kind === 'status' && isStrandedStatus(legal, cur)}
            <div class="field">
              <label for={`f-${f.key}`}>{f.label}</label>
              {#if $level !== 'editing'}
                <span class="value" class:stranded>{cur || '—'}</span>
              {:else if f.kind === 'status' && legal}
                <select id={`f-${f.key}`} bind:value={draft[f.key]} disabled={saving}>
                  {#each statusChoices(legal, cur) as opt (opt)}
                    <option value={opt}>{opt}{opt === cur && stranded ? ' — not legal for this type' : ''}</option>
                  {/each}
                </select>
              {:else if f.kind === 'doctype' && types}
                <select id={`f-${f.key}`} bind:value={draft[f.key]} disabled={saving}>
                  {#if cur && !types.includes(cur)}
                    <option value={cur}>{cur} — not a known type</option>
                  {/if}
                  {#each types as t (t)}<option value={t}>{t}</option>{/each}
                </select>
              {:else}
                <!-- Free text: either the field is declared free, or the vocabulary could not be
                     reached. A picker with nothing in it and no way to type is worse than both. -->
                <input id={`f-${f.key}`} type="text" bind:value={draft[f.key]} disabled={saving} />
              {/if}
              {#if stranded}
                <span class="stranded-note">
                  “{cur}” is not a status this type allows. Left exactly as it is — pick a
                  replacement, or put the type back.
                </span>
              {/if}
            </div>
          {/each}
          {#if $level === 'editing'}
            <button class="primary" type="submit" disabled={saving}>
              <Icon name="save" />
              {saving ? 'Saving…' : `Save — ${MOD_LABEL}↵`}
            </button>
          {/if}
        </form>
        {#if saveError}<p class="error">{saveError}</p>{/if}
        {#if saveNote}<p class="success">{saveNote}</p>{/if}
        {#if conflict}
          <div class="conflict">
            <p class="note">
              This file changed on disk since it was loaded, so nothing was saved. Reload to take
              the version below, or overwrite it with your edit.
            </p>
            <pre class="preview-block">{conflict.current_content ?? ''}</pre>
            <div class="actions">
              <button type="button" onclick={reload} disabled={saving}>Reload</button>
              <button
                type="button"
                onclick={() => save(String(conflict?.current_checksum ?? ''))}
                disabled={saving}
              >
                Overwrite anyway
              </button>
            </div>
          </div>
        {/if}
      {/if}

      <dl>
        {#each shortFields(selected) as [k, v]}
          {#if v !== ''}
            <dt>{k}</dt>
            <dd>{v}</dd>
          {/if}
        {/each}
      </dl>
      {#each config.detailBlocks as key}
        {#if selected[key]}
          <h4>{key}</h4>
          <pre>{selected[key]}</pre>
        {/if}
      {/each}
      {#if config.children}
        {@const kids = config.children}
        {#if childRows.length}
          <h4>{kids.collection}</h4>
          {#each childRows as row (row.id)}
            <div class="child">
              {#each kids.columns as col (col.key)}
                {@const v = cell(row, col)}
                {#if v !== ''}
                  <span class="child-label">{col.label}</span>
                  <pre>{v}</pre>
                {/if}
              {/each}
            </div>
          {/each}
        {/if}
      {/if}
    </div>
  {/if}
</div>

<style>
  .browse {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }
  .list,
  .instance {
    flex: 1;
    overflow: auto;
    padding: 1rem;
    min-height: 0;
  }
  .head,
  .instance-head {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    margin-bottom: 0.6rem;
    flex-wrap: wrap;
  }
  .spacer {
    flex: 1;
  }
  h2 {
    margin: 0;
    font-size: 1rem;
  }
  .search {
    font-size: 0.8rem;
    padding: 0.25rem 0.5rem;
    width: min(22rem, 40%);
  }
  .pager {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.8rem;
    color: var(--muted);
  }
  tbody tr {
    cursor: pointer;
  }
  tbody:hover {
    background: color-mix(in srgb, var(--accent) 8%, transparent);
  }
  tbody.on {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
    box-shadow: inset 2px 0 0 var(--accent);
  }
  .cell {
    max-width: 22rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .preview-row td {
    padding-top: 0;
    border-top: 0;
  }
  .preview {
    margin: 0;
    color: var(--muted);
    font-size: 0.76rem;
    line-height: 1.35;
    /* Two lines, then ellipsis: a long synopsis truncates rather than reflowing the row into a
       paragraph and losing the list's scannability. */
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .empty,
  .error,
  .hint,
  .walk {
    color: var(--muted);
    font-size: 0.8rem;
  }
  .error {
    color: var(--error);
    max-width: 60rem;
    word-break: break-word;
  }
  .hint {
    margin-top: 0.8rem;
  }
  .verb {
    display: inline-flex;
    align-items: center;
    gap: 0.32rem;
    font-size: 0.78rem;
    padding: 0.25rem 0.6rem;
    border-radius: 0.35rem;
    text-decoration: none;
    color: inherit;
    border: 1px solid var(--border);
    background: transparent;
    line-height: 1.2;
  }
  .verb:hover {
    background: color-mix(in srgb, var(--accent) 12%, transparent);
  }
  .verb.danger:hover,
  .verb.armed {
    border-color: var(--error);
    color: var(--error);
    background: color-mix(in srgb, var(--error) 12%, transparent);
  }
  h3 {
    margin: 0;
    font-size: 0.85rem;
    color: var(--muted);
    font-weight: 500;
  }
  dl {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 0.2rem 0.8rem;
    font-size: 0.8rem;
    max-width: 60rem;
  }
  dt {
    color: var(--muted);
  }
  dd {
    margin: 0;
    word-break: break-word;
  }
  h4 {
    margin: 1rem 0 0.3rem;
    font-size: 0.8rem;
    color: var(--muted);
  }
  pre {
    white-space: pre-wrap;
    word-break: break-word;
    font-size: 0.78rem;
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 0.5rem;
    padding: 0.6rem;
    margin: 0 0 0.5rem;
    max-width: 60rem;
  }
  .edit {
    display: flex;
    align-items: flex-start;
    flex-wrap: wrap;
    gap: 0.4rem 0.9rem;
    margin: 0.6rem 0 0.4rem;
    font-size: 0.8rem;
  }
  .field {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    flex-wrap: wrap;
  }
  .edit label {
    color: var(--muted);
  }
  .edit input,
  .edit select {
    min-width: 0;
    font-size: 0.8rem;
    padding: 0.3rem 0.45rem;
  }
  .edit input {
    flex: 0 1 18rem;
  }
  .edit .value {
    font-weight: 600;
  }
  .edit .value.stranded {
    color: var(--error);
  }
  .stranded-note {
    color: var(--error);
    font-size: 0.74rem;
    flex: 1 0 100%;
    max-width: 40rem;
  }
  .edit button {
    display: inline-flex;
    align-items: center;
    gap: 0.32rem;
    font-size: 0.8rem;
    padding: 0.3rem 0.7rem;
  }
  .success {
    color: var(--accent);
    font-size: 0.8rem;
    margin: 0 0 0.4rem;
  }
  .conflict .note {
    margin: 0 0 0.4rem;
    font-size: 0.8rem;
    color: var(--muted);
  }
  .conflict .preview-block {
    max-height: 14rem;
    overflow: auto;
  }
  .conflict .actions {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }
  .conflict .actions button {
    font-size: 0.8rem;
    padding: 0.3rem 0.7rem;
  }
  .child {
    border-left: 2px solid var(--border);
    padding-left: 0.6rem;
    margin-bottom: 0.5rem;
  }
  .child-label {
    font-size: 0.75rem;
    color: var(--muted);
    display: block;
    margin: 0.2rem 0 0.15rem;
  }
</style>
