//go:build linux

package terminal

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/identity"
	"golang.org/x/sys/unix"
)

type factoryGitUnit struct {
	state, invocation, group string
	mainPID                  int
}

func factoryGitUnitName(id string) string { return "soda-git-factory-" + id + ".service" }

func parseFactoryGitUnit(body []byte) factoryGitUnit {
	var out factoryGitUnit
	for _, line := range strings.Split(string(body), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "ActiveState":
			out.state = value
		case "InvocationID":
			out.invocation = value
		case "ControlGroup":
			out.group = value
		case "MainPID":
			out.mainPID, _ = strconv.Atoi(value)
		}
	}
	return out
}

func (f *FactoryGitRuntime) unit(ctx context.Context, id string) (factoryGitUnit, error) {
	runtime := &MuseRuntime{Exec: f.Exec}
	body, err := runtime.userSystemctl(ctx, "show", "--property=ActiveState", "--property=InvocationID", "--property=ControlGroup", "--property=MainPID", factoryGitUnitName(id))
	if err != nil {
		return factoryGitUnit{}, err
	}
	return parseFactoryGitUnit(body), nil
}

func validFactoryGitBinding(b identity.Binding) bool {
	pid, err := strconv.Atoi(b.ChildID)
	return factoryGitBindingIdentity(b) && b.UID == 1000 && b.GID == 1000 && err == nil && pid > 0
}

func factoryGitBindingIdentity(b identity.Binding) bool {
	return b.Kind == identity.Factory && b.Scope == "git-factory" && terminalID.MatchString(b.ID) && containerID.MatchString(b.ContainerID) && terminalID.MatchString(b.InvocationID) && b.Project != ""
}

func factoryGitUnitMatches(unit factoryGitUnit, b identity.Binding) bool {
	pid, _ := strconv.Atoi(b.ChildID)
	return unit.state == "active" && unit.invocation == b.InvocationID && unit.mainPID == pid && strings.HasSuffix(unit.group, "/"+factoryGitUnitName(b.ID))
}

func (f *FactoryGitRuntime) Validate(ctx context.Context, l identity.Lease, workerPID int) error {
	if l.Binding == nil || !validFactoryGitBinding(*l.Binding) || workerPID <= 0 {
		return identity.ErrDenied
	}
	unit, err := f.unit(ctx, l.Binding.ID)
	if err != nil || !factoryGitUnitMatches(unit, *l.Binding) {
		return identity.ErrStale
	}
	pid, _ := strconv.Atoi(l.Binding.ChildID)
	if err := factoryGitNamespaces(pid, workerPID); err != nil {
		return err
	}
	unit, err = f.unit(ctx, l.Binding.ID)
	if err != nil || !factoryGitUnitMatches(unit, *l.Binding) {
		return identity.ErrStale
	}
	return nil
}

func factoryGitNamespaces(child, worker int) error {
	childFD, err := unix.PidfdOpen(child, 0)
	if err != nil {
		return identity.ErrStale
	}
	defer unix.Close(childFD)
	workerFD, err := unix.PidfdOpen(worker, 0)
	if err != nil {
		return identity.ErrStale
	}
	defer unix.Close(workerFD)
	if !factoryGitNamespaceMatch(child, worker) || !factoryGitRootMatch(child, worker) || !factoryGitPIDAlive(childFD) || !factoryGitPIDAlive(workerFD) {
		return identity.ErrStale
	}
	return nil
}

func factoryGitNamespaceMatch(child, worker int) bool {
	for _, namespace := range []string{"user", "mnt", "net"} {
		a, err := os.Readlink("/proc/" + strconv.Itoa(child) + "/ns/" + namespace)
		if err != nil {
			return false
		}
		b, err := os.Readlink("/proc/" + strconv.Itoa(worker) + "/ns/" + namespace)
		if err != nil || a != b {
			return false
		}
	}
	return true
}

func factoryGitRootMatch(child, worker int) bool {
	a, err := os.Stat("/proc/" + strconv.Itoa(child) + "/root")
	if err != nil {
		return false
	}
	b, err := os.Stat("/proc/" + strconv.Itoa(worker) + "/root")
	return err == nil && os.SameFile(a, b)
}

func factoryGitPIDAlive(fd int) bool {
	process := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	n, err := unix.Poll(process, 0)
	return err == nil && n == 0
}

func (f *FactoryGitRuntime) Stop(ctx context.Context, b identity.Binding) error {
	if !validFactoryGitBinding(b) {
		return identity.ErrDenied
	}
	unit, err := f.unit(ctx, b.ID)
	if err != nil {
		return identity.ErrUncertain
	}
	if unit.invocation != "" && unit.invocation != b.InvocationID {
		return identity.ErrStale
	}
	runtime := &MuseRuntime{Exec: f.Exec}
	_, _ = runtime.userSystemctl(ctx, "stop", factoryGitUnitName(b.ID))
	unit, err = f.unit(ctx, b.ID)
	if err != nil || unit.state != "inactive" || unit.group != "" {
		return identity.ErrUncertain
	}
	return factoryGitChildStopped(b)
}

func factoryGitChildStopped(b identity.Binding) error {
	pid, _ := strconv.Atoi(b.ChildID)
	group, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/cgroup")
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || strings.Contains(string(group), filepath.Base(factoryGitUnitName(b.ID))) {
		return identity.ErrUncertain
	}
	return nil
}
