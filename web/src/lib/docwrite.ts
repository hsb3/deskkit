// The SPA's one door from browser to disk: POST /desk/doc/write
// (internal/modules/librarian/web/write.go). The server records the original first and
// the write is reversible with `deskkit restore --by-path`; the browser never rewrites
// YAML itself — it names the fields to set and the server edits frontmatter in place.
//
// Concurrency is compare-and-swap on the file's checksum: a 409 means the file changed
// on disk since it was loaded and NOTHING was written. It is returned, not thrown, so
// the surface can show the disk's current state; overwriting is an explicit re-submit
// with `current_checksum` as the new base, never automatic.

export interface DocWriteResult {
  path: string
  outcome: 'written' | 'noop' | 'conflict'
  checksum?: string
  revision_id?: string
  current_checksum?: string
  current_content?: string
}

/** Sends the auth token when present (required in public mode, harmless on loopback),
 * matching the chat routes. Throws on 400/other failures with the server's error text. */
export async function writeDocField(
  path: string,
  baseChecksum: string,
  set: Record<string, string>,
  token: string | null,
): Promise<DocWriteResult> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) headers['Authorization'] = token
  const resp = await fetch('/desk/doc/write', {
    method: 'POST',
    headers,
    body: JSON.stringify({ path, base_checksum: baseChecksum, set }),
  })
  let body: Partial<DocWriteResult> & { error?: string } = {}
  try {
    body = (await resp.json()) as typeof body
  } catch {
    /* non-JSON body: fall through to the status-bearing message below */
  }
  if (resp.status === 409) return { ...body, path, outcome: 'conflict' }
  if (!resp.ok) throw new Error(body.error || `save failed (HTTP ${resp.status})`)
  return body as DocWriteResult
}
