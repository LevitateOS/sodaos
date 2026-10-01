//go:build linux

package terminal

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/identity"
	domain "github.com/levitateos/sodaos/internal/project"
)

// factoryCodexOutputBinding checks that a recorded binding still addresses
// its run directory and that the current Project container still equals the
// recorded incarnation. Anything else refuses: a viewer must never receive
// another execution's bytes.
func (s *Service) factoryCodexOutputBinding(ctx context.Context, projectID string, b *identity.Binding) (FactoryCodexPaths, error) {
	if b == nil || b.Kind != identity.Factory || b.Scope != domain.FactoryScopeCodex || !terminalID.MatchString(b.ID) {
		return FactoryCodexPaths{}, identity.ErrDenied
	}
	if !containerID.MatchString(b.Project) || !domain.ValidFactoryRole(b.Login) || !terminalID.MatchString(b.InvocationID) {
		return FactoryCodexPaths{}, identity.ErrDenied
	}
	if !domain.ValidPreparationID(b.ChildID) {
		return FactoryCodexPaths{}, identity.ErrDenied
	}
	checkout, runDir, home, codex := domain.FactoryRunPaths(b.Login, b.ChildID, b.ID)
	if checkout == "" || filepath.Clean(b.CredentialRoot) != runDir {
		return FactoryCodexPaths{}, identity.ErrDenied
	}
	container, err := s.factoryProjectContainer(ctx, projectID, true)
	if err != nil || container != b.Project {
		return FactoryCodexPaths{}, identity.ErrStale
	}
	return FactoryCodexPaths{
		Checkout: checkout, RunDir: runDir, Home: home, Codex: codex,
		Stdout: runDir + "/stdout.log",
	}, nil
}

func factoryOutputSize(out []byte) (int64, bool) {
	size, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	return size, err == nil && size >= 0 && len(out) <= 64
}

// FactoryCodexOutput reads one bounded slice of the run's recorded CLI
// output log at a byte cursor. A cursor past the recorded size reports a
// gap; a zero cursor on an over-window log serves the trailing window with
// truncated set. A missing log reads empty: the CLI may not have written
// yet. Only the recorded container incarnation is ever read.
func (s *Service) FactoryCodexOutput(ctx context.Context, projectID string, b *identity.Binding, offset int64, limit int) (FactoryCodexOutputSlice, error) {
	var empty FactoryCodexOutputSlice
	if offset < 0 || offset > domain.MaxFactoryOutputOffset || limit < 1 || limit > domain.MaxFactoryOutputRead {
		return empty, identity.ErrDenied
	}
	p, err := s.factoryCodexOutputBinding(ctx, projectID, b)
	if err != nil {
		return empty, err
	}
	var total int64
	if out, err := s.podman(ctx, nil, "--remote=false", "exec", b.Project, "/usr/bin/stat", "-c", "%s", p.Stdout); err == nil {
		if size, ok := factoryOutputSize(out); ok {
			total = size
		}
	}
	if offset > total {
		return FactoryCodexOutputSlice{Total: total, Offset: total, Gap: true}, nil
	}
	start := offset
	truncated := false
	if start == 0 && total > domain.MaxFactoryOutputWindow {
		start, truncated = total-domain.MaxFactoryOutputWindow, true
	}
	if start >= total {
		return FactoryCodexOutputSlice{Total: total, Offset: start, Truncated: truncated}, nil
	}
	read := "/usr/bin/tail -c +" + strconv.FormatInt(start+1, 10) + " " + shellQuote(p.Stdout) + " | /usr/bin/head -c " + strconv.Itoa(limit) + "\n"
	out, err := s.podman(ctx, nil, "--remote=false", "exec", b.Project, "/usr/bin/sh", "-c", read)
	if err != nil {
		return FactoryCodexOutputSlice{Total: total, Offset: start, Truncated: truncated}, nil
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return FactoryCodexOutputSlice{Data: out, Total: total, Offset: start, Truncated: truncated}, nil
}
