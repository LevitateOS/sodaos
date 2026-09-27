// soda-fetch-muse stages the checksum-pinned native Muse executable for builds.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/levitateos/sodaos/internal/release/build"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "soda-fetch-muse:", err)
		os.Exit(1)
	}
}

func run() error {
	flags := flag.NewFlagSet("soda-fetch-muse", flag.ContinueOnError)
	arch := flags.String("arch", "", "native architecture")
	manifest := flags.String("manifest", "project-os/muse-release.json", "release manifest")
	out := flags.String("out", "", "native executable destination")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return build.FetchMuse(ctx, *manifest, *arch, *out)
}
