package main

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"regexp"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/host/terminal"
	"github.com/levitateos/sodaos/internal/host/workspace"
	"github.com/levitateos/sodaos/internal/identity"
)

type nativeRuntime struct {
	Host *host.Client
	Muse *terminal.MuseRuntime
}

var nativeID = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (n nativeRuntime) Finish(ctx context.Context, l identity.Lease) ([]byte, error) {
	if l.Binding == nil {
		return nil, identity.ErrDenied
	}
	if l.Kind == identity.Terminal {
		out, err := n.Host.IdentityFinish(ctx, l)
		return out.Credential, err
	}
	if err := n.Validate(ctx, l); err != nil {
		return nil, err
	}
	r := factory.Run{ID: l.ExecutionID, Resources: []factory.Resource{{Kind: "workspace", ID: l.Binding.ID, Name: factory.ResourceName(l.ExecutionID, "workspace")}}}
	w := workspace.Runtime{Exec: workspace.Native{}}
	return w.CaptureCredential(ctx, r)
}

func (n nativeRuntime) Validate(ctx context.Context, l identity.Lease) error {
	// Forgejo requires its own supervised Git execution, never the workspace boundary.
	if l.ProviderID == identity.Forgejo {
		return n.projectGitOperation(ctx, "validate", l)
	}
	if l.Binding == nil {
		return identity.ErrDenied
	}
	if isFactoryMuseLease(l) {
		return n.factoryMuseOperation(ctx, "validate", l)
	}
	if l.Kind == identity.Terminal {
		return n.Host.IdentityValidate(ctx, l)
	}
	observed, exists, err := observeFactory(ctx, l)
	if err != nil {
		return err
	}
	if !exists || !observed.Running {
		return identity.ErrDenied
	}
	return nil
}

type factoryObservation struct {
	ID, Owner string
	Running   bool
}

func observeFactory(ctx context.Context, l identity.Lease) (factoryObservation, bool, error) {
	var observed factoryObservation
	if l.Kind != identity.Factory || l.Binding == nil {
		return observed, false, identity.ErrDenied
	}
	if !nativeID.MatchString(l.Binding.ID) {
		return observed, false, identity.ErrDenied
	}
	exists, err := factoryExists(ctx, l.Binding.ID)
	if err != nil || !exists {
		return observed, exists, err
	}
	data, err := podman(ctx, "inspect", "--format", `{"id":{{json .ID}},"owner":{{json (index .Config.Labels "io.soda.factory.run")}},"running":{{json .State.Running}}}`, l.Binding.ID)
	if err != nil {
		return observed, true, errors.New("factory execution observation unavailable")
	}
	if !factoryObservationMatches(data, &observed, l) {
		return observed, true, identity.ErrDenied
	}
	return observed, true, nil
}

func factoryExists(ctx context.Context, id string) (bool, error) {
	_, err := podman(ctx, "container", "exists", id)
	if err == nil {
		return true, nil
	}
	var exited *exec.ExitError
	if errors.As(err, &exited) && exited.ExitCode() == 1 {
		return false, nil
	}
	return false, errors.New("factory runtime observation unavailable")
}

func podman(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "podman", append([]string{"--remote=false"}, args...)...).Output()
}

func (n nativeRuntime) Stop(ctx context.Context, l identity.Lease) error {
	if l.ProviderID == identity.Forgejo {
		return n.projectGitOperation(ctx, "stop", l)
	}
	if l.Binding == nil {
		return nil
	}
	if isFactoryMuseLease(l) {
		return n.factoryMuseOperation(ctx, "stop", l)
	}
	if l.Kind == identity.Terminal {
		return n.Host.IdentityStop(ctx, l)
	}
	return stopFactory(ctx, l)
}

func (n nativeRuntime) projectGitOperation(ctx context.Context, action string, l identity.Lease) error {
	if n.Host == nil || l.Kind != identity.Terminal || l.Binding == nil || l.Binding.Scope != "git" {
		if action == "stop" {
			return identity.ErrUncertain
		}
		return identity.ErrDenied
	}
	if action == "validate" {
		return n.Host.IdentityValidate(ctx, l)
	}
	return n.Host.IdentityStop(ctx, l)
}

func stopFactory(ctx context.Context, l identity.Lease) error {
	observed, exists, err := observeFactory(ctx, l)
	if err != nil || !exists {
		return err
	}
	if observed.Running {
		if _, err = podman(ctx, "kill", "--signal=KILL", l.Binding.ID); err != nil {
			return errors.New("factory execution termination unconfirmed")
		}
	}
	observed, _, err = observeFactory(ctx, l)
	if err != nil || observed.Running {
		return errors.New("factory execution is not confirmed stopped")
	}
	return nil
}

func factoryObservationMatches(data []byte, out *factoryObservation, l identity.Lease) bool {
	return json.Unmarshal(data, out) == nil && out.ID == l.Binding.ID && out.Owner == l.ExecutionID
}

func isFactoryMuseLease(l identity.Lease) bool {
	return l.ProviderID == identity.Muse && l.Kind == identity.Factory && l.Binding != nil && l.Binding.Scope == "muse-factory"
}

func (n nativeRuntime) factoryMuseOperation(ctx context.Context, action string, l identity.Lease) error {
	if n.Muse == nil {
		return identity.ErrDenied
	}
	_, err := n.Muse.Muse(ctx, action, identity.Delivery{Lease: l})
	return err
}
