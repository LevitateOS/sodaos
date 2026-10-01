package project

import (
	"bytes"
	"context"
	"errors"
	"path"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/identity"
	domain "github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/strictjson"
)

// PrepareCandidate reuses the protected approved snapshot, never role-owned
// setup or Git state. Prepare retains ownership of source-bundle verification,
// fresh clone, installed-tool checks and detached setup execution.
func (r *Runtime) PrepareCandidate(ctx context.Context, in domain.FactoryCandidate) (domain.PrepareState, error) {
	var empty domain.PrepareState
	if err := in.Validate(); err != nil {
		return empty, err
	}
	container, err := r.prepareContainer(ctx, in.Preparation.Project, true)
	if err != nil {
		return empty, err
	}
	source, err := r.inspectPreparationState(ctx, domain.PrepareInspect{Project: in.Preparation.Project, ID: in.SourcePreparation}, container)
	if err != nil {
		return empty, err
	}
	if !source.Ready || source.Stopped || source.Role != in.Preparation.Role || source.SetupDigest != in.Preparation.SetupDigest {
		return empty, errors.New("candidate source preparation is not ready for this role and setup")
	}
	files, err := r.candidateApprovedFiles(ctx, container, in, source.SourceCommit)
	if err != nil {
		return empty, err
	}
	return r.Prepare(ctx, domain.Prepare{Preparation: in.Preparation, Setup: domain.ApprovedSetup{Files: files, Bundle: in.Bundle}})
}

// candidateProtectedFile reads only a fixed protected snapshot location.
// Metadata checks reject role-writable files, links and non-regular sources;
// head bounds the bytes returned over the executor transport.
func (r *Runtime) candidateProtectedFile(ctx context.Context, container, filename string, limit int) ([]byte, error) {
	meta, err := r.podman(ctx, nil, "exec", container, "/usr/bin/env", "LC_ALL=C", "/usr/bin/stat", "-c", "%u:%g:%a:%h:%F", "--", filename)
	if err != nil || string(meta) != "0:0:644:1:regular file\n" {
		return nil, errors.New("candidate approved file is not protected")
	}
	raw, err := r.podman(ctx, nil, "exec", container, "/usr/bin/head", "-c", strconv.Itoa(limit+1), "--", filename)
	if err != nil || len(raw) == 0 || len(raw) > limit {
		return nil, errors.New("candidate approved file is unavailable or exceeds bounds")
	}
	return raw, nil
}

func (r *Runtime) candidateApprovedFiles(ctx context.Context, container string, in domain.FactoryCandidate, sourceCommit string) (map[string][]byte, error) {
	dir := path.Join(domain.FactoryPreparationsDir, in.SourcePreparation)
	snapshot := path.Join(dir, "snapshot")
	meta, err := r.podman(ctx, nil, "exec", container, "/usr/bin/stat", "-c", "%u:%g:%a", "--", domain.FactoryDir, domain.FactoryPreparationsDir, dir, snapshot)
	if err != nil || string(meta) != strings.Repeat("0:0:755\n", 4) {
		return nil, errors.New("candidate approved snapshot is not protected")
	}
	raw, err := r.candidateProtectedFile(ctx, container, path.Join(dir, "request.json"), 4096)
	if err != nil {
		return nil, err
	}
	var saved struct {
		ID           string `json:"id"`
		Role         string `json:"role"`
		SetupDigest  string `json:"setup_digest"`
		SourceCommit string `json:"source_commit"`
		Credential   string `json:"credential"`
	}
	if strictjson.Decode(bytes.NewReader(raw), &saved) != nil || saved.ID != in.SourcePreparation || saved.Role != in.Preparation.Role ||
		saved.SetupDigest != in.Preparation.SetupDigest || saved.SourceCommit != sourceCommit || saved.Credential != in.Preparation.Credential {
		return nil, errors.New("candidate approved snapshot differs from its recorded inputs")
	}
	listing, err := r.podman(ctx, nil, "exec", container, "/usr/bin/ls", "-1A", "--", snapshot)
	if err != nil || len(listing) > 1024 {
		return nil, errors.New("candidate approved snapshot is unavailable")
	}
	files := map[string][]byte{}
	for name := range strings.SplitSeq(strings.TrimSpace(string(listing)), "\n") {
		if name == "source.bundle" {
			continue
		}
		if !domain.ValidApprovedName(name) || len(files) >= domain.MaxApprovedFiles {
			return nil, errors.New("candidate approved snapshot has an invalid file set")
		}
		files[name], err = r.candidateProtectedFile(ctx, container, path.Join(snapshot, name), domain.MaxApprovedFileSize)
		if err != nil {
			return nil, err
		}
	}
	if domain.SetupDigestOf(files) != in.Preparation.SetupDigest {
		return nil, errors.New("candidate approved snapshot differs from its digest")
	}
	return files, nil
}

