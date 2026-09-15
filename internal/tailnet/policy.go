package tailnet

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"syscall"

	"github.com/levitateos/sodaos/internal/filelock"
	"github.com/levitateos/sodaos/internal/strictjson"
	"golang.org/x/sys/unix"
)

// policyStore has one stable directory lock. Production anchors at /var/lib;
// tests supply a private descriptor, never a browser/native request path.
// Enrollment policy lives in policy_enrollment.go, project policy in
// policy_project.go; this file holds only the shared locked-file plumbing.
type policyStore struct {
	runtime bool // Fixed at construction after native runtime configuration admission.
	parent  func() (*os.Root, error)
	uid     uint32
	syncDir func(*os.File) error
}

func newRevision() string { b := make([]byte, 16); _, _ = rand.Read(b); return hex.EncodeToString(b) }

func owned(info os.FileInfo, uid uint32, directory bool) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uid || info.Mode().Perm()&0o077 != 0 {
		return false
	}
	if directory {
		return info.IsDir()
	}
	return info.Mode().IsRegular() && stat.Nlink == 1 && info.Size() <= 65536
}

func openOwnedLibDir(root *os.Root, name string) (*os.Root, error) {
	info, e := root.Lstat(name)
	if e != nil || !info.IsDir() || info.Mode().Perm()&0o022 != 0 {
		return nil, ErrUnavailable
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != 0 {
		return nil, ErrUnavailable
	}
	next, e := root.OpenRoot(name)
	if e != nil {
		return nil, ErrUnavailable
	}
	return next, nil
}

func policyParent() (*os.Root, error) {
	if os.Geteuid() != 0 {
		return nil, ErrUnavailable
	}
	root, err := os.OpenRoot("/")
	if err != nil {
		return nil, ErrUnavailable
	}
	for _, name := range []string{"var", "lib"} {
		next, e := openOwnedLibDir(root, name)
		root.Close()
		if e != nil {
			return nil, ErrUnavailable
		}
		root = next
	}
	return root, nil
}

func createPolicyDir(ctx context.Context, parent *os.Root) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	err := parent.Mkdir("soda-tailnet", 0o700)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return ErrUnavailable
	}
	directory, e := parent.Open(".")
	if e != nil {
		return ErrUnconfirmed
	}
	e = directory.Sync()
	directory.Close()
	if e != nil {
		return ErrUnconfirmed
	}
	return nil
}

func (p *policyStore) ensurePolicyDir(ctx context.Context, parent *os.Root, create bool) (os.FileInfo, error) {
	info, err := parent.Lstat("soda-tailnet")
	if errors.Is(err, os.ErrNotExist) && create {
		if e := createPolicyDir(ctx, parent); e != nil {
			return nil, e
		}
		info, err = parent.Lstat("soda-tailnet")
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, os.ErrNotExist
	}
	if err != nil || !owned(info, p.uid, true) {
		return nil, ErrUnavailable
	}
	return info, nil
}

func acquirePolicyLock(ctx context.Context, root *os.Root, uid uint32) (*os.File, error) {
	lock, err := root.Open(".")
	if err == nil {
		var st os.FileInfo
		st, err = lock.Stat()
		if err == nil && !owned(st, uid, true) {
			err = ErrUnavailable
		}
	}
	if err == nil {
		err = filelock.Acquire(ctx, lock, unix.LOCK_EX)
	}
	if err != nil {
		if lock != nil {
			lock.Close()
		}
		return nil, ErrUnavailable
	}
	return lock, nil
}

func (p *policyStore) lock(ctx context.Context, create bool) (*os.Root, *os.File, error) {
	parent, err := p.parent()
	if err != nil {
		return nil, nil, ErrUnavailable
	}
	defer parent.Close()
	if _, err = p.ensurePolicyDir(ctx, parent, create); err != nil {
		return nil, nil, err
	}
	root, err := parent.OpenRoot("soda-tailnet")
	if err != nil {
		return nil, nil, ErrUnavailable
	}
	lock, err := acquirePolicyLock(ctx, root, p.uid)
	if err != nil {
		root.Close()
		return nil, nil, err
	}
	return root, lock, nil
}

func (p *policyStore) read(root *os.Root, name string, out any) error {
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if errors.Is(err, os.ErrNotExist) {
		return os.ErrNotExist
	}
	if err != nil {
		return ErrUnavailable
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !owned(st, p.uid, false) {
		return ErrUnavailable
	}
	b, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil || len(b) > 65536 || strictjson.Decode(bytes.NewReader(b), out) != nil {
		return ErrUnavailable
	}
	return nil
}

func refuseUnownedPolicy(root *os.Root, name string, uid uint32) error {
	st, e := root.Lstat(name)
	if e == nil {
		if !owned(st, uid, false) {
			return ErrUnavailable
		}
		return nil
	}
	if !errors.Is(e, os.ErrNotExist) {
		return ErrUnavailable
	}
	return nil
}

func writePolicyFile(root *os.Root, name string, b []byte) error {
	temp := "pending-" + newRevision()
	f, err := root.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return ErrUnconfirmed
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return ErrUnconfirmed
	}
	if root.Rename(temp, name) != nil {
		return ErrUnconfirmed
	}
	return nil
}

func (p *policyStore) syncPolicy(lock *os.File) error {
	if p.syncDir != nil {
		return p.syncDir(lock)
	}
	return lock.Sync()
}

func (p *policyStore) publish(root *os.Root, lock *os.File, name string, v any) error {
	// Refuse special/relinked occupants before atomic publication. Only host root
	// can alter this directory; noncooperating root writers aren't a CAS guarantee.
	if err := refuseUnownedPolicy(root, name, p.uid); err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil || len(b) > 65536 {
		return ErrInvalid
	}
	if err = writePolicyFile(root, name, b); err != nil {
		return err
	}
	if err = p.syncPolicy(lock); err != nil {
		return ErrUnconfirmed
	}
	return nil
}
