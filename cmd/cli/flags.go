package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

func HelpRequested(args []string) bool {
	return len(args) == 1 && IsHelpArg(args[0])
}

func IsHelpArg(arg string) bool {
	return arg == "help" || arg == "-h" || arg == "--help"
}

func NewFlagSet(name, usage string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s\n\n", usage)
		fmt.Fprintln(fs.Output(), "Options:")
		fs.PrintDefaults()
	}
	return fs
}

func Parse(fs *flag.FlagSet, args []string) (bool, error) {
	if HelpRequested(args) {
		fs.Usage()
		return true, nil
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}
