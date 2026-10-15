package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/levitateos/sodaos/internal/host"
)

func main() { os.Exit(run()) }

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client := host.NewClient("/run/soda/host.sock")
	if err := execute(ctx, os.Stdout, client.TailnetSettings); err != nil {
		fmt.Fprintln(os.Stderr, "soda-tailnet:", err)
		return 1
	}
	return 0
}
