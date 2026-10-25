import { describe, expect, it } from 'vitest';
import { waitingLink } from './waiting';

const expiresAt = '2026-10-25T10:10:00Z';
const http = 'http://203.0.113.5:24364/oldPath/';
const https = 'https://panel.example.com:24364/oldPath/';

describe('waitingLink', () => {
  it('opens a new path started here on its Access tab over HTTPS', () => {
    const pending = { port: 24364, pathChanged: true, expiresAt, fromNew: false };
    const started = { port: 24364, path: 'n3w', expiresAt };
    expect(waitingLink(pending, started, https)).toEqual({
      url: 'https://panel.example.com:24364/n3w/settings/access',
      signInFirst: false,
    });
  });

  it('opens a new path over plain HTTP at sign-in, since the session stays behind', () => {
    const pending = { port: 24364, pathChanged: true, expiresAt, fromNew: false };
    const started = { port: 24364, path: 'n3w', expiresAt };
    expect(waitingLink(pending, started, http)).toEqual({
      url: 'http://203.0.113.5:24364/n3w/',
      signInFirst: true,
    });
  });

  it('cannot link a new path it did not start', () => {
    const pending = { port: 24364, pathChanged: true, expiresAt, fromNew: false };
    expect(waitingLink(pending, null, https).url).toBeNull();
    const other = { port: 24364, path: 'old-start', expiresAt: '2026-10-25T10:01:00Z' };
    expect(waitingLink(pending, other, https).url).toBeNull();
  });

  it('links a new port with this page path, even after a reload', () => {
    const pending = { port: 31000, pathChanged: false, expiresAt, fromNew: false };
    expect(waitingLink(pending, null, http)).toEqual({
      url: 'http://203.0.113.5:31000/oldPath/settings/access',
      signInFirst: false,
    });
  });
});
