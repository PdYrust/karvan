/**
 * Where to go after signing in: the page the operator was on, when it is on
 * this origin and under the hidden path, otherwise the hidden path root. It
 * reads only the page's own address, so nothing from outside can steer it.
 */
export function returnTarget(location: Pick<Location, 'href'>, baseURI: string): string {
  const base = new URL(baseURI);
  let here: URL;
  try {
    here = new URL(location.href);
  } catch {
    return base.href;
  }
  const path = here.pathname;
  if (
    here.origin !== base.origin ||
    !path.startsWith(base.pathname) ||
    path.includes('//') ||
    /%2f|%5c|\\/i.test(path)
  ) {
    return base.href;
  }
  return path + here.search;
}
