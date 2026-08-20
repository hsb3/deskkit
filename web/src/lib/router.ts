// Minimal hash router: `#/<mode>`, where the modes are the rail's buttons (lib/shell.ts owns
// that list and the aliases for the segments the pre-rail SPA used). A hand-rolled store keeps
// the bundle free of a router dependency; upgrade only if the SPA ever needs nested layouts or
// guards per route.
//
// An empty hash yields an empty page, deliberately: which mode you land on is the shell's call,
// not the router's, and shell.resolveMode falls an unknown segment back to the landing mode.
import { readable } from 'svelte/store'

export interface Route {
  page: string
  args: string[]
}

function parse(): Route {
  const parts = window.location.hash.replace(/^#\/?/, '').split('/').filter(Boolean)
  return { page: parts[0] ?? '', args: parts.slice(1) }
}

export const route = readable<Route>(parse(), (set) => {
  const onChange = () => set(parse())
  window.addEventListener('hashchange', onChange)
  return () => window.removeEventListener('hashchange', onChange)
})
