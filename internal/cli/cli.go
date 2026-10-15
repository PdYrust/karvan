// Package cli implements the karvan command line.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

// IO is the set of streams a command reads from and writes to.
type IO struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type command struct {
	name    string
	summary string
	run     func(ctx context.Context, streams IO, args []string) error
}

// errUsage marks errors caused by wrong arguments; they exit with status 2.
var errUsage = errors.New("usage")

func commands() []command {
	return []command{
		{name: "version", summary: "Print version information", run: runVersion},
	}
}

// Run executes the command named by args[0] and returns the process exit code.
func Run(ctx context.Context, args []string, streams IO) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		printUsage(streams.Stdout)
		return 0
	}
	for _, c := range commands() {
		if c.name != args[0] {
			continue
		}
		err := c.run(ctx, streams, args[1:])
		switch {
		case err == nil:
			return 0
		case errors.Is(err, flag.ErrHelp):
			return 0
		case errors.Is(err, errUsage):
			fmt.Fprintf(streams.Stderr, "karvan %s: %v\n", c.name, err)
			return 2
		default:
			fmt.Fprintf(streams.Stderr, "karvan %s: %v\n", c.name, err)
			return 1
		}
	}
	fmt.Fprintf(streams.Stderr, "karvan: unknown command %q\n\n", args[0])
	printUsage(streams.Stderr)
	return 2
}

func printUsage(w io.Writer) {
	var b strings.Builder
	b.WriteString("Usage: karvan <command> [flags]\n\nCommands:\n")
	for _, c := range commands() {
		fmt.Fprintf(&b, "  %-9s %s\n", c.name, c.summary)
	}
	b.WriteString("\nRun \"karvan <command> -h\" for the flags of a command.\n")
	fmt.Fprint(w, b.String())
}

func newFlagSet(name string, streams IO) *flag.FlagSet {
	fs := flag.NewFlagSet("karvan "+name, flag.ContinueOnError)
	fs.SetOutput(streams.Stderr)
	return fs
}

func parseFlags(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return fmt.Errorf("%w: %w", errUsage, err)
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("%w: unexpected argument %q", errUsage, fs.Arg(0))
	}
	return nil
}
