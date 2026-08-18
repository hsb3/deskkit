<script lang="ts">
  import type { RecordModel } from 'pocketbase'
  import { pb } from '../lib/pb'
  import type { BrowseConfig } from '../lib/collections'

  let { config }: { config: BrowseConfig } = $props()

  let records = $state<RecordModel[]>([])
  let page = $state(1)
  let totalPages = $state(1)
  let error = $state('')
  let loading = $state(false)
  let selected = $state<RecordModel | null>(null)
  let runMessages = $state<RecordModel[]>([])

  const PAGE_SIZE = 50

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
    if (config.collection === 'agent_runs') {
      try {
        const res = await pb.collection('messages').getList(1, 200, {
          filter: `run = "${rec.id}"`,
          sort: 'seq',
        })
        runMessages = res.items
      } catch {
        // messages unavailable: the runs row still renders
      }
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
  .turn .role {
    font-size: 0.75rem;
    color: var(--muted);
    margin: 0.4rem 0 0.2rem;
  }
</style>
