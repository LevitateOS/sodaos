package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/runners"
	"github.com/levitateos/sodaos/internal/tailnet"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var errTailnetPreparation = errors.New("Tailnet preparation unconfirmed; observe and explicitly retry")

func exitStatus(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, errTailnetPreparation) {
		return 78
	}
	return 1
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitStatus(err))
	}
}
func run() error {
	path := flag.String("config", "/etc/soda/host.json", "operator-owned runtime configuration")
	action := flag.String("tailnet-action", "", "fixed native companion phase: run or stop")
	project := flag.String("project", "", "exact native project ID for a companion phase")
	flag.Parse()
	if *action != "" {
		if os.Geteuid() != 0 || !tailnet.ValidProject(*project) || (*action != "run" && *action != "stop") || flag.NArg() != 0 {
			return tailnet.ErrInvalid
		}
		c, e := host.LoadConfig(*path)
		if e != nil {
			return tailnet.ErrUnavailable
		}
		if !c.TailnetManagement || c.TailnetImage == "" {
			return nil
		}
		d := &host.Daemon{Config: c, Exec: host.Native{}, Tailnet: tailnet.NewProjectManagement()}
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer stop()
		phase, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		if *action == "stop" {
			bounded, done := context.WithTimeout(phase, 30*time.Second)
			defer done()
			return d.StopTailnet(bounded, *project)
		}
		cid, e := d.StartTailnet(phase, *project)
		if e != nil {
			return errors.Join(errTailnetPreparation, e)
		}
		cancel()
		return d.WaitTailnet(ctx, cid)
	}
	if *project != "" || flag.NArg() != 0 {
		return tailnet.ErrInvalid
	}
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
	runnerNative := runners.NewNative()
	daemon := &host.Daemon{Config: c, Exec: host.Native{}, Runners: &runners.Operations{Local: runnerNative, Lifecycle: runnerNative}}
	if c.TailnetManagement {
		daemon.Tailnet = tailnet.NewManagement()
		if c.TailnetImage != "" {
			daemon.Tailnet = tailnet.NewProjectManagement()
		}
	}
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
