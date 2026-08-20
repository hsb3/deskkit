// Settings API: the SPA's first write surface. Reads/writes the single-row `settings`
// collection (superuser-only via PocketBase API rules — no hand-rolled auth check here),
// reads the model catalog from GET /desk/models (internal/core/spa/catalog.go), and reads
// GET /desk/settings/resolved (internal/core/spa/settings_resolved.go) so the panel can show
// which source is currently winning per field and lock the ones an environment variable
// already controls.
//
// The `settings` row is SEEDED by the store migration and the collection is a singleton, so
// this module only ever updates it — it never creates one. Both fetches still degrade to null
// rather than fake data when their endpoint 404s or errors (an older binary, an unmigrated
// store), so a missing backend piece reads as "we don't know", never as a false env-lock or a
// phantom saved value.
import { ClientResponseError } from 'pocketbase'
import { pb } from './pb'

export interface SettingsRecord {
  id: string
  llm_provider: string
  llm_model: string
  llm_api_key_hint: string
  /** Keep the finder minimised between items. Ruled a user setting, default on — so a store
   * whose migrations predate the field reads as on rather than as off. */
  sticky_finder: boolean
}

export interface ModelCatalogEntry {
  id: string
  name: string
  provider: string
}

export interface Catalog {
  providers: string[]
  models: ModelCatalogEntry[]
}

/** A resolution leg name as the backend reports it: env | profile | store | central | default
 * (config.go's Source* constants), or "" for an API key no leg supplies. Unrecognized strings
 * still render, generically. */
export type ResolvedSource = string

export interface ResolvedField {
  value: string
  source: ResolvedSource
}

export interface ResolvedSettings {
  provider: ResolvedField
  model: ResolvedField
  api_key: { source: ResolvedSource }
  /** URL template for handing a document's body to where the operator actually writes it —
   * e.g. `obsidian://open?path={abs}`. Absent/empty on a desk that declares no editor, and the
   * Open-body verb then does not render at all. Optional on the type because an older binary's
   * /desk/settings/resolved has no such key. */
  editor_url?: ResolvedField
  /** The desk's absolute root, needed to expand `{abs}` from a desk-relative path. */
  desk_root?: ResolvedField
}

export interface SettingsPatch {
  llm_provider: string
  llm_model: string
  llm_api_key?: string
}

/** Plain-language rendering of a resolution leg, for the "currently winning" note. A leg name
 * this build doesn't recognize still renders, quoted, rather than breaking the panel. */
export function sourceLabel(source: string | undefined): string {
  switch (source) {
    case 'env':
      return 'an environment variable'
    case 'profile':
      return 'the desk profile file'
    case 'store':
      return 'a setting saved on this desk'
    case 'central':
      return 'the machine-wide config file'
    case 'default':
      return 'the built-in default'
    case '':
      return 'nothing — none is set'
    default:
      return source ? `"${source}"` : 'an unknown source'
  }
}

/** Given a loaded value, decide whether a known-option dropdown can show it directly or the
 * "Custom…" escape hatch has to hold it (an aged/unknown model or a hand-set provider id). */
export function pickChoice(value: string, knownIds: string[]): { choice: string; custom: string } {
  if (value && knownIds.includes(value)) return { choice: value, custom: '' }
  return { choice: 'custom', custom: value }
}

/** Expand an editor URL template against one document. Nothing shells out: the browser renders
 * an anchor and the OS resolves the scheme, so the two placeholders are URL-encoded, not shell-
 * quoted. Returns '' when there is no template or no path, which is the signal not to render
 * the verb at all. */
export function editorHref(template: string, deskRoot: string, path: string): string {
  if (!template || !path) return ''
  const root = deskRoot.replace(/\/+$/, '')
  const abs = root ? `${root}/${path}` : path
  return template
    .replaceAll('{path}', encodeURIComponent(path))
    .replaceAll('{abs}', encodeURIComponent(abs))
}

function isNotFound(e: unknown): boolean {
  return e instanceof ClientResponseError && e.status === 404
}

/** Returns null when this store has no settings row to read — an older binary whose migrations
 * predate the collection. A migrated store always has exactly one, seeded by the migration. */
export async function fetchSettings(): Promise<SettingsRecord | null> {
  try {
    const rec = await pb.collection('settings').getFirstListItem('')
    return {
      id: rec.id,
      llm_provider: String(rec.llm_provider ?? ''),
      llm_model: String(rec.llm_model ?? ''),
      llm_api_key_hint: String(rec.llm_api_key_hint ?? ''),
      sticky_finder: Boolean(rec.sticky_finder ?? true),
    }
  } catch (e) {
    if (isNotFound(e)) return null
    throw e
  }
}

export async function saveSettings(id: string, patch: SettingsPatch): Promise<void> {
  await pb.collection('settings').update(id, patch)
}

/** The sticky-finder toggle saves on its own rather than riding the panel's form: it changes
 * behaviour the moment it is flipped, and the form's Save is gated on a provider and model this
 * preference has nothing to do with. */
export async function saveStickyFinder(id: string, on: boolean): Promise<void> {
  await pb.collection('settings').update(id, { sticky_finder: on })
}

/** The preference alone, for the shell to read at startup. Anything that goes wrong — no row, an
 * older store, no permission — yields the ruled default rather than an error: a preference is
 * never worth failing to boot over. */
export async function fetchStickyFinder(): Promise<boolean> {
  try {
    const rec = await fetchSettings()
    return rec ? rec.sticky_finder : true
  } catch {
    return true
  }
}

/** GET /desk/models (internal/core/spa/catalog.go): unauthenticated, no secrets, registered in
 * both bind modes so the picker works before login. A fetch failure degrades to an empty
 * catalog — both dropdowns fall back to their "Custom…" free-text escape hatch. */
export async function fetchCatalog(): Promise<Catalog> {
  try {
    const resp = await fetch('/desk/models')
    if (!resp.ok) return { providers: [], models: [] }
    const body = (await resp.json()) as { providers?: string[]; models?: ModelCatalogEntry[] }
    return { providers: body.providers ?? [], models: body.models ?? [] }
  } catch {
    return { providers: [], models: [] }
  }
}

/** Best-effort: returns null (never fake data) when the endpoint is absent, errors, or refuses
 * the caller. Superuser-gated on a public bind, so an unauthenticated browser gets null and the
 * panel says it cannot confirm the winning source rather than guessing one. */
export async function fetchResolvedSettings(): Promise<ResolvedSettings | null> {
  try {
    return await pb.send<ResolvedSettings>('/desk/settings/resolved', { method: 'GET' })
  } catch {
    return null
  }
}
