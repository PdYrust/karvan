/**
 * The digits of typed text: Persian and Arabic digits count as the same
 * digits, since a Persian keyboard types them, and everything else goes.
 */
export function latinDigits(typed: string): string {
  return typed
    .replace(/[\u06F0-\u06F9]/g, (d) => String(d.charCodeAt(0) - 0x06f0))
    .replace(/[\u0660-\u0669]/g, (d) => String(d.charCodeAt(0) - 0x0660))
    .replace(/\D/g, '');
}

/** A typed code from an authenticator app: at most six digits. */
export const codeDigits = (typed: string) => latinDigits(typed).slice(0, 6);

/** A two-step secret in groups of four, as authenticator apps show it. */
export function groupSecret(secret: string): string {
  return (secret.match(/.{1,4}/g) ?? []).join(' ');
}
