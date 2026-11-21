package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help")) {
		fmt.Fprintln(stdout, "Usage: gh corkboard [--help]")
		fmt.Fprintln(stdout, "A corkboard for essential notes.")
		return 0
	}

	fmt.Fprintf(stderr, "gh corkboard: unknown argument %q. Run gh corkboard --help.\n", args[0])
	return 2
}
