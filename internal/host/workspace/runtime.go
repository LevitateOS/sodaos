package workspace

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
)

type Runtime struct {
	Config Config
	Exec   Executor
}

func ownership(id string) string { return "io.soda.factory.run=" + id }

// Create requires a pre-recorded run and persists each returned native ID before
// the next allocation. Exact recorded intent covers interruption during create.
func (w *Runtime) Create(ctx context.Context, r *factory.Run, persist func(factory.Run) error) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if err := w.Config.Validate(); err != nil {
		return err
	}
	if err := w.validatePreparedRun(*r); err != nil {
		return err
	}
	for i := range r.Resources {
		if err := w.createResource(ctx, r, i); err != nil {
			return err
		}
		if err := persist(*r); err != nil {
			return err
		}
	}
	return nil
}

func (w *Runtime) validatePreparedRun(r factory.Run) error {
	if !freshResourceIntents(r.Resources) {
		return errors.New("workspace needs its exact recorded resource intents")
	}
	if r.Image != w.Config.Image || r.Harness != "codex-"+w.Config.HarnessVersion || r.Model != w.Config.Model {
		return errors.New("run profile differs from qualified runtime")
	}
	info, err := os.Lstat(filepath.Join(w.Config.Root, r.ID, "input"))
	if err != nil || !info.IsDir() {
		return errors.New("fresh private task inputs are required before create")
	}
	return nil
}

func (w *Runtime) createResource(ctx context.Context, r *factory.Run, index int) error {
	resource := &r.Resources[index]
	switch resource.Kind {
	case "network":
		return w.createNetwork(ctx, r, resource)
	case "proxy":
		if err := w.createProxy(ctx, r, resource, w.resource(*r, "network").Name); err != nil {
			return err
		}
		_, err := w.Exec.Run(ctx, nil, "podman", "start", resource.ID)
		return err
	case "workspace":
		return w.createWorkspace(ctx, r, resource)
	default:
		return errors.New("unknown recorded workspace resource")
	}
}

func (w *Runtime) resource(r factory.Run, kind string) factory.Resource {
	for _, resource := range r.Resources {
		if resource.Kind == kind {
			return resource
		}
	}
	return factory.Resource{}
}

func (w *Runtime) containerArguments(r *factory.Run, resource *factory.Resource) []string {
	return []string{"create", "--name", resource.Name, "--label", ownership(r.ID), "--pull=never", "--userns=keep-id:uid=1000,gid=1000", "--user=1000:1000", "--read-only", "--cap-drop=all", "--security-opt=no-new-privileges", "--pids-limit=" + strconv.Itoa(w.Config.PIDs), "--cpus=" + strconv.Itoa(w.Config.CPUs), "--memory=" + strconv.FormatInt(w.Config.MemoryBytes, 10), "--memory-swap=" + strconv.FormatInt(w.Config.MemoryBytes, 10)}
}

func (w *Runtime) createContainer(ctx context.Context, resource *factory.Resource, args []string) error {
	out, err := w.Exec.Run(ctx, nil, "podman", args...)
	if err != nil {
		return err
	}
	resource.ID = strings.TrimSpace(string(out))
	if !factory.ValidDigest(resource.ID) {
		return errors.New("native container did not return an immutable ID")
	}
	return nil
}

func (w *Runtime) createWorkspace(ctx context.Context, r *factory.Run, resource *factory.Resource) error {
	network := w.resource(*r, "network").Name
	proxy := w.resource(*r, "proxy").ID
	format := fmt.Sprintf(`{{(index .NetworkSettings.Networks %q).IPAddress}}`, network)
	out, err := w.Exec.Run(ctx, nil, "podman", "inspect", "--format", format, proxy)
	if err != nil {
		return err
	}
	ip, err := netip.ParseAddr(strings.TrimSpace(string(out)))
	if err != nil || !ip.IsPrivate() {
		return errors.New("proxy has no private address on the run network")
	}
	url := "http://" + ip.String() + ":3128"
	args := w.containerArguments(r, resource)
	args = append(args, "--network="+network, "--tmpfs", "/tmp:rw,nosuid,nodev,size=32m", "--tmpfs", fmt.Sprintf("/workspace:rw,nosuid,nodev,size=%d,mode=0700,uid=1000,gid=1000", w.Config.WritableBytes), "--env", "HOME=/workspace/home", "--env", "CODEX_HOME=/run/codex", "--env", "CODEX_SQLITE_HOME=/workspace/.codex-state", "--env", "PATH=/opt/codex/bin:/opt/codex/codex-path:/usr/bin:/bin", "--env", "HTTPS_PROXY="+url, "--env", "HTTP_PROXY="+url, "--volume", w.Config.HarnessDirectory+":/opt/codex:ro,Z", "--volume", w.Config.CredentialHome+":/run/codex:rw,Z", "--volume", filepath.Join(w.Config.Root, r.ID, "input")+":/input:ro,Z", "--workdir", "/workspace", w.Config.Image, "sleep", "infinity")
	return w.createContainer(ctx, resource, args)
}

// Initialize uses only a supplied immutable bundle and a pinned source revision.
// No remote Git destination or Forgejo credential enters the workspace.
func (w *Runtime) Initialize(ctx context.Context, r factory.Run) error {
	id := w.resource(r, "workspace").ID
	if _, err := w.Exec.Run(ctx, nil, "podman", "start", id); err != nil {
		return err
	}
	script := `set -eu
mkdir /workspace/home
git -c core.hooksPath=/dev/null clone --no-checkout /input/source.bundle /workspace/repo
git -C /workspace/repo -c core.hooksPath=/dev/null checkout --detach "$1"
git -C /workspace/repo config user.name soda-agent
git -C /workspace/repo config user.email soda-agent@localhost
`
	_, err := w.Exec.Run(ctx, nil, "podman", "exec", id, "sh", "-c", script, "initialize-workspace", r.InputSHA)
	return err
}

func freshResourceIntents(resources []factory.Resource) bool {
	if len(resources) != 3 {
		return false
	}
	for i, kind := range []string{"network", "proxy", "workspace"} {
		if resources[i].Kind != kind || resources[i].ID != "" {
			return false
		}
	}
	return true
}
