# Constraints

The quality bar for every change. A failing check stops the change until it is fixed.
This file does not get weakened to make a change pass; loosening a rule is its own commit with a reason.

Last reviewed: 2026-10-15.

## Why these rules exist
- The file tree reads like a map: one package or folder, one responsibility, no god files, no dumping-ground folders.
- The web panel looks polished on a phone and on a desktop, in Persian and English, in dark and light.
- Karvan holds tokens and database access for whole panels, so security checks are never optional.

## Floor (always enforced)
- No suppression comments: `@ts-ignore`, `@ts-nocheck`, `eslint-disable`, `istanbul ignore`. Go `//nolint` only with a specific linter and a reason.
- No skipped or focused tests (`t.Skip`, `.skip`, `.only`).
- No unfinished work: no `TODO`/`FIXME`, no stubs that fail at runtime.
- No secrets in source. No secret in a log line.
- No reference to AI or automation in files, commit messages or authors. Every commit is authored by `YrustPd <PdYrust@outlook.com>`.
- Colors only come from `web/src/styles/tokens.css`. No color literals anywhere else in `web/src`.

## Enforced with numbers
| Dimension | Rule | Checked by | Runs at |
|---|---|---|---|
| Floor | all floor rules above | `scripts/check-floor.sh` | `make check` |
| Go format and lint | zero issues | `golangci-lint run` (`.golangci.yml`) | `make check` |
| Go file size | 500 lines or less per file (aim for 300) | revive `file-length-limit` | `make check` |
| Go function size | 60 lines or less, 40 statements or less | `funlen` | `make check` |
| Go complexity | cyclomatic 15 or less | `gocyclo` | `make check` |
| Go tests | all pass | `go test -cover ./...` (inside `scripts/check-coverage.sh`) | `make check` |
| Go races | none found | `go test -race ./...` | `make check-full` |
| Go coverage | 80% or more statements in `config`, `secret`, `store`, `settings`, `auth`, `audit`, `httpapi` and its subpackages | `scripts/check-coverage.sh` | `make check` |
| Go vulnerabilities | no findings | `govulncheck ./...` | `make check-full` |
| TypeScript | zero errors, `strict` on | `tsc --noEmit` | `make check` |
| Fast gate time | `make check` finishes in about 90 s or less on a 2-core machine | `time make check` | when checks are added |
| Web lint and format | zero errors | `eslint`, `prettier --check` | `make check` |
| Component size | 250 lines or less per `.tsx` (aim for 200) | ESLint `max-lines` | `make check` |
| Web tests and coverage | all pass; 80% or more lines in `web/src/lib` and `web/src/i18n` | `vitest run --coverage` | `make check` |
| Bundle (gzip, fonts excluded) | sign-in JS 145 KB or less; 3D logo chunk 180 KB or less; app JS 260 KB or less; stylesheet 35 KB or less | `web/scripts/bundle-size.mjs` (follows the Vite manifest) | `make web` |
| Web dependencies | nothing at high or above | `pnpm audit --audit-level high` | `make check-full` |

The coverage floor (80%) is high enough that new logic needs a test, and low enough to leave room for wiring code.
The bundle numbers keep the sign-in page usable on a slow mobile connection. They were first guessed at 110 KB for the sign-in page; measured, React 19 alone is about 60 KB and Kumo's Button and Input about 55 KB, so the budget is today's 132 KB plus a little room. Fonts are left out because they cache across visits.
The Lighthouse numbers are the "good" band of Core Web Vitals; accessibility is 100 because the panel must work with a keyboard and a screen reader.

## Reviewed by a person, every page
- Tested at 360 px and 1440 px, in fa and en, dark and light, keyboard only.
- Follows `DESIGN.md`. If `DESIGN.md` is wrong, it is fixed first.
- Lighthouse mobile on the sign-in page and the overview: performance 90 or more, accessibility 100, best practices 95 or more. It needs a running install and Chrome, so it runs by hand (`lighthouse <url> --form-factor=mobile`) until CI can start one. On a slow machine, set `--throttling.cpuSlowdownMultiplier` so the host plus the multiplier matches a mid-range phone.

## Measured, not yet meeting the bar
| Metric | Today | Target | Direction |
|---|---|---|---|
| Lighthouse mobile performance, sign-in | 70 to 83 (2026-10-16, host benchmark 340 to 550, CPU multiplier 1) | 90 | must not fall |
| Lighthouse mobile performance, overview | 67 to 72 (same run) | 90 | must not fall |

## Exceptions
| ID | Rule | Path | Reason | Owner | Expires |
|---|---|---|---|---|---|
