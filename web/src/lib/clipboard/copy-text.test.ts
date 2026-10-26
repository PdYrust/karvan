import { afterEach, describe, expect, it, vi } from 'vitest';
import { copyText } from './copy-text';

/** A page served over HTTPS (or localhost) has the Clipboard API; one over plain HTTP has none. */
function secure(writeText?: (text: string) => Promise<void>) {
  vi.stubGlobal('isSecureContext', writeText !== undefined);
  vi.stubGlobal('navigator', { ...navigator, clipboard: writeText ? { writeText } : undefined });
}

/** Records what document.execCommand('copy') would copy: the selected field's text. */
function execCommand(works: boolean) {
  const copied: string[] = [];
  const exec = vi.fn((command: string) => {
    const field = document.activeElement;
    if (command === 'copy' && field instanceof HTMLTextAreaElement) copied.push(field.value);
    return works;
  });
  Object.defineProperty(document, 'execCommand', { value: exec, configurable: true });
  return { exec, copied };
}

afterEach(() => {
  vi.unstubAllGlobals();
  Reflect.deleteProperty(document, 'execCommand');
  document.body.replaceChildren();
});

describe('copyText', () => {
  it('uses the Clipboard API where the page is secure', async () => {
    const writeText = vi.fn(() => Promise.resolve());
    secure(writeText);
    const { exec } = execCommand(true);
    await expect(copyText('JBSW Y3DP')).resolves.toBe(true);
    expect(writeText).toHaveBeenCalledWith('JBSW Y3DP');
    expect(exec).not.toHaveBeenCalled();
  });

  it('copies through a hidden field over plain HTTP, and removes it', async () => {
    secure();
    const { copied } = execCommand(true);
    const button = document.createElement('button');
    document.body.append(button);
    button.focus();
    await expect(copyText('JBSWY3DP')).resolves.toBe(true);
    expect(copied).toEqual(['JBSWY3DP']);
    expect(document.querySelector('textarea')).toBeNull();
    expect(document.activeElement).toBe(button);
  });

  it('falls back when the Clipboard API refuses', async () => {
    secure(() => Promise.reject(new Error('denied')));
    const { copied } = execCommand(true);
    await expect(copyText('secret')).resolves.toBe(true);
    expect(copied).toEqual(['secret']);
  });

  it('says so when the browser copies nothing', async () => {
    secure();
    execCommand(false);
    await expect(copyText('secret')).resolves.toBe(false);
    expect(document.querySelector('textarea')).toBeNull();
  });
});
