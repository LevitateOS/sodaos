package tailnet

import (
	"context"
	"errors"
	"os"
	"slices"
)

// ProjectSelection is reviewed explicitly by the creating human. Omission or
// Enabled=false means Off regardless of a subsequently changed operator default.
type ProjectSelection struct {
	Enabled  bool   `json:"enabled"`
	Revision string `json:"revision,omitempty"`
	Binding  string `json:"binding,omitempty"`
}

func (s ProjectSelection) Validate() error {
	if !s.Enabled {
		if s.Revision != "" || s.Binding != "" {
			return ErrInvalid
		}
	} else if !revisionPattern.MatchString(s.Revision) || !revisionPattern.MatchString(s.Binding) {
		return ErrInvalid
	}
	return nil
}

// RunBinding is a native-only public-metadata projection, not a credential or a
// browser-selected target. Existing connected nodes keep their saved binding when
// global admission closes; only new key issuance needs Admission=true.
type RunBinding struct {
	Enabled, Admission bool
	Tailnet            string
	Tags               []string
}

func (m *Management) RunBinding(ctx context.Context, target RunTarget) (RunBinding, error) {
	var out RunBinding
	if !target.valid() {
		return out, ErrInvalid
	}
	root, lock, e := m.policy.lock(ctx, false)
	if errors.Is(e, os.ErrNotExist) {
		return out, nil
	}
	if e != nil {
		return out, e
	}
	defer root.Close()
	defer lock.Close()
	project, e := m.policy.loadProject(root, target.Project, target.Container)
	if e != nil {
		return out, e
	}
	out.Enabled = project.Enabled
	if !project.Enabled {
		return out, nil
	}
	p, e := m.policy.load(root)
	if e != nil {
		return out, e
	}
	if p.Binding != project.Binding {
		return out, ErrConflict
	}
	out.Admission = p.Admission
	out.Tailnet = p.Tailnet
	out.Tags = slices.Clone(p.Tags)
	return out, nil
}
