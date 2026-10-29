import { describe, expect, it } from 'vitest';
import { returnTarget } from './return-target';

const base = 'https://host.example:8443/abcDEF2345/';
const at = (href: string) => returnTarget({ href }, base);

describe('returnTarget', () => {
  it('keeps a page under the hidden path with its query', () => {
    expect(at('https://host.example:8443/abcDEF2345/domains')).toBe('/abcDEF2345/domains');
    expect(at('https://host.example:8443/abcDEF2345/rules/3?tab=ports#x')).toBe(
      '/abcDEF2345/rules/3?tab=ports',
    );
    expect(at(base)).toBe('/abcDEF2345/');
  });

  it('falls back to the hidden path root for another origin', () => {
    expect(at('https://evil.example/abcDEF2345/domains')).toBe(base);
    expect(at('http://host.example:8443/abcDEF2345/domains')).toBe(base);
    expect(at('https://host.example:9443/abcDEF2345/domains')).toBe(base);
  });

  it('falls back for a path outside the hidden path', () => {
    expect(at('https://host.example:8443/domains')).toBe(base);
    expect(at('https://host.example:8443/abcDEF2345x/domains')).toBe(base);
    expect(at('https://host.example:8443/abcDEF2345/../domains')).toBe(base);
    expect(at('https://host.example:8443/abcDEF2345')).toBe(base);
  });

  it('never returns a path another host could read as its own', () => {
    expect(at('https://host.example:8443//evil.example/abcDEF2345/')).toBe(base);
    expect(at('https://host.example:8443/abcDEF2345//evil.example/x')).toBe(base);
    expect(at('https://host.example:8443/abcDEF2345/%2F%2Fevil.example')).toBe(base);
    expect(at('https://host.example:8443/abcDEF2345/a%2fb')).toBe(base);
    expect(at('https://host.example:8443/abcDEF2345/a%5Cb')).toBe(base);
    expect(at('not a url')).toBe(base);
  });
});
