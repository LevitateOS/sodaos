package project

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	domain "github.com/levitateos/sodaos/internal/project"
)

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

// osReleaseLines splits like Python splitlines: blank lines are skipped by the
// caller either way, so separator runs collapse without changing the result.
func osReleaseLines(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case '\n', '\r', '\v', '\f', '\u001c', '\u001d', '\u001e', '\u0085', '\u2028', '\u2029':
			return true
		}
		return false
	})
}

// parseOSRelease reads the first 4096 bytes of an os-release file. Values
// keep their literal bytes except one pair of surrounding single or double
// quotes; there is no escape processing. Only ID, VERSION_ID and PRETTY_NAME
// are kept; a repeated or malformed kept field is ambiguous.
func parseOSRelease(raw []byte) (domain.OSRelease, error) {
	var out domain.OSRelease
	if len(raw) > 4096 {
		return out, errors.New("oversized OS release file")
	}
	if !utf8.Valid(raw) {
		return out, errors.New("invalid OS release encoding")
	}
	values := map[string]string{}
	for _, line := range osReleaseLines(string(raw)) {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimLeftFunc(line, unicode.IsSpace), "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if key != "ID" && key != "VERSION_ID" && key != "PRETTY_NAME" {
			continue
		}
		if !found {
			return out, errors.New("ambiguous OS release field")
		}
		if _, seen := values[key]; seen {
			return out, errors.New("ambiguous OS release field")
		}
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[0] == value[len(value)-1] {
			value = value[1 : len(value)-1]
		}
		values[key] = value
	}
	id, ok := values["ID"]
	if !ok {
		return out, errors.New("missing OS identity")
	}
	version, ok := values["VERSION_ID"]
	if !ok {
		return out, errors.New("missing OS version")
	}
	name, ok := values["PRETTY_NAME"]
	if !ok {
		name = id
	}
	return domain.OSRelease{ID: id, Version: version, Name: name}, nil
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
	if _, err := r.podman(ctx, nil, "exec", "soda-"+id, "/usr/bin/test", "-f", "/etc/os-release"); err != nil {
		return result, nil
	}
	raw, err := r.podman(ctx, nil, "exec", "soda-"+id, "/usr/bin/head", "-c", "4097", "/etc/os-release")
	if err != nil {
		return result, nil
	}
	release, err := parseOSRelease(raw)
	if err != nil || !validOSRelease(release) {
		return result, nil
	}
	result.Release = &release
	result.Unavailable = false
	return result, nil
}
