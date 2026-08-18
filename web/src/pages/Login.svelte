<script lang="ts">
  import { login } from '../lib/pb'

  let email = $state('')
  let password = $state('')
  let error = $state('')
  let busy = $state(false)

  async function submit(evt: SubmitEvent) {
    evt.preventDefault()
    busy = true
    error = ''
    try {
      await login(email, password)
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    } finally {
      busy = false
    }
  }
</script>

<div class="wrap">
  <form onsubmit={submit}>
    <h1>deskkit</h1>
    <p class="note">Sign in with a superuser account to open this desk.</p>
    <label for="email">Email</label>
    <input id="email" type="email" bind:value={email} required autocomplete="username" />
    <label for="password">Password</label>
    <input id="password" type="password" bind:value={password} required autocomplete="current-password" />
    {#if error}<p class="error">{error}</p>{/if}
    <button class="primary" type="submit" disabled={busy}>Sign in</button>
  </form>
</div>

<style>
  .wrap {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  form {
    width: min(22rem, 90vw);
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 0.8rem;
    padding: 1.5rem;
  }
  h1 {
    margin: 0;
    font-size: 1.2rem;
  }
  .note {
    margin: 0 0 0.5rem;
    font-size: 0.8rem;
    color: var(--muted);
  }
  label {
    font-size: 0.78rem;
    color: var(--muted);
  }
  .error {
    color: var(--error);
    font-size: 0.8rem;
    margin: 0;
  }
</style>
