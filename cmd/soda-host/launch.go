package main

import (
	"context"
	"errors"
	"net"

	"github.com/levitateos/sodaos/internal/host"
)

// Both launch interfaces retire their own executions before the host exits.
func startRuntimeLaunchers(ctx context.Context, stop context.CancelFunc, daemon *host.Daemon) (func() error, error) {
	services := []struct {
		open     func() (*net.UnixListener, error)
		serve    func(context.Context, *net.UnixListener) error
		listener *net.UnixListener
	}{{open: daemon.OpenMuseListener, serve: daemon.ServeMuse}, {open: daemon.OpenGitListener, serve: daemon.ServeGit}}
	var results []<-chan error
	for index, service := range services {
		listener, err := service.open()
		if err != nil {
			stop()
			for _, previous := range services[:index] {
				if previous.listener != nil {
					_ = previous.listener.Close()
				}
			}
			return nil, err
		}
		services[index].listener = listener
	}
	for _, service := range services {
		if service.listener == nil {
			continue
		}
		result := make(chan error, 1)
		results = append(results, result)
		go func() {
			err := service.serve(ctx, service.listener)
			result <- err
			if err != nil {
				stop()
			}
		}()
	}
	return func() error {
		var failures []error
		for _, result := range results {
			failures = append(failures, awaitMuseShutdown(result))
		}
		return errors.Join(failures...)
	}, nil
}
