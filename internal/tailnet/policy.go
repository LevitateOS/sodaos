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
	"slices"
	"strings"
	"syscall"

	"github.com/levitateos/sodaos/internal/filelock"
	"github.com/levitateos/sodaos/internal/strictjson"
	"golang.org/x/sys/unix"
)

// policyStore has one stable directory lock. Production anchors at /var/lib;
// tests supply a private descriptor, never a browser/native request path.
type policyStore struct {
	parent  func() (*os.Root, error)
	uid     uint32
	syncDir func(*os.File) error
}
type enrollmentPolicy struct {
	Version       int      `json:"version"`
	Revision      string   `json:"revision"`
	Binding       string   `json:"binding"`
	Tailnet       string   `json:"tailnet"`
	Tags          []string `json:"tags"`
	Preauthorized bool     `json:"preauthorized"`
	Admission     bool     `json:"admission"`
	Default       bool     `json:"default"`
	Credential    string   `json:"credential"`
}
type credential struct {
	ClientID string `json:"client_id"`
	Secret   string `json:"secret"`
}
type projectPolicy struct {
	Version   int    `json:"version"`
	Revision  string `json:"revision"`
	Project   string `json:"project"`
	Container string `json:"container"`
	Binding   string `json:"binding"`
	Enabled   bool   `json:"enabled"`
	ActiveRun string `json:"active_run,omitempty"`
}

