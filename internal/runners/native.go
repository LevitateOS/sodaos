package runners

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/levitateos/sodaos/internal/filelock"
	"github.com/levitateos/sodaos/internal/strictjson"
	"golang.org/x/sys/unix"
)

type Command struct {
	Name string
	Args []string
}

type CommandResult struct {
	Stdout string
}

type CommandRunner interface {
	Run(context.Context, Command) (CommandResult, error)
}

type ExecCommandRunner struct{}

func (ExecCommandRunner) Run(ctx context.Context, request Command) (CommandResult, error) {
	command := exec.CommandContext(ctx, request.Name, request.Args...)
	var stdout bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = io.Discard
	if err := command.Run(); err != nil {
		return CommandResult{}, err
	}
	return CommandResult{Stdout: stdout.String()}, nil
}

type Native struct {
	RootPath string
	LockPath string
	Runner   CommandRunner
}

func NewNative() *Native {
	return &Native{RootPath: DefaultRootPath, LockPath: DefaultLockPath, Runner: ExecCommandRunner{}}
}

func (native *Native) List(ctx context.Context) ([]RunnerView, error) {
	lock, err := native.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer lock.Close()
	entries, err := os.ReadDir(native.rootPath())
	if errors.Is(err, os.ErrNotExist) {
		return []RunnerView{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read local runner directory: %w", err)
	}
	views := make([]RunnerView, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || ValidateID(entry.Name()) != nil {
			continue
		}
		view, viewErr := native.runnerView(ctx, entry.Name())
		if viewErr != nil {
			return nil, viewErr
		}
		views = append(views, view)
	}
	sort.Slice(views, func(i, j int) bool { return views[i].ID < views[j].ID })
	return views, nil
}

func (native *Native) runnerView(ctx context.Context, id string) (RunnerView, error) {
	descriptor, err := native.readDescriptor(id)
	if err != nil {
		return RunnerView{}, err
	}
	service, err := native.serviceState(ctx, id)
	if err != nil {
		return RunnerView{}, err
	}
	version, err := native.forgejoVersion(ctx)
	if err != nil {
		return RunnerView{}, err
	}
	return RunnerView{Descriptor: descriptor, Version: version, Capacity: RunnerCapacity, Service: service}, nil
}

func (native *Native) Start(ctx context.Context, id string) error {
	lock, err := native.lock(ctx)
	if err != nil {
		return err
	}
	defer lock.Close()
	return native.serviceAction(ctx, id, "enable", "--now")
}

func (native *Native) Stop(ctx context.Context, id string) error {
	lock, err := native.lock(ctx)
	if err != nil {
		return err
	}
	defer lock.Close()
	return native.serviceAction(ctx, id, "disable", "--now")
}

func (native *Native) Restart(ctx context.Context, id string) error {
	lock, err := native.lock(ctx)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := native.serviceAction(ctx, id, "enable"); err != nil {
		return err
	}
	return native.serviceAction(ctx, id, "restart")
}

func (native *Native) Remove(ctx context.Context, id string) error {
	lock, err := native.lock(ctx)
	if err != nil {
		return err
	}
	defer lock.Close()
	descriptor, err := native.readDescriptor(id)
	if err != nil {
		return err
	}
	if _, err = native.run(ctx, "systemctl", "disable", "--now", native.unit(id)); err != nil {
		return errors.New("listener stop is unconfirmed; account and state removal were not attempted")
	}
	if _, err = native.run(ctx, "userdel", descriptor.Account); err != nil {
		return errors.New("account removal is unconfirmed; local state was not removed")
	}
	if err = os.RemoveAll(filepath.Join(native.rootPath(), id)); err != nil {
		return errors.New("account was removed, but local state was not fully removed")
	}
	return nil
}

func (native *Native) serviceAction(ctx context.Context, id, action string, options ...string) error {
	if _, err := native.readDescriptor(id); err != nil {
		return err
	}
	arguments := append([]string{action}, options...)
	if _, err := native.run(ctx, "systemctl", append(arguments, native.unit(id))...); err != nil {
		return fmt.Errorf("%s local runner listener", action)
	}
	return nil
}

func (native *Native) readDescriptor(id string) (Descriptor, error) {
	if err := ValidateID(id); err != nil {
		return Descriptor{}, err
	}
	contents, err := os.ReadFile(native.descriptorPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return Descriptor{}, fmt.Errorf("local runner %s does not exist", id)
	}
	if err != nil {
		return Descriptor{}, err
	}
	var descriptor Descriptor
	if err = strictjson.Decode(bytes.NewReader(contents), &descriptor); err != nil {
		return Descriptor{}, fmt.Errorf("local runner %s descriptor is invalid", id)
	}
	account, accountErr := AccountName(id)
	if accountErr != nil || descriptor.ID != id || descriptor.Account != account {
		return Descriptor{}, fmt.Errorf("local runner %s descriptor is invalid", id)
	}
	if descriptor.Provider != ProviderForgejo {
		return Descriptor{}, fmt.Errorf("local runner %s has an unsupported provider; only Forgejo runners are supported", id)
	}
	return descriptor, nil
}

func (native *Native) writeDescriptor(descriptor Descriptor) error {
	path := native.descriptorPath(descriptor.ID)
	contents, err := json.MarshalIndent(descriptor, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".descriptor-*")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if err = temporary.Chmod(0o644); err == nil {
		_, err = temporary.Write(append(contents, '\n'))
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(temporary.Name(), path)
}

func (native *Native) serviceState(ctx context.Context, id string) (ServiceState, error) {
	result, err := native.run(ctx, "systemctl", "show", "--no-pager", "--property=LoadState,ActiveState,SubState,UnitFileState", native.unit(id))
	if err != nil {
		return ServiceState{}, fmt.Errorf("inspect local runner %s service: %w", id, err)
	}
	values := map[string]string{}
	for _, line := range strings.Split(result.Stdout, "\n") {
		key, value, found := strings.Cut(line, "=")
		if found {
			if _, duplicate := values[key]; duplicate {
				return ServiceState{}, fmt.Errorf("local runner %s service observation is ambiguous", id)
			}
			values[key] = value
		}
	}
	// systemctl show can succeed with absent/empty properties (for example a
	// missing unit). That is unavailable inventory, not an observed boot policy.
	// Keep systemd's state vocabulary upstream-owned; require the observations,
	// not a copied enumeration of every possible state.
	for _, key := range []string{"LoadState", "ActiveState", "SubState", "UnitFileState"} {
		if strings.TrimSpace(values[key]) == "" {
			return ServiceState{}, fmt.Errorf("local runner %s service observation is incomplete", id)
		}
	}
	return ServiceState{Load: values["LoadState"], Active: values["ActiveState"], Sub: values["SubState"], Enabled: values["UnitFileState"]}, nil
}

func (native *Native) forgejoVersion(ctx context.Context) (string, error) {
	result, err := native.run(ctx, "forgejo-runner", "--version")
	if err != nil || strings.TrimSpace(result.Stdout) == "" {
		return "", errors.New("read Forgejo runner version")
	}
	return strings.TrimSpace(result.Stdout), nil
}

func (native *Native) lock(ctx context.Context) (*os.File, error) {
	file, err := os.OpenFile(native.lockPath(), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err = filelock.Acquire(ctx, file, unix.LOCK_EX); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func (native *Native) run(ctx context.Context, name string, args ...string) (CommandResult, error) {
	return native.runner().Run(ctx, Command{Name: name, Args: args})
}

func (native *Native) runner() CommandRunner {
	if native.Runner == nil {
		return ExecCommandRunner{}
	}
	return native.Runner
}

func (native *Native) rootPath() string {
	if native.RootPath != "" {
		return native.RootPath
	}
	return DefaultRootPath
}
func (native *Native) lockPath() string {
	if native.LockPath != "" {
		return native.LockPath
	}
	return DefaultLockPath
}
func (native *Native) statePath(id string) string {
	return filepath.Join(native.rootPath(), id, "state")
}
func (native *Native) descriptorPath(id string) string {
	return filepath.Join(native.rootPath(), id, "descriptor.json")
}
func (native *Native) unit(id string) string { return "soda-runner@" + id + ".service" }

func writeOwnedFile(path string, contents []byte, mode os.FileMode, owner identity) error {
	if err := os.WriteFile(path, contents, mode); err != nil {
		return err
	}
	return os.Chown(path, int(owner.UID), int(owner.GID))
}
