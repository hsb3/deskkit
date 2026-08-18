import { describe, expect, it } from 'vitest'
import { splitFrames } from './sse'

describe('splitFrames', () => {
  it('parses complete frames and keeps the trailing partial', () => {
    const { events, rest } = splitFrames(
      'data: {"kind":"token","token":"hi"}\n\ndata: {"kind":"final","content":"hi"}\n\ndata: {"kind":"tok',
    )
    expect(events).toEqual([
      { kind: 'token', token: 'hi' },
      { kind: 'final', content: 'hi' },
    ])
    expect(rest).toBe('data: {"kind":"tok')
  })

  it('drops malformed and non-data frames without dying', () => {
    const { events, rest } = splitFrames('noise\n\ndata: {broken\n\ndata: {"kind":"error","err":"x"}\n\n')
    expect(events).toEqual([{ kind: 'error', err: 'x' }])
    expect(rest).toBe('')
  })
})
