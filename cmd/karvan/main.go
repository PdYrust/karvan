// Command karvan moves users between Marzban and PasarGuard panels and serves
// its web panel.
package main

import (
	"context"
	"os"

	"github.com/PdYrust/karvan/internal/cli"
)

func main() {
	os.Exit(cli.Run(context.Background(), os.Args[1:], cli.IO{
		Stdout:           os.Stdout,
		Stderr:           os.Stderr,
		StdoutIsTerminal: isTerminal(os.Stdout),
	}))
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
