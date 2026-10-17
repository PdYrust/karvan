import { describe, expect, it } from 'vitest';
import { isTyping, sequenceWindowMs, shortcutReader } from './shortcuts';

describe('shortcutReader', () => {
  const read = () => shortcutReader({ KeyO: '/', KeyS: '/settings' });

  it('opens the help and the page search', () => {
    const r = read();
    expect(r('Slash', true, 0)).toEqual({ type: 'help' });
    expect(r('Slash', false, 0)).toEqual({ type: 'page-search' });
  });

  it('goes to a page after g', () => {
    const r = read();
    expect(r('KeyG', false, 100)).toBeNull();
    expect(r('KeyS', false, 400)).toEqual({ type: 'goto', to: '/settings' });
    // The sequence is used up.
    expect(r('KeyS', false, 500)).toBeNull();
  });

  it('forgets g after the window or another key', () => {
    const r = read();
    r('KeyG', false, 0);
    expect(r('KeyO', false, sequenceWindowMs + 1)).toBeNull();
    r('KeyG', false, 10_000);
    expect(r('KeyX', false, 10_100)).toBeNull();
    expect(r('KeyO', false, 10_200)).toBeNull();
    expect(r('KeyG', true, 20_000)).toBeNull();
    expect(r('KeyO', false, 20_100)).toBeNull();
  });
});

describe('isTyping', () => {
  it('knows fields from buttons', () => {
    const text = document.createElement('input');
    const box = document.createElement('input');
    box.type = 'checkbox';
    const area = document.createElement('textarea');
    const div = document.createElement('div');
    const combo = document.createElement('div');
    combo.setAttribute('role', 'combobox');
    expect(isTyping(text)).toBe(true);
    expect(isTyping(box)).toBe(false);
    expect(isTyping(area)).toBe(true);
    expect(isTyping(div)).toBe(false);
    expect(isTyping(combo)).toBe(true);
    expect(isTyping(null)).toBe(false);
  });
});
