package host

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

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

var (
	osID      = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	osVersion = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)
)

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

func validOSEnvironment(env Environment) bool {
	if env.Image != "" && (!strings.HasPrefix(env.Image, "sha256:") || !imageID.MatchString(env.Image)) {
		return false
	}
	if env.IP != "" && !validAddress(env.IP) {
		return false
	}
	return env.Profile == nil || env.Profile.Validate() == nil
}

func validOSObservation(id string, out OSObservation) bool {
	if out.Environment.ID != id || (out.Release == nil) != out.Unavailable {
		return false
	}
	if !validOSEnvironment(out.Environment) {
		return false
	}
	if out.Release != nil && (!out.Environment.Running || !validOSRelease(*out.Release)) {
		return false
	}
	return true
}

func (c *Client) ObserveOS(ctx context.Context, id string) (OSObservation, error) {
	var out OSObservation
	if err := c.call(ctx, "/os", Create{ID: id}, &out); err != nil {
		return OSObservation{}, err
	}
	if !validOSObservation(id, out) {
		return OSObservation{}, errors.New("invalid native OS observation")
	}
	return out, nil
}
