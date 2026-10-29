import { describe, expect, it } from 'vitest';
import { ApiError } from '../../lib/api/client';
import type { LinkSite } from '../../lib/api/rules';
import { inputOf, isPlainHttp, newValues, parseDomains, refusedField, valuesOf } from './site-form';

const saved: LinkSite = {
  id: 3,
  name: 'old-443',
  panelId: 2,
  sourceKind: 'pasarguard',
  sourceUrl: 'https://old.example.com:8443',
  oldPath: 'sub',
  domains: ['a.example.com', 'b.example.com'],
  port: 443,
  lazy: true,
  plainHttpOk: false,
  https: true,
};

describe('the site form', () => {
  it('starts a new site from a migration with HTTPS on 443 and lazy checks on', () => {
    const v = newValues({ panelId: 2, sourceKind: 'pasarguard', sourceUrl: 'https://old' }, 1);
    expect(v).toMatchObject({ panelId: 2, sourceKind: 'pasarguard', sourceUrl: 'https://old' });
    expect(v).toMatchObject({ oldPath: 'sub', port: '443', lazy: true });
    expect(newValues({}, 1).panelId).toBe(1);
  });

  it('opens a saved site with its domains one per line', () => {
    expect(valuesOf(saved).domains).toBe('a.example.com\nb.example.com');
    expect(valuesOf({ ...saved, domains: null }).domains).toBe('');
  });

  it('reads the port as a number and the domains as a list', () => {
    const input = inputOf({ ...valuesOf(saved), port: '8443', domains: ' a.x, b.x\n\nc.x ' });
    expect(input.port).toBe(8443);
    expect(input.domains).toEqual(['a.x', 'b.x', 'c.x']);
    expect(parseDomains('')).toEqual([]);
  });

  it('accepts a plain http old panel only for an http address', () => {
    expect(isPlainHttp(' HTTP://old')).toBe(true);
    expect(isPlainHttp('https://old')).toBe(false);
    const v = { ...valuesOf(saved), plainHttpOk: true };
    expect(inputOf(v).plainHttpOk).toBe(false);
    expect(inputOf({ ...v, sourceUrl: 'http://old' }).plainHttpOk).toBe(true);
  });

  it('finds the field a refused save names', () => {
    expect(refusedField(new ApiError(400, 'site_invalid', undefined, { field: 'port' }))).toBe(
      'port',
    );
    expect(
      refusedField(new ApiError(400, 'site_invalid', undefined, { field: 'other' })),
    ).toBeUndefined();
    expect(refusedField(new ApiError(409, 'name_taken'))).toBeUndefined();
  });
});
