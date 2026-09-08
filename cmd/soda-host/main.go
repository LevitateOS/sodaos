package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/levitateos/sodaos/internal/host"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	path := flag.String("config", "/etc/soda/host.json", "operator-owned runtime configuration")
	flag.Parse()
	if os.Geteuid() != 0 || os.Getenv("LISTEN_PID") != strconv.Itoa(os.Getpid()) || os.Getenv("LISTEN_FDS") != "1" {
		return fmt.Errorf("requires root and the soda-host systemd Unix socket")
	}
	c, err := host.LoadConfig(*path)
	if err != nil {
		return err
	}
	f := os.NewFile(3, "soda-host.socket")
	listener, err := net.FileListener(f)
	f.Close()
	if err != nil {
		return err
	}
	defer listener.Close()
	daemon := &host.Daemon{Config: c, Exec: host.Native{}}
	server := &http.Server{Handler: daemon, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 20 * time.Second, MaxHeaderBytes: 8192}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		daemon.CloseTerminals() // Includes hijacked streams, unlike HTTP Shutdown.
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
		close(done)
	}()
	err = server.Serve(listener)
	stop()
	<-done
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
