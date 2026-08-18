// SSE frame handling for /desk/chat/stream. The server writes one `data: <json>`
// frame per agent event; frames are separated by a blank line. Ported from the
// pre-SPA standalone chat page.

/** One agent event as emitted by the stream (kinds: token, tool_start, tool_end, final, error). */
export interface AgentEvent {
  kind: string
  token?: string
  tool?: string
  call_id?: string
  err?: string
  content?: string
  partial?: string
  canceled?: boolean
  total_tokens?: number
}

/**
 * Splits accumulated stream text into complete SSE frames, returning the parsed
 * events and the trailing partial frame to carry into the next chunk.
 */
export function splitFrames(buffer: string): { events: AgentEvent[]; rest: string } {
  const frames = buffer.split('\n\n')
  const rest = frames.pop() ?? ''
  const events: AgentEvent[] = []
  for (const frame of frames) {
    if (!frame.startsWith('data: ')) continue
    const payload = frame.slice(6)
    if (!payload) continue
    try {
      events.push(JSON.parse(payload))
    } catch {
      // a malformed frame is dropped, matching the old page's behavior
    }
  }
  return { events, rest }
}

/**
 * POSTs one chat turn and yields agent events until the terminal (final|error)
 * event. Sends the auth token when present (required in public mode, harmless
 * on loopback). Throws on a non-OK pre-stream HTTP response.
 */
export async function* streamTurn(message: string, token: string | null): AsyncGenerator<AgentEvent> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) headers['Authorization'] = token
  const resp = await fetch('/desk/chat/stream', {
    method: 'POST',
    headers,
    body: JSON.stringify({ message }),
  })
  if (!resp.ok || !resp.body) {
    let detail = ''
    try {
      detail = ((await resp.json()) as { error?: string }).error ?? ''
    } catch {
      /* non-JSON error body */
    }
    throw new Error(detail || `stream request failed (HTTP ${resp.status})`)
  }
  const reader = resp.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  for (;;) {
    const chunk = await reader.read()
    if (chunk.done) return
    buffer += decoder.decode(chunk.value, { stream: true })
    const { events, rest } = splitFrames(buffer)
    buffer = rest
    for (const evt of events) {
      yield evt
      if (evt.kind === 'final' || evt.kind === 'error') return
    }
  }
}
