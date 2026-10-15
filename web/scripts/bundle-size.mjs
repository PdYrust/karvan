// Checks the gzip size of what each page loads against CONSTRAINTS.md.
// It follows the Vite manifest, so shared chunks count for every page that
// loads them. Fonts are left out: they are budgeted apart and cached.
import { readFileSync } from 'node:fs';
import { gzipSync } from 'node:zlib';

const budgetsKB = {
  'login (JS)': 145,
  'login 3D logo (JS, lazy)': 180,
  'app (JS)': 260,
  'stylesheet (CSS)': 35,
};

const dist = new URL('../dist/', import.meta.url);
const manifest = JSON.parse(readFileSync(new URL('.vite/manifest.json', dist), 'utf8'));
const gz = (file) => gzipSync(readFileSync(new URL(file, dist))).length;

function closure(key, { dynamic = false } = {}, seen = new Set()) {
  if (seen.has(key)) return seen;
  seen.add(key);
  const c = manifest[key];
  for (const k of [...(c.imports ?? []), ...(dynamic ? (c.dynamicImports ?? []) : [])]) {
    closure(k, { dynamic }, seen);
  }
  return seen;
}

const files = (keys) => new Set([...keys].map((k) => manifest[k].file));
const sum = (set) => [...set].reduce((n, f) => n + gz(f), 0);

const loginStatic = files(closure('login.html'));
const loginAll = files(closure('login.html', { dynamic: true }));
const lazy = new Set([...loginAll].filter((f) => !loginStatic.has(f)));
const css = new Set(Object.values(manifest).flatMap((c) => c.css ?? []));

const measured = {
  'login (JS)': sum(loginStatic),
  'login 3D logo (JS, lazy)': sum(lazy),
  'app (JS)': sum(files(closure('index.html'))),
  'stylesheet (CSS)': sum(css),
};

let failed = false;
for (const [name, bytes] of Object.entries(measured)) {
  const kb = bytes / 1024;
  const over = kb > budgetsKB[name];
  failed ||= over;
  console.log(
    `${over ? 'OVER' : 'ok  '} ${name.padEnd(26)} ${kb.toFixed(1).padStart(6)} KB / ${budgetsKB[name]} KB`,
  );
}
process.exit(failed ? 1 : 0);
