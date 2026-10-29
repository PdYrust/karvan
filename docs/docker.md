# Karvan in Docker

The image `ghcr.io/pdyrust/karvan:<version>` (linux/amd64 and linux/arm64) holds one file on top of `gcr.io/distroless/static-debian12`: the same static `karvan` binary as the release archive, with the web panel built in. There is no shell, no Docker CLI and nothing to install. It runs as root, like the archive install.

**Warning:** the compose file mounts `/var/run/docker.sock`. Whoever controls this container controls Docker on the host, and through it the whole host. Karvan needs the socket to stop and start a panel for a Safe Mode restore. To run without it, remove that line and set `"panelControl": "off"` in `/etc/karvan/config.json`; you then stop and start the panel yourself around a full backup restore.

## Set up

Copy [`packaging/docker/compose.yml`](../packaging/docker/compose.yml) to the server, set the image tag, then:

```sh
docker compose run --rm karvan init          # or: init -port 8443
docker compose up -d
```

`init` prints the address, username and password once. Save them. A second `init` refuses to run. To get new details, stop the container and run `docker compose run --rm karvan reset -password` (see `reset -h` for the path, username and port).

## Volumes

Each path is mounted at the same path inside the container, so the paths Karvan finds in panel ENV files are right on both sides.

| Host path | Mode | Why |
|---|---|---|
| `/var/lib/karvan` | read-write | Karvan's data: its database and Safe Mode backups |
| `/etc/karvan` | read-write | `config.json` and `secret.key`; `init` writes them |
| `/opt` | read only | panel folders (`.env`, compose file); Karvan scans here for panels |
| `/var/lib/marzban` | read-write | Marzban's data: its SQLite database and Xray config, which a migration writes |
| `/var/lib/pasarguard` | read-write | the same for PasarGuard |
| `/root/cert` | read only | panel certificates the bridge reads |
| `/etc/letsencrypt` | read only | certificates from certbot |
| `/var/run/docker.sock` | read-write | stop and start panels (see the warning above) |

Docker creates a missing host folder as an empty one. Remove the line of a panel you do not run. If a panel lives elsewhere (another folder than `/opt/<panel>`, or its data outside `/var/lib`), add a mount for that path at the same path. Karvan only reads panel ENV files and never changes them.

## Ports

The container uses the host network (`network_mode: host`), so nothing is published with `ports:`:

- the panel port that `init` chose or `-port` set,
- each rule's port, which the bridge opens while Karvan runs (Docker cannot publish ports added later),
- port 80, while a Let's Encrypt HTTP-01 order runs or when "keep port 80" is on.

Panel databases that listen on `127.0.0.1` stay reachable. Open the panel port, the rule ports and 80 in the host firewall as needed.

## Health

The image's health check runs `karvan health`, which passes while `karvan serve` keeps writing its heartbeat. `docker compose ps` shows it.

## Update

Before an update, stop the container and copy `/var/lib/karvan`. Set the new tag in `compose.yml`, then `docker compose pull && docker compose up -d`. An older version does not open a database that a newer one has migrated: to go back, put the copy back with the container stopped, then start the older tag.
