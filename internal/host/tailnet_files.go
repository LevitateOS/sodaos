package host

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/levitateos/sodaos/internal/filelock"
	"github.com/levitateos/sodaos/internal/tailnet"
)

// Every ancestor is appliance-owned; shifted root can access only the two
// explicit companion mounts, never this directory, its lock or the host policy.
// Existing unsafe paths are refused, not chmod/chown/recreated as a repair.
type runFiles struct {
	root     *os.Root
	lock     *os.File
	uid, gid uint32
}

func rootDirectory(info os.FileInfo, uid uint32) bool {
	s, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.IsDir() && s.Uid == uid && info.Mode().Perm() == 0o700
}

func openRuntimeProject(ctx context.Context, base, project string) (*runFiles, error) {
	return openRuntimeProjectOwned(ctx, base, project, 0, 0)
}

func openRuntimeRoot(base string, uid uint32) (*os.Root, error) {
	info, e := os.Lstat(base)
	if e != nil || !rootDirectory(info, uid) {
		return nil, tailnet.ErrUnavailable
	}
	root, e := os.OpenRoot(base)
	if e != nil {
		return nil, tailnet.ErrUnavailable
	}
	opened, e := root.Stat(".")
	if e != nil || !os.SameFile(info, opened) {
		root.Close()
		return nil, tailnet.ErrUnavailable
	}
	return root, nil
}

