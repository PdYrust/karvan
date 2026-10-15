# Karvan

Karvan moves and merges users between Marzban and PasarGuard panels, keeping every user's UUID and credentials intact.

It ships as one binary with a bilingual (Persian and English) web panel.

> Status: early development. Not ready for production use.

## Build from source

Requirements: Go 1.27, Node.js 24, pnpm 12.

```sh
make build        # bin/karvan
make check        # lint, tests, coverage and floor rules
```

## Contributing

Read [CONSTRAINTS.md](CONSTRAINTS.md) before changing code. It defines the quality bar every change must meet.

## License

[GNU Affero General Public License v3.0](LICENSE)
