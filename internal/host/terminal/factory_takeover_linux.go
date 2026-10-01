//go:build linux

package terminal

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/identity"
	domain "github.com/levitateos/sodaos/internal/project"
)

// FactoryTakeoverCopy copies one retired run's retained role-checkout work
// into the admitted member's own derived checkout destination. The recorded
// container incarnation must equal the current running Project container;
// a replaced container refuses instead of copying from the wrong
// incarnation. Role Git state and the private run homes holding
// credentials, prompts and output are excluded. All paths derive from
// validated identities; every step uses fixed argv.
func (s *Service) FactoryTakeoverCopy(ctx context.Context, projectID, recorded, role, preparation, member, run string) (string, bool, error) {
	dest := domain.TakeoverDestination(member, run)
	src := "/home/" + role + "/checkouts/" + preparation
	if dest == "" || !domain.TakeoverSource(src, role, preparation) || !containerID.MatchString(recorded) {
		return "", false, identity.ErrDenied
	}
	current, err := s.factoryProjectContainer(ctx, projectID, true)
	if err != nil || current != recorded {
		return "", false, identity.ErrStale
	}
	if _, err = s.podman(ctx, nil, "--remote=false", "exec", current, "/usr/bin/test", "-d", dest); err == nil {
		return dest, true, nil
	}
	partial := dest + ".partial"
	parent := "/home/" + member + "/" + domain.TakeoverDirName
	if _, err := s.podman(ctx, nil, "--remote=false", "exec", current, "/usr/bin/test", "-d", src); err != nil {
		return "", false, errors.New("takeover found no retained work")
	}
	steps := [][]string{
		{"/usr/bin/mkdir", "-p", parent},
		{"/usr/bin/rm", "-rf", partial},
		{"/usr/bin/mkdir", partial},
		{"/usr/bin/cp", "-a", src + "/.", partial + "/"},
		{"/usr/bin/rm", "-rf", partial + "/.git", partial + "/.soda-home"},
		{"/usr/bin/git", "-C", partial, "init", "-q"},
		{"/usr/bin/chown", "-R", "--reference=/home/" + member, partial},
	}
	for _, step := range steps {
		args := append([]string{"--remote=false", "exec", current}, step...)
		out, err := s.podman(ctx, nil, args...)
		if err != nil {
			return "", false, err
		}
		if len(out) > 65536 {
			return "", false, errors.New("takeover step returned excessive output")
		}
	}
	if _, err = s.podman(ctx, nil, "--remote=false", "exec", current, "/usr/bin/test", "-e", dest); err == nil {
		_, _ = s.podman(ctx, nil, "--remote=false", "exec", current, "/usr/bin/rm", "-rf", partial)
		return dest, true, nil
	}
	if _, err = s.podman(ctx, nil, "--remote=false", "exec", current, "/usr/bin/mv", "-T", partial, dest); err != nil {
		_, _ = s.podman(ctx, nil, "--remote=false", "exec", current, "/usr/bin/rm", "-rf", partial)
		return "", false, errors.New("takeover destination was not confirmed")
	}
	return dest, false, nil
}
