package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/linuxhost"
	"github.com/levitateos/sodaos/internal/runners"
)

func main() { os.Exit(run()) }

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// A signal must also unblock a request decoder waiting on stdin.
	stopInput := context.AfterFunc(ctx, func() { _ = os.Stdin.Close() })
	defer stopInput()
	err := execute(ctx, os.Args[1:], os.Stdin, os.Stdout, func(ctx context.Context, action string, input io.Reader) (any, error) {
		// Preserve the helper's real/effective-root and original pkexec caller gate
		// before reading configuration, input or native runner state.
		actor, err := linuxhost.PKExecCaller()
		if err != nil {
			return nil, err
		}
		cfg, err := config.Load("/etc/soda/dashboard.json")
		if err != nil {
			return nil, errors.New("runner integration configuration unavailable")
		}
		native := runners.NewNative()
		coordinator := runners.Coordinator{
			ForgejoURL:       cfg.ForgejoInternalURL,
			ForgejoPublicURL: cfg.ForgejoURL,
			Authorizer:       runners.LinuxAuthorizer{Accounts: linuxhost.NewNative()},
			Local:            native,
			Lifecycle:        native,
		}
		return coordinator.Execute(ctx, actor, action, input)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "soda-runners:", err)
		if errors.Is(err, errUsage) {
			return 2
		}
		return 1
	}
	return 0
}
