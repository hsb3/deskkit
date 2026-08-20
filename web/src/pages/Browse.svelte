<script lang="ts">
  import type { RecordModel } from 'pocketbase'
  import { pb } from '../lib/pb'
  import type { BrowseConfig } from '../lib/collections'
  import { writeDocField, type DocWriteResult } from '../lib/docwrite'

  let { config }: { config: BrowseConfig } = $props()

  let records = $state<RecordModel[]>([])
  let page = $state(1)
  let totalPages = $state(1)
  let error = $state('')
  let loading = $state(false)
  let selected = $state<RecordModel | null>(null)
  let runMessages = $state<RecordModel[]>([])

  // Documents-only edit state (the one writable field, write-through phase 0).
  let statusDraft = $state('')
  let saving = $state(false)
  let saveNote = $state('')
  let saveError = $state('')
  let conflict = $state<DocWriteResult | null>(null)

  const PAGE_SIZE = 50
  const editable = $derived(config.collection === 'files')

  async function load(cfg: BrowseConfig, p: number) {
    loading = true
    error = ''
    selected = null
    try {
      const res = await pb.collection(cfg.collection).getList(p, PAGE_SIZE, {
        sort: cfg.sort,
        filter: cfg.filter,
        expand: cfg.expand,
      })
      records = res.items
      totalPages = Math.max(res.totalPages, 1)
      page = p
    } catch (e) {
      records = []
      error = e instanceof Error ? e.message : String(e)
    } finally {
      loading = false
    }
  }

  $effect(() => {
    load(config, 1)
  })

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
  <div class="list">
    <div class="head">
      <h2>{config.title}</h2>
      {#if totalPages > 1}
        <div class="pager">
          <button disabled={page <= 1 || loading} onclick={() => load(config, page - 1)}>‹</button>
          <span>{page} / {totalPages}</span>
          <button disabled={page >= totalPages || loading} onclick={() => load(config, page + 1)}>›</button>
        </div>
      {/if}
    </div>
    {#if error}
      <p class="error">{error}</p>
    {:else if !loading && records.length === 0}
      <p class="empty">Nothing here yet.</p>
    {:else}
      <table>
        <thead>
          <tr>
            {#each config.columns as col}<th>{col.label}</th>{/each}
          </tr>
        </thead>
        <tbody>
          {#each records as rec (rec.id)}
            <tr class:selected={selected?.id === rec.id} onclick={() => select(rec)}>
              {#each config.columns as col}
                <td class="cell">{cell(rec, col)}</td>
              {/each}
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
  {#if selected}
    <aside>
      <div class="aside-head">
        <h3>{selected.id}</h3>
        <button onclick={() => (selected = null)}>Close</button>
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
          <input id="status" type="text" bind:value={statusDraft} disabled={saving} />
          <button class="primary" type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save'}</button>
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
    </aside>
  {/if}
</div>

<style>
  .browse {
    display: flex;
    height: 100%;
    min-height: 0;
  }
  .list {
    flex: 1;
    overflow: auto;
    padding: 1rem;
  }
  .head {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  h2 {
    margin: 0 0 0.6rem;
    font-size: 1rem;
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
  tr.selected {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
  }
  .cell {
    max-width: 18rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .empty,
  .error {
    color: var(--muted);
    font-size: 0.85rem;
  }
  .error {
    color: var(--error);
  }
  aside {
    width: min(34rem, 45vw);
    border-left: 1px solid var(--border);
    background: var(--panel);
    overflow: auto;
    padding: 1rem;
  }
  .aside-head {
    display: flex;
    justify-content: space-between;
    align-items: center;
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
    flex: 1;
    min-width: 0;
    font-size: 0.8rem;
    padding: 0.3rem 0.45rem;
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
