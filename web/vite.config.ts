import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { configDefaults, defineConfig } from 'vitest/config';

const domTests = [
  'src/**/*.test.tsx',
  'src/lib/api/**/*.test.ts',
  'src/lib/clipboard/**/*.test.ts',
  'src/lib/keys/**/*.test.ts',
  'src/lib/preferences/**/*.test.ts',
  'src/i18n/change-locale.test.ts',
  'src/features/live/use-dots.test.ts',
];

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
    // Only these tests need a DOM; the rest run in plain Node, which starts
    // much faster than happy-dom. A test that needs a DOM and is not listed
    // fails in Node (no document, window or storage), so add it here.
    projects: [
      {
        extends: true,
        test: {
          name: 'node',
          environment: 'node',
          include: ['src/**/*.test.{ts,tsx}'],
          exclude: [...configDefaults.exclude, ...domTests],
        },
      },
      { extends: true, test: { name: 'dom', environment: 'happy-dom', include: domTests } },
    ],
    coverage: {
      provider: 'v8',
      include: ['src/lib/**/*.ts', 'src/i18n/**/*.ts'],
      exclude: ['**/*.test.ts'],
      thresholds: { lines: 80 },
    },
  },
});
