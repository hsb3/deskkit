// Minimal hash router: `#/chat`, `#/browse/<collection>`, `#/browse/<collection>/<id>`.
// A hand-rolled store keeps the bundle free of a router dependency; upgrade only
// if the SPA ever needs nested layouts or guards per route.
import { readable } from 'svelte/store'

export interface Route {
  page: string
  args: string[]
}

function parse(): Route {
  const parts = window.location.hash.replace(/^#\/?/, '').split('/').filter(Boolean)
  return { page: parts[0] || 'chat', args: parts.slice(1) }
}

export const route = readable<Route>(parse(), (set) => {
  const onChange = () => set(parse())
  window.addEventListener('hashchange', onChange)
  return () => window.removeEventListener('hashchange', onChange)
})
