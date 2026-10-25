import { randomPort } from '../../../lib/api/account';
import { latinDigits } from '../security/code';

export type PortChoice = { kind: 'random' } | { kind: 'typed'; text: string };

/** A typed port: at most five digits, Persian ones read as the same digits. */
export const portDigits = (typed: string) => latinDigits(typed).slice(0, 5);

/** The port to ask the server for, or null while the choice cannot be sent. */
export function chosenPort(choice: PortChoice, current: number): number | null {
  if (choice.kind === 'random') return randomPort;
  if (!/^\d{1,5}$/.test(choice.text)) return null;
  const port = Number(choice.text);
  return port >= 1 && port <= 65535 && port !== current ? port : null;
}
