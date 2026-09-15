package project

import (
	"context"
	_ "embed"
	"encoding/json"
	"regexp"
	"unicode"
	"unicode/utf8"

	domain "github.com/levitateos/sodaos/internal/project"
)

//go:embed project_os.py
var projectOSProgram string

var (
	osID      = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	osVersion = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)
)

func validOSRelease(p domain.OSRelease) bool {
	if !osID.MatchString(p.ID) || !osVersion.MatchString(p.Version) || p.Name == "" || len(p.Name) > 256 || !utf8.ValidString(p.Name) {
		return false
	}
	for _, r := range p.Name {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	return true
}

func (r *Runtime) ObserveOS(ctx context.Context, id string) (domain.OSObservation, error) {
	env, _, err := r.Inspect(ctx, id)
	result := domain.OSObservation{Environment: env, Unavailable: true}
	if err != nil {
		return result, err
	}
	if !env.Running {
		return result, nil
	} // Never start a stopped root to inspect a file.
	raw, err := r.podman(ctx, nil, "exec", "soda-"+id, "/usr/bin/python3", "-I", "-S", "-B", "-c", projectOSProgram)
	if err != nil || len(raw) > 2048 {
		return result, nil
	}
	var release domain.OSRelease
	if json.Unmarshal(raw, &release) != nil || !validOSRelease(release) {
		return result, nil
	}
	result.Release = &release
	result.Unavailable = false
	return result, nil
}
