// Checks the gzip size of what each page loads against CONSTRAINTS.md.
// It follows the Vite manifest, so shared chunks count for every page that
// loads them. Fonts are left out: they are budgeted apart and cached.
import { readFileSync } from 'node:fs';
import { gzipSync } from 'node:zlib';

const budgetsKB = {
  'login (JS)': 145,
  'login globe (JS, lazy)': 15,
  'app (JS)': 260,
  'stylesheet (CSS)': 35,
};

const dist = new URL('../dist/', import.meta.url);
const manifest = JSON.parse(readFileSync(new URL('.vite/manifest.json', dist), 'utf8'));
const gz = (file) => gzipSync(readFileSync(new URL(file, dist))).length;

function closure(key, seen = new Set()) {
  if (seen.has(key)) return seen;
  seen.add(key);
  const c = manifest[key];
  for (const k of c.imports ?? []) {
    closure(k, seen);
  }
  return seen;
}

const files = (keys) => new Set([...keys].map((k) => manifest[k].file));
const sum = (set) => [...set].reduce((n, f) => n + gz(f), 0);

const loginStatic = files(closure('login.html'));
// What the sign-in page loads later, after the form (the globe).
const loginLazy = new Set(
  (manifest['login.html'].dynamicImports ?? [])
    .flatMap((k) => [...files(closure(k))])
    .filter((f) => !loginStatic.has(f)),
);
const css = new Set(Object.values(manifest).flatMap((c) => c.css ?? []));

const measured = {
  'login (JS)': sum(loginStatic),
  'login globe (JS, lazy)': sum(loginLazy),
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
