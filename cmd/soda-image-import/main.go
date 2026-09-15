// soda-image-import is the image-based appliance's fixed native import phase.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
)

func run() error {
	if os.Geteuid() != 0 || len(os.Args) != 1 {
		return fmt.Errorf("root and no arguments required")
	}
	p, err := deliver.Load(deliver.Path)
	if err != nil {
		return fmt.Errorf("appliance release metadata unavailable")
	}
	if err = build.RequireNative(p.Architecture); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	return deliver.NativeImport(ctx, p)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
