import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// The dist is go:embed'ded by the spa package and is never committed:
// `make build` runs this build before compiling the binary.
export default defineConfig({
  plugins: [svelte()],
  build: {
    // emptyOutDir stays off so the tracked dist/.gitkeep survives; the Makefile's
    // spa target owns cleanup of stale build outputs.
    outDir: '../librarian/internal/core/spa/dist',
    emptyOutDir: false,
  },
  server: {
    // Local dev against a running `deskkit serve` on the default port.
    proxy: {
      '/api': 'http://127.0.0.1:8090',
      '/desk': 'http://127.0.0.1:8090',
    },
  },
})
