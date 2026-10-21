/** A piece of a translated sentence: plain words, or a machine value. */
export interface Span {
  text: string;
  code: boolean;
}

/**
 * Splits a message into its words and the machine values marked with
 * backticks ("Add `SQLALCHEMY_DATABASE_URL` to it."), so each value can be
 * kept left to right inside a Persian sentence. Without that, ".env" next
 * to Persian words reads "env.".
 */
export function codeSpans(text: string): Span[] {
  return text
    .split('`')
    .map((part, i) => ({ text: part, code: i % 2 === 1 }))
    .filter((s) => s.text !== '');
}
