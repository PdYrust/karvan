//go:build !dbtest

package paneldbtest

// serverFlavors is empty in the fast gate; `make test-db` adds the servers.
var serverFlavors []string
