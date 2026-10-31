package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/PdYrust/karvan/internal/netports"
)

// runPorts prints the TCP ports this host listens on, or with --free one
// random port nothing listens on, for the installer to propose.
func runPorts(_ context.Context, streams IO, args []string) error {
	fs := newFlagSet("ports", streams)
	free := fs.Bool("free", false, fmt.Sprintf("print one free port from %d to %d", netports.FreeLow, netports.FreeHigh))
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if *free {
		p, err := netports.Free()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(streams.Stdout, p)
		return err
	}
	ports, err := netports.Listening()
	if err != nil {
		return err
	}
	var b strings.Builder
	for _, p := range ports {
		fmt.Fprintln(&b, p)
	}
	_, err = fmt.Fprint(streams.Stdout, b.String())
	return err
}
