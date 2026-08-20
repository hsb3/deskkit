// The keyboard map, as data. The rail IS the shortcut legend, so the map has
// to be one small readable table rather than handlers scattered across components: every rail
// button has a key and every key has a button, and that promise is only checkable if the whole
// map lives in one place.
//
// Nothing here touches the DOM — `resolveKey` turns a keystroke into an intent and stops. Who
// carries out the intent is App's business (modes, levels) or the active finder's (rows, edit),
// which is what makes the map testable without a browser.

/** An intent, not a keystroke. `mode` is zero-based: index 0 is the first rail button. */
export type Action =
  | { kind: 'mode'; index: number }
  | { kind: 'finder' }
  | { kind: 'search' }
  | { kind: 'next' }
  | { kind: 'prev' }
  | { kind: 'open' }
  | { kind: 'modify' }
  | { kind: 'save' }
  | { kind: 'body' }
  | { kind: 'delete' }
  | { kind: 'back' }

/** The parts of a KeyboardEvent this map reads. Declared structurally so tests can hand it a
 * plain object and so nothing here depends on a DOM lib being present. */
export interface KeyLike {
  key: string
  metaKey?: boolean
  ctrlKey?: boolean
  altKey?: boolean
}

/** How many rail buttons the digit shortcuts cover. The rail's own list is the source of truth
 * for what those buttons ARE (shell.ts); this is only the ceiling the map will emit. */
export const MODE_KEY_COUNT = 6

/**
 * Resolve one keystroke to an intent, or null to let the browser have it.
 *
 * `typing` is whether the keystroke landed in a text field. The bare keys (j/k/e/o/Enter) are
 * letters while you are typing and must not be stolen — and THREE modified ones are deliberately
 * exempt from that suppression, each for a concrete reason: ⌘K focuses search "from anywhere
 * including mid-edit", ⌘↵ saves from inside the field being edited, and ESC backs out of the
 * edit that field belongs to. Those three are the whole reason this takes a `typing` flag
 * instead of just refusing to fire inside inputs.
 *
 * The exemption is a short, closed list — ⌘⌫ is NOT on it. Ctrl+Backspace is the standard
 * delete-previous-word chord on Windows and Linux, and Meta/Control are one modifier here, so
 * exempting it would mean a user deleting a word in the search box arms a document delete
 * instead (and App's preventDefault swallows the native behaviour). Delete has no reason to
 * fire from inside an input the way the other three do.
 *
 * Meta and Control are treated as the same modifier so one map serves a Mac and everything else.
 * Alt-modified keystrokes are left alone: they compose characters.
 */
export function resolveKey(e: KeyLike, typing: boolean): Action | null {
  if (e.altKey) return null
  const mod = Boolean(e.metaKey || e.ctrlKey)

  if (mod) {
    const digit = Number(e.key)
    if (e.key.length === 1 && Number.isInteger(digit) && digit >= 1 && digit <= MODE_KEY_COUNT) {
      return { kind: 'mode', index: digit - 1 }
    }
    switch (e.key.toLowerCase()) {
      case 'b':
        return { kind: 'finder' }
      case 'k':
        return { kind: 'search' }
      case 'enter':
        return { kind: 'save' }
    }
    // The delete chord is deliberately NOT in that switch. Ctrl+Backspace is the OS
    // delete-previous-word shortcut, so unlike the three above it has no business firing inside
    // a text field — it drops through to BELOW the typing guard instead. Every other modified
    // chord stays the browser's.
    if (e.key !== 'Backspace') return null
  }

  if (e.key === 'Escape') return { kind: 'back' }
  if (typing) return null

  // Destructive, so it is a chord rather than a bare letter — and it arms the same two-step
  // confirm the button does, so a stray press costs a second press, never a file.
  if (mod) return e.key === 'Backspace' ? { kind: 'delete' } : null

  switch (e.key) {
    case 'j':
      return { kind: 'next' }
    case 'k':
      return { kind: 'prev' }
    case 'Enter':
      return { kind: 'open' }
    case 'e':
      return { kind: 'modify' }
    case 'o':
      return { kind: 'body' }
  }
  return null
}

/** Is this element a place where a keystroke is text rather than a command? Anything editable
 * counts — an unknown custom element that reports contentEditable included. */
export function isTyping(target: EventTarget | null): boolean {
  const el = target as (HTMLElement & { tagName?: string }) | null
  if (!el || !el.tagName) return false
  if (el.isContentEditable) return true
  return el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.tagName === 'SELECT'
}
