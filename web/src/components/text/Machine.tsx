import type { ReactNode } from 'react';

/** A machine value (host name, kernel, path) kept left to right inside Persian text. */
export function Machine({ children }: { children: ReactNode }) {
  return (
    <bdi dir="ltr" className="font-mono text-sm break-all">
      {children}
    </bdi>
  );
}
