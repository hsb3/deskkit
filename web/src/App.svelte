<script lang="ts">
  import { onMount } from 'svelte'
  import { auth, initAuth, logout } from './lib/pb'
  import { route } from './lib/router'
  import { browsePages } from './lib/collections'
  import Login from './pages/Login.svelte'
  import Chat from './pages/Chat.svelte'
  import Browse from './pages/Browse.svelte'

  const nav = [
    { page: 'chat', label: 'Chat' },
    { page: 'documents', label: 'Documents' },
    { page: 'findings', label: 'Findings' },
    { page: 'runs', label: 'Agent runs' },
    { page: 'pm', label: 'PM items' },
  ]

  onMount(() => {
    initAuth()
  })
</script>

{#if $auth === 'checking'}
  <div class="center muted">Connecting…</div>
{:else if $auth === 'login'}
  <Login />
{:else}
  <div class="shell">
    <nav>
      <div class="brand">deskkit</div>
      {#each nav as item}
        <a href={`#/${item.page}`} class:active={$route.page === item.page}>{item.label}</a>
      {/each}
      <div class="spacer"></div>
      <button class="signout" onclick={logout}>Sign out</button>
    </nav>
    <section>
      {#if $route.page === 'chat'}
        <Chat />
      {:else if browsePages[$route.page]}
        <Browse config={browsePages[$route.page]} />
      {:else}
        <div class="center muted">Not found.</div>
      {/if}
    </section>
  </div>
{/if}

<style>
  .shell {
    display: flex;
    height: 100vh;
  }
  nav {
    width: 11rem;
    border-right: 1px solid var(--border);
    background: var(--panel);
    display: flex;
    flex-direction: column;
    padding: 0.8rem 0.6rem;
    gap: 0.15rem;
  }
  .brand {
    font-weight: 700;
    padding: 0.3rem 0.6rem 0.8rem;
  }
  nav a {
    color: var(--fg);
    text-decoration: none;
    padding: 0.4rem 0.6rem;
    border-radius: 0.45rem;
    font-size: 0.88rem;
  }
  nav a:hover {
    background: color-mix(in srgb, var(--accent) 10%, transparent);
  }
  nav a.active {
    background: var(--accent);
    color: #fff;
  }
  .spacer {
    flex: 1;
  }
  .signout {
    font-size: 0.8rem;
  }
  section {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  section > :global(*) {
    flex: 1;
    min-height: 0;
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
