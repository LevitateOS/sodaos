// soda-install is an explicit operator console adapter, not a boot-time installer
// daemon. The disk subcommand refuses non-live CoreOS hosts.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/levitateos/sodaos/internal/installer"
)

func main() {
	if len(os.Args) != 2 || (os.Args[1] != "disk" && os.Args[1] != "continue") {
		fmt.Fprintln(os.Stderr, "usage: soda-install disk|continue (interactive native CoreOS root only)")
		os.Exit(2)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()
	if err := installer.Run(ctx, os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
