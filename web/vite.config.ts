import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

// In development the Go server runs separately; set KARVAN_DEV_TARGET to its
// address including the hidden path, e.g. http://127.0.0.1:34567/abcDEF2345ghijkm
const devTarget = process.env.KARVAN_DEV_TARGET;
const devBase = devTarget ? new URL(devTarget).pathname.replace(/\/$/, '') : '';

export default defineConfig({
  // Relative URLs; the server writes the hidden path into <base> at serve time.
  base: './',
  plugins: [react(), tailwindcss()],
  build: {
    manifest: true,
    target: 'es2022',
    assetsInlineLimit: 0,
    rollupOptions: {
      input: { login: 'login.html', app: 'index.html' },
    },
  },
  server: devTarget
    ? {
        proxy: {
          '/api': {
            target: new URL(devTarget).origin,
            rewrite: (path) => devBase + path,
            cookiePathRewrite: { '*': '/' },
          },
        },
      }
    : undefined,
  test: {
    environment: 'happy-dom',
    include: ['src/**/*.test.{ts,tsx}'],
    coverage: {
      provider: 'v8',
      include: ['src/lib/**/*.ts', 'src/i18n/**/*.ts'],
      exclude: ['**/*.test.ts'],
      thresholds: { lines: 80 },
    },
  },
});
