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

func run() (runErr error) {
	path := flag.String("config", "/etc/soda/dashboard.json", "configuration file")
	extensionSocket := flag.String("extension-socket", "", "optional private native-extension Unix socket")
	operatorSocket := flag.String("operator-socket", "", "optional private factory-operator Unix socket")
	operatorUID := flag.Int("operator-uid", -1, "required OS peer UID when the operator socket is set")
	flag.Parse()
	app, database, err := openDashboard(*path)
	if err != nil {
		return err
	}
	defer func() { runErr = errors.Join(runErr, database.Close()) }()
	if err = app.StartCoordinator(context.Background()); err != nil {
		return err
	}
	defer func() { runErr = errors.Join(runErr, app.CloseCoordinator()) }()
	server := &http.Server{Addr: app.Config.Listen, Handler: app.Coordinator.AdmitHTTP(app), ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	serveErrors := make(chan error, 2)
	privateServer, listener, err := extensionHTTPServer(*extensionSocket, app)
	if err != nil {
		return err
	}
	if listener != nil {
		defer listener.Close()
		serveExtensionSocket(privateServer, listener, stop, serveErrors)
	}
	operatorServer, operatorListener, err := operatorEndpoint(*operatorSocket, *operatorUID, app)
	if err != nil {
		return err
	}
	if operatorListener != nil {
		defer operatorListener.Close()
		serveOperatorSocket(operatorServer, operatorListener, stop, serveErrors)
	}
	stopped := beginShutdown(ctx, app, server, privateServer, operatorServer)
	serveErr := server.ListenAndServe()
	stop()
	shutdownErr := <-stopped
	var listenerErr error
	for {
		select {
		case err := <-serveErrors:
			listenerErr = errors.Join(listenerErr, err)
		default:
			if errors.Is(serveErr, http.ErrServerClosed) {
				serveErr = nil
			}
			return errors.Join(serveErr, listenerErr, shutdownErr)
		}
	}
}

func operatorEndpoint(socket string, uid int, app *web.Server) (*http.Server, net.Listener, error) {
	if socket == "" {
		return nil, nil, nil
	}
	if uid < 0 || uid > 0xffffffff {
		return nil, nil, errors.New("operator socket requires an explicit OS peer UID")
	}
	server, listener, err := operatorHTTPServer(socket, uint32(uid), app.Coordinator)
	if err != nil {
		return nil, nil, err
	}
	server.Handler = app.Coordinator.AdmitHTTP(server.Handler)
	return server, listener, nil
}

func serveOperatorSocket(server *http.Server, listener net.Listener, stop context.CancelFunc, serveErrors chan<- error) {
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Error("operator listener stopped", "error", err)
			serveErrors <- err
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
	server, listener, err := extensionListener(socket, app.ExtensionHandler())
	if err != nil {
		return nil, nil, err
	}
	server.Handler = app.Coordinator.AdmitHTTP(server.Handler)
	return server, listener, nil
}

func serveExtensionSocket(server *http.Server, listener net.Listener, stop context.CancelFunc, serveErrors chan<- error) {
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Error("extension listener stopped", "error", err)
			serveErrors <- err
			stop()
		}
	}()
}

func beginShutdown(ctx context.Context, app *web.Server, server, private, operator *http.Server) <-chan error {
	stopped := make(chan error, 1)
	go func() {
		<-ctx.Done()
		app.Coordinator.StopAdmission()
		shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		shutdownDone := make(chan error, 1)
		go func() { shutdownDone <- shutdownServers(shutdown, server, private, operator) }()
		app.CloseTerminals()
		shutdownErr := <-shutdownDone
		stopped <- errors.Join(shutdownErr, app.CloseCoordinator())
	}()
	return stopped
}

func shutdownServers(ctx context.Context, servers ...*http.Server) error {
	active := make([]*http.Server, 0, len(servers))
	for _, server := range servers {
		if server != nil {
			active = append(active, server)
		}
	}
	type shutdownResult struct{ err error }
	results := make(chan shutdownResult, len(active))
	for _, server := range active {
		go func(server *http.Server) { results <- shutdownResult{err: server.Shutdown(ctx)} }(server)
	}
	var shutdownErr error
	for range active {
		shutdownErr = errors.Join(shutdownErr, (<-results).err)
	}
	if shutdownErr == nil {
		return nil
	}
	var closeErr error
	for _, server := range active {
		closeErr = errors.Join(closeErr, server.Close())
	}
	return errors.Join(shutdownErr, closeErr)
}
