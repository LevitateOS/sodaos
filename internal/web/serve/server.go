package serve

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/avatar"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/web"
)

type Product string

const (
	Forge   Product = "forge"
	Factory Product = "factory"
)

// Run owns one product's listeners and process lifetime. Forge never starts a
// coordinator, claims its ledger lock or opens a worker MCP listener.
func Run(product Product) (runErr error) {
	defaultConfig := "/etc/soda/dashboard.json"
	if product == Factory {
		defaultConfig = "/etc/soda/factory.json"
	}
	path := flag.String("config", defaultConfig, "configuration file")
	extensionSocket := flag.String("extension-socket", "", "optional private native-extension Unix socket")
	operatorSocket := flag.String("operator-socket", "", "optional private operator Unix socket")
	operatorUID := flag.Int("operator-uid", -1, "required OS peer UID when the operator socket is set")
	listen := flag.String("listen", "", "override the configured loopback HTTP listener")
	factoryControl := flag.String("factory-control-socket", "", "Forge integration: private Factory Project control socket")
	var controlSocket *string
	if product == Factory {
		controlSocket = flag.String("control-socket", "", "private Forge integration socket; admits the service OS UID only")
	}
	flag.Parse()
	app, database, err := openProduct(product, *path, *listen, *factoryControl)
	if err != nil {
		return err
	}
	defer func() { runErr = errors.Join(runErr, database.Close()) }()
	server := &http.Server{Addr: app.Config.Listen, Handler: admitHTTP(app, app), ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	serveErrors := make(chan error, 4)
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
	var factoryMCPServer, controlServer *http.Server
	if product == Factory {
		var workerListener net.Listener
		factoryMCPServer, workerListener, err = factoryMCPHTTPServer(app)
		if err != nil {
			return err
		}
		defer workerListener.Close()
		serveFactoryMCPSocket(factoryMCPServer, workerListener, stop, serveErrors)
		if *controlSocket != "" {
			var controlListener net.Listener
			controlServer, controlListener, err = factoryControlHTTPServer(*controlSocket, app)
			if err != nil {
				return err
			}
			defer controlListener.Close()
			serveOperatorSocket(controlServer, controlListener, stop, serveErrors)
		}
		// Worker MCP must listen before the coordinator can launch an agent.
		if err = app.StartCoordinator(context.Background()); err != nil {
			return err
		}
	}
	stopped := beginShutdown(ctx, app, server, privateServer, operatorServer, factoryMCPServer, controlServer)
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
	var operations http.Handler
	if app.Coordinator == nil {
		operations = app.API.OperatorHandler()
	}
	server, listener, err := operatorHTTPServer(socket, uint32(uid), app.Coordinator, operations)
	if err != nil {
		return nil, nil, err
	}
	server.Handler = admitHTTP(app, server.Handler)
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

func openProduct(product Product, path, listen, factoryControl string) (*web.Server, *store.Store, error) {
	if product == Forge {
		if err := avatar.Validate(); err != nil {
			return nil, nil, err
		}
	}
	load := config.Load
	if product == Factory {
		load = config.LoadFactory
	}
	c, err := load(path)
	if err != nil {
		return nil, nil, err
	}
	if listen != "" {
		host, _, parseErr := net.SplitHostPort(listen)
		if parseErr != nil || !net.ParseIP(host).IsLoopback() {
			return nil, nil, errors.New("listen must be a loopback IP:port")
		}
		c.Listen = listen
	}
	if factoryControl != "" {
		if !filepath.IsAbs(factoryControl) {
			return nil, nil, errors.New("Factory control socket must be absolute")
		}
		c.FactoryControlSocket = factoryControl
	}
	dsn, err := config.Secret(c.DatabaseDSNFile)
	if err != nil {
		return nil, nil, err
	}
	database, err := store.Open(dsn)
	if err != nil {
		return nil, nil, err
	}
	if product == Factory {
		app, err := web.NewFactory(c, database)
		if err != nil {
			_ = database.Close()
			return nil, nil, err
		}
		return app, database, nil
	}
	return web.NewForge(c, database), database, nil
}

func extensionHTTPServer(socket string, app *web.Server) (*http.Server, net.Listener, error) {
	if socket == "" {
		return nil, nil, nil
	}
	server, listener, err := extensionListener(socket, app.ExtensionHandler())
	if err != nil {
		return nil, nil, err
	}
	server.Handler = admitHTTP(app, server.Handler)
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

func beginShutdown(ctx context.Context, app *web.Server, servers ...*http.Server) <-chan error {
	stopped := make(chan error, 1)
	go func() {
		<-ctx.Done()
		if app.Coordinator != nil {
			app.Application.StopAdmission()
		}
		shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		shutdownDone := make(chan error, 1)
		go func() { shutdownDone <- shutdownServers(shutdown, servers...) }()
		app.CloseTerminals()
		shutdownErr := <-shutdownDone
		if app.Coordinator != nil {
			shutdownErr = errors.Join(shutdownErr, app.CloseCoordinator())
		}
		stopped <- shutdownErr
	}()
	return stopped
}

func admitHTTP(app *web.Server, handler http.Handler) http.Handler {
	if app.Coordinator == nil {
		return handler
	}
	return app.Coordinator.AdmitHTTP(handler)
}

func factoryControlHTTPServer(socket string, app *web.Server) (*http.Server, net.Listener, error) {
	server, listener, err := extensionListener(socket, app.Coordinator.ProjectHandler())
	if err != nil {
		return nil, nil, err
	}
	if err := os.Chmod(socket, 0o600); err != nil {
		_ = listener.Close()
		return nil, nil, err
	}
	gated, err := gateOperatorPeers(listener, uint32(os.Geteuid()))
	if err != nil {
		_ = listener.Close()
		return nil, nil, err
	}
	server.Handler = admitHTTP(app, server.Handler)
	return server, gated, nil
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
