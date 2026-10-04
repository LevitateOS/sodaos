package project

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/levitateos/sodaos/internal/host/terminal"
	domain "github.com/levitateos/sodaos/internal/project"
	"golang.org/x/crypto/ssh"
)

var keyRevision = regexp.MustCompile(`^[0-9a-f]{64}$`)

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

func validAccessKeysRequest(in domain.AccessKeys) bool {
	if !domain.ValidLogin(in.Login) || in.Login == "root" || in.Identity <= 0 {
		return false
	}
	if in.Apply {
		return keyRevision.MatchString(in.Revision)
	}
	return in.Revision == "" && len(in.Keys) == 0
}

func (r *Runtime) previewAccessKeys(ctx context.Context, in domain.AccessKeys) error {
	if !in.Apply {
		return nil
	}
	preview := in
	preview.Apply = false
	preview.Keys = nil
	preview.Revision = ""
	observed, e := r.AccessKeys(ctx, preview)
	if e != nil || observed.Revision != in.Revision {
		return errors.New("native keys changed or are not managed canonical keys")
	}
	return nil
}

func decodeAccessKeyState(data []byte) (domain.AccessKeyState, error) {
	var out domain.AccessKeyState
	if len(data) > 65536 {
		return out, errors.New("native key operation not confirmed")
	}
	if json.Unmarshal(data, &out) != nil || !keyRevision.MatchString(out.Revision) || out.Keys == nil {
		return out, errors.New("invalid native key observation")
	}
	if _, err := canonicalKeys(out.Keys); err != nil {
		return out, err
	}
	return out, nil
}

func (r *Runtime) AccessKeys(ctx context.Context, in domain.AccessKeys) (domain.AccessKeyState, error) {
	var out domain.AccessKeyState
	if !validAccessKeysRequest(in) {
		return out, errors.New("invalid own-account key operation")
	}
	keys, err := canonicalKeys(in.Keys)
	if err != nil {
		return out, err
	}
	cid, err := r.ProjectContainer(ctx, in.Project, true)
	if err != nil {
		return out, err
	}
	if err = r.previewAccessKeys(ctx, in); err != nil {
		return out, err
	}
	// The fixed agent owns key validation; the request travels only over stdin.
	body, _ := json.Marshal(map[string]any{"login": in.Login, "identity": in.Identity, "apply": in.Apply, "revision": in.Revision, "keys": keys})
	agent := terminal.AgentExec(cid, body, "keys")
	data, err := r.podman(ctx, body, agent.Args[1:]...)
	if err != nil {
		return out, errors.New("native key operation not confirmed")
	}
	out, err = decodeAccessKeyState(data)
	if err != nil {
		return out, err
	}
	if in.Apply && strings.Join(out.Keys, "\n") != strings.Join(keys, "\n") {
		return out, errors.New("native key result differs from requested set")
	}
	return out, nil
}
