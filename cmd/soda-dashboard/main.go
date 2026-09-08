package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/levitateos/sodaos/internal/avatar"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web"
)

func main() {
	path := flag.String("config", "/etc/soda/dashboard.json", "configuration file")
	flag.Parse()
	if err := avatar.Validate(); err != nil {
		slog.Error("embedded avatar style is invalid")
		os.Exit(1)
	}
	c, err := config.Load(*path)
	if err != nil {
		slog.Error("configuration", "error", err)
		os.Exit(1)
	}
	key, err := config.GrantKey(c.GrantKeyFile)
	if err != nil {
		slog.Error("grant key startup failed; database not opened")
		os.Exit(1)
	}
	db, err := store.OpenEncrypted(c.Database, key)
	if err != nil {
		slog.Error("database startup failed")
		os.Exit(1)
	}
	defer db.Close()
	app := web.New(c, db)
	server := &http.Server{Addr: c.Listen, Handler: app, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		<-ctx.Done()
		app.CloseTerminals()
		shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		server.Shutdown(shutdown)
	}()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("HTTP server stopped", "error", err)
		os.Exit(1)
	}
	stop()
	<-stopped
}