// InspectCandidate resolves role and checkout from the settled host receipt.
// An old run never reads a replacement Project container.
func (f *Factory) InspectCandidate(ctx context.Context, in domain.FactoryCandidateInspect) (domain.FactoryCandidateState, error) {
	var empty domain.FactoryCandidateState
	if err := in.Validate(); err != nil {
		return empty, err
	}
	file, err := f.lockRun(ctx, in.Project, in.ID)
	if err != nil {
		return empty, err
	}
	receipt, exists, err := f.loadReceipt(in.Project, in.ID)
	_ = file.Close()
	if err != nil {
		return empty, err
	}
	if !exists {
		return empty, identity.ErrNotFound
	}
	if !receiptTerminal(receipt.Phase) || receipt.Run.Validate() != nil || receipt.Binding == nil {
		return empty, errors.New("candidate inspection requires a settled run")
	}
	runtime := &Runtime{Exec: f.terminal.Exec}
	container, err := runtime.prepareContainer(ctx, in.Project, true)
	if err != nil || container != receipt.Binding.Project {
		return empty, identity.ErrStale
	}
	checkout, _, _, _ := domain.FactoryRunPaths(receipt.Run.Role, receipt.Run.Preparation, receipt.Run.ID)
	out, err := runtime.podman(ctx, nil, "exec", "--user", receipt.Run.Role, container,
		"/usr/bin/env", "-i", "PATH=/usr/bin:/bin", "HOME=/home/"+receipt.Run.Role, "LC_ALL=C",
		"TMPDIR=/home/"+receipt.Run.Role+"/checkouts", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_NO_REPLACE_OBJECTS=1", "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0",
		"/usr/bin/sh", "-c", candidateInspectScript, "soda-candidate", checkout)
	if err != nil || len(out) > 64 {
		return empty, errors.New("candidate checkout inspection unconfirmed")
	}
	fields := strings.Fields(string(out))
	if len(fields) != 2 || !domain.ValidCommit(fields[0]) || (fields[1] != "clean" && fields[1] != "dirty") {
		return empty, errors.New("candidate checkout observation is invalid")
	}
	return domain.FactoryCandidateState{ID: in.ID, Project: in.Project, Container: container, Candidate: fields[0], Dirty: fields[1] == "dirty"}, nil
}

// Only rev-parse reads source metadata; it executes no filters or hooks. Every
// worktree/index comparison uses a clean repository, source objects and a
// disposable index. No agent configuration or optional external diff runs.
const candidateInspectScript = `set -eu
umask 077
src=$1
head=$(/usr/bin/git --git-dir="$src/.git" rev-parse --verify 'HEAD^{commit}')
dir=$(/usr/bin/mktemp -d "$TMPDIR/.soda-candidate-XXXXXX")
trap '/usr/bin/rm -rf "$dir"' EXIT HUP INT TERM
/usr/bin/git -c core.hooksPath=/dev/null init --bare --template= "$dir/repo.git" >/dev/null 2>/dev/null
export GIT_OBJECT_DIRECTORY="$src/.git/objects"
git_candidate() {
  /usr/bin/git -c core.hooksPath=/dev/null -c core.fsmonitor=false --git-dir="$dir/repo.git" --work-tree="$src" "$@"
}
dirty=clean
export GIT_INDEX_FILE="$src/.git/index"
if git_candidate diff-index --cached --quiet --no-ext-diff --no-textconv "$head" --; then :; else
  code=$?; [ "$code" = 1 ] || exit "$code"; dirty=dirty
fi
export GIT_INDEX_FILE="$dir/index"
git_candidate read-tree "$head"
if git_candidate update-index --refresh >/dev/null; then :; else
  code=$?; [ "$code" = 1 ] || exit "$code"
fi
if git_candidate diff-files --quiet --no-ext-diff --no-textconv --; then :; else
  code=$?; [ "$code" = 1 ] || exit "$code"; dirty=dirty
fi
# Standard exclusions read worktree .gitignore with this clean repository's
# empty info/exclude and no global configuration. A new visible .gitignore must
# itself remain visible, even if its rules try to ignore that file. Already
# ignored build directories stay excluded without traversing their contents.
git_candidate ls-files --others --exclude-standard --exclude='!**/.gitignore' --exclude=.git/ --exclude=.soda-home/ >"$dir/untracked"
[ ! -s "$dir/untracked" ] || dirty=dirty
printf '%s %s\n' "$head" "$dirty"
`
