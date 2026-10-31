import { describe, expect, it } from 'vitest';
import type { Domain } from '../../lib/api/domains';
import type { Https } from '../../lib/api/https';
import { httpsStart } from './https-form';
import { askedForm, panelForm } from './panel-form';
import { panelLink, panelUseOf } from './panel-use';

const facts = { domains: ['panel.example.com'], issuer: 'R11', warnings: [] };
const domain = (over: Partial<Domain> = {}): Domain => ({
  id: 3,
  name: 'panel.example.com',
  certificate: { id: 5, name: 'panel', origin: 'acme', facts },
  certificates: [],
  pins: [],
  acme: null,
  rules: [],
  warnings: [],
  ...over,
});
const https = (over: Partial<Https> = {}): Https => ({
  tls: 'self',
  domainId: 0,
  shared: false,
  port: 8443,
  pending: null,
  sharedPorts: [],
  ...over,
});

describe('Use for the Karvan panel', () => {
  it('is offered for a domain with a certificate for its name', () => {
    expect(panelUseOf(domain(), https())).toEqual({ kind: 'use' });
  });

  it('says why it cannot be picked without a certificate that covers the name', () => {
    expect(panelUseOf(domain({ certificate: null }), https())).toEqual({
      kind: 'use',
      blocked: 'no_certificate',
    });
    expect(panelUseOf(domain({ warnings: ['domain_mismatch'] }), https())).toEqual({
      kind: 'use',
      blocked: 'domain_mismatch',
    });
  });

  it('turns into Stop while the panel is served with this domain', () => {
    expect(panelUseOf(domain(), https({ tls: 'domain', domainId: 3 }))).toEqual({ kind: 'stop' });
    expect(panelUseOf(domain(), https({ tls: 'domain', domainId: 9 }))).toEqual({ kind: 'use' });
  });

  it('starts domain mode on the same port', () => {
    const h = https();
    expect(httpsStart(panelForm(domain(), h, 'use'), h)).toEqual({
      tls: 'domain',
      domainId: 3,
      port: 0,
      shared: false,
    });
  });

  it('opens the HTTPS tab with the change for the domain', () => {
    const h = https();
    const link = new URL(panelLink(domain(), 'use'), 'https://k.example/');
    expect(link.pathname).toBe('/settings/https');
    expect(askedForm(link.searchParams, [domain()], h)).toEqual(panelForm(domain(), h, 'use'));
    expect(askedForm(new URLSearchParams('domain=3&panel=other'), [domain()], h)).toBeNull();
    expect(askedForm(new URLSearchParams('domain=9&panel=use'), [domain()], h)).toBeNull();
    expect(askedForm(new URLSearchParams(''), [domain()], h)).toBeNull();
  });

  it('asks for a port of its own when leaving a shared port or stopping', () => {
    const shared = https({ tls: 'domain', domainId: 9, shared: true, port: 443 });
    expect(panelForm(domain(), shared, 'use').where).toBe('typed');
    const on = https({ tls: 'domain', domainId: 3 });
    const stop = panelForm(domain(), on, 'stop');
    expect(stop).toMatchObject({ tls: 'self', domainId: 0, where: 'typed' });
    expect(httpsStart(stop, on)).toBeNull();
    expect(httpsStart({ ...stop, typed: '9443' }, on)).toEqual({
      tls: 'self',
      domainId: 0,
      port: 9443,
      shared: false,
    });
  });
});
