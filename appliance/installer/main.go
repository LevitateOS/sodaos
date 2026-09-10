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
	if len(os.Args) != 2 || !action(os.Args[1]) {
		fmt.Fprintln(os.Stderr, "usage: soda-install disk|continue|configure|enroll-key")
		os.Exit(2)
	}
	signals := []os.Signal{syscall.SIGTERM, syscall.SIGHUP}
	if os.Args[1] != "disk" {
		signals = append(signals, syscall.SIGINT)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), signals...)
	defer cancel()
	if err := installer.Run(ctx, os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func action(value string) bool {
	switch value {
	case "disk", "continue", "configure", "enroll-key", "enrollment-serve", "enrollment-receive":
		return true
	default:
		return false
	}
}
