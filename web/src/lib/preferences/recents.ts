const recentsKey = 'karvan.recents';

/** How many pages a browser remembers. */
export const maxRecents = 8;

// A route inside the panel, such as /panels/2. Paths are relative to the
// hidden path, which is never stored; "//host" would leave the panel.
const pagePath = /^\/(?!\/)[\w\-/]*$/;

/** Pages this browser opened, newest first. */
export function readRecents(): string[] {
  let raw: unknown;
  try {
    raw = JSON.parse(localStorage.getItem(recentsKey) ?? '[]');
  } catch {
    return [];
  }
  if (!Array.isArray(raw)) return [];
  const paths = raw.filter((p): p is string => typeof p === 'string' && pagePath.test(p));
  return [...new Set(paths)].slice(0, maxRecents);
}

/** Puts a page first, drops its earlier visit and anything past the limit. */
export function recordRecent(path: string): string[] {
  const before = readRecents();
  if (!pagePath.test(path)) return before;
  const next = [path, ...before.filter((p) => p !== path)].slice(0, maxRecents);
  try {
    localStorage.setItem(recentsKey, JSON.stringify(next));
  } catch {
    // Blocked storage only means the visit is not remembered.
  }
  return next;
}
