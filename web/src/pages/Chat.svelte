<script lang="ts">
  import { pb } from '../lib/pb'
  import { streamTurn, type AgentEvent } from '../lib/sse'

  interface Step {
    callId: string
    label: string
    cls: '' | 'running' | 'error'
  }
  interface Message {
    role: 'user' | 'assistant' | 'system'
    text: string
    steps: Step[]
    meta: string
  }

  let messages = $state<Message[]>([])
  let input = $state('')
  let busy = $state(false)
  let scroller: HTMLElement

  function scrollToEnd() {
    requestAnimationFrame(() => scroller?.scrollTo({ top: scroller.scrollHeight }))
  }

  function setStep(msg: Message, callId: string, label: string, cls: Step['cls']) {
    const found = msg.steps.find((s) => s.callId === callId)
    if (found) {
      found.label = label
      found.cls = cls
    } else {
      msg.steps.push({ callId, label, cls })
    }
  }

  // Applies one stream event to the assistant bubble; kinds per the stream
  // contract: token, tool_start, tool_end, final, error.
  function handleEvent(evt: AgentEvent, msg: Message) {
    switch (evt.kind) {
      case 'token':
        msg.text += evt.token ?? ''
        break
      case 'tool_start':
        setStep(msg, evt.call_id ?? '', `running tool ${evt.tool}…`, 'running')
        break
      case 'tool_end':
        if (evt.err) setStep(msg, evt.call_id ?? '', `tool ${evt.tool} failed: ${evt.err}`, 'error')
        else setStep(msg, evt.call_id ?? '', `tool ${evt.tool} done`, '')
        break
      case 'final':
        msg.text = evt.content || msg.text
        if (typeof evt.total_tokens === 'number') msg.meta = `tokens: ${evt.total_tokens}`
        break
      case 'error':
        msg.text = evt.partial || msg.text
        msg.steps.push({
          callId: `err-${msg.steps.length}`,
          label: (evt.canceled ? 'canceled: ' : 'error: ') + evt.err,
          cls: 'error',
        })
        break
    }
    scrollToEnd()
  }

  async function send(evt: SubmitEvent) {
    evt.preventDefault()
    const text = input.trim()
    if (busy || !text) return
    input = ''
    messages.push({ role: 'user', text, steps: [], meta: '' })
    const reply: Message = $state({ role: 'assistant', text: '', steps: [], meta: '' })
    messages.push(reply)
    busy = true
    scrollToEnd()
    try {
      for await (const ev of streamTurn(text, pb.authStore.token || null)) {
        handleEvent(ev, reply)
      }
    } catch (e) {
      reply.steps.push({
        callId: 'conn',
        label: `connection error: ${e instanceof Error ? e.message : e}`,
        cls: 'error',
      })
    } finally {
      busy = false
    }
  }

  async function reset() {
    if (busy) return
    busy = true
    try {
      const headers: Record<string, string> = {}
      if (pb.authStore.token) headers['Authorization'] = pb.authStore.token
      await fetch('/desk/chat/reset', { method: 'POST', headers })
      messages = [{ role: 'system', text: 'New conversation started.', steps: [], meta: '' }]
    } catch (e) {
      messages.push({ role: 'system', text: `reset failed: ${e}`, steps: [], meta: '' })
    } finally {
      busy = false
    }
  }

  function onKeydown(evt: KeyboardEvent) {
    if (evt.key === 'Enter' && !evt.shiftKey) {
      evt.preventDefault()
      ;(evt.currentTarget as HTMLTextAreaElement).form?.requestSubmit()
    }
  }
</script>

<div class="chat">
  <main bind:this={scroller}>
    <div class="messages" aria-live="polite">
      {#each messages as msg}
        <div class="msg {msg.role}">
          <div class="text">{msg.text}</div>
          {#if msg.steps.length}
            <div class="steps">
              {#each msg.steps as step}
                <div class="step {step.cls}">{step.label}</div>
              {/each}
            </div>
          {/if}
          {#if msg.meta}<div class="meta">{msg.meta}</div>{/if}
        </div>
      {/each}
    </div>
  </main>
  <footer>
    <form onsubmit={send}>
      <label for="input">Message to the librarian</label>
      <textarea
        id="input"
        rows="2"
        placeholder="Ask about this desk…"
        bind:value={input}
        onkeydown={onKeydown}
        disabled={busy}
      ></textarea>
      <div class="actions">
        <button type="button" onclick={reset} disabled={busy}>New conversation</button>
        <button type="submit" class="primary" disabled={busy}>Send</button>
      </div>
    </form>
  </footer>
</div>

<style>
  .chat {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }
  main {
    flex: 1;
    overflow-y: auto;
    padding: 1rem;
  }
  .messages {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    max-width: 46rem;
    margin: 0 auto;
  }
  .msg {
    padding: 0.55rem 0.8rem;
    border-radius: 0.6rem;
    line-height: 1.4;
    white-space: pre-wrap;
    word-break: break-word;
  }
  .msg.user {
    align-self: flex-end;
    background: var(--accent);
    color: #fff;
  }
  .msg.assistant {
    align-self: flex-start;
    background: var(--panel);
    border: 1px solid var(--border);
  }
  .msg.system {
    align-self: center;
    background: transparent;
    color: var(--muted);
    font-size: 0.8rem;
  }
  .steps {
    margin-top: 0.4rem;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  .step {
    font-size: 0.78rem;
    color: var(--muted);
    border-left: 2px solid var(--border);
    padding-left: 0.5rem;
  }
  .step.running {
    color: var(--accent);
  }
  .step.error {
    color: var(--error);
  }
  .meta {
    margin-top: 0.3rem;
    font-size: 0.72rem;
    color: var(--muted);
  }
  footer {
    border-top: 1px solid var(--border);
    background: var(--panel);
    padding: 0.75rem 1rem 1rem;
  }
  form {
    max-width: 46rem;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }
  label {
    font-size: 0.78rem;
    color: var(--muted);
  }
  textarea {
    resize: vertical;
    min-height: 2.4rem;
  }
  .actions {
    display: flex;
    gap: 0.5rem;
    justify-content: flex-end;
  }
</style>
