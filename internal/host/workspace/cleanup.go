package workspace

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

// Cleanup reconciles only this ledger record. It never adopts a prefix-matching
// resource or removes anything lacking the exact execution ownership label.
func (w *Runtime) Cleanup(ctx context.Context, r *factory.Run) error {
	if err := r.Validate(); err != nil {
		return err
	}
	var failures []error
	for i := len(r.Resources) - 1; i >= 0; i-- {
		if err := w.removeResource(ctx, *r, r.Resources[i]); err != nil {
			failures = append(failures, err)
		}
	}
	err := errors.Join(failures...)
	r.CleanupComplete = err == nil
	return err
}

func (w *Runtime) observeResource(ctx context.Context, r factory.Run, resource factory.Resource) (string, bool, error) {
	target := resource.ID
	if target == "" {
		target = resource.Name
	}
	command := []string{"container"}
	if resource.Kind == "network" {
		command = []string{"network"}
	}
	if _, err := w.Exec.Run(ctx, nil, "podman", append(command, "exists", target)...); err != nil {
		var exited *ExitError
		if errors.As(err, &exited) && exited.Code == 1 {
			return "", false, nil
		}
		return "", false, err
	}
	var format string
	if resource.Kind == "network" {
		format = `{"id":{{json .ID}},"name":{{json .Name}},"owner":{{json (index .Labels "io.soda.factory.run")}}}`
	} else {
		format = `{"id":{{json .ID}},"name":{{json .Name}},"owner":{{json (index .Config.Labels "io.soda.factory.run")}}}`
	}
	args := append(command, "inspect", "--format", format, target)
	out, err := w.Exec.Run(ctx, nil, "podman", args...)
	if err != nil {
		return "", false, err
	}
	return resourceIdentity(out, r, resource)
}

func resourceIdentity(out []byte, r factory.Run, resource factory.Resource) (string, bool, error) {
	var identity struct{ ID, Name, Owner string }
	if err := json.Unmarshal(out, &identity); err != nil {
		return "", false, errors.New("invalid native resource identity")
	}
	if identity.Name != resource.Name || identity.Owner != r.ID || !factory.ValidDigest(identity.ID) {
		return "", false, errors.New("resource ownership does not match ledger")
	}
	if resource.ID != "" && identity.ID != resource.ID {
		return "", false, errors.New("resource ID changed")
	}
	return identity.ID, true, nil
}

func (w *Runtime) removeResource(ctx context.Context, r factory.Run, resource factory.Resource) error {
	id, exists, err := w.observeResource(ctx, r, resource)
	if err != nil || !exists {
		return err
	}
	if resource.Kind == "network" {
		_, err := w.Exec.Run(ctx, nil, "podman", "network", "rm", id)
		return err
	}
	// Stop the whole container, rather than signalling only the harness leader.
	if _, err := w.Exec.Run(ctx, nil, "podman", "stop", "--time", "3", id); err != nil {
		return err
	}
	_, err = w.Exec.Run(ctx, nil, "podman", "rm", id)
	return err
}
