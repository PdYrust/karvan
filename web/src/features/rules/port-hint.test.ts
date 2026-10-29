import { describe, expect, it } from 'vitest';
import type { Ports } from '../../lib/api/ports';
import { oldPort, portHint } from './port-hint';

const ports: Ports = {
  ports: [
    { port: 443, owner: 'other' },
    { port: 8443, owner: 'rule' },
    { port: 24364, owner: 'panel' },
  ],
  proposal: 2053,
};

describe('portHint', () => {
  it('proposes a free port for an empty field', () => {
    expect(portHint(ports, '')).toEqual({ kind: 'free', free: 2053 });
  });
  it('names a port another program or the panel holds', () => {
    expect(portHint(ports, '443')).toEqual({ kind: 'taken', port: 443, free: 2053 });
    expect(portHint(ports, '24364')).toEqual({ kind: 'taken', port: 24364, free: 2053 });
  });
  it('says nothing for a free port or one rules share', () => {
    expect(portHint(ports, '2096')).toBeNull();
    expect(portHint(ports, '8443')).toBeNull();
    expect(portHint(ports, '2053')).toBeNull();
    expect(portHint(undefined, '443')).toBeNull();
    expect(portHint({ ...ports, proposal: 0 }, '443')).toBeNull();
  });
});

describe('oldPort', () => {
  it("is the old address's port when the rule takes its domain", () => {
    expect(oldPort('https://old.example.com:2096/sub/', ['old.example.com'])).toBe(2096);
    expect(oldPort('https://Old.example.com/sub/', ['old.example.com'])).toBe(443);
  });
  it('is 0 for another domain, plain HTTP without a port, or no address', () => {
    expect(oldPort('https://old.example.com:2096/', ['new.example.com'])).toBe(0);
    expect(oldPort('http://old.example.com/', ['old.example.com'])).toBe(0);
    expect(oldPort('', ['old.example.com'])).toBe(0);
  });
});
