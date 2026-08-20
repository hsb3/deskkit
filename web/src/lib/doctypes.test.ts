import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  doctypeNames,
  fetchDoctypes,
  isStrandedStatus,
  legalStatuses,
  resetDoctypeCache,
  statusChoices,
  type Doctypes,
} from './doctypes'

const vocab: Doctypes = {
  status: {
    decision: ['proposed', 'accepted', 'rejected', 'superseded'],
    spec: ['draft', 'in-review', 'approved', 'building', 'shipped', 'shelved'],
    meta: ['final'],
  },
  types: {
    decision: { family: 'decision', lightweight: false, required: ['decided_by'], optional: [] },
    spec: { family: 'spec', lightweight: false, required: [], optional: [] },
    journal: { family: '', lightweight: true, required: [], optional: [] },
    orphan: { family: 'nosuchfamily', lightweight: false, required: [], optional: [] },
  },
}

afterEach(() => {
  vi.unstubAllGlobals()
  resetDoctypeCache()
})

describe('legalStatuses', () => {
  it('answers with the family’s statuses for a full doctype', () => {
    expect(legalStatuses(vocab, 'decision')).toEqual([
      'proposed',
      'accepted',
      'rejected',
      'superseded',
    ])
  })

  // Null is the signal to render a text input. Every one of these must degrade that way rather
  // than to an empty picker with no way to type a value.
  it('falls back to free text for a lightweight type, an unknown one, or no vocabulary', () => {
    expect(legalStatuses(vocab, 'journal')).toBeNull()
    expect(legalStatuses(vocab, 'not-a-type')).toBeNull()
    expect(legalStatuses(vocab, '')).toBeNull()
    expect(legalStatuses(null, 'decision')).toBeNull()
  })

  it('falls back to free text when the family names no statuses', () => {
    expect(legalStatuses(vocab, 'orphan')).toBeNull()
  })

  // '' is a legal JSON key, so the no-family check has to come before the lookup rather than
  // rely on it missing: a lightweight document must never be handed a picker keyed on nothing.
  it('still gives a lightweight type free text even if the server lists an empty family', () => {
    const odd: Doctypes = { ...vocab, status: { ...vocab.status, '': ['bogus'] } }
    expect(legalStatuses(odd, 'journal')).toBeNull()
  })
})

describe('statusChoices', () => {
  it('offers the legal set unchanged when the current value is already in it', () => {
    expect(statusChoices(['a', 'b'], 'b')).toEqual(['a', 'b'])
    expect(statusChoices(['a', 'b'], '')).toEqual(['a', 'b'])
  })

  // Changing the type strands the saved status outside the new family. Showing it is the whole
  // point: silently dropping it would rewrite someone's frontmatter behind their back.
  it('keeps a stranded value visible at the front rather than dropping it', () => {
    expect(statusChoices(['proposed', 'accepted'], 'shipped')).toEqual([
      'shipped',
      'proposed',
      'accepted',
    ])
  })
})

describe('isStrandedStatus', () => {
  it('is true only when there is a legal set the value is outside of', () => {
    expect(isStrandedStatus(['a', 'b'], 'c')).toBe(true)
    expect(isStrandedStatus(['a', 'b'], 'a')).toBe(false)
    expect(isStrandedStatus(['a', 'b'], '')).toBe(false)
    expect(isStrandedStatus(null, 'c')).toBe(false)
  })

  // The scenario the correction called out: a decision saved as `accepted`, retyped as a spec.
  it('flags the status left behind when the drafted type changes family', () => {
    const before = legalStatuses(vocab, 'decision')
    const after = legalStatuses(vocab, 'spec')
    expect(isStrandedStatus(before, 'accepted')).toBe(false)
    expect(isStrandedStatus(after, 'accepted')).toBe(true)
    expect(statusChoices(after!, 'accepted')[0]).toBe('accepted')
  })
})

describe('doctypeNames', () => {
  it('lists every known type, sorted', () => {
    expect(doctypeNames(vocab)).toEqual(['decision', 'journal', 'orphan', 'spec'])
  })

  it('is null when there is nothing to pick from, so the field degrades to text', () => {
    expect(doctypeNames(null)).toBeNull()
    expect(doctypeNames({ status: {}, types: {} })).toBeNull()
  })
})

describe('fetchDoctypes', () => {
  it('reads the served vocabulary and caches it for the page', async () => {
    const spy = vi.fn(async () => new Response(JSON.stringify(vocab), { status: 200 }))
    vi.stubGlobal('fetch', spy)
    expect(await fetchDoctypes()).toEqual(vocab)
    expect(await fetchDoctypes()).toEqual(vocab)
    expect(spy).toHaveBeenCalledTimes(1)
    expect((spy.mock.calls[0] as unknown as [string])[0]).toBe('/desk/doctypes')
  })

  it('degrades to null on a 404, a network error, or junk — never throws at the caller', async () => {
    // A JSON body on the 404 on purpose: it must be the STATUS that rejects this, not the
    // parse failing by luck.
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => new Response(JSON.stringify(vocab), { status: 404 })),
    )
    expect(await fetchDoctypes()).toBeNull()

    resetDoctypeCache()
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => {
        throw new Error('offline')
      }),
    )
    expect(await fetchDoctypes()).toBeNull()

    resetDoctypeCache()
    vi.stubGlobal('fetch', vi.fn(async () => new Response('<html>', { status: 200 })))
    expect(await fetchDoctypes()).toBeNull()
  })

  it('fills in either half the server left out, so no caller needs an absent-field branch', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('{}', { status: 200 })))
    expect(await fetchDoctypes()).toEqual({ status: {}, types: {} })
  })
})
