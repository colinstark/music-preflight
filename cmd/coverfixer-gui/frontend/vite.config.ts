import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// Vite config for the coverfixer GUI. Build output goes to ./dist which Wails
// embeds via `//go:embed all:frontend/dist` in main.go. The dev server
// (started by Wails' `wails dev` through the `dev` script) must match the
// `frontend:dev:serverUrl` in wails.json.
export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    target: 'es2020',
  },
  server: {
    host: '127.0.0.1',
    port: 5173,
    strictPort: true,
  },
});
