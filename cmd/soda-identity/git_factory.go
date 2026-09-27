package main

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/host/terminal"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/identity/control"
	"github.com/levitateos/sodaos/internal/store"
)

func factoryGitRuntime(db *store.Store) *terminal.FactoryGitRuntime {
	runtime := &terminal.FactoryGitRuntime{}
	runtime.Exec = host.Native{}
	runtime.Resolve = func(ctx context.Context, peer terminal.MusePeer) (terminal.FactoryGitCaller, error) {
		return resolveFactoryGit(ctx, db, peer)
	}
	return runtime
}

func wireFactoryGit(runtime *terminal.FactoryGitRuntime, broker *control.Controller) {
	runtime.Acquire = broker.AcquireGit
	runtime.Register = broker.RegisterGit
	runtime.End = broker.EndLease
	runtime.Proxy = broker.FactoryGitProxy
	runtime.Select = func(ctx context.Context, actor int64, project, selected string) (string, error) {
		available, err := broker.Available(ctx, actor, project)
		if err != nil {
			return "", err
		}
		return identity.SelectForgejoConnection(available, selected)
	}
	runtime.Authorize = func(ctx context.Context, actor int64, project string) error {
		_, err := runtime.Select(ctx, actor, project, "")
		return err
	}
}

func resolveFactoryGit(ctx context.Context, db *store.Store, peer terminal.MusePeer) (terminal.FactoryGitCaller, error) {
	var caller terminal.FactoryGitCaller
	namespace, err := factoryPeerNamespace(peer)
	if err != nil {
		return caller, err
	}
	leases, err := db.IdentityLeases(ctx)
	if err != nil {
		return caller, err
	}
	for _, lease := range leases {
		if !factoryGitCandidate(ctx, db, lease) {
			continue
		}
		found, err := matchFactoryMuse(ctx, lease, namespace)
		if err != nil {
			return caller, err
		}
		if !found {
			continue
		}
		pid, err := factoryContainerPID(ctx, lease.Binding.ID)
		if err != nil {
			return caller, err
		}
		return terminal.FactoryGitCaller{Container: lease.Binding.ID, Project: lease.ProjectID, ExecutionID: lease.ExecutionID, Actor: lease.ActorID, RepositoryID: lease.RepositoryID, HostPID: pid, Deadline: lease.Deadline}, nil
	}
	return caller, identity.ErrDenied
}

func factoryGitCandidate(ctx context.Context, db *store.Store, lease identity.Lease) bool {
	return baseFactoryLease(lease) && lease.RepositoryID > 0 && factoryRunID(lease.ExecutionID) && factoryLeaseCurrent(ctx, db, lease) == nil
}

func factoryContainerPID(ctx context.Context, container string) (int, error) {
	body, err := podman(ctx, "inspect", "--format", "{{.State.Pid}}", container)
	if err != nil {
		return 0, identity.ErrDenied
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(body)))
	if err != nil || pid <= 0 {
		return 0, identity.ErrDenied
	}
	return pid, nil
}

func validateFactoryGitSettings(c settings) error {
	if c.GitWorkerSocket == "" {
		return nil
	}
	if !filepath.IsAbs(c.GitWorkerSocket) || filepath.Base(c.GitWorkerSocket) != "launch.sock" {
		return errors.New("explicit Git worker launch socket required")
	}
	root := filepath.Dir(c.GitWorkerSocket)
	if root == filepath.Dir(c.AdminSocket) || root == filepath.Dir(c.RuntimeSocket) || root == filepath.Dir(c.MuseWorkerSocket) {
		return errors.New("git worker launch socket must be separate")
	}
	return nil
}

func openFactoryGitSocket(c settings) (*net.UnixListener, error) {
	if err := validateFactoryGitSettings(c); err != nil || c.GitWorkerSocket == "" {
		return nil, err
	}
	if err := prepareWorkerInterface(c.GitWorkerSocket); err != nil {
		return nil, err
	}
	if _, err := os.Lstat(c.GitWorkerSocket); !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("git worker socket is occupied")
	}
	listener, err := listenWorkerSocket(c.GitWorkerSocket)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(c.GitWorkerSocket, 0o666); err != nil {
		_ = listener.Close()
		return nil, err
	}
	return listener, nil
}

func closeWorkerSocket(listener *net.UnixListener) {
	if listener != nil {
		_ = listener.Close()
	}
}

func (n nativeRuntime) factoryGitOperation(ctx context.Context, action string, l identity.Lease) error {
	if !n.factoryGitValid(l) {
		if action == "stop" {
			return identity.ErrUncertain
		}
		return identity.ErrDenied
	}
	if action == "stop" {
		return n.Git.Stop(ctx, *l.Binding)
	}
	pid, err := n.currentFactoryGitPID(ctx, l)
	if err != nil {
		return err
	}
	if err := n.Git.Validate(ctx, l, pid); err != nil {
		return err
	}
	current, err := n.currentFactoryGitPID(ctx, l)
	if err != nil || current != pid {
		return identity.ErrStale
	}
	return nil
}

func (n nativeRuntime) factoryGitValid(l identity.Lease) bool {
	return n.Git != nil && n.Store != nil && l.Binding != nil && l.ProviderID == identity.Forgejo && l.Kind == identity.Factory && l.Binding.Scope == "git-factory" && l.Binding.Project == l.ProjectID && l.RepositoryID > 0
}

func (n nativeRuntime) currentFactoryGitPID(ctx context.Context, l identity.Lease) (int, error) {
	leases, err := n.Store.IdentityLeases(ctx)
	if err != nil {
		return 0, err
	}
	for _, base := range leases {
		if !factoryGitBaseMatches(base, l) || factoryLeaseCurrent(ctx, n.Store, base) != nil {
			continue
		}
		if pid, ok := currentFactoryWorkerPID(ctx, base); ok {
			return pid, nil
		}
	}
	return 0, identity.ErrDenied
}

func currentFactoryWorkerPID(ctx context.Context, base identity.Lease) (int, bool) {
	observed, exists, err := observeFactory(ctx, base)
	if err != nil || !exists || !observed.Running {
		return 0, false
	}
	pid, err := factoryContainerPID(ctx, base.Binding.ID)
	return pid, err == nil
}

func factoryGitBaseMatches(base, git identity.Lease) bool {
	return baseFactoryLease(base) && base.Binding.ID == git.Binding.ContainerID && base.ActorID == git.ActorID && base.ProjectID == git.ProjectID && base.RepositoryID == git.RepositoryID && base.Deadline.After(time.Now())
}
