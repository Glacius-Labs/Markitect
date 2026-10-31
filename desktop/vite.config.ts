import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

// The build goes where the Go package embeds it. The dev server forwards the
// JSON routes to a running markitect desktop (MARKITECT_DESKTOP, its address).
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: '../internal/adapters/desktop/ui',
    emptyOutDir: true,
    assetsInlineLimit: 0,
  },
  server: {
    proxy: { '/api': process.env.MARKITECT_DESKTOP ?? 'http://127.0.0.1:7340' },
  },
  test: {
    environment: 'node',
  },
});
