package acceptance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// Worker runs an admitted executable under a separate systemd service identity.
// It is a local privilege boundary, not a build recipe or a command service.
// The trusted controller supplies the executable and mounts; build output must
// never select them. systemd owns the namespace/cgroup and all descendants.
type Worker struct {
	Name, User, Executable, Directory string
	ReadOnly, Writable                []string // systemd host:guest bind pairs
	Environment                       []string
	Arguments                         []string
}

func validWorkerIdentity(w Worker) error {
	if !regexp.MustCompile(`^soda-(build|qualify)-[a-z0-9-]{1,48}$`).MatchString(w.Name) {
		return errors.New("exact task worker name required")
	}
	if w.User != "soda-build-worker" && w.User != "soda-qualifier" {
		return errors.New("separate approved worker identity required")
	}
	if !filepath.IsAbs(w.Directory) || strings.ContainsAny(w.Directory, "\n\r:") {
		return errors.New("absolute worker directory required")
	}
	return TrustedExecutable(w.Executable)
}

func appendBindPaths(args []string, property string, paths []string) ([]string, error) {
	for _, pair := range paths {
		parts := strings.Split(pair, ":")
		if len(parts) != 2 || !filepath.IsAbs(parts[0]) || !filepath.IsAbs(parts[1]) || strings.ContainsAny(pair, "\n\r\t %") {
			return nil, errors.New("explicit absolute worker bind pair required")
		}
		args = append(args, "--property="+property+"="+pair)
	}
	return args, nil
}

func allowedWorkerEnvKey(key string) bool {
	switch key {
	case "HOME", "PATH", "XDG_RUNTIME_DIR", "GOTOOLCHAIN", "GOCACHE", "GOMODCACHE", "BUN_INSTALL_CACHE_DIR", "PLAYWRIGHT_BROWSERS_PATH", "SODA_BUILD_START_NS":
		return true
	}
	return false
}

func appendWorkerEnv(args, environment []string) ([]string, error) {
	for _, env := range environment {
		key, _, ok := strings.Cut(env, "=")
		if !allowedWorkerEnvKey(key) {
			return nil, errors.New("worker environment key refused")
		}
		if !ok || strings.ContainsAny(env, "\n\r\x00") {
			return nil, errors.New("invalid worker environment")
		}
		args = append(args, "--setenv="+env)
	}
	return args, nil
}

func (w Worker) arguments() ([]string, error) {
	if err := validWorkerIdentity(w); err != nil {
		return nil, err
	}
	args := []string{
		"--quiet", "--wait", "--pipe", "--collect", "--service-type=exec", "--unit=" + w.Name,
		"--property=User=" + w.User, "--property=Group=" + w.User,
		"--property=WorkingDirectory=" + w.Directory,
		"--property=ProtectHome=tmpfs", "--property=ProtectSystem=strict",
		"--property=PrivateTmp=yes", "--property=PrivateMounts=yes",
		"--property=Delegate=yes", "--property=CPUQuota=400%", "--property=MemoryMax=16G",
		"--property=CPUAffinity=0 1 2 3", "--property=KillMode=control-group",
		"--property=TimeoutStopSec=20s", "--property=UMask=0077",
		"--property=InaccessiblePaths=-/var/lib/soda-release -/root",
	}
	var err error
	args, err = appendBindPaths(args, "BindReadOnlyPaths", w.ReadOnly)
	if err != nil {
		return nil, err
	}
	args, err = appendBindPaths(args, "BindPaths", w.Writable)
	if err != nil {
		return nil, err
	}
	args, err = appendWorkerEnv(args, w.Environment)
	if err != nil {
		return nil, err
	}
	return append(append(args, "--", w.Executable), w.Arguments...), nil
}

// TrustedExecutable refuses a builder-writable executable or parent, including
// symlinks. A digest supplied by the builder is not an executable admission grant.
func trustedPathMode(st os.FileInfo, path, current string) error {
	s, ok := st.Sys().(*syscall.Stat_t)
	if !ok || s.Uid != 0 || st.Mode().Perm()&0o022 != 0 || st.Mode()&os.ModeSymlink != 0 {
		return errors.New("worker executable and parents must be root-owned and not group/world writable")
	}
	if current == path && (!st.Mode().IsRegular() || st.Mode().Perm()&0o111 == 0) {
		return errors.New("admitted regular executable required")
	}
	return nil
}

func TrustedExecutable(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("absolute admitted executable required")
	}
	for p := path; ; p = filepath.Dir(p) {
		st, err := os.Lstat(p)
		if err != nil {
			return err
		}
		if err = trustedPathMode(st, path, p); err != nil {
			return err
		}
		if p == "/" {
			break
		}
	}
	return nil
}

func (w Worker) Run(ctx context.Context, out, stderr io.Writer) error {
	if os.Geteuid() != 0 {
		return errors.New("trusted root controller required for worker dispatch")
	}
	args, err := w.arguments()
	if err != nil {
		return err
	}
	// Refuse an existing unit instead of adopting or replacing its processes.
	probe := exec.CommandContext(ctx, "/usr/bin/systemctl", "show", w.Name+".service", "--property=LoadState", "--value")
	state, err := probe.Output()
	if err != nil || strings.TrimSpace(string(state)) != "not-found" {
		return errors.New("worker unit is already present or could not be checked")
	}
	cmd := exec.Command("/usr/bin/systemd-run", args...)
	cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C.UTF-8"}
	// Give systemd anonymous pipes, not caller-owned log file descriptors.
	// Passing a protected/home log FD across the service boundary can make
	// StartTransientUnit fail; the controller retains and owns the log files.
	cmd.Stdout, cmd.Stderr = struct{ io.Writer }{out}, struct{ io.Writer }{stderr}
	p, err := StartCommand(ctx, cmd)
	if err != nil {
		return err
	}
	err = p.Wait(ctx)
	if ctx.Err() != nil {
		// Stopping only systemd-run does not stop its service. Own the exact unit
		// even when the controller's request context has already been cancelled.
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		stop := exec.CommandContext(cleanup, "/usr/bin/systemctl", "stop", w.Name+".service")
		stop.Env = cmd.Env
		stopErr := stop.Run()
		err = errors.Join(ctx.Err(), err, stopErr, p.Stop())
	}
	if err != nil {
		return fmt.Errorf("worker %s failed: %w", w.Name, err)
	}
	return nil
}