func newRevision() string { b := make([]byte, 16); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func owned(info os.FileInfo, uid uint32, directory bool) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uid || info.Mode().Perm()&0077 != 0 {
		return false
	}
	if directory {
		return info.IsDir()
	}
	return info.Mode().IsRegular() && stat.Nlink == 1 && info.Size() <= 65536
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
		info, e := root.Lstat(name)
		if e != nil || !info.IsDir() || info.Mode().Perm()&0022 != 0 {
			root.Close()
			return nil, ErrUnavailable
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || st.Uid != 0 {
			root.Close()
			return nil, ErrUnavailable
		}
		next, e := root.OpenRoot(name)
		root.Close()
		if e != nil {
			return nil, ErrUnavailable
		}
		root = next
	}
	return root, nil
}
func (p *policyStore) lock(ctx context.Context, create bool) (*os.Root, *os.File, error) {
	parent, err := p.parent()
	if err != nil {
		return nil, nil, ErrUnavailable
	}
	defer parent.Close()
	info, err := parent.Lstat("soda-tailnet")
	if errors.Is(err, os.ErrNotExist) && create {
		if e := ctx.Err(); e != nil {
			return nil, nil, e
		}
		err = parent.Mkdir("soda-tailnet", 0700)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return nil, nil, ErrUnavailable
		}
		directory, e := parent.Open(".")
		if e != nil {
			return nil, nil, ErrUnconfirmed
		}
		e = directory.Sync()
		directory.Close()
		if e != nil {
			return nil, nil, ErrUnconfirmed
		}
		info, err = parent.Lstat("soda-tailnet")
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, os.ErrNotExist
	}
	if err != nil || !owned(info, p.uid, true) {
		return nil, nil, ErrUnavailable
	}
	root, err := parent.OpenRoot("soda-tailnet")
	if err != nil {
		return nil, nil, ErrUnavailable
	}
	lock, err := root.Open(".")
	if err == nil {
		var st os.FileInfo
		st, err = lock.Stat()
		if err == nil && !owned(st, p.uid, true) {
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
		root.Close()
		return nil, nil, ErrUnavailable
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
func (p *policyStore) publish(root *os.Root, lock *os.File, name string, v any) error {
	// Refuse special/relinked occupants before atomic publication. Only host root
	// can alter this directory; noncooperating root writers aren't a CAS guarantee.
	if st, e := root.Lstat(name); e == nil {
		if strings.HasPrefix(name, "credential-") {
			return ErrConflict
		}
		if !owned(st, p.uid, false) {
			return ErrUnavailable
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return ErrUnavailable
	}
	b, err := json.Marshal(v)
	if err != nil || len(b) > 65536 {
		return ErrInvalid
	}
	temp := "pending-" + newRevision()
	f, err := root.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
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
	if p.syncDir != nil {
		err = p.syncDir(lock)
	} else {
		err = lock.Sync()
	}
	if err != nil {
		return ErrUnconfirmed
	}
	return nil
}
func (p *policyStore) load(root *os.Root) (enrollmentPolicy, error) {
	var v enrollmentPolicy
	err := p.read(root, "policy.json", &v)
	if errors.Is(err, os.ErrNotExist) {
		return enrollmentPolicy{Revision: "0", Tags: []string{}}, nil
	}
	if err != nil {
		return v, err
	}
	probe := EnrollmentRequest{Action: "save", Revision: v.Revision, Tailnet: v.Tailnet, Tags: v.Tags, Preauthorized: &v.Preauthorized}
	var c credential
	if v.Version != 1 || !revisionPattern.MatchString(v.Revision) || !revisionPattern.MatchString(v.Binding) || !revisionPattern.MatchString(v.Credential) || (v.Default && !v.Admission) {
		return v, ErrUnavailable
	}
	if p.read(root, "credential-"+v.Credential+".json", &c) != nil {
		return v, ErrUnavailable
	}
	probe.ClientID, probe.ClientSecret = c.ClientID, c.Secret
	if probe.Validate() != nil {
		return v, ErrUnavailable
	}
	return v, nil
}
func (v enrollmentPolicy) view() EnrollmentView {
	return EnrollmentView{Revision: v.Revision, Binding: v.Binding, Tailnet: v.Tailnet, Tags: slices.Clone(v.Tags), Configured: v.Revision != "0", Admission: v.Admission, Default: v.Default, Preauthorized: v.Preauthorized, CredentialChecked: v.Revision != "0", RuntimeSupported: false}
}
func (p *policyStore) enrollment(ctx context.Context) (EnrollmentView, error) {
	root, lock, err := p.lock(ctx, false)
	if errors.Is(err, os.ErrNotExist) {
		return (enrollmentPolicy{Revision: "0", Tags: []string{}}).view(), nil
	}
	if err != nil {
		return EnrollmentView{}, err
	}
	defer root.Close()
	defer lock.Close()
	v, err := p.load(root)
	return v.view(), err
}
func (p *policyStore) update(ctx context.Context, r EnrollmentRequest, check func(context.Context, EnrollmentRequest) error) (EnrollmentResult, error) {
	if r.Validate() != nil {
		return EnrollmentResult{}, ErrInvalid
	}
	if r.Action == "default" && *r.Default {
		return EnrollmentResult{}, ErrUnsupported
	}
	// A pure credential check must not create directories, files or device keys.
	if r.Action == "check" {
		before, err := p.enrollment(ctx)
		if err != nil {
			return EnrollmentResult{}, err
		}
		if before.Revision != r.Revision {
			return EnrollmentResult{}, ErrConflict
		}
		if err = check(ctx, r); err != nil {
			return EnrollmentResult{}, err
		}
		after, err := p.enrollment(ctx)
		if err != nil {
			return EnrollmentResult{}, err
		}
		if after.Revision != before.Revision {
			return EnrollmentResult{}, ErrConflict
		}
		return EnrollmentResult{Outcome: "confirmed", CredentialChecked: true, Enrollment: after}, nil
	}
	root, lock, err := p.lock(ctx, r.Action == "save")
	if errors.Is(err, os.ErrNotExist) {
		return EnrollmentResult{}, ErrConflict
	}
	if err != nil {
		return EnrollmentResult{}, err
	}
	defer root.Close()
	defer lock.Close()
	v, err := p.load(root)
	if err != nil {
		return EnrollmentResult{}, err
	}
	if v.Revision != r.Revision {
		return EnrollmentResult{}, ErrConflict
	}
	switch r.Action {
	case "save", "rotate":
		if r.Action == "rotate" && (v.Revision == "0" || v.Tailnet != r.Tailnet || !slices.Equal(v.Tags, r.Tags) || v.Preauthorized != *r.Preauthorized) {
			return EnrollmentResult{}, ErrConflict
		}
		if err = check(ctx, r); err != nil {
			return EnrollmentResult{}, err
		}
		if err = ctx.Err(); err != nil {
			return EnrollmentResult{}, ErrUnconfirmed
		}
		ref := newRevision()
		// Immutable, root-only credentials: old inputs remain in custody on rotation.
		if err = p.publish(root, lock, "credential-"+ref+".json", credential{r.ClientID, r.ClientSecret}); err != nil {
			return EnrollmentResult{}, err
		}
		if r.Action == "save" {
			v.Binding = newRevision()
			v.Default = false
			v.Admission = true
		}
		v.Version, v.Tailnet, v.Tags, v.Preauthorized, v.Credential = 1, r.Tailnet, slices.Clone(r.Tags), *r.Preauthorized, ref
	case "default":
		if v.Revision == "0" {
			return EnrollmentResult{}, ErrConflict
		}
		v.Default = false
	case "disable":
		if v.Revision == "0" {
			return EnrollmentResult{}, ErrConflict
		}
		v.Admission = false
		v.Default = false
	}
	if ctx.Err() != nil {
		return EnrollmentResult{}, ErrUnconfirmed
	}
	v.Revision = newRevision()
	if err = p.publish(root, lock, "policy.json", v); err != nil {
		return EnrollmentResult{}, err
	}
	return EnrollmentResult{Outcome: "confirmed", Saved: true, CredentialChecked: v.view().CredentialChecked, Enrollment: v.view()}, nil
}
func (p *policyStore) loadProject(root *os.Root, project, cid string) (projectPolicy, error) {
	v := projectPolicy{Project: project, Container: cid, Revision: "0"}
	var loaded projectPolicy
	e := p.read(root, "project-"+project+".json", &loaded)
	if errors.Is(e, os.ErrNotExist) {
		return v, nil
	}
	if e != nil {
		return v, e
	}
	if loaded.Version != 1 || !revisionPattern.MatchString(loaded.Revision) || (loaded.Binding != "" && !revisionPattern.MatchString(loaded.Binding)) || (loaded.Enabled && loaded.Binding == "") || (loaded.ActiveRun != "" && !containerPattern.MatchString(loaded.ActiveRun)) {
		return v, ErrUnavailable
	}
	if loaded.Project != project || loaded.Container != cid {
		return v, ErrConflict
	}
	return loaded, nil
}
func (p *policyStore) project(ctx context.Context, r ProjectRequest, cid string) (ProjectView, error) {
	if r.Validate() != nil || !containerPattern.MatchString(cid) {
		return ProjectView{}, ErrInvalid
	}
	// No dormant launcher, pretend success or automatic key creation in stage 2.
	if r.Action == "enable" || r.Action == "retry" {
		return ProjectView{}, ErrUnsupported
	}
	root, lock, err := p.lock(ctx, r.Action != "inspect")
	v := projectPolicy{Project: r.Project, Container: cid, Revision: "0"}
	if !errors.Is(err, os.ErrNotExist) {
		if err != nil {
			return ProjectView{}, err
		}
		defer root.Close()
		defer lock.Close()
		v, err = p.loadProject(root, r.Project, cid)
		if err != nil {
			return ProjectView{}, err
		}
	}
	if r.Action == "disable" {
		if v.Revision != r.Revision {
			return ProjectView{}, ErrConflict
		}
		if ctx.Err() != nil {
			return ProjectView{}, ErrUnconfirmed
		}
		v.Enabled = false
		v.Version = 1
		v.Revision = newRevision()
		if err = p.publish(root, lock, "project-"+r.Project+".json", v); err != nil {
			return ProjectView{}, err
		}
	}
	state := "runtime-unsupported" // Policy Off does not prove a current connection is gone.
	outcome := "observed"
	if r.Action == "disable" {
		outcome = "disconnect-unconfirmed"
	} // No companion observer yet.
	return ProjectView{Saved: r.Action == "disable", Project: r.Project, Revision: v.Revision, Binding: v.Binding, Enabled: v.Enabled, State: state, Outcome: outcome}, nil
}
