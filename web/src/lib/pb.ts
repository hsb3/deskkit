// One PocketBase client for the whole SPA, served same-origin by the binary.
//
// Auth model: every domain collection is nil-rule (superuser-only), so reads
// always need a superuser token.
//  - Loopback serve registers GET /desk/bootstrap, which mints one; the SPA
//    fetches it on load and the operator never sees a login screen.
//  - Public serve has no bootstrap route; the SPA shows a login form and
//    authenticates against _superusers via the SDK.
import PocketBase from 'pocketbase'
import { writable } from 'svelte/store'

export const pb = new PocketBase('/')
pb.autoCancellation(false)

export type AuthState = 'checking' | 'authed' | 'login'

export const auth = writable<AuthState>('checking')

/** Resolve auth on load: reuse a stored token, else try the loopback bootstrap, else login. */
export async function initAuth(): Promise<void> {
  if (pb.authStore.isValid) {
    auth.set('authed')
    return
  }
  try {
    const resp = await fetch('/desk/bootstrap')
    // In public mode the route is unregistered and the SPA's index fallback answers with
    // the HTML shell — a 200 that is NOT a token. Only a JSON response counts.
    if (resp.ok && (resp.headers.get('content-type') ?? '').includes('application/json')) {
      const body = (await resp.json()) as { token: string }
      pb.authStore.save(body.token, null)
      auth.set('authed')
      return
    }
  } catch {
    /* no bootstrap → fall through to login */
  }
  auth.set('login')
}

export async function login(email: string, password: string): Promise<void> {
  await pb.collection('_superusers').authWithPassword(email, password)
  auth.set('authed')
}

export function logout(): void {
  pb.authStore.clear()
  auth.set('login')
}
