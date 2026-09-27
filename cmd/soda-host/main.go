package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/tailnet"
)

var errTailnetPreparation = errors.New("tailnet preparation unconfirmed; observe and explicitly retry")

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

func validTailnetActionArgs(action, project string) bool {
	return os.Geteuid() == 0 && tailnet.ValidProject(project) && (action == "run" || action == "stop") && flag.NArg() == 0
}

func runTailnetAction(path, action, project string) error {
	if !validTailnetActionArgs(action, project) {
		return tailnet.ErrInvalid
	}
	c, e := host.LoadConfig(path)
	if e != nil {
		return tailnet.ErrUnavailable
	}
	if !c.TailnetManagement || c.TailnetImage == "" {
		return nil
	}
	d := host.NewCompanionDaemon(c)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	phase, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if action == "stop" {
		bounded, done := context.WithTimeout(phase, 30*time.Second)
		defer done()
		return d.Companion.StopTailnet(bounded, project)
	}
	cid, e := d.Companion.StartTailnet(phase, project)
	if e != nil {
		return errors.Join(errTailnetPreparation, e)
	}
	cancel()
	return d.Companion.WaitTailnet(ctx, cid)
}

func serveHostSocket(c host.Config) error {
	if os.Geteuid() != 0 || os.Getenv("LISTEN_PID") != strconv.Itoa(os.Getpid()) || os.Getenv("LISTEN_FDS") != "1" {
		return fmt.Errorf("requires root and the soda-host systemd Unix socket")
	}
	f := os.NewFile(3, "soda-host.socket")
	listener, err := net.FileListener(f)
	f.Close()
	if err != nil {
		return err
	}
	defer listener.Close()
	daemon := host.NewDaemon(c)
	server := &http.Server{Handler: daemon, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 20 * time.Second, MaxHeaderBytes: 8192}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	museListener, err := daemon.OpenMuseListener()
	if err != nil {
		return err
	}
	museErrors := make(chan error, 1)
	if museListener != nil {
		defer func() { _ = museListener.Close() }()
		go func() {
			err := daemon.ServeMuse(ctx, museListener)
			museErrors <- err
			if err != nil {
				stop()
			}
		}()
	}
	done := make(chan struct{})
	var museFailure error
	go func() {
		<-ctx.Done()
		if museListener != nil {
			museFailure = awaitMuseShutdown(museErrors)
		}
		daemon.CloseTerminals()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
		close(done)
	}()
	err = server.Serve(listener)
	stop()
	<-done
	return hostSocketResult(err, museFailure)
}

func run() error {
	path := flag.String("config", "/etc/soda/host.json", "operator-owned runtime configuration")
	action := flag.String("tailnet-action", "", "fixed native companion phase: run or stop")
	project := flag.String("project", "", "exact native project ID for a companion phase")
	flag.Parse()
	if *action != "" {
		return runTailnetAction(*path, *action, *project)
	}
	if *project != "" || flag.NArg() != 0 {
		return tailnet.ErrInvalid
	}
	c, err := host.LoadConfig(*path)
	if err != nil {
		return err
	}
	return serveHostSocket(c)
}

func awaitMuseShutdown(result <-chan error) error {
	timer := time.NewTimer(35 * time.Second)
	defer timer.Stop()
	select {
	case err := <-result:
		return err
	case <-timer.C:
		return errors.New("muse execution retirement remains unconfirmed")
	}
}

func hostSocketResult(err, museFailure error) error {
	if museFailure != nil {
		return museFailure
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
