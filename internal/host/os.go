package host

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

//go:embed project_os.py
var projectOSProgram string

// OSRelease is an observation of the mutable root, never a creation profile.
type OSRelease struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Name    string `json:"name"`
}
type OSObservation struct {
	Environment Environment `json:"environment"`
	Release     *OSRelease  `json:"os_release"`
	Unavailable bool        `json:"os_release_unavailable"`
}

var osID = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
var osVersion = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

func validOSRelease(p OSRelease) bool {
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
func (d *Daemon) observeOS(ctx context.Context, id string) (OSObservation, error) {
	env, _, err := d.inspect(ctx, id)
	result := OSObservation{Environment: env, Unavailable: true}
	if err != nil {
		return result, err
	}
	if !env.Running {
		return result, nil
	} // Never start a stopped root to inspect a file.
	raw, err := d.podman(ctx, nil, "exec", "soda-"+id, "/usr/bin/python3", "-I", "-S", "-B", "-c", projectOSProgram)
	if err != nil || len(raw) > 2048 {
		return result, nil
	}
	var release OSRelease
	if json.Unmarshal(raw, &release) != nil || !validOSRelease(release) {
		return result, nil
	}
	result.Release = &release
	result.Unavailable = false
	return result, nil
}
func (c *Client) ObserveOS(ctx context.Context, id string) (OSObservation, error) {
	var out OSObservation
	if err := c.call(ctx, "/os", Create{ID: id}, &out); err != nil {
		return OSObservation{}, err
	}
	env := out.Environment
	if env.ID != id ||
		(env.Image != "" && (!strings.HasPrefix(env.Image, "sha256:") || !imageID.MatchString(env.Image))) ||
		(env.IP != "" && !validAddress(env.IP)) ||
		(env.Profile != nil && env.Profile.Validate() != nil) ||
		(out.Release == nil) != out.Unavailable ||
		(out.Release != nil && (!env.Running || !validOSRelease(*out.Release))) {
		return OSObservation{}, errors.New("invalid native OS observation")
	}
	return out, nil
}
