import { describe, expect, it } from 'vitest'
import { editorHref, pickChoice, sourceLabel } from './settings'

describe('sourceLabel', () => {
  it('renders known resolution legs in plain language', () => {
    expect(sourceLabel('env')).toBe('an environment variable')
    expect(sourceLabel('profile')).toBe('the desk profile file')
    expect(sourceLabel('store')).toBe('a setting saved on this desk')
    expect(sourceLabel('central')).toBe('the machine-wide config file')
    expect(sourceLabel('default')).toBe('the built-in default')
  })

  it('renders the API key\'s no-leg-supplies-it case as its own sentence', () => {
    expect(sourceLabel('')).toBe('nothing — none is set')
  })

  it('falls back for an unknown leg name instead of breaking', () => {
    expect(sourceLabel('something-new')).toBe('"something-new"')
    expect(sourceLabel(undefined)).toBe('an unknown source')
  })
})

describe('pickChoice', () => {
  it('selects the known option when the value is in the list', () => {
    expect(pickChoice('anthropic', ['anthropic', 'openai'])).toEqual({ choice: 'anthropic', custom: '' })
  })

  it('falls back to the custom escape hatch for an unlisted or empty value', () => {
    expect(pickChoice('some-old-model', ['anthropic', 'openai'])).toEqual({
      choice: 'custom',
      custom: 'some-old-model',
    })
    expect(pickChoice('', ['anthropic', 'openai'])).toEqual({ choice: 'custom', custom: '' })
  })
})

describe('editorHref', () => {
  it('expands {abs} against the desk root, URL-encoded', () => {
    expect(editorHref('obsidian://open?path={abs}', '/desks/ops', '_knowledge/a b.md')).toBe(
      'obsidian://open?path=%2Fdesks%2Fops%2F_knowledge%2Fa%20b.md',
    )
  })

  it('expands {path} as the desk-relative path, and both placeholders in one template', () => {
    expect(editorHref('x://{path}?abs={abs}', '/root', 'notes/a.md')).toBe(
      'x://notes%2Fa.md?abs=%2Froot%2Fnotes%2Fa.md',
    )
  })

  it('does not double the separator when the desk root has a trailing slash', () => {
    expect(editorHref('e://{abs}', '/root/', 'a.md')).toBe('e://%2Froot%2Fa.md')
  })

  it('falls back to the relative path when no desk root is known', () => {
    expect(editorHref('e://{abs}', '', 'a.md')).toBe('e://a.md')
  })

  // '' is the signal not to render the Open-body verb at all: a desk that declares no editor
  // gets no button, rather than a button that goes nowhere.
  it('returns nothing when there is no template or no path', () => {
    expect(editorHref('', '/root', 'a.md')).toBe('')
    expect(editorHref('e://{abs}', '/root', '')).toBe('')
  })
})
