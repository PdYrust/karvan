import { codeSpans } from '../../lib/format/code-spans';

/**
 * A translated sentence with machine values marked in backticks. Each value
 * keeps left to right in mono at about 0.9 em, so `.env` reads the same in a
 * Persian sentence as in an English one.
 */
export function CodeText({ text }: { text: string }) {
  return (
    <>
      {codeSpans(text).map((s, i) =>
        s.code ? (
          <bdi key={i} dir="ltr" className="font-mono text-[0.9em] wrap-anywhere">
            {s.text}
          </bdi>
        ) : (
          s.text
        ),
      )}
    </>
  );
}
