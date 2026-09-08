package host

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

type Lifecycle struct {
	Project string `json:"project"`
	Action  string `json:"action"`
}
type LifecycleState struct {
	Environment Environment `json:"environment"`
	BootEnabled bool        `json:"boot_enabled"`
}
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

//go:embed project_keys.py
var projectKeys string

func (c *Client) Lifecycle(ctx context.Context, in Lifecycle) (LifecycleState, error) {
	var out LifecycleState
	err := c.call(ctx, "/lifecycle", in, &out)
	if err == nil && (out.Environment.ID != in.Project || (out.Environment.IP != "" && !validAddress(out.Environment.IP)) || (in.Action == "start" && (!out.Environment.Running || !out.BootEnabled)) || (in.Action == "stop" && (out.Environment.Running || out.BootEnabled))) {
		err = errors.New("native lifecycle outcome not confirmed")
	}
	return out, err
}
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

func (d *Daemon) lifecycle(ctx context.Context, in Lifecycle) (LifecycleState, error) {
	var result LifecycleState
	if in.Action != "inspect" && in.Action != "start" && in.Action != "stop" {
		return result, errors.New("invalid lifecycle operation")
	}
	cid, err := d.projectContainer(ctx, in.Project, false)
	if err != nil {
		return result, err
	}
	unit := "soda-project@" + in.Project + ".service"
	readUnit := func() (bool, error) {
		b, e := d.Exec.Run(ctx, nil, "/usr/bin/systemctl", "show", unit, "--property=LoadState,FragmentPath,DropInPaths,UnitFileState")
		if e != nil || len(b) > 4096 {
			return false, errors.New("native unit unavailable")
		}
		fields := map[string]string{}
		for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				return false, errors.New("invalid native unit observation")
			}
			if _, exists := fields[key]; exists {
				return false, errors.New("ambiguous native unit")
			}
			if key != "LoadState" && key != "FragmentPath" && key != "DropInPaths" && key != "UnitFileState" {
				return false, errors.New("unexpected unit property")
			}
			fields[key] = value
		}
		// Fedora's stock systemd package applies this global stop-timeout policy
		// to every service. Like FragmentPath, this trusts installed host-root
		// configuration, not arbitrary per-project overrides or caller paths.
		dropIns := fields["DropInPaths"]
		if len(fields) != 4 || fields["LoadState"] != "loaded" || fields["FragmentPath"] != "/etc/systemd/system/soda-project@.service" || (dropIns != "" && dropIns != "/usr/lib/systemd/system/service.d/10-timeout-abort.conf") || (fields["UnitFileState"] != "enabled" && fields["UnitFileState"] != "disabled") {
			return false, errors.New("native unit is not the selected project unit")
		}
		return fields["UnitFileState"] == "enabled", nil
	}
	if _, err = readUnit(); err != nil {
		return result, err
	}
	if in.Action != "inspect" {
		verb := "enable"
		if in.Action == "stop" {
			verb = "disable"
		}
		// Stop also disables next-boot start. Start restores it. No persistent Soda
		// desired-state copy or direct Podman stop competing with systemd Restart.
		if _, err = d.Exec.Run(ctx, nil, "/usr/bin/systemctl", verb, "--now", unit); err != nil {
			return result, errors.New("native lifecycle outcome unconfirmed")
		}
	}
	after, err := d.projectContainer(ctx, in.Project, false)
	if err != nil || after != cid {
		return result, errors.New("project identity changed during operation")
	}
	result.BootEnabled, err = readUnit()
	if err != nil {
		return result, err
	}
	result.Environment, _, err = d.inspect(ctx, in.Project)
	if err == nil && ((in.Action == "start" && (!result.Environment.Running || !result.BootEnabled)) || (in.Action == "stop" && (result.Environment.Running || result.BootEnabled))) {
		err = errors.New("native lifecycle outcome unconfirmed")
	}
	return result, err
}

func (d *Daemon) accessKeys(ctx context.Context, in AccessKeys) (AccessKeyState, error) {
	var out AccessKeyState
	if !loginName.MatchString(in.Login) || in.Login == "root" || in.Identity <= 0 || (in.Apply && !keyRevision.MatchString(in.Revision)) || (!in.Apply && (in.Revision != "" || len(in.Keys) != 0)) {
		return out, errors.New("invalid own-account key operation")
	}
	keys, err := canonicalKeys(in.Keys)
	if err != nil {
		return out, err
	}
	cid, err := d.terminalContainer(ctx, in.Project)
	if err != nil {
		return out, err
	}
	// Load the existing fixed identity validator as an in-memory Python module.
	// No project file installation/import search, new image or duplicated validator.
	program := "import sys,types\nm=types.ModuleType('project_terminal')\nexec(" + strconv.Quote(projectTerminal) + ",m.__dict__)\nsys.modules['project_terminal']=m\n" + projectKeys
	if in.Apply {
		preview := in
		preview.Apply = false
		preview.Keys = nil
		preview.Revision = ""
		observed, e := d.accessKeys(ctx, preview)
		if e != nil || observed.Revision != in.Revision {
			return out, errors.New("native keys changed or are not managed canonical keys")
		}
	}
	body, _ := json.Marshal(map[string]any{"login": in.Login, "identity": in.Identity, "apply": in.Apply, "revision": in.Revision, "keys": keys})
	data, err := d.podman(ctx, body, "--remote=false", "exec", "--interactive", cid, "/usr/bin/python3", "-I", "-c", program)
	if err != nil || len(data) > 65536 {
		return out, errors.New("native key operation not confirmed")
	}
	if json.Unmarshal(data, &out) != nil || !keyRevision.MatchString(out.Revision) || out.Keys == nil {
		return out, errors.New("invalid native key observation")
	}
	if _, err = canonicalKeys(out.Keys); err != nil {
		return out, err
	}
	if in.Apply && strings.Join(out.Keys, "\n") != strings.Join(keys, "\n") {
		return out, errors.New("native key result differs from requested set")
	}
	return out, nil
}
