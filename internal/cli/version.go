package cli

import (
	"context"
	"fmt"

	"github.com/PdYrust/karvan/internal/buildinfo"
)

func runVersion(_ context.Context, streams IO, args []string) error {
	if err := parseFlags(newFlagSet("version", streams), args); err != nil {
		return err
	}
	_, err := fmt.Fprintln(streams.Stdout, buildinfo.String())
	return err
}
