// Hand-written SVG paths, on purpose: an icon set is a dependency, a build step and a licence
// for thirteen glyphs. All of them are stroke-only on one 24×24 grid, so they inherit
// `currentColor` and line up with each other at any size.
//
// Icons are decoration here, never the label — the rail keeps its digit and every verb button
// keeps its word. So Icon.svelte renders them aria-hidden and nothing depends on recognising one.
//
// Plain .ts rather than a component module block so shell.ts can name an icon per mode without
// importing a Svelte component into the module the keyboard tests load.
export const ICONS = {
  // modes, in rail order
  queue: 'M3 13h5l1.5 2.5h5L16 13h5M5 5h14l2 8v6H3v-6z',
  library: 'M4 5a2 2 0 0 1 2-2h13v18H6a2 2 0 0 1-2-2zM8 3v18',
  patrol: 'M10.5 4a6.5 6.5 0 1 0 0 13 6.5 6.5 0 0 0 0-13zM15.2 15.2 20 20',
  work: 'M4 4h4v13H4zM10 4h4v16h-4zM16 4h4v9h-4z',
  agent: 'M4 5h16v10h-9l-5 4v-4H4z',
  config: 'M4 7h4M12 7h8M4 12h10M18 12h2M4 17h6M14 17h6M10 5v4M16 10v4M12 15v4',
  // the finder, and the verbs
  finder: 'M4 6h16M4 12h16M4 18h10',
  modify: 'M4 20h4.5L20 8.5 15.5 4 4 15.5zM14 5.5 18.5 10',
  body: 'M14 4h6v6M20 4l-8.5 8.5M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5',
  remove: 'M4 7h16M9.5 7V4h5v3M6.5 7 7.5 20h9L17.5 7M10 10.5v6M14 10.5v6',
  save: 'M5 12.5 9.5 17 19 7',
  revert: 'M4 9h9.5a5 5 0 0 1 0 10H8M4 9l4.5-4.5M4 9l4.5 4.5',
  back: 'M14.5 5.5 8 12l6.5 6.5',
} as const

export type IconName = keyof typeof ICONS
