import { afterEach, describe, expect, it, vi } from 'vitest';
import { readLocale } from '../lib/preferences/preferences';
import { changeLocale } from './change-locale';
afterEach(() => localStorage.clear());
describe('changeLocale', () => {
  it('saves the new language and reloads the page', () => {
    const reload = vi.fn();
    changeLocale('en', 'fa', reload);
    expect(readLocale()).toBe('en');
    expect(reload).toHaveBeenCalledOnce();
  });
  it('does nothing for the language already shown', () => {
    const reload = vi.fn();
    changeLocale('fa', 'fa', reload);
    expect(reload).not.toHaveBeenCalled();
  });
});
