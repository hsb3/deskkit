<script lang="ts">
  // The rail: the one element that never moves. It is the app's navigation, its window manager
  // and its shortcut legend in the same 34px strip — which is why the buttons are labelled with
  // their own keys. Every button here has a key and every key has a button; verbs and icons are
  // a later, content-stage call, so the shapes carry the digits for now.
  import { MODES, MOD_LABEL, type Level } from './shell'

  let {
    mode,
    level,
    finderAvailable,
    onSelect,
    onFinder,
  }: {
    mode: string
    level: Level
    /** Whether the mode on screen has a finder at all. A thread and a settings form do not. */
    finderAvailable: boolean
    onSelect: (id: string) => void
    onFinder: () => void
  } = $props()
</script>

<nav class="rail" aria-label="Work modes">
  {#each MODES as m, i (m.id)}
    <button
      class="btn"
      class:on={mode === m.id}
      aria-current={mode === m.id ? 'page' : undefined}
      aria-label={`${m.label} (${MOD_LABEL}${i + 1})`}
      title={`${m.label} — ${m.hint}  ·  ${MOD_LABEL}${i + 1}`}
      onclick={() => onSelect(m.id)}>{i + 1}</button
    >
  {/each}
  <div class="spacer"></div>
  <!-- Where the finder went. Lit while it is minimised, so the way back is visible rather than
       remembered — the button and the key are the same target. -->
  <button
    class="btn finder"
    class:lit={finderAvailable && level !== 'finder'}
    disabled={!finderAvailable}
    aria-pressed={finderAvailable && level === 'finder'}
    aria-label={`Finder (${MOD_LABEL}B)`}
    title={finderAvailable
      ? `Finder — the list for this mode  ·  ${MOD_LABEL}B`
      : 'This mode has no finder'}
    onclick={onFinder}>F</button
  >
</nav>

<style>
  .rail {
    width: 34px;
    flex: 0 0 34px;
    border-right: 1px solid var(--border);
    background: var(--panel);
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 0.35rem 0;
    gap: 0.25rem;
  }
  .btn {
    width: 26px;
    height: 26px;
    padding: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.72rem;
    border-radius: 0.35rem;
    background: transparent;
    border: 1px solid transparent;
    color: var(--muted);
  }
  .btn:hover:not(:disabled) {
    background: color-mix(in srgb, var(--accent) 12%, transparent);
    color: var(--fg);
  }
  .btn.on {
    background: var(--accent);
    border-color: var(--accent);
    color: #fff;
  }
  .btn.finder {
    border-color: var(--border);
  }
  .btn.finder.lit {
    background: color-mix(in srgb, var(--accent) 22%, transparent);
    border-color: var(--accent);
    color: var(--fg);
  }
  .spacer {
    flex: 1;
  }
</style>
