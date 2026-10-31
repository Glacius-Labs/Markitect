import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

// The running markitect desktop the dev server forwards /api to.
const service = process.env.MARKITECT_DESKTOP ?? 'http://127.0.0.1:7340';

// The build goes where the Go package embeds it. The dev server runs on
// 127.0.0.1, the host of the service's cookie, and forwards /api to the
// service under the service's own host and origin, which it checks.
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: '../internal/adapters/desktop/ui',
    emptyOutDir: true,
    assetsInlineLimit: 0,
  },
  server: {
    host: '127.0.0.1',
    proxy: { '/api': { target: service, changeOrigin: true, headers: { origin: service } } },
  },
  test: {
    environment: 'node',
  },
});
