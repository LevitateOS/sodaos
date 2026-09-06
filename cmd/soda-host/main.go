package main

import (
	"flag"
	"fmt"
	"github.com/levitateos/sodaos/internal/host"
	"net"
	"net/http"
	"os"
	"strconv"
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
	server := &http.Server{Handler: &host.Daemon{Config: c, Exec: host.Native{}}, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 20 * time.Second, MaxHeaderBytes: 8192}
	return server.Serve(listener)
}
