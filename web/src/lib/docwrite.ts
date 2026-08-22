// The SPA's two doors from browser to disk: POST /desk/doc/write and POST /desk/doc/delete
// (internal/modules/librarian/web/). The server records the original first and BOTH are
// reversible with `deskkit restore --by-path`; the browser never rewrites YAML itself — it
// names the fields to set and the server edits frontmatter in place.
//
// Concurrency is compare-and-swap on the file's checksum: a 409 means the file changed
// on disk since it was loaded and NOTHING was written or removed. It is returned, not thrown,
// so the surface can show the disk's current state; overwriting is an explicit re-submit
// with `current_checksum` as the new base, never automatic.
//
// A 400 is the server refusing, and its text is the answer — "…is write-protected
// (.deskkitignore)" for a path the desk protects, or the frontmatter editor declining a
// block array. Those are correct behaviours, so the message is thrown verbatim for the surface
// to show; there is deliberately no client-side guard second-guessing them.

export interface DocWriteResult {
  path: string
  outcome: 'written' | 'noop' | 'conflict' | 'deleted'
  checksum?: string
  revision_id?: string
  current_checksum?: string
  current_content?: string
}

/** Sends the auth token when present (required in public mode, harmless on loopback),
 * matching the chat routes. Throws on 400/other failures with the server's error text. */
async function postDoc(
  url: string,
  verb: string,
  body: unknown,
  token: string | null,
): Promise<DocWriteResult> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) headers['Authorization'] = token
  const resp = await fetch(url, { method: 'POST', headers, body: JSON.stringify(body) })
  let parsed: Partial<DocWriteResult> & { error?: string } = {}
  try {
    parsed = (await resp.json()) as typeof parsed
  } catch {
    /* non-JSON body: fall through to the status-bearing message below */
  }
  const path = String((body as { path?: string }).path ?? '')
  if (resp.status === 409) return { ...parsed, path, outcome: 'conflict' }
  if (!resp.ok) throw new Error(parsed.error || `${verb} failed (HTTP ${resp.status})`)
  return parsed as DocWriteResult
}

export function writeDocField(
  path: string,
  baseChecksum: string,
  set: Record<string, string>,
  token: string | null,
): Promise<DocWriteResult> {
  return postDoc('/desk/doc/write', 'save', { path, base_checksum: baseChecksum, set }, token)
}

/** Reversible with `deskkit restore --by-path` — the server records the original into the
 * revisions collection before it removes anything, so this is a soft delete on disk terms.
 * Same CAS posture as the write: a 409 comes back as a conflict result, not an exception. */
export function deleteDoc(
  path: string,
  baseChecksum: string,
  token: string | null,
): Promise<DocWriteResult> {
  return postDoc('/desk/doc/delete', 'delete', { path, base_checksum: baseChecksum }, token)
}
