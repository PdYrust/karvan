import type { ReactNode } from 'react';

/**
 * A machine value (host name, kernel, path) kept left to right inside
 * Persian text. A long one wraps at a hyphen or a space when it can.
 */
export function Machine({ children }: { children: ReactNode }) {
  return (
    <bdi dir="ltr" className="font-mono text-sm wrap-anywhere">
      {children}
    </bdi>
  );
}
