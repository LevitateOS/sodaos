// soda-factory is the unprivileged operator interface to bounded software work.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/levitateos/sodaos/internal/factory/control"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "soda-factory:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("soda-factory", flag.ContinueOnError)
	path := flags.String("config", "", "private operator configuration file")
	if err := flags.Parse(args); err != nil {
		return err
	}
	config, policy, err := control.Load(*path)
	if err != nil {
		return err
	}
	controller, err := openController(ctx, config, policy, flags.Args())
	if err != nil {
		return err
	}
	defer func() { _ = controller.Store.Close() }()
	return dispatch(ctx, controller, flags.Args())
}

func dispatch(ctx context.Context, c *control.Controller, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: soda-factory --config FILE admit ISSUE DELIVERY | run ATTEMPT | status ATTEMPT | cancel ATTEMPT | recover")
	}
	switch args[0] {
	case "admit":
		return admit(ctx, c, args[1:])
	case "recover":
		if len(args) != 1 {
			return errors.New("recover takes no arguments")
		}
		return c.Recover(ctx)
	case "run", "cancel", "status":
		return attempt(ctx, c, args)
	default:
		return errors.New("unknown factory command")
	}
}

func admit(ctx context.Context, c *control.Controller, args []string) error {
	if len(args) != 2 {
		return errors.New("admit requires issue number and stable delivery ID")
	}
	number, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return errors.New("invalid issue number")
	}
	a, created, err := c.Admit(ctx, number, args[1])
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"attempt": a, "created": created})
}

func attempt(ctx context.Context, c *control.Controller, args []string) error {
	if len(args) != 2 {
		return errors.New("command requires one attempt ID")
	}
	switch args[0] {
	case "run":
		return c.Execute(ctx, args[1])
	case "cancel":
		return c.Cancel(ctx, args[1])
	default:
		a, err := c.Store.FactoryAttempt(ctx, args[1])
		if err != nil {
			return err
		}
		runs, err := c.Store.FactoryRuns(ctx, args[1])
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"attempt": a, "runs": runs})
	}
}

func openController(ctx context.Context, c control.Config, policy string, args []string) (*control.Controller, error) {
	if len(args) > 0 {
		switch args[0] {
		case "status", "cancel", "recover":
			return control.OpenLocal(c, policy)
		}
	}
	return control.Open(ctx, c, policy)
}
