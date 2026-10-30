# ADR 0001: One Go binary with an embedded React panel

## Status
Accepted, 2026-10-16

## Context
Karvan runs on the destination panel's server, often a small VPS with 1 to 2 GB of memory, amd64 or arm64. Operators install it with one command and expect updates without a toolchain. It holds secrets for whole panels, so every dependency is attack surface.

## Decision
- **Go, standard library first.** `net/http` routing, `flag`, `log/slog`, `database/sql`. Three Go dependencies: `golang.org/x/crypto` (argon2id, XChaCha20-Poly1305), `modernc.org/sqlite`, and nothing else.
- **SQLite through a pure-Go driver** (`modernc.org/sqlite`), so cross-compiling needs no cgo toolchain. Own forward-only migration runner over embedded SQL files.
- **Web panel:** React 19, Vite, Tailwind 4 and Kumo 2.14, built into `web/dist` and embedded with `go:embed`. pnpm with exact pins; it refuses releases younger than a day and blocks install scripts.
- **No i18n or date library.** Two locales with typed dictionaries; `Intl` gives the Jalali calendar, Tehran time and Persian digits.

## Alternatives considered
- **cgo SQLite (`mattn/go-sqlite3`):** faster, but every release would need a C cross-compiler per architecture.
- **A router or CLI framework (chi, cobra):** not needed for about ten routes and four commands.
- **react-i18next and a date library:** larger bundles for what `Intl` already does.

## Consequences
- The release is one static binary per architecture.
- Kumo's Button pulls in its tooltip code, and React 19 alone is about 60 KB gzip, so the sign-in page is about 136 KB of JS (budget 145 KB in `web/scripts/bundle-size.mjs`).
- ESLint is 10.12.0 instead of the planned 9.39.5: ESLint 9 is deprecated upstream, and `eslint-plugin-jsx-a11y` 6.10.2 works with 10 even though its peer range stops at 9. TypeScript stays at 6.0.3 because typescript-eslint supports `<6.1`.
