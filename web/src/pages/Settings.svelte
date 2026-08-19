<script lang="ts">
  import { onMount } from 'svelte'
  import {
    fetchSettings,
    saveSettings,
    fetchCatalog,
    fetchResolvedSettings,
    sourceLabel,
    pickChoice,
    type ModelCatalogEntry,
    type ResolvedSettings,
  } from '../lib/settings'

  let loading = $state(true)
  let loadError = $state('')

  let recordId = $state<string | null>(null)
  let apiKeyHint = $state('')
  let resolved = $state<ResolvedSettings | null>(null)
  let providers = $state<string[]>([])
  let catalog = $state<ModelCatalogEntry[]>([])

  let providerChoice = $state('custom')
  let providerCustom = $state('')
  let modelChoice = $state('')
  let modelCustom = $state('')
  let apiKeyInput = $state('')

  let saving = $state(false)
  let saveMsg = $state('')
  let saveErr = $state('')

  const providerValue = $derived(providerChoice === 'custom' ? providerCustom : providerChoice)
  const modelValue = $derived(modelChoice === 'custom' ? modelCustom : modelChoice)
  const modelsForProvider = $derived(catalog.filter((m) => m.provider === providerValue))

  const providerLocked = $derived(resolved?.provider.source === 'env')
  const modelLocked = $derived(resolved?.model.source === 'env')
  const keyLocked = $derived(resolved?.api_key.source === 'env')

  function seedProvider(v: string, known: string[]) {
    const picked = pickChoice(v, known)
    providerChoice = picked.choice
    providerCustom = picked.custom
  }

  function seedModel(v: string, models: ModelCatalogEntry[]) {
    const picked = pickChoice(v, models.map((m) => m.id))
    modelChoice = picked.choice
    modelCustom = picked.custom
  }

  async function load() {
    loading = true
    loadError = ''
    try {
      const [rec, cat, res] = await Promise.all([fetchSettings(), fetchCatalog(), fetchResolvedSettings()])
      providers = cat.providers
      catalog = cat.models
      resolved = res
      const fallbackProvider = providers[0] ?? ''
      if (rec) {
        recordId = rec.id
        apiKeyHint = rec.llm_api_key_hint
        seedProvider(rec.llm_provider || resolved?.provider.value || fallbackProvider, providers)
        seedModel(rec.llm_model || resolved?.model.value || '', catalog)
      } else {
        recordId = null
        apiKeyHint = ''
        seedProvider(resolved?.provider.value || fallbackProvider, providers)
        seedModel(resolved?.model.value || '', catalog)
      }
    } catch (e) {
      loadError = e instanceof Error ? e.message : String(e)
    } finally {
      loading = false
    }
  }

  onMount(() => {
    load()
  })

  async function save(evt: SubmitEvent) {
    evt.preventDefault()
    saving = true
    saveMsg = ''
    saveErr = ''
    const patch: { llm_provider: string; llm_model: string; llm_api_key?: string } = {
      llm_provider: providerValue,
      llm_model: modelValue,
    }
    if (!keyLocked && apiKeyInput.trim()) patch.llm_api_key = apiKeyInput.trim()
    try {
      // Update only: the collection is a singleton whose one row the store migration seeds, and
      // its create hook rejects any other record. No row means an unmigrated store, which is a
      // state to report rather than to paper over with a create that would be refused anyway.
      if (!recordId) throw new Error('This desk has no settings row yet — run a newer binary against it.')
      await saveSettings(recordId, patch)
      apiKeyInput = ''
      saveMsg = 'Saved.'
      await load()
    } catch (e) {
      saveErr = e instanceof Error ? e.message : String(e)
    } finally {
      saving = false
    }
  }
</script>

<div class="settings">
  <h2>Settings</h2>
  {#if loading}
    <p class="muted">Loading…</p>
  {:else if loadError}
    <p class="error">{loadError}</p>
  {:else}
    {#if resolved === null}
      <p class="muted small">
        This desk doesn't expose which source is winning yet, so provider/model/key below show
        only their stored values — none can be confirmed locked by an environment variable.
      </p>
    {/if}
    <form onsubmit={save}>
      <div class="field">
        <label for="provider">Provider</label>
        {#if providerLocked}
          <div class="locked">
            <span class="value">{resolved?.provider.value}</span>
            <p class="note">An environment variable supplies this — edits here will not take effect.</p>
          </div>
        {:else}
          <select id="provider" bind:value={providerChoice}>
            {#each providers as p (p)}<option value={p}>{p}</option>{/each}
            <option value="custom">Custom…</option>
          </select>
          {#if providerChoice === 'custom'}
            <input type="text" bind:value={providerCustom} placeholder="Provider id" />
          {/if}
          <p class="source">Currently set by {sourceLabel(resolved?.provider.source)}.</p>
        {/if}
      </div>

      <div class="field">
        <label for="model">Model</label>
        {#if modelLocked}
          <div class="locked">
            <span class="value">{resolved?.model.value}</span>
            <p class="note">An environment variable supplies this — edits here will not take effect.</p>
          </div>
        {:else}
          <select id="model" bind:value={modelChoice}>
            {#each modelsForProvider as m (m.id)}<option value={m.id}>{m.name}</option>{/each}
            <option value="custom">Custom…</option>
          </select>
          {#if modelChoice === 'custom'}
            <input type="text" bind:value={modelCustom} placeholder="Model id" />
          {/if}
          <p class="source">Currently set by {sourceLabel(resolved?.model.source)}.</p>
        {/if}
      </div>

      <div class="field">
        <label for="apikey">API key</label>
        {#if keyLocked}
          <div class="locked">
            <span class="value">Set by an environment variable</span>
            <p class="note">An environment variable supplies the key — edits here will not take effect.</p>
          </div>
        {:else}
          <p class="key-status">{apiKeyHint ? `Set, ending in ${apiKeyHint}` : 'Not set.'}</p>
          <input
            id="apikey"
            type="password"
            bind:value={apiKeyInput}
            placeholder="Leave blank to keep the current key"
            autocomplete="off"
          />
          <p class="source">Currently set by {sourceLabel(resolved?.api_key.source)}.</p>
        {/if}
      </div>

      {#if saveErr}<p class="error">{saveErr}</p>{/if}
      {#if saveMsg}<p class="success">{saveMsg}</p>{/if}
      <button class="primary" type="submit" disabled={saving || !providerValue || !modelValue}>
        {saving ? 'Saving…' : 'Save'}
      </button>
    </form>
  {/if}
</div>

<style>
  .settings {
    padding: 1rem;
    overflow: auto;
    max-width: 34rem;
  }
  h2 {
    margin: 0 0 0.8rem;
    font-size: 1rem;
  }
  form {
    display: flex;
    flex-direction: column;
    gap: 1.1rem;
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }
  label {
    font-size: 0.8rem;
    color: var(--muted);
  }
  select,
  input {
    width: 100%;
  }
  .source,
  .key-status {
    margin: 0;
    font-size: 0.76rem;
    color: var(--muted);
  }
  .locked {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 0.5rem;
    padding: 0.5rem 0.7rem;
  }
  .locked .value {
    font-weight: 600;
  }
  .locked .note {
    margin: 0.25rem 0 0;
    font-size: 0.76rem;
    color: var(--muted);
  }
  .error {
    color: var(--error);
    font-size: 0.85rem;
    margin: 0;
  }
  .success {
    color: var(--accent);
    font-size: 0.85rem;
    margin: 0;
  }
  .muted {
    color: var(--muted);
  }
  .small {
    font-size: 0.8rem;
  }
</style>
