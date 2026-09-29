package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
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
	if err := run(); err != nil {
		slog.Error("dashboard stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	path := flag.String("config", "/etc/soda/dashboard.json", "configuration file")
	extensionSocket := flag.String("extension-socket", "", "optional private native-extension Unix socket")
	flag.Parse()
	app, database, err := openDashboard(*path)
	if err != nil {
		return err
	}
	defer func() {
		if err := database.Close(); err != nil {
			slog.Error("database close failed", "error", err)
		}
	}()
	server := &http.Server{Addr: app.Config.Listen, Handler: app, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	privateServer, listener, err := extensionHTTPServer(*extensionSocket, app)
	if err != nil {
		return err
	}
	if listener != nil {
		defer listener.Close()
		serveExtensionSocket(privateServer, listener, stop)
	}
	stopped := beginShutdown(ctx, app, server, privateServer)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	stop()
	<-stopped
	return nil
}

func openDashboard(path string) (*web.Server, *store.Store, error) {
	if err := avatar.Validate(); err != nil {
		return nil, nil, err
	}
	c, err := config.Load(path)
	if err != nil {
		return nil, nil, err
	}
	key, err := config.GrantKey(c.GrantKeyFile)
	if err != nil {
		return nil, nil, err
	}
	database, err := store.OpenEncrypted(c.Database, key)
	if err != nil {
		return nil, nil, err
	}
	return web.New(c, database), database, nil
}

func extensionHTTPServer(socket string, app *web.Server) (*http.Server, net.Listener, error) {
	if socket == "" {
		return nil, nil, nil
	}
	return extensionListener(socket, app.ExtensionHandler())
}

func serveExtensionSocket(server *http.Server, listener net.Listener, stop context.CancelFunc) {
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Error("extension listener stopped", "error", err)
			stop()
		}
	}()
}

func beginShutdown(ctx context.Context, app *web.Server, server, private *http.Server) <-chan struct{} {
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		<-ctx.Done()
		app.CloseTerminals()
		shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		shutdownServer(shutdown, server, "HTTP")
		if private != nil {
			shutdownServer(shutdown, private, "extension")
		}
	}()
	return stopped
}

func shutdownServer(ctx context.Context, server *http.Server, name string) {
	if err := server.Shutdown(ctx); err != nil {
		slog.Error(name+" server shutdown failed", "error", err)
	}
}