// Tests use their own UID in a private directory; production always passes 0:0.
func openRuntimeProjectOwned(ctx context.Context, base, project string, uid, gid uint32) (*runFiles, error) {
	if !projectID.MatchString(project) {
		return nil, tailnet.ErrInvalid
	}
	parent, e := openRuntimeRoot(base, uid)
	if e != nil {
		return nil, tailnet.ErrUnavailable
	}
	defer parent.Close()
	info, e := parent.Stat(".")
	if e != nil || !rootDirectory(info, uid) {
		return nil, tailnet.ErrUnavailable
	}
	if e = parent.Mkdir(project, 0o700); e != nil && !errors.Is(e, os.ErrExist) {
		return nil, tailnet.ErrUnavailable
	}
	info, e = parent.Lstat(project)
	if e != nil || !rootDirectory(info, uid) {
		return nil, tailnet.ErrUnavailable
	}
	root, e := parent.OpenRoot(project)
	if e != nil {
		return nil, tailnet.ErrUnavailable
	}
	lock, e := root.OpenFile("lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0o600)
	if e != nil {
		root.Close()
		return nil, tailnet.ErrUnavailable
	}
	info, e = lock.Stat()
	if e != nil || !runtimeFile(info, uid, gid, 0o600) {
		lock.Close()
		root.Close()
		return nil, tailnet.ErrUnavailable
	}
	if e = filelock.Acquire(ctx, lock, syscall.LOCK_EX); e != nil {
		lock.Close()
		root.Close()
		return nil, e
	}
	return &runFiles{root: root, lock: lock, uid: uid, gid: gid}, nil
}

func runtimeFile(info os.FileInfo, uid, gid uint32, mode os.FileMode) bool {
	s, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.Mode().IsRegular() && info.Mode().Perm() == mode && s.Uid == uid && s.Gid == gid && s.Nlink == 1
}
func (f *runFiles) Close() { f.lock.Close(); f.root.Close() }
func ownedRunDir(info os.FileInfo, uid, gid uint32) bool {
	s, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.IsDir() && info.Mode().Perm() == 0o700 && s.Uid == uid && s.Gid == gid
}

func prepareRunSubdir(root *os.Root, path string, run projectRun, fresh bool) error {
	if fresh {
		e := root.Mkdir(path, 0o700)
		if e == nil {
			e = root.Chown(path, int(run.UID), int(run.GID))
		}
		if e != nil {
			return tailnet.ErrUnavailable
		}
	}
	info, e := root.Lstat(path)
	if e != nil {
		return tailnet.ErrUnavailable
	}
	if !ownedRunDir(info, run.UID, run.GID) {
		return tailnet.ErrUnavailable
	}
	return nil
}

func (f *runFiles) openPreparedRoot(name string) (*os.Root, error) {
	info, e := f.root.Lstat(name)
	if e != nil || !rootDirectory(info, f.uid) {
		return nil, tailnet.ErrUnavailable
	}
	root, e := f.root.OpenRoot(name)
	if e != nil {
		return nil, tailnet.ErrUnavailable
	}
	return root, nil
}

func (f *runFiles) prepare(run projectRun, fresh bool) (*os.Root, error) {
	name := run.Target.Run
	if !containerID.MatchString(name) || run.UID == 0 || run.GID == 0 {
		return nil, tailnet.ErrInvalid
	}
	if fresh {
		// A pre-existing directory is an incomplete attempt, never an empty workspace.
		if e := f.root.Mkdir(name, 0o700); e != nil {
			return nil, tailnet.ErrConflict
		}
	}
	root, e := f.openPreparedRoot(name)
	if e != nil {
		return nil, e
	}
	for _, path := range []string{"control", "input"} {
		if e = prepareRunSubdir(root, path, run, fresh); e != nil {
			root.Close()
			return nil, e
		}
	}
	return root, nil
}

// The single-use key never enters argv, environment, a pipe, journal or project
// filesystem. Removal is allowed only after native exec completion is observed.
func writeRunKey(root *os.Root, run projectRun, key string) (*os.File, error) {
	if !strings.HasPrefix(key, "tskey-auth-") || len(key) > 1024 || strings.ContainsAny(key, "\r\n\x00") { // slop-audit-allow: production shape check for real Tailscale-shaped auth keys
		return nil, tailnet.ErrInvalid
	}
	file, e := root.OpenFile("input/key", os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o600)
	if e != nil {
		return nil, tailnet.ErrConflict
	}
	fail := func() (*os.File, error) { file.Close(); return nil, tailnet.ErrUnconfirmed }
	if e = file.Chown(int(run.UID), int(run.GID)); e != nil {
		return fail()
	}
	if _, e = io.WriteString(file, key); e != nil {
		return fail()
	}
	if e = file.Sync(); e != nil {
		return fail()
	}
	return file, nil
}

func retireRunKey(root *os.Root, file *os.File) error {
	before, e := file.Stat()
	after, other := root.Lstat("input/key")
	if e != nil || other != nil || !os.SameFile(before, after) {
		return tailnet.ErrUnconfirmed
	}
	if file.Truncate(0) != nil || file.Sync() != nil || root.Remove("input/key") != nil {
		return tailnet.ErrUnconfirmed
	}
	return nil
}

// Called only under the runtime lock after observing no outstanding native exec.
// This retires the exact completed consumer's input, not arbitrary retained files.
func retirePendingRunKey(root *os.Root, run projectRun) error {
	before, e := root.Lstat("input/key")
	if errors.Is(e, os.ErrNotExist) {
		return nil
	}
	if e != nil || !runtimeFile(before, run.UID, run.GID, 0o600) || before.Size() > 1024 {
		return tailnet.ErrUnconfirmed
	}
	file, e := root.OpenFile("input/key", os.O_WRONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if e != nil {
		return tailnet.ErrUnconfirmed
	}
	defer file.Close()
	info, e := file.Stat()
	if e != nil || !os.SameFile(before, info) || !runtimeFile(info, run.UID, run.GID, 0o600) || info.Size() > 1024 {
		return tailnet.ErrUnconfirmed
	}
	return retireRunKey(root, file)
}

func writeCompanionID(root *os.Root, id string) error {
	if !containerID.MatchString(id) {
		return tailnet.ErrUnconfirmed
	}
	file, e := root.OpenFile("companion-id", os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o600)
	if e != nil {
		return tailnet.ErrConflict
	}
	defer file.Close()
	if _, e = file.WriteString(id); e != nil {
		return tailnet.ErrUnconfirmed
	}
	if file.Sync() != nil {
		return tailnet.ErrUnconfirmed
	}
	dir, e := root.Open(".")
	if e != nil {
		return tailnet.ErrUnconfirmed
	}
	defer dir.Close()
	if dir.Sync() != nil {
		return tailnet.ErrUnconfirmed
	}
	return nil
}

func readCompanionID(base string, run projectRun, uid, gid uint32) (string, error) {
	if !projectID.MatchString(run.Target.Project) || !containerID.MatchString(run.Target.Run) {
		return "", tailnet.ErrInvalid
	}
	root, e := openRuntimeRoot(base, uid)
	if e != nil {
		return "", tailnet.ErrUnavailable
	}
	defer root.Close()
	for _, path := range []string{".", run.Target.Project, filepath.Join(run.Target.Project, run.Target.Run)} {
		info, e := root.Lstat(path)
		if e != nil || !rootDirectory(info, uid) {
			return "", tailnet.ErrUnavailable
		}
	}
	file, e := root.OpenFile(filepath.Join(run.Target.Project, run.Target.Run, "companion-id"), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if e != nil {
		return "", tailnet.ErrUnavailable
	}
	defer file.Close()
	info, e := file.Stat()
	if e != nil || !runtimeFile(info, uid, gid, 0o600) || info.Size() != 64 {
		return "", tailnet.ErrUnavailable
	}
	b, e := io.ReadAll(io.LimitReader(file, 65))
	if e != nil || !containerID.Match(b) {
		return "", tailnet.ErrUnavailable
	}
	return string(b), nil
}

// Validate the real shared resolver inode, not an arbitrary path obtained from
// the project. Tailscale owns backup/restore in the retained companion root.
func freshTailscaleConflict(devices string) bool {
	for _, line := range strings.Split(devices, "\n") {
		name, _, ok := strings.Cut(line, ":")
		if ok && strings.TrimSpace(name) == "tailscale0" {
			return true
		}
	}
	return false
}

func validateResolverText(data []byte, fresh bool) error {
	if len(data) > 16384 || len(data) == 0 {
		return tailnet.ErrUnsupported
	}
	text := strings.ToLower(string(data))
	if strings.Contains(text, "systemd-resolved") || strings.Contains(text, "resolvconf") || (fresh && strings.Contains(text, "tailscale")) {
		return tailnet.ErrUnsupported
	}
	return nil
}

func validateResolverInode(run projectRun, stat func(string) (os.FileInfo, error)) error {
	expected := filepath.Join("/var/lib/containers/storage/overlay-containers", run.Target.Container, "userdata/resolv.conf")
	if run.Resolver != expected {
		return tailnet.ErrUnsupported
	}
	info, e := stat(expected)
	if e != nil || !info.Mode().IsRegular() || info.Size() > 16384 {
		return tailnet.ErrUnsupported
	}
	actual, e := stat("/proc/" + strconv.Itoa(run.PID) + "/root/etc/resolv.conf")
	if e != nil || !os.SameFile(info, actual) {
		return tailnet.ErrUnsupported
	}
	return nil
}

func validateRunResolver(run projectRun, read func(string) ([]byte, error), stat func(string) (os.FileInfo, error), fresh bool) error {
	if e := validateResolverInode(run, stat); e != nil {
		return e
	}
	expected := filepath.Join("/var/lib/containers/storage/overlay-containers", run.Target.Container, "userdata/resolv.conf")
	data, e := read(expected)
	if e != nil {
		return tailnet.ErrUnsupported
	}
	if e = validateResolverText(data, fresh); e != nil {
		return e
	}
	devices, e := read("/proc/" + strconv.Itoa(run.PID) + "/net/dev")
	if e != nil || len(devices) > 65536 {
		return tailnet.ErrUnavailable
	}
	if fresh && freshTailscaleConflict(string(devices)) {
		return tailnet.ErrConflict
	}
	return nil
}
