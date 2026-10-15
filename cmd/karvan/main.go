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
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}))
}
