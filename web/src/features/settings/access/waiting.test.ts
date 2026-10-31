import { describe, expect, it } from 'vitest';
import { waitingLink } from './waiting';

const expiresAt = '2026-10-25T10:10:00Z';
const http = 'http://panel.example.com:24364/oldPath/';
const https = 'https://panel.example.com:24364/oldPath/';
const waiting = (port: number, pathChanged: boolean) => ({
  port,
  pathChanged,
  expiresAt,
  fromNew: false,
});

describe('waitingLink', () => {
  it('opens a new path started here on its Access tab over HTTPS', () => {
    const started = { port: 24364, path: 'n3w', expiresAt };
    expect(waitingLink(waiting(24364, true), started, https)).toEqual({
      url: 'https://panel.example.com:24364/n3w/settings/access',
      signInFirst: false,
    });
  });

  it('opens a new path over plain HTTP at sign-in, since the session stays behind', () => {
    const started = { port: 24364, path: 'n3w', expiresAt };
    expect(waitingLink(waiting(24364, true), started, http)).toEqual({
      url: 'http://panel.example.com:24364/n3w/',
      signInFirst: true,
    });
  });

  it('cannot link a new path it did not start', () => {
    expect(waitingLink(waiting(24364, true), null, https).url).toBeNull();
    const other = { port: 24364, path: 'old-start', expiresAt: '2026-10-25T10:01:00Z' };
    expect(waitingLink(waiting(24364, true), other, https).url).toBeNull();
  });

  it("links a new port with this page's scheme, host and path, even after a reload", () => {
    expect(waitingLink(waiting(31000, false), null, http)).toEqual({
      url: 'http://panel.example.com:31000/oldPath/settings/access',
      signInFirst: false,
    });
    expect(waitingLink(waiting(31000, false), null, https)).toEqual({
      url: 'https://panel.example.com:31000/oldPath/settings/access',
      signInFirst: false,
    });
  });
});
