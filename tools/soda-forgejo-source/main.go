// soda-forgejo-source is a build-only tool; never an appliance service.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/levitateos/sodaos/internal/forgejobuild"
)

func main() {
	lock := flag.String("lock", "appliance/forgejo/source.lock.json", "reviewed Forgejo source lock")
	archive := flag.String("archive", "", "existing archive to verify; otherwise fetch the locked public commit")
	out := flag.String("out", "", "fresh output directory under an existing private/owned parent")
	flag.Parse()
	if *out == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := forgejobuild.Prepare(ctx, *lock, *archive, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Forgejo source prepared; no compilation, installation or interface conformance claimed.")
}
