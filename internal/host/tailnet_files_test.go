package host

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/tailnet"
)

func runtimeTestRoot(t *testing.T) string {
	t.Helper()
	path := t.TempDir()
	if e := os.Chmod(path, 0700); e != nil {
		t.Fatal(e)
	}
	return path
}
func fileRun() projectRun {
	return projectRun{Target: tailnet.RunTarget{Project: "p" + strings.Repeat("a", 24), Container: strings.Repeat("b", 64), Run: strings.Repeat("c", 64)}, UID: uint32(os.Geteuid()), GID: uint32(os.Getegid()), PID: 77}
}
func TestTailnetRunFilesExclusiveSecretRetirementAndIndependentLocks(t *testing.T) {
	base := runtimeTestRoot(t)
	run := fileRun()
	if run.UID == 0 {
		run.UID = 100000
		run.GID = 100000
	}
	open := func(project string) (*runFiles, error) {
		return openRuntimeProjectOwned(t.Context(), base, project, uint32(os.Geteuid()), uint32(os.Getegid()))
	}
	f, e := open(run.Target.Project)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	if _, e = openRuntimeProjectOwned(ctx, base, run.Target.Project, uint32(os.Geteuid()), uint32(os.Getegid())); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal("lock waiter not cancellable", e)
	}
	other, e := open("p" + strings.Repeat("d", 24))
	if e != nil {
		t.Fatal("unrelated project blocked", e)
	}
	other.Close()
	root, e := f.prepare(run, true)
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	if _, e = f.prepare(run, true); !errors.Is(e, tailnet.ErrConflict) {
		t.Fatal("run root recreated", e)
	}
	if e = f.saveCurrent(run); e != nil {
		t.Fatal(e)
	}
	restored, e := f.current()
	if e != nil || restored != run {
		t.Fatal(restored, e)
	}
	cid := strings.Repeat("e", 64)
	if e = writeCompanionID(root, cid); e != nil {
		t.Fatal(e)
	}
	if e = writeCompanionID(root, strings.Repeat("f", 64)); !errors.Is(e, tailnet.ErrConflict) {
		t.Fatal("companion identity replaced", e)
	}
	if observed, e := readCompanionID(base, run, uint32(os.Geteuid()), uint32(os.Getegid())); e != nil || observed != cid {
		t.Fatal("companion identity not retained", observed, e)
	}
	key, e := writeRunKey(root, run, "tskey-auth-synthetic-one-use-only")
	if e != nil {
		t.Fatal(e)
	}
	defer key.Close()
	info, e := key.Stat()
	if e != nil || !runtimeFile(info, run.UID, run.GID, 0600) {
		t.Fatal("unsafe key metadata", e)
	}
	if _, e = writeRunKey(root, run, "tskey-auth-synthetic-another"); !errors.Is(e, tailnet.ErrConflict) {
		t.Fatal("key replaced", e)
	}
	if e = retireRunKey(root, key); e != nil {
		t.Fatal(e)
	}
	if info, e = key.Stat(); e != nil || info.Size() != 0 {
		t.Fatal("retired inode still contains key", e)
	}
	if _, e = root.Lstat("input/key"); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("key not retired", e)
	}
	// No reusable credentials are stored with the runtime record.
	b, e := os.ReadFile(filepath.Join(base, run.Target.Project, "current.json"))
	if e != nil || strings.Contains(string(b), "tskey") || strings.Contains(string(b), "secret") {
		t.Fatal("unsafe runtime projection", e)
	}
}
func TestTailnetRuntimeRootRefusesSymlinkWithoutWritingThroughIt(t *testing.T) {
	base := runtimeTestRoot(t)
	link := filepath.Join(t.TempDir(), "runtime")
	if e := os.Symlink(base, link); e != nil {
		t.Fatal(e)
	}
	if f, e := openRuntimeProjectOwned(t.Context(), link, fileRun().Target.Project, uint32(os.Geteuid()), uint32(os.Getegid())); e == nil {
		f.Close()
		t.Fatal("runtime root link accepted")
	}
	entries, e := os.ReadDir(base)
	if e != nil || len(entries) != 0 {
		t.Fatal("wrote through unsafe root", e)
	}
}

