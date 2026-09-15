package host

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"
)

type AccessKeys struct {
	Project  string   `json:"project"`
	Login    string   `json:"login"`
	Identity int64    `json:"identity"`
	Revision string   `json:"revision,omitempty"`
	Keys     []string `json:"keys,omitempty"`
	Apply    bool     `json:"apply"`
}
type AccessKeyState struct {
	Revision string   `json:"revision"`
	Keys     []string `json:"keys"`
}

var keyRevision = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (c *Client) AccessKeys(ctx context.Context, in AccessKeys) (AccessKeyState, error) {
	var out AccessKeyState
	err := c.call(ctx, "/access-keys", in, &out)
	if err == nil {
		_, err = canonicalKeys(out.Keys)
		if !keyRevision.MatchString(out.Revision) || out.Keys == nil {
			err = errors.New("invalid native key revision")
		}
		if in.Apply && strings.Join(out.Keys, "\n") != strings.Join(in.Keys, "\n") {
			err = errors.New("native key result differs from request")
		}
	}
	return out, err
}

func canonicalKeys(values []string) ([]string, error) {
	if len(values) > 32 {
		return nil, errors.New("too many development keys")
	}
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	size := 0
	for _, value := range values {
		key, _, options, rest, err := ssh.ParseAuthorizedKey([]byte(value))
		if err != nil || len(options) != 0 || len(strings.TrimSpace(string(rest))) != 0 {
			return nil, errors.New("invalid development key")
		}
		canonical := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
		if strings.TrimSpace(value) != canonical || seen[canonical] {
			return nil, errors.New("noncanonical or duplicate development key")
		}
		size += len(canonical)
		if size > 48000 {
			return nil, errors.New("development key set too large")
		}
		seen[canonical] = true
		out = append(out, canonical)
	}
	return out, nil
}
