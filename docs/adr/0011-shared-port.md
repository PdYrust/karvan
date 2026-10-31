# ADR 0011: The bridge serves the web panel on a shared HTTPS port
## Context
Many servers have one HTTPS port open to the world, usually 443, and the subscription bridge already listens there for the old panel links. An operator who wants the panel at `https://panel.example.com/<path>/` on that port needs something to split the requests between the panel and the bridge. Panel and bridge are one process; the bridge owns the rule listeners, their SNI picks and certificates. Some operators already run nginx or haproxy in front of their panels.
## Decision
- Karvan's own bridge listener hosts the panel. A domain-mode address with `shared` set has no listener of its own: the address server hands it to the bridge (`Bridge.SharePanel`) when the change starts, keeps it there while the change waits and after the confirm, and takes it back when the change lapses or the server stops.
- On that port, before the rule router, a request goes to the panel when it came over TLS, its `Host` (lower case, no port) is the panel domain and its path is under `/<path>` (`middleware.UnderBase`, constant time, as on the panel's own port). Everything else goes to the rules as before, the plain 404 included, so a wrong hidden path is answered exactly like any unknown link and reveals nothing.
- The confirm, cookies, HSTS (`domain` mode) and audit work as for any address change (ADR 0009, ADR 0010).
- Handshakes: a name a rule on the port serves keeps the rule's certificate, byte for byte. Only a name no rule serves gets the panel domain's certificate.
- The port stays open while the panel is on it, even after its last rule is deleted, so deleting a rule never locks the operator out. A shared port is not one of the panel's own ports, so rules can still be saved on it (`port_taken` does not count it).
- `karvan reset -port` or `-port-number` gives the panel a port of its own again: the way back if the bridge cannot open the shared port.
## Alternatives considered
- **An external nginx or haproxy in front, with a config snippet written by Karvan** (option a). More software to install and keep in step with the bridge's rules and certificates, and a second place that terminates TLS. Rejected.
- **The panel's listener hosts the bridge rules instead.** It would move the bridge's SNI and certificate logic into the panel's address server, which changes every rule's handshake path. Rejected.
## Consequences
- One process, one listener per port, one certificate store.
- Operators who already have nginx or haproxy in front keep it: they send `/sub/` (or the old panel's subscription path) to Karvan's bridge port and list the proxy's address in `trustedProxies` in `config.json`. Karvan writes no proxy config.
- A rule turned to plain HTTP on the shared port would make the port plain and the panel unreachable there. Saving such a rule is refused while an HTTPS rule holds the port (`port_mixed`); with none left, `karvan reset -port` is the way back.
