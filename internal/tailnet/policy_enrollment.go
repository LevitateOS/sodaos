package tailnet

import (
	"context"
	"errors"
	"os"
	"slices"
)

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
