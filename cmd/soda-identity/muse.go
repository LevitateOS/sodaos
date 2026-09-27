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

func factoryMuseRuntime(c settings, db *store.Store) *terminal.MuseRuntime {
	runtime := &terminal.MuseRuntime{Exec: host.Native{}, BinaryVersion: c.Muse.Version, BinarySHA256: c.Muse.SHA256, FactoryRoot: c.MuseWorkerRoot}
	runtime.FactoryResolve = func(ctx context.Context, peer terminal.MusePeer) (terminal.MuseFactoryCaller, error) {
		return resolveFactoryMuse(ctx, db, c.MuseWorkerRoot, peer)
	}
	return runtime
}

func wireFactoryMuse(runtime *terminal.MuseRuntime, broker *control.Controller) {
	runtime.Acquire = broker.Acquire
	runtime.Attach = broker.Register
	runtime.End = broker.EndLease
	runtime.Select = func(ctx context.Context, actor int64, project, selected string) (string, error) {
		available, err := broker.Available(ctx, actor, project)
		if err != nil {
			return "", err
		}
		return identity.SelectMuseConnection(available, selected)
	}
}

func openFactoryMuseSocket(c settings) (*net.UnixListener, error) {
	if err := validateWorkerSettings(c); err != nil {
		return nil, err
	}
	if c.MuseWorkerSocket == "" {
		return nil, nil
	}
	if err := workerRuntimeAvailable(c.MuseWorkerRoot); err != nil {
		return nil, err
	}
	if _, err := os.Lstat(c.MuseWorkerSocket); !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("muse worker socket is occupied")
	}
	listener, err := net.ListenUnix("unixpacket", &net.UnixAddr{Name: c.MuseWorkerSocket, Net: "unixpacket"})
	if err != nil {
		return nil, err
	}
	if err = os.Chmod(c.MuseWorkerSocket, 0o666); err != nil {
		_ = listener.Close()
		return nil, err
	}
	return listener, nil
}

func resolveFactoryMuse(ctx context.Context, db *store.Store, root string, peer terminal.MusePeer) (terminal.MuseFactoryCaller, error) {
	var caller terminal.MuseFactoryCaller
	namespace, err := factoryPeerNamespace(peer)
	if err != nil {
		return caller, err
	}
	leases, err := db.IdentityLeases(ctx)
	if err != nil {
		return caller, err
	}
	for _, lease := range leases {
		if !baseFactoryLease(lease) {
			continue
		}
		if err := factoryLeaseCurrent(ctx, db, lease); err != nil {
			continue
		}
		found, err := matchFactoryMuse(ctx, lease, namespace)
		if err != nil {
			return caller, err
		}
		if !found {
			continue
		}
		return factoryMuseCaller(ctx, root, lease)
	}
	return caller, identity.ErrDenied
}

func matchFactoryMuse(ctx context.Context, lease identity.Lease, namespace string) (bool, error) {
	if _, exists, err := observeFactory(ctx, lease); err != nil || !exists {
		return false, err
	}
	out, err := podman(ctx, "inspect", "--format", "{{.State.Pid}}", lease.Binding.ID)
	if err != nil {
		return false, identity.ErrDenied
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || pid <= 0 {
		return false, identity.ErrDenied
	}
	observed, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/ns/pid")
	return err == nil && observed == namespace, nil
}

func factoryMuseCaller(ctx context.Context, root string, lease identity.Lease) (terminal.MuseFactoryCaller, error) {
	var caller terminal.MuseFactoryCaller
	if !lease.Deadline.After(time.Now()) || !nativeID.MatchString(lease.Binding.ID) || !factoryRunID(lease.ExecutionID) {
		return caller, identity.ErrDenied
	}
	body, err := podman(ctx, "inspect", "--format", "{{.State.Pid}}", lease.Binding.ID)
	if err != nil {
		return caller, identity.ErrDenied
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(body)))
	if err != nil || pid <= 0 {
		return caller, identity.ErrDenied
	}
	caller = terminal.MuseFactoryCaller{Container: lease.Binding.ID, Project: lease.ProjectID, Actor: lease.ActorID, Deadline: lease.Deadline, Login: "soda-worker", UID: 1000, GID: 1000, HostPID: pid, CredentialRoot: filepath.Join(root, lease.ExecutionID)}
	return caller, nil
}

func factoryRunID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, ch := range id {
		if !(ch >= '0' && ch <= '9') && !(ch >= 'a' && ch <= 'f') {
			return false
		}
	}
	return true
}

func validateWorkerSettings(c settings) error {
	if c.MuseWorkerSocket == "" && c.MuseWorkerRoot == "" {
		return nil
	}
	if !filepath.IsAbs(c.MuseWorkerSocket) || !filepath.IsAbs(c.MuseWorkerRoot) || c.Muse.Binary == "" {
		return errors.New("explicit Muse worker runtime paths required")
	}
	if c.MuseWorkerSocket == c.AdminSocket || c.MuseWorkerSocket == c.RuntimeSocket {
		return errors.New("muse launch socket must be separate from broker sockets")
	}
	return nil
}

func baseFactoryLease(l identity.Lease) bool {
	return l.Kind == identity.Factory && l.Binding != nil && l.Binding.Scope == ""
}

func factoryLeaseCurrent(ctx context.Context, db *store.Store, l identity.Lease) error {
	if !l.Deadline.After(time.Now()) {
		return identity.ErrDenied
	}
	conn, err := db.IdentityConnection(ctx, l.ConnectionID)
	if err != nil {
		return err
	}
	if conn.State != identity.Ready || conn.Generation != l.Generation || conn.ProviderID != l.ProviderID {
		return identity.ErrDenied
	}
	if l.GrantID == "" && conn.OwnerID != l.ActorID {
		return identity.ErrDenied
	}
	return factoryGrantCurrent(ctx, db, l)
}

func factoryGrantCurrent(ctx context.Context, db *store.Store, l identity.Lease) error {
	if l.GrantID == "" {
		return nil
	}
	grant, err := db.IdentityGrant(ctx, l.GrantID)
	if err != nil {
		return err
	}
	if grant.ConnectionID != l.ConnectionID || grant.Revoked || grant.Revision != l.GrantRevision || grant.UserID != l.ActorID || grant.ProjectID != l.ProjectID {
		return identity.ErrDenied
	}
	return nil
}

func factoryPeerNamespace(peer terminal.MusePeer) (string, error) {
	if peer.UID != uint32(os.Geteuid()) || peer.PID <= 0 {
		return "", identity.ErrDenied
	}
	ns, err := os.Readlink("/proc/" + strconv.Itoa(peer.PID) + "/ns/pid")
	if err != nil {
		return "", identity.ErrDenied
	}
	return ns, nil
}
