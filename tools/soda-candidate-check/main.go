// soda-candidate-check verifies one delivered soda-build candidate directory
// through the existing release validation owner. It never builds, installs,
// publishes or changes trust.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/levitateos/sodaos/internal/release/deliver"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type checkFlags struct {
	candidate, arch, soda, forgejo string
}

func parseCheckFlags(args []string) (checkFlags, error) {
	f := flag.NewFlagSet("soda-candidate-check", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	candidate := f.String("candidate", "", "delivered candidate artifacts directory")
	arch := f.String("arch", "", "requested native architecture")
	soda := f.String("soda-revision", "", "requested Soda source revision")
	forgejo := f.String("forgejo-revision", "", "requested Fountain source revision")
	if err := f.Parse(args); err != nil || len(f.Args()) != 0 {
		return checkFlags{}, errors.New("invalid candidate check flags")
	}
	flags := checkFlags{*candidate, *arch, *soda, *forgejo}
	if flags.candidate == "" || flags.arch == "" || flags.soda == "" || flags.forgejo == "" {
		return checkFlags{}, errors.New("candidate, arch, soda-revision and forgejo-revision are required")
	}
	return flags, nil
}

func run(args []string) error {
	flags, err := parseCheckFlags(args)
	if err != nil {
		return err
	}
	return deliver.CheckCandidate(flags.candidate, flags.arch, flags.soda, flags.forgejo)
}
