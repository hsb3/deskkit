import { afterEach, describe, expect, it, vi } from 'vitest'
import { deleteDoc, writeDocField } from './docwrite'

function stubFetch(status: number, body: unknown) {
  const spy = vi.fn(async () => new Response(JSON.stringify(body), { status }))
  vi.stubGlobal('fetch', spy)
  return spy
}

afterEach(() => vi.unstubAllGlobals())

describe('writeDocField', () => {
  it('posts the field-mode contract body and returns the written outcome', async () => {
    const spy = stubFetch(200, { path: 'a.md', outcome: 'written', checksum: 'new', revision_id: 'r1' })
    const res = await writeDocField('a.md', 'old', { status: 'active' }, 'tok')
    expect(res).toEqual({ path: 'a.md', outcome: 'written', checksum: 'new', revision_id: 'r1' })
    const [url, init] = spy.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe('/desk/doc/write')
    expect(init.method).toBe('POST')
    expect(JSON.parse(String(init.body))).toEqual({
      path: 'a.md',
      base_checksum: 'old',
      set: { status: 'active' },
    })
    expect(init.headers).toEqual({ 'Content-Type': 'application/json', Authorization: 'tok' })
  })

  it('omits the Authorization header when there is no token', async () => {
    const spy = stubFetch(200, { path: 'a.md', outcome: 'noop' })
    await writeDocField('a.md', 'old', { status: 'x' }, null)
    const [, init] = spy.mock.calls[0] as unknown as [string, RequestInit]
    expect(init.headers).toEqual({ 'Content-Type': 'application/json' })
  })

  it('returns a 409 as a conflict result with the disk state instead of throwing', async () => {
    stubFetch(409, {
      path: 'a.md',
      outcome: 'conflict',
      current_checksum: 'disk',
      current_content: '---\nstatus: other\n---\n',
    })
    const res = await writeDocField('a.md', 'old', { status: 'x' }, null)
    expect(res.outcome).toBe('conflict')
    expect(res.current_checksum).toBe('disk')
    expect(res.current_content).toContain('status: other')
  })

  it("throws the server's error text on a 400", async () => {
    stubFetch(400, { error: 'write_doc: a.md is write-protected (.librarian-ignore)' })
    await expect(writeDocField('a.md', 'old', { status: 'x' }, null)).rejects.toThrow(
      'write_doc: a.md is write-protected (.librarian-ignore)',
    )
  })

  it('throws a status-bearing message when the error body is not JSON', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => new Response('<html>nope', { status: 500 })),
    )
    await expect(writeDocField('a.md', 'old', { status: 'x' }, null)).rejects.toThrow('HTTP 500')
  })
})

describe('deleteDoc', () => {
  it('posts the path and the checksum it is staked on, and reports the removal', async () => {
    const spy = stubFetch(200, { outcome: 'deleted', path: 'a.md', revision_id: 'r9' })
    const res = await deleteDoc('a.md', 'old', 'tok')
    expect(res.outcome).toBe('deleted')
    expect(res.revision_id).toBe('r9')
    const [url, init] = spy.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe('/desk/doc/delete')
    expect(init.method).toBe('POST')
    expect(JSON.parse(String(init.body))).toEqual({ path: 'a.md', base_checksum: 'old' })
    expect(init.headers).toEqual({ 'Content-Type': 'application/json', Authorization: 'tok' })
  })

  it('sends no set map — a delete names a file, it does not edit fields', async () => {
    const spy = stubFetch(200, { outcome: 'deleted', path: 'a.md' })
    await deleteDoc('a.md', 'old', null)
    const [, init] = spy.mock.calls[0] as unknown as [string, RequestInit]
    expect(JSON.parse(String(init.body))).not.toHaveProperty('set')
    expect(init.headers).toEqual({ 'Content-Type': 'application/json' })
  })

  it('returns a 409 as a conflict with the disk state, so nothing was removed', async () => {
    stubFetch(409, { current_checksum: 'disk', current_content: 'still here' })
    const res = await deleteDoc('a.md', 'stale', null)
    expect(res.outcome).toBe('conflict')
    expect(res.path).toBe('a.md')
    expect(res.current_checksum).toBe('disk')
  })

  it("throws the server's refusal verbatim — a write-protected path is not a bug", async () => {
    stubFetch(400, { error: 'delete_doc: a.md is write-protected (.librarian-ignore)' })
    await expect(deleteDoc('a.md', 'old', null)).rejects.toThrow(
      'delete_doc: a.md is write-protected (.librarian-ignore)',
    )
  })

  it('names the verb when the error body is not JSON', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => new Response('<html>nope', { status: 500 })),
    )
    await expect(deleteDoc('a.md', 'old', null)).rejects.toThrow('delete failed (HTTP 500)')
  })
})
