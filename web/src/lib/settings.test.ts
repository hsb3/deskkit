import { describe, expect, it } from 'vitest'
import { pickChoice, sourceLabel } from './settings'

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
