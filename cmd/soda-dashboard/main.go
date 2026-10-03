package main

import (
	"context"
	"errors"
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
	operatorSocket := flag.String("operator-socket", "", "optional private factory-operator Unix socket")
	operatorUID := flag.Int("operator-uid", -1, "required OS peer UID when the operator socket is set")
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
	if err = app.StartCoordinator(context.Background()); err != nil {
		return err
	}
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
	operatorServer, operatorListener, err := operatorEndpoint(*operatorSocket, *operatorUID, app)
	if err != nil {
		return err
	}
	if operatorListener != nil {
		defer operatorListener.Close()
		serveOperatorSocket(operatorServer, operatorListener, stop)
	}
	stopped := beginShutdown(ctx, app, server, privateServer, operatorServer)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	stop()
	<-stopped
	return nil
}

func operatorEndpoint(socket string, uid int, app *web.Server) (*http.Server, net.Listener, error) {
	if socket == "" {
		return nil, nil, nil
	}
	if uid < 0 || uid > 0xffffffff {
		return nil, nil, errors.New("operator socket requires an explicit OS peer UID")
	}
	return operatorHTTPServer(socket, uint32(uid), app.Coordinator)
}

func serveOperatorSocket(server *http.Server, listener net.Listener, stop context.CancelFunc) {
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Error("operator listener stopped", "error", err)
			stop()
		}
	}()
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
	dsn, err := config.Secret(c.DatabaseDSNFile)
	if err != nil {
		return nil, nil, err
	}
	database, err := store.OpenEncrypted(dsn, key)
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

func beginShutdown(ctx context.Context, app *web.Server, server, private, operator *http.Server) <-chan struct{} {
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		<-ctx.Done()
		app.CloseTerminals()
		app.CloseCoordinator()
		shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		shutdownServer(shutdown, server, "HTTP")
		if private != nil {
			shutdownServer(shutdown, private, "extension")
		}
		if operator != nil {
			shutdownServer(shutdown, operator, "operator")
		}
	}()
	return stopped
}

func shutdownServer(ctx context.Context, server *http.Server, name string) {
	if err := server.Shutdown(ctx); err != nil {
		slog.Error(name+" server shutdown failed", "error", err)
	}
}
