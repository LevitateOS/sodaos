package tailnet

import (
	"context"
	"errors"
	"os"
)

type projectPolicy struct {
	Version   int    `json:"version"`
	Revision  string `json:"revision"`
	Project   string `json:"project"`
	Container string `json:"container"`
	Binding   string `json:"binding"`
	Enabled   bool   `json:"enabled"`
}

func validLoadedProject(loaded projectPolicy) bool {
	return loaded.Version == 1 && revisionPattern.MatchString(loaded.Revision) && (loaded.Binding == "" || revisionPattern.MatchString(loaded.Binding)) && (!loaded.Enabled || loaded.Binding != "")
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
	if !validLoadedProject(loaded) {
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
