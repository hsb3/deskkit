<script lang="ts">
  // The finder, and the instance it opens into. One component, three allocations of the screen:
  // while you are looking, the list IS the screen and no detail pane is reserved; once a row is
  // open the instance takes the canvas and the list minimises into the rail's F button; editing
  // is that same allocation with different verbs and never navigates.
  //
  // Every mouse target here has a key and vice versa — the keys arrive as intents on the shell's
  // action bus (App owns the keyboard; this component owns what rows and edits mean).
  import { onMount, tick } from 'svelte'
  import type { RecordModel } from 'pocketbase'
  import { pb } from '../lib/pb'
  import type { BrowseConfig } from '../lib/collections'
  import { writeDocField, type DocWriteResult } from '../lib/docwrite'
  import type { Action } from '../lib/keys'
  import { MOD_LABEL, level, onAction, outOf, setLevel, stickyFinder } from '../lib/shell'

  let { config }: { config: BrowseConfig } = $props()

  let records = $state<RecordModel[]>([])
  let page = $state(1)
  let totalPages = $state(1)
  let error = $state('')
  let loading = $state(false)
  let selected = $state<RecordModel | null>(null)
  let runMessages = $state<RecordModel[]>([])

  // Finder state: which row the keyboard is on, and what is being searched for.
  let cursor = $state(0)
  let query = $state('')
  let searchEl = $state<HTMLInputElement | null>(null)
  let statusEl = $state<HTMLInputElement | null>(null)
  let listEl = $state<HTMLElement | null>(null)
  let searchTimer: ReturnType<typeof setTimeout> | undefined

  // Documents-only edit state (the one writable field, write-through phase 0).
  let statusDraft = $state('')
  let saving = $state(false)
  let saveNote = $state('')
  let saveError = $state('')
  let conflict = $state<DocWriteResult | null>(null)

  const PAGE_SIZE = 50
  const editable = $derived(config.collection === 'files')

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
    runMessages = []
    statusDraft = String(rec.status ?? '')
    saveNote = ''
    saveError = ''
    conflict = null
    if (config.collection === 'agent_runs') {
      try {
        const res = await pb.collection('messages').getList(1, 200, {
          filter: pb.filter('run = {:id}', { id: rec.id }),
          sort: 'seq',
        })
        runMessages = res.items
      } catch {
        // messages unavailable: the runs row still renders
      }
    }
  }

  function scrollCursorIntoView() {
    tick().then(() => {
      listEl?.querySelectorAll('tbody tr')[cursor]?.scrollIntoView({ block: 'nearest' })
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
    statusEl?.focus()
    statusEl?.select()
  }

  /** ESC: exactly one level out, and the abandoned draft goes with the level it belonged to. */
  function back() {
    const next = outOf($level)
    if ($level === 'editing') {
      statusDraft = String(selected?.status ?? '')
      conflict = null
      saveError = ''
    }
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
      case 'save':
        if ($level === 'editing') save(String(selected?.checksum ?? ''))
        break
      case 'back':
        back()
        break
    }
  }

  onMount(() => onAction(handle))

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
      // Only adopt the incoming status into the input when the operator hasn't typed.
      if (statusDraft === String(selected.status ?? '')) statusDraft = String(rec.status ?? '')
      Object.assign(selected, rec)
    }
    const row = records.find((r) => r.id === rec.id)
    if (row) Object.assign(row, rec)
  }

  // Live-follow the selected document: the binary watches the desk tree, so an edit made
  // in another editor lands here. Older servers without realtime degrade silently.
  const watchId = $derived(editable ? (selected?.id ?? '') : '')
  $effect(() => {
    const id = watchId
    if (!id) return
    let stop: (() => void) | null = null
    let dropped = false
    pb.collection('files')
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

  /** Save the status field through the server's write-through route. `base` is the
   * checksum the save is staked on — the loaded one normally, the 409's fresh one when
   * the operator explicitly chooses to overwrite. */
  async function save(base: string) {
    const rec = selected
    if (!rec || saving) return
    saving = true
    saveNote = ''
    saveError = ''
    try {
      const res = await writeDocField(
        String(rec.path ?? ''),
        base,
        { status: statusDraft },
        pb.authStore.token || null,
      )
      if (res.outcome === 'conflict') {
        conflict = res
        return
      }
      conflict = null
      applyIncoming({ ...rec, status: statusDraft, checksum: res.checksum ?? rec.checksum } as RecordModel)
      saveNote = res.outcome === 'noop' ? 'Already saved — no change on disk.' : 'Saved to disk.'
      // A finished save is a finished edit: back to reading, the level ⌘↵ was invoked from.
      setLevel('reading')
    } catch (e) {
      saveError = e instanceof Error ? e.message : String(e)
    } finally {
      saving = false
    }
  }

  /** Conflict resolution "Reload": adopt what the server has now and drop the edit. */
  async function reload() {
    const rec = selected
    if (!rec) return
    try {
      const fresh = await pb.collection('files').getOne(rec.id)
      statusDraft = String(fresh.status ?? '')
      applyIncoming(fresh)
      conflict = null
      saveNote = ''
      saveError = ''
    } catch (e) {
      saveError = e instanceof Error ? e.message : String(e)
    }
  }

  function cell(rec: RecordModel, col: { key: string; expandKey?: string }): string {
    if (col.expandKey) {
      const rel = (rec.expand as Record<string, RecordModel> | undefined)?.[col.key]
      return rel ? String(rel[col.expandKey] ?? '') : ''
    }
    const v = rec[col.key]
    if (v === null || v === undefined) return ''
    return typeof v === 'object' ? JSON.stringify(v) : String(v)
  }

  function shortFields(rec: RecordModel): [string, string][] {
    return Object.entries(rec)
      .filter(([k, v]) => !['expand', 'collectionId', 'collectionName'].includes(k))
      .filter(([k]) => !config.detailBlocks.includes(k))
      .filter(([k]) => !(editable && k === 'status'))
      .map(([k, v]) => [k, typeof v === 'object' && v !== null ? JSON.stringify(v) : String(v ?? '')])
  }
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
        <table>
          <thead>
            <tr>
              {#each config.columns as col}<th>{col.label}</th>{/each}
            </tr>
          </thead>
          <tbody>
            {#each records as rec, i (rec.id)}
              <tr
                class:on={i === cursor}
                aria-selected={i === cursor}
                onclick={() => openRow(rec, i)}
              >
                {#each config.columns as col}
                  <td class="cell">{cell(rec, col)}</td>
                {/each}
              </tr>
            {/each}
          </tbody>
        </table>
        <p class="hint">j / k walk the rows · ↵ opens · {MOD_LABEL}K searches</p>
      {/if}
    </div>
  {:else if selected}
    <div class="instance">
      <div class="instance-head">
        <button onclick={back}>‹ {$level === 'editing' ? 'Reading' : config.title} — esc</button>
        <h3>{selected.path ?? selected.title ?? selected.id}</h3>
        <div class="spacer"></div>
        {#if editable && $level === 'reading'}
          <button onclick={startEdit}>Modify — e</button>
        {/if}
        <span class="walk">j / k for the next one</span>
      </div>

      {#if editable}
        <form
          class="edit"
          onsubmit={(e) => {
            e.preventDefault()
            save(String(selected?.checksum ?? ''))
          }}
        >
          <label for="status">status</label>
          {#if $level === 'editing'}
            <input id="status" type="text" bind:this={statusEl} bind:value={statusDraft} disabled={saving} />
            <button class="primary" type="submit" disabled={saving}>
              {saving ? 'Saving…' : `Save — ${MOD_LABEL}↵`}
            </button>
          {:else}
            <span class="value">{statusDraft || '—'}</span>
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
            <pre class="preview">{conflict.current_content ?? ''}</pre>
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
      {#if runMessages.length}
        <h4>messages</h4>
        {#each runMessages as m (m.id)}
          <div class="turn">
            <div class="role">{m.seq} · {m.role}{m.tool_name ? ` · ${m.tool_name}` : ''}</div>
            {#if m.content}<pre>{m.content}</pre>{/if}
            {#if m.tool_calls}<pre>{JSON.stringify(m.tool_calls, null, 2)}</pre>{/if}
          </div>
        {/each}
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
  tbody tr:hover {
    background: color-mix(in srgb, var(--accent) 8%, transparent);
  }
  tr.on {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
    box-shadow: inset 2px 0 0 var(--accent);
  }
  .cell {
    max-width: 18rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
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
  }
  .hint {
    margin-top: 0.8rem;
  }
  .instance-head button {
    font-size: 0.78rem;
    padding: 0.25rem 0.6rem;
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
    align-items: center;
    gap: 0.4rem;
    margin: 0.6rem 0 0.4rem;
    font-size: 0.8rem;
  }
  .edit label {
    color: var(--muted);
  }
  .edit input {
    flex: 0 1 22rem;
    min-width: 0;
    font-size: 0.8rem;
    padding: 0.3rem 0.45rem;
  }
  .edit .value {
    font-weight: 600;
  }
  .edit button {
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
  .conflict .preview {
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
  .turn .role {
    font-size: 0.75rem;
    color: var(--muted);
    margin: 0.4rem 0 0.2rem;
  }
</style>
