// The desk's document vocabulary: GET /desk/doctypes. Product-neutral (it is a vocabulary, not
// desk data), so it is unauthenticated on both binds and can be fetched before login.
//
// Fetched once per page load and cached here, because every open record asks the same question
// and the answer cannot change without a restart. A failed fetch caches as `null`, and every
// picker below then degrades to a free text input — a vocabulary the browser could not reach
// must never make a record uneditable.

export interface DoctypeInfo {
  /** '' for a lightweight type — it has no status family, so its status is free text. */
  family: string
  lightweight: boolean
  required: string[]
  optional: string[]
}

export interface Doctypes {
  /** family name → the statuses legal for it. */
  status: Record<string, string[]>
  /** doctype name → what it is. */
  types: Record<string, DoctypeInfo>
}

let pending: Promise<Doctypes | null> | null = null

async function load(): Promise<Doctypes | null> {
  try {
    const resp = await fetch('/desk/doctypes')
    if (!resp.ok) return null
    const body = (await resp.json()) as Partial<Doctypes>
    if (!body || typeof body !== 'object') return null
    return { status: body.status ?? {}, types: body.types ?? {} }
  } catch {
    return null
  }
}

/** The vocabulary, or null if this server has no `/desk/doctypes` (an older binary) or the
 * fetch failed. Cached including the null: retrying per record would be one dead request per
 * click for no new information. */
export function fetchDoctypes(): Promise<Doctypes | null> {
  pending ??= load()
  return pending
}

/** Drop the cache. For tests — nothing in the app re-fetches. */
export function resetDoctypeCache(): void {
  pending = null
}

/** The statuses legal for a doctype, or null when the answer is "free text": no vocabulary, an
 * unknown doctype, a lightweight type with no family, or a family the vocabulary lists no
 * statuses for. Null is the signal to render an input rather than a select. */
export function legalStatuses(vocab: Doctypes | null, doctype: string): string[] | null {
  if (!vocab || !doctype) return null
  const t = vocab.types[doctype]
  if (!t) return null
  // A lightweight type has no family and therefore no legal set — checked BEFORE the lookup,
  // because '' is a perfectly legal JSON key and a server that ever emitted one would otherwise
  // hand a lightweight document someone else's picker.
  if (!t.family) return null
  const list = vocab.status[t.family]
  return list && list.length ? list : null
}

/** What the picker offers: the legal statuses, plus the value currently on the record when that
 * value is not among them. Changing a document's type can strand its saved status outside the
 * new type's family — the answer to that is to SHOW the stranded value and let a person choose,
 * never to silently drop or rewrite someone's frontmatter. */
export function statusChoices(legal: string[], current: string): string[] {
  return current && !legal.includes(current) ? [current, ...legal] : legal
}

/** Whether `current` is outside the legal set — what the "no longer legal for this type" mark
 * is drawn from. False whenever there is no legal set to be outside of. */
export function isStrandedStatus(legal: string[] | null, current: string): boolean {
  return legal != null && current !== '' && !legal.includes(current)
}

/** Every doctype the desk knows, sorted, or null when there is no vocabulary to pick from (in
 * which case the type field degrades to a text input like the status one). */
export function doctypeNames(vocab: Doctypes | null): string[] | null {
  if (!vocab) return null
  const names = Object.keys(vocab.types).sort()
  return names.length ? names : null
}
