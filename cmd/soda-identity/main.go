// soda-identity owns the private subscription connection service.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/host/terminal"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/identity/codex"
	"github.com/levitateos/sodaos/internal/identity/control"
	identityforgejo "github.com/levitateos/sodaos/internal/identity/forgejo"
	"github.com/levitateos/sodaos/internal/identity/muse"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/strictjson"
)

type settings struct {
	Forgejo           identityforgejo.Config `json:"forgejo"`
	ForgejoSecretFile string                 `json:"forgejo_secret_file"`
	MuseWorkerRoot    string                 `json:"muse_worker_root"`
	MuseWorkerSocket  string                 `json:"muse_worker_socket"`
	GitWorkerSocket   string                 `json:"git_worker_socket"`
	Database          string                 `json:"database"`
	KeyFile           string                 `json:"key_file"`
	AdminSocket       string                 `json:"admin_socket"`
	RuntimeSocket     string                 `json:"runtime_socket"`
	HostSocket        string                 `json:"host_socket"`
	Codex             codex.Config           `json:"codex"`
	Muse              muse.Config            `json:"muse"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "soda-identity:", err)
		os.Exit(1)
	}
}

func load(path string) (settings, error) {
	var c settings
	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer f.Close()
	if err = strictjson.Decode(f, &c); err != nil {
		return c, err
	}
	for _, p := range []string{c.Database, c.KeyFile, c.AdminSocket, c.RuntimeSocket, c.HostSocket} {
		if !filepath.IsAbs(p) {
			return c, errors.New("explicit absolute service paths required")
		}
	}
	if c.AdminSocket == c.RuntimeSocket {
		return c, errors.New("administration and runtime sockets must differ")
	}
	if err := validateWorkerSettings(c); err != nil {
		return c, err
	}
	return c, validateFactoryGitSettings(c)
}

func listen(path string, mode os.FileMode) (net.Listener, error) {
	// An occupied path is evidence of another process, never disposable intent.
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("broker socket path is occupied")
	}
	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err = os.Chmod(path, mode); err != nil {
		_ = l.Close()
		return nil, err
	}
	return l, nil
}

func run() error {
	flags := flag.NewFlagSet("soda-identity", flag.ContinueOnError)
	path := flags.String("config", "/etc/soda/identity.json", "restricted operator configuration")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	c, err := load(*path)
	if err != nil {
		return err
	}
	admin, runtime, err := serviceListeners(c)
	if err != nil {
		return err
	}
	defer func() { _ = admin.Close(); _ = runtime.Close() }()
	broker, db, worker, gitWorker, err := openService(c)
	if err != nil {
		return err
	}
	defer func() { _ = broker.Close(); _ = db.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	launch, err := openFactoryMuseSocket(c)
	if err != nil {
		return err
	}
	defer closeWorkerSocket(launch)
	gitLaunch, err := openFactoryGitSocket(c)
	if err != nil {
		return err
	}
	defer closeWorkerSocket(gitLaunch)
	if err = broker.Reconcile(ctx); err != nil {
		slog.Warn("identity execution termination remains unconfirmed")
	}
	return serve(ctx, broker, admin, runtime, worker, launch, gitWorker, gitLaunch)
}

func serve(ctx context.Context, b *control.Controller, admin, runtime net.Listener, worker *terminal.MuseRuntime, launch *net.UnixListener, gitWorker *terminal.FactoryGitRuntime, gitLaunch *net.UnixListener) error {
	servers := []*http.Server{
		{Handler: b.Handler(false), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192},
		{Handler: b.Handler(true), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192},
	}
	errs := make(chan error, 4)
	go func() { errs <- servers[0].Serve(admin) }()
	go func() { errs <- servers[1].Serve(runtime) }()
	if launch != nil {
		launcher := &terminal.MuseLaunch{Start: worker.StartFactory}
		go func() { errs <- launcher.Serve(ctx, launch) }()
	}
	if gitLaunch != nil {
		go func() { errs <- gitWorker.ServeFactory(ctx, gitLaunch) }()
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			for _, s := range servers {
				bounded, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_ = s.Shutdown(bounded)
				cancel()
			}
			return nil
		case err := <-errs:
			return err
		case <-ticker.C:
			bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
			err := b.Sweep(bounded)
			cancel()
			if err != nil {
				slog.Warn("identity execution termination remains unconfirmed")
			}
		}
	}
}

func openService(c settings) (*control.Controller, *store.Store, *terminal.MuseRuntime, *terminal.FactoryGitRuntime, error) {
	db, err := openStore(c)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	worker := factoryMuseRuntime(c, db)
	gitWorker := factoryGitRuntime(db)
	b, err := openBroker(c, db, worker, gitWorker)
	if err != nil {
		_ = db.Close()
		return nil, nil, nil, nil, err
	}
	wireFactoryMuse(worker, b)
	wireFactoryGit(gitWorker, b)
	return b, db, worker, gitWorker, nil
}

func openStore(c settings) (*store.Store, error) {
	key, err := config.GrantKey(c.KeyFile)
	if err != nil {
		return nil, err
	}
	return store.OpenEncrypted(c.Database, key)
}

func openBroker(c settings, db *store.Store, worker *terminal.MuseRuntime, gitWorker *terminal.FactoryGitRuntime) (*control.Controller, error) {
	providers := map[string]identity.Provider{}
	if err := configureForgejo(c, providers); err != nil {
		return nil, err
	}
	if c.Codex.Binary != "" {
		if err := os.MkdirAll(c.Codex.Root, 0o700); err != nil {
			return nil, err
		}
		p, err := codex.New(c.Codex)
		if err != nil {
			return nil, err
		}
		providers[identity.Codex] = p
	}
	if c.Muse.Binary != "" {
		if err := os.MkdirAll(c.Muse.Root, 0o700); err != nil {
			return nil, err
		}
		p, err := muse.New(c.Muse)
		if err != nil {
			return nil, err
		}
		providers[identity.Muse] = p
	}
	return control.New(db, providers, nativeRuntime{Host: host.NewClient(c.HostSocket), Muse: worker, Git: gitWorker, Store: db})
}

func serviceListeners(c settings) (net.Listener, net.Listener, error) {
	if os.Getenv("LISTEN_PID") == strconv.Itoa(os.Getpid()) {
		return activatedListeners(c)
	}
	admin, err := listen(c.AdminSocket, 0o660)
	if err != nil {
		return nil, nil, err
	}
	runtime, err := listen(c.RuntimeSocket, 0o600)
	if err != nil {
		_ = admin.Close()
		return nil, nil, err
	}
	return admin, runtime, nil
}

func activatedListeners(c settings) (net.Listener, net.Listener, error) {
	if os.Getenv("LISTEN_FDS") != "2" {
		return nil, nil, errors.New("two broker sockets required")
	}
	names := strings.Split(os.Getenv("LISTEN_FDNAMES"), ":")
	if len(names) != 2 {
		return nil, nil, errors.New("named broker sockets required")
	}
	listeners := map[string]net.Listener{}
	for i, name := range names {
		f := os.NewFile(uintptr(3+i), name)
		l, err := net.FileListener(f)
		_ = f.Close()
		if err != nil {
			closeListeners(listeners)
			return nil, nil, err
		}
		listeners[name] = l
	}
	if !activatedPathsMatch(c, listeners) {
		closeListeners(listeners)
		return nil, nil, errors.New("broker socket configuration differs from systemd")
	}
	return listeners["admin"], listeners["runtime"], nil
}

func activatedPathsMatch(c settings, ls map[string]net.Listener) bool {
	admin, runtime := ls["admin"], ls["runtime"]
	return admin != nil && runtime != nil && admin.Addr().Network() == "unix" && runtime.Addr().Network() == "unix" && admin.Addr().String() == c.AdminSocket && runtime.Addr().String() == c.RuntimeSocket
}

func closeListeners(ls map[string]net.Listener) {
	for _, l := range ls {
		_ = l.Close()
	}
}
