import { describe, expect, it } from 'vitest';
import { chosenPort, portDigits } from './port';

describe('chosenPort', () => {
  it('asks the server to pick for a random port', () => {
    expect(chosenPort({ kind: 'random' }, 24364)).toBe(-1);
  });

  it('takes a typed port from 1 to 65535', () => {
    expect(chosenPort({ kind: 'typed', text: '8443' }, 24364)).toBe(8443);
    expect(chosenPort({ kind: 'typed', text: '1' }, 24364)).toBe(1);
    expect(chosenPort({ kind: 'typed', text: '65535' }, 24364)).toBe(65535);
  });

  it('refuses the current port, an empty field and numbers out of range', () => {
    expect(chosenPort({ kind: 'typed', text: '24364' }, 24364)).toBeNull();
    expect(chosenPort({ kind: 'typed', text: '' }, 24364)).toBeNull();
    expect(chosenPort({ kind: 'typed', text: '0' }, 24364)).toBeNull();
    expect(chosenPort({ kind: 'typed', text: '65536' }, 24364)).toBeNull();
  });
});

describe('portDigits', () => {
  it('keeps up to five digits, Persian ones too', () => {
    expect(portDigits('۸۴۴۳')).toBe('8443');
    expect(portDigits('84a43')).toBe('8443');
    expect(portDigits('1234567')).toBe('12345');
  });
});
