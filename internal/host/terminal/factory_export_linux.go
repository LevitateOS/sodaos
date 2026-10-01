//go:build linux

package terminal

import (
	"context"
	"errors"
	"strconv"

	"github.com/levitateos/sodaos/internal/identity"
	domain "github.com/levitateos/sodaos/internal/project"
)

// The source contributes objects only. A clean bare repository supplies
// configuration and HEAD, so export neither trusts agent Git configuration
// nor changes the agent's refs. Positional arguments are validated paths,
// the exact candidate and the maximum output length.
const factoryExportScript = `set -eu
src=$1
candidate=$2
limit=$3
if ! /usr/bin/test -d "$src/.git/objects"; then
  printf 'soda-export-missing\n'
  exit 0
fi
dir=$(/usr/bin/mktemp -d "$TMPDIR/.soda-export-XXXXXX")
trap '/usr/bin/rm -rf "$dir"' EXIT HUP INT TERM
/usr/bin/git -c core.hooksPath=/dev/null init --bare --template= "$dir/repo.git" >/dev/null 2>/dev/null
export GIT_OBJECT_DIRECTORY="$src/.git/objects"
git_export() {
  /usr/bin/git -c core.hooksPath=/dev/null --git-dir="$dir/repo.git" "$@"
}
if ! actual=$(git_export rev-parse --verify "$candidate^{commit}" 2>/dev/null); then
  printf 'soda-export-invalid\n'
  exit 0
fi
if [ "$actual" != "$candidate" ]; then
  printf 'soda-export-invalid\n'
  exit 0
fi
git_export update-ref HEAD "$candidate" 2>/dev/null
git_export bundle create "$dir/candidate.bundle" HEAD 2>/dev/null
/usr/bin/head -c "$limit" "$dir/candidate.bundle"
`

// FactoryExportBundle exports the exact candidate from the recorded role
// checkout in the same running container incarnation. Git runs as the
// role, with a clean environment and private temporary storage under its
// home. Only a bounded HEAD bundle crosses the daemon boundary.
func (s *Service) FactoryExportBundle(ctx context.Context, projectID, recorded, role, preparation, candidate string) ([]byte, error) {
	src := "/home/" + role + "/checkouts/" + preparation
	if !domain.TakeoverSource(src, role, preparation) || !domain.ValidCommit(candidate) || !containerID.MatchString(recorded) {
		return nil, identity.ErrDenied
	}
	current, err := s.factoryProjectContainer(ctx, projectID, true)
	if err != nil || current != recorded {
		return nil, identity.ErrStale
	}
	bundle, err := s.podman(ctx, nil, "--remote=false", "exec", "--user", role, current,
		"/usr/bin/env", "-i", "PATH=/usr/bin:/bin", "HOME=/home/"+role, "LC_ALL=C",
		"TMPDIR=/home/"+role+"/checkouts", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_NO_REPLACE_OBJECTS=1", "GIT_TERMINAL_PROMPT=0",
		"/usr/bin/sh", "-c", factoryExportScript, "soda-export", src, candidate,
		strconv.Itoa(domain.MaxFactoryExportBundle+1))
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("candidate export execution unconfirmed")
	}
	switch string(bundle) {
	case "soda-export-missing\n":
		return nil, identity.ErrNotFound
	case "soda-export-invalid\n":
		return nil, domain.ErrFactoryExportCandidate
	}
	if len(bundle) == 0 || len(bundle) > domain.MaxFactoryExportBundle {
		return nil, domain.ErrFactoryExportBounds
	}
	return bundle, nil
}
