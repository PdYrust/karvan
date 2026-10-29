import { describe, expect, it } from 'vitest';
import type { Domain } from '../../lib/api/domains';
import { warningLines, warningTone } from './rule-view';

function domain(id: number, name: string, warnings: Domain['warnings']): Domain {
  return {
    id,
    name,
    certificate: null,
    certificates: [],
    pins: [],
    acme: null,
    rules: [],
    warnings,
  };
}

const domains = [
  domain(1, 'a.example.com', ['cert_conflict', 'expires_soon']),
  domain(2, 'b.example.com', ['no_certificate']),
  domain(3, 'c.example.com', ['domain_mismatch']),
];

describe('a rule card', () => {
  it('shows the warnings of its own domains, each with its domain', () => {
    const lines = warningLines(['a.example.com', 'b.example.com'], [], domains);
    expect(lines.map((l) => [l.domain?.id, l.code])).toEqual([
      [1, 'cert_conflict'],
      [1, 'expires_soon'],
    ]);
  });

  it('leaves out plain HTTP, which the card already says', () => {
    expect(warningLines(['b.example.com'], [], domains)).toEqual([]);
  });

  it('puts a rule not adopted first, without a domain', () => {
    const lines = warningLines(['c.example.com'], ['certificate_not_adopted'], domains);
    expect(lines.map((l) => [l.domain?.id, l.code])).toEqual([
      [undefined, 'certificate_not_adopted'],
      [3, 'domain_mismatch'],
    ]);
  });

  it('shows nothing for a rule with no domains or an unknown one', () => {
    expect(warningLines(null, [], domains)).toEqual([]);
    expect(warningLines(['gone.example.com'], [], domains)).toEqual([]);
  });

  it('marks what stops HTTPS as danger and the rest as a warning', () => {
    expect(warningTone('expired')).toBe('danger');
    expect(warningTone('renewal_failed')).toBe('danger');
    expect(warningTone('cert_conflict')).toBe('warning');
    expect(warningTone('certificate_not_adopted')).toBe('warning');
  });
});
