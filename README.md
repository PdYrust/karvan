<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/brand/karvan-mark-dark.svg">
    <img src="docs/brand/karvan-mark.svg" width="112" height="112" alt="Karvan">
  </picture>
</p>

<h1 align="center">Karvan</h1>

<p align="center">Move and merge users between Marzban and PasarGuard panels, with every UUID and credential kept.</p>

---

Karvan ships as one binary with a bilingual (Persian and English) web panel.

> Status: early development. Not ready for production use.

## Build from source

Requirements: Go 1.27, Node.js 24, pnpm 12.

```sh
make build        # web panel + bin/karvan
make check        # lint, tests, coverage and floor rules
```

First run on a server:

```sh
karvan init       # prints the address, username and password once
karvan serve
```

Lost access? `karvan reset -password` (also `-user`, `-path`, `-port`, `-totp`).

## Contributing

Read [CONSTRAINTS.md](CONSTRAINTS.md) before changing code and [DESIGN.md](DESIGN.md) before changing the web panel.

## License

[GNU Affero General Public License v3.0](LICENSE)
