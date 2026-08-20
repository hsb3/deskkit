<script lang="ts">
  import { onMount } from 'svelte'
  import { auth, initAuth } from './lib/pb'
  import { route } from './lib/router'
  import { browsePages } from './lib/collections'
  import { fetchStickyFinder } from './lib/settings'
  import { isTyping, resolveKey } from './lib/keys'
  import {
    MODES,
    MOD_LABEL,
    dispatch,
    level,
    resolveMode,
    setLevel,
    stickyFinder,
    toggleFinder,
  } from './lib/shell'
  import Rail from './lib/Rail.svelte'
  import Login from './pages/Login.svelte'
  import Chat from './pages/Chat.svelte'
  import Browse from './pages/Browse.svelte'
  import Settings from './pages/Settings.svelte'

  const modeId = $derived(resolveMode($route.page))
  const mode = $derived(MODES.find((m) => m.id === modeId) ?? MODES[0])

  // Agent is a thread, but its runs have to stay reachable — they are not a work mode of their
  // own (nothing else earns a rail button), so they live inside this one.
  let agentView = $state<'thread' | 'runs'>($route.page === 'runs' ? 'runs' : 'thread')

  /** Which browse config the screen is showing, if any. Finder-ness is a property of what is on
   * screen rather than of the mode: Agent has no finder on its thread and one on its runs. */
  const browseConfig = $derived(
    modeId === 'library'
      ? browsePages.documents
      : modeId === 'patrol'
        ? browsePages.findings
        : modeId === 'work'
          ? browsePages.pm
          : modeId === 'agent' && agentView === 'runs'
            ? browsePages.runs
            : null,
  )

  function goto(id: string) {
    window.location.hash = `#/${id}`
  }

  // A mode change always lands you in that mode's finder: arriving somewhere already zoomed into
  // an item you did not choose is disorienting, and it would strand the rail's lit F button on a
  // screen with nothing behind it.
  $effect(() => {
    void modeId
    void agentView
    setLevel('finder')
  })

  function onKeydown(e: KeyboardEvent) {
    const action = resolveKey(e, isTyping(e.target))
    if (!action) return
    switch (action.kind) {
      case 'mode': {
        const m = MODES[action.index]
        if (m) {
          e.preventDefault()
          goto(m.id)
        }
        return
      }
      case 'finder':
        e.preventDefault()
        if (browseConfig) toggleFinder()
        return
      case 'back':
        // ESC leaves the field you are in as well as the level it belongs to; the finder still
        // hears it, so it can drop an abandoned draft.
        if (isTyping(e.target)) (e.target as HTMLElement).blur()
        dispatch(action)
        return
      default:
        // Rows, search and saving belong to whichever finder is on screen. Nothing listening
        // means the keystroke is simply not ours — leave it to the browser.
        if (dispatch(action)) e.preventDefault()
    }
  }

  onMount(() => {
    initAuth()
  })

  // The sticky-finder preference is per DESK, so it is read from the store once there is a token
  // to read it with. An unreachable or older store leaves the default (on) in place.
  $effect(() => {
    if ($auth !== 'authed') return
    fetchStickyFinder().then((on) => stickyFinder.set(on))
  })
</script>

<svelte:window onkeydown={onKeydown} />

{#if $auth === 'checking'}
  <div class="center muted">Connecting…</div>
{:else if $auth === 'login'}
  <Login />
{:else}
  <div class="shell">
    <Rail
      mode={modeId}
      level={$level}
      finderAvailable={browseConfig !== null}
      onSelect={goto}
      onFinder={toggleFinder}
    />
    <section>
      <header>
        <h1>{mode.label}</h1>
        {#if modeId === 'agent'}
          <div class="tabs" role="tablist" aria-label="Agent views">
            <button role="tab" aria-selected={agentView === 'thread'} onclick={() => (agentView = 'thread')}>
              Thread
            </button>
            <button role="tab" aria-selected={agentView === 'runs'} onclick={() => (agentView = 'runs')}>
              Runs
            </button>
          </div>
        {/if}
        <div class="spacer"></div>
        <span class="legend">{MOD_LABEL}1–{MOD_LABEL}6 modes · {MOD_LABEL}B finder · {MOD_LABEL}K search · j/k rows · ↵ open · e modify · {MOD_LABEL}↵ save · esc back</span>
      </header>
      <div class="body">
        {#if browseConfig}
          <Browse config={browseConfig} />
        {:else if modeId === 'agent'}
          <Chat />
        {:else if modeId === 'config'}
          <Settings />
        {:else}
          <div class="soon">
            <p>
              The landing queue — the findings and work items waiting on you, in one adjudicable
              list — arrives with the inbox template.
            </p>
            <p class="muted">Until then, Patrol and Work hold the same rows.</p>
          </div>
        {/if}
      </div>
    </section>
  </div>
{/if}

<style>
  .shell {
    display: flex;
    height: 100vh;
  }
  section {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  header {
    display: flex;
    align-items: center;
    gap: 0.7rem;
    padding: 0.4rem 0.8rem;
    border-bottom: 1px solid var(--border);
    background: var(--panel);
  }
  h1 {
    margin: 0;
    font-size: 0.9rem;
    font-weight: 600;
  }
  .tabs {
    display: flex;
    gap: 0.2rem;
  }
  .tabs button {
    font-size: 0.75rem;
    padding: 0.15rem 0.55rem;
    border-radius: 0.35rem;
    background: transparent;
    border-color: transparent;
    color: var(--muted);
  }
  .tabs button[aria-selected='true'] {
    background: color-mix(in srgb, var(--accent) 16%, transparent);
    color: var(--fg);
  }
  .spacer {
    flex: 1;
  }
  .legend {
    font-size: 0.68rem;
    color: var(--muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .body {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }
  .body > :global(*) {
    flex: 1;
    min-height: 0;
  }
  .soon {
    padding: 1.2rem;
    max-width: 34rem;
    font-size: 0.9rem;
  }
  .center {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100vh;
  }
  .muted {
    color: var(--muted);
  }
</style>
