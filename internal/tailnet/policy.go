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
	"syscall"

	"github.com/levitateos/sodaos/internal/filelock"
	"github.com/levitateos/sodaos/internal/strictjson"
	"golang.org/x/sys/unix"
)

// policyStore has one stable directory lock. Production anchors at /var/lib;
// tests supply a private descriptor, never a browser/native request path.
type policyStore struct {
	runtime bool // Fixed at construction after native runtime configuration admission.
	parent  func() (*os.Root, error)
	uid     uint32
	syncDir func(*os.File) error
}
type enrollmentPolicy struct {
	Version       int        `json:"version"`
	Revision      string     `json:"revision"`
	Binding       string     `json:"binding"`
	Tailnet       string     `json:"tailnet"`
	Tags          []string   `json:"tags"`
	Preauthorized bool       `json:"preauthorized"`
	Admission     bool       `json:"admission"`
	Default       bool       `json:"default"`
	Credential    credential `json:"credential"`
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
		if e != nil || !info.IsDir() || info.Mode().Perm()&0o022 != 0 {
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

func (p *policyStore) publish(root *os.Root, lock *os.File, name string, v any) error {
	// Refuse special/relinked occupants before atomic publication. Only host root
	// can alter this directory; noncooperating root writers aren't a CAS guarantee.
	if st, e := root.Lstat(name); e == nil {
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
	if v.Version != 2 || !revisionPattern.MatchString(v.Revision) || !revisionPattern.MatchString(v.Binding) || (v.Default && !v.Admission) {
		return v, ErrUnavailable
	}
	probe.ClientID, probe.ClientSecret = v.Credential.ClientID, v.Credential.Secret
	if probe.Validate() != nil {
		return v, ErrUnavailable
	}
	return v, nil
}

func (v enrollmentPolicy) view() EnrollmentView {
	return EnrollmentView{Revision: v.Revision, Binding: v.Binding, Tailnet: v.Tailnet, Tags: slices.Clone(v.Tags), Configured: v.Revision != "0", Admission: v.Admission, Default: v.Default, Preauthorized: v.Preauthorized, CredentialChecked: v.Revision != "0", RuntimeSupported: false}
}

func (p *policyStore) view(v enrollmentPolicy) EnrollmentView {
	out := v.view()
	out.RuntimeSupported = p.runtime
	return out
}

func (p *policyStore) enrollment(ctx context.Context) (EnrollmentView, error) {
	root, lock, err := p.lock(ctx, false)
	if errors.Is(err, os.ErrNotExist) {
		return p.view(enrollmentPolicy{Revision: "0", Tags: []string{}}), nil
	}
	if err != nil {
		return EnrollmentView{}, err
	}
	defer root.Close()
	defer lock.Close()
	v, err := p.load(root)
	return p.view(v), err
}

func validateUpdateRequest(r EnrollmentRequest, runtime bool) error {
	if r.Validate() != nil {
		return ErrInvalid
	}
	if r.Action == "default" && *r.Default && !runtime {
		return ErrUnsupported
	}
	return nil
}

func (p *policyStore) checkEnrollment(ctx context.Context, r EnrollmentRequest, check func(context.Context, EnrollmentRequest) error) (EnrollmentResult, error) {
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

func (p *policyStore) lockAndLoad(ctx context.Context, r EnrollmentRequest) (*os.Root, *os.File, enrollmentPolicy, error) {
	root, lock, err := p.lock(ctx, r.Action == "save")
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, enrollmentPolicy{}, ErrConflict
	}
	if err != nil {
		return nil, nil, enrollmentPolicy{}, err
	}
	v, err := p.load(root)
	if err != nil {
		root.Close()
		lock.Close()
		return nil, nil, enrollmentPolicy{}, err
	}
	if v.Revision != r.Revision {
		root.Close()
		lock.Close()
		return nil, nil, enrollmentPolicy{}, ErrConflict
	}
	return root, lock, v, nil
}

func canRotateEnrollment(v enrollmentPolicy, r EnrollmentRequest) bool {
	if v.Revision == "0" || v.Tailnet != r.Tailnet || !slices.Equal(v.Tags, r.Tags) || v.Preauthorized != *r.Preauthorized {
		return false
	}
	return true
}

func applySaveOrRotate(ctx context.Context, v enrollmentPolicy, r EnrollmentRequest, check func(context.Context, EnrollmentRequest) error) (enrollmentPolicy, error) {
	if r.Action == "rotate" && !canRotateEnrollment(v, r) {
		return enrollmentPolicy{}, ErrConflict
	}
	if err := check(ctx, r); err != nil {
		return enrollmentPolicy{}, err
	}
	if err := ctx.Err(); err != nil {
		return enrollmentPolicy{}, ErrUnconfirmed
	}
	if r.Action == "save" {
		v.Binding = newRevision()
		v.Default = false
		v.Admission = true
	}
	// Policy and credential are one restricted atomic publication. No archive,
	// provider revocation or automatic conversion of retained v1 files.
	v.Version, v.Tailnet, v.Tags, v.Preauthorized, v.Credential = 2, r.Tailnet, slices.Clone(r.Tags), *r.Preauthorized, credential{r.ClientID, r.ClientSecret}
	return v, nil
}

func applyDefault(v enrollmentPolicy, r EnrollmentRequest) (enrollmentPolicy, error) {
	if v.Revision == "0" {
		return enrollmentPolicy{}, ErrConflict
	}
	if *r.Default && !v.Admission {
		return enrollmentPolicy{}, ErrConflict
	}
	v.Default = *r.Default
	return v, nil
}

func applyDisable(v enrollmentPolicy) (enrollmentPolicy, error) {
	if v.Revision == "0" {
		return enrollmentPolicy{}, ErrConflict
	}
	v.Admission = false
	v.Default = false
	return v, nil
}

func applyPolicyMutation(ctx context.Context, v enrollmentPolicy, r EnrollmentRequest, check func(context.Context, EnrollmentRequest) error) (enrollmentPolicy, error) {
	switch r.Action {
	case "save", "rotate":
		return applySaveOrRotate(ctx, v, r, check)
	case "default":
		return applyDefault(v, r)
	case "disable":
		return applyDisable(v)
	default:
		return v, nil
	}
}

func (p *policyStore) update(ctx context.Context, r EnrollmentRequest, check func(context.Context, EnrollmentRequest) error) (EnrollmentResult, error) {
	if err := validateUpdateRequest(r, p.runtime); err != nil {
		return EnrollmentResult{}, err
	}
	// A pure credential check must not create directories, files or device keys.
	if r.Action == "check" {
		return p.checkEnrollment(ctx, r, check)
	}
	root, lock, v, err := p.lockAndLoad(ctx, r)
	if err != nil {
		return EnrollmentResult{}, err
	}
	defer root.Close()
	defer lock.Close()
	v, err = applyPolicyMutation(ctx, v, r, check)
	if err != nil {
		return EnrollmentResult{}, err
	}
	if ctx.Err() != nil {
		return EnrollmentResult{}, ErrUnconfirmed
	}
	v.Revision = newRevision()
	if err = p.publish(root, lock, "policy.json", v); err != nil {
		return EnrollmentResult{}, err
	}
	return EnrollmentResult{Outcome: "confirmed", Saved: true, CredentialChecked: v.view().CredentialChecked, Enrollment: p.view(v)}, nil
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
	if loaded.Version != 1 || !revisionPattern.MatchString(loaded.Revision) || (loaded.Binding != "" && !revisionPattern.MatchString(loaded.Binding)) || (loaded.Enabled && loaded.Binding == "") {
		return v, ErrUnavailable
	}
	if loaded.Project != project || loaded.Container != cid {
		return v, ErrConflict
	}
	return loaded, nil
}

func validateProjectRequest(r ProjectRequest, cid string, runtime bool) error {
	if r.Validate() != nil || !containerPattern.MatchString(cid) {
		return ErrInvalid
	}
	if !runtime && (r.Action == "enable" || r.Action == "retry") {
		return ErrUnsupported
	}
	return nil
}

func (p *policyStore) lockAndLoadProject(ctx context.Context, r ProjectRequest, cid string) (*os.Root, *os.File, projectPolicy, error) {
	root, lock, err := p.lock(ctx, r.Action != "inspect")
	v := projectPolicy{Project: r.Project, Container: cid, Revision: "0"}
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, v, nil
	}
	if err != nil {
		return nil, nil, v, err
	}
	v, err = p.loadProject(root, r.Project, cid)
	if err != nil {
		root.Close()
		lock.Close()
		return nil, nil, v, err
	}
	return root, lock, v, nil
}

func bindingMismatch(v projectPolicy, r ProjectRequest) bool {
	if r.Action == "retry" && (!v.Enabled || v.Binding != r.Binding) {
		return true
	}
	return v.Binding != "" && v.Binding != r.Binding
}

func (p *policyStore) validateProjectBinding(root *os.Root, v projectPolicy, r ProjectRequest) error {
	policy, err := p.load(root)
	if err != nil {
		return err
	}
	if !policy.Admission || policy.Binding != r.Binding || bindingMismatch(v, r) {
		return ErrConflict
	}
	return nil
}

func (p *policyStore) mutateProject(root *os.Root, lock *os.File, v projectPolicy, r ProjectRequest, ctx context.Context) (projectPolicy, error) {
	if v.Revision != r.Revision {
		return v, ErrConflict
	}
	if ctx.Err() != nil {
		return v, ErrUnconfirmed
	}
	if r.Action == "enable" || r.Action == "retry" {
		if err := p.validateProjectBinding(root, v, r); err != nil {
			return v, err
		}
		v.Binding = r.Binding
	}
	v.Enabled = r.Action != "disable"
	v.Version = 1
	v.Revision = newRevision()
	if err := p.publish(root, lock, "project-"+r.Project+".json", v); err != nil {
		return v, err
	}
	return v, nil
}

func initialProjectStateAndOutcome(action string, runtime bool) (string, string) {
	state := "runtime-unsupported"
	outcome := "observed"
	if action == "disable" {
		outcome = "disconnect-unconfirmed"
	}
	if runtime {
		state = "unconfirmed"
		if action != "inspect" {
			outcome = "runtime-unconfirmed"
		}
	}
	return state, outcome
}

func (p *policyStore) buildProjectView(root *os.Root, v projectPolicy, r ProjectRequest) (ProjectView, error) {
	state, outcome := initialProjectStateAndOutcome(r.Action, p.runtime)
	result := ProjectView{
		Saved:    r.Action != "inspect",
		Project:  r.Project,
		Revision: v.Revision,
		Binding:  v.Binding,
		Enabled:  v.Enabled,
		State:    state,
		Outcome:  outcome,
	}
	if p.runtime && root != nil {
		policy, err := p.load(root)
		if err != nil {
			return ProjectView{}, err
		}
		if policy.Admission {
			result.AvailableBinding = policy.Binding
			result.AvailableNetwork = policy.Tailnet
		}
	}
	return result, nil
}

func (p *policyStore) project(ctx context.Context, r ProjectRequest, cid string) (ProjectView, error) {
	if err := validateProjectRequest(r, cid, p.runtime); err != nil {
		return ProjectView{}, err
	}
	root, lock, v, err := p.lockAndLoadProject(ctx, r, cid)
	if err != nil {
		return ProjectView{}, err
	}
	if root != nil {
		defer root.Close()
		defer lock.Close()
	}
	if r.Action != "inspect" {
		v, err = p.mutateProject(root, lock, v, r, ctx)
		if err != nil {
			return ProjectView{}, err
		}
	}
	return p.buildProjectView(root, v, r)
}