func TestTailnetRunFilesRefuseSymlinksModesAndChangedKeyInode(t *testing.T) {
	for _, kind := range []string{"project-link", "lock-link", "lock-hardlink", "unsafe-mode"} {
		t.Run(kind, func(t *testing.T) {
			base := runtimeTestRoot(t)
			outside := t.TempDir()
			run := fileRun()
			project := filepath.Join(base, run.Target.Project)
			if kind == "project-link" {
				if e := os.Symlink(outside, project); e != nil {
					t.Fatal(e)
				}
			} else {
				if e := os.Mkdir(project, 0700); e != nil {
					t.Fatal(e)
				}
				target := filepath.Join(outside, "retained")
				if e := os.WriteFile(target, []byte("preserve"), 0600); e != nil {
					t.Fatal(e)
				}
				var e error
				switch kind {
				case "lock-link":
					e = os.Symlink(target, filepath.Join(project, "lock"))
				case "lock-hardlink":
					e = os.Link(target, filepath.Join(project, "lock"))
				case "unsafe-mode":
					e = os.Chmod(project, 0755)
				}
				if e != nil {
					t.Fatal(e)
				}
			}
			f, e := openRuntimeProjectOwned(t.Context(), base, run.Target.Project, uint32(os.Geteuid()), uint32(os.Getegid()))
			if e == nil {
				f.Close()
				t.Fatal("unsafe path accepted", kind)
			}
		})
	}
	base := runtimeTestRoot(t)
	run := fileRun()
	if run.UID == 0 {
		run.UID = 100000
		run.GID = 100000
	}
	f, e := openRuntimeProjectOwned(t.Context(), base, run.Target.Project, uint32(os.Geteuid()), uint32(os.Getegid()))
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	root, e := f.prepare(run, true)
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	key, e := writeRunKey(root, run, "tskey-auth-synthetic-secret")
	if e != nil {
		t.Fatal(e)
	}
	defer key.Close()
	if root.Rename("input/key", "input/retained") != nil || root.WriteFile("input/key", []byte("later write"), 0600) != nil {
		t.Fatal("fixture")
	}
	if e = retireRunKey(root, key); !errors.Is(e, tailnet.ErrUnconfirmed) {
		t.Fatal("replaced input retired", e)
	}
	b, e := root.ReadFile("input/key")
	if e != nil || string(b) != "later write" {
		t.Fatal("later write lost", e)
	}
}
func TestTailnetResolverRequiresOriginalInodeAndNoConflictingManager(t *testing.T) {
	run := fileRun()
	run.Resolver = "/var/lib/containers/storage/overlay-containers/" + run.Target.Container + "/userdata/resolv.conf"
	path := filepath.Join(t.TempDir(), "resolver")
	if os.WriteFile(path, []byte("nameserver 192.0.2.1\n"), 0600) != nil {
		t.Fatal("fixture")
	}
	info, e := os.Stat(path)
	if e != nil {
		t.Fatal(e)
	}
	stat := func(string) (os.FileInfo, error) { return info, nil }
	contents := "nameserver 192.0.2.1\n"
	devices := "eth0: 0 0\n"
	read := func(path string) ([]byte, error) {
		if path == run.Resolver {
			return []byte(contents), nil
		}
		return []byte(devices), nil
	}
	if e = validateRunResolver(run, read, stat, true); e != nil {
		t.Fatal(e)
	}
	for _, text := range []string{"# Generated by systemd-resolved\n", "# resolvconf\n", "# resolv.conf(5) file generated by tailscale\n"} {
		contents = text
		if e = validateRunResolver(run, read, stat, true); e == nil {
			t.Fatal("conflicting resolver accepted")
		}
	}
	contents = "nameserver 192.0.2.1\n"
	devices = "tailscale0: 0 0\n"
	if e = validateRunResolver(run, read, stat, true); !errors.Is(e, tailnet.ErrConflict) {
		t.Fatal("foreign TUN accepted", e)
	}
	if e = validateRunResolver(run, read, stat, false); e != nil {
		t.Fatal("owned same-run interface refused", e)
	}
	other := filepath.Join(t.TempDir(), "other")
	if os.WriteFile(other, []byte(contents), 0600) != nil {
		t.Fatal("fixture")
	}
	stat = func(p string) (os.FileInfo, error) {
		if p == run.Resolver {
			return info, nil
		}
		return os.Stat(other)
	}
	if e = validateRunResolver(run, read, stat, false); e == nil {
		t.Fatal("different resolver inode accepted")
	}
}
func TestTailnetCompletedKeyInputCanBeRetiredForExplicitRetry(t *testing.T) {
	base := runtimeTestRoot(t)
	run := fileRun()
	if run.UID == 0 {
		run.UID, run.GID = 100000, 100000
	}
	f, e := openRuntimeProjectOwned(t.Context(), base, run.Target.Project, uint32(os.Geteuid()), uint32(os.Getegid()))
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	root, e := f.prepare(run, true)
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	if _, e = root.Lstat("state"); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("node identity directory created")
	}
	key, e := writeRunKey(root, run, "tskey-auth-synthetic-completed-input")
	if e != nil {
		t.Fatal(e)
	}
	defer key.Close()
	if e = retirePendingRunKey(root, run); e != nil {
		t.Fatal(e)
	}
	if info, e := key.Stat(); e != nil || info.Size() != 0 {
		t.Fatal("old input not scrubbed", e)
	}
	if e = retirePendingRunKey(root, run); e != nil {
		t.Fatal("absent input became a permanent retry veto", e)
	}
	if e = root.Symlink("../companion-id", "input/key"); e != nil {
		t.Fatal(e)
	}
	if e = retirePendingRunKey(root, run); e == nil {
		t.Fatal("substituted input accepted")
	}
}
