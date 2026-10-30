import type { CertFacts } from './rules';

/**
 * A list as the server sent it, or [] for null: an older server writes an empty Go list as null,
 * and a page must never see one.
 */
export function list<T>(xs: T[] | null | undefined): T[] {
  return xs ?? [];
}

/** Certificate facts with their lists. */
export function factsOf(f: CertFacts): CertFacts {
  return { ...f, domains: list(f.domains), warnings: list(f.warnings) };
}
