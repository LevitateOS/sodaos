package host

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/installlayout"
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

func validLifecycleAction(action string) bool {
	return action == "inspect" || action == "start" || action == "stop"
}

func parseUnitShowProperties(b []byte) (map[string]string, error) {
	if len(b) > 4096 {
		return nil, errors.New("native unit unavailable")
	}
	fields := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, errors.New("invalid native unit observation")
		}
		if _, exists := fields[key]; exists {
			return nil, errors.New("ambiguous native unit")
		}
		switch key {
		case "LoadState", "FragmentPath", "DropInPaths", "UnitFileState":
			fields[key] = value
		default:
			return nil, errors.New("unexpected unit property")
		}
	}
	return fields, nil
}

func validateUnitProperties(fields map[string]string) (bool, error) {
	dropIns := fields["DropInPaths"]
	if len(fields) != 4 || fields["LoadState"] != "loaded" || fields["FragmentPath"] != installlayout.ProjectUnit {
		return false, errors.New("native unit is not the selected project unit")
	}
	if dropIns != "" && dropIns != "/usr/lib/systemd/system/service.d/10-timeout-abort.conf" {
		return false, errors.New("native unit is not the selected project unit")
	}
	if fields["UnitFileState"] != "enabled" && fields["UnitFileState"] != "disabled" {
		return false, errors.New("native unit is not the selected project unit")
	}
	return fields["UnitFileState"] == "enabled", nil
}

func (d *Daemon) readProjectUnit(ctx context.Context, unit string) (bool, error) {
	b, err := d.Exec.Run(ctx, nil, "/usr/bin/systemctl", "show", unit, "--property=LoadState,FragmentPath,DropInPaths,UnitFileState")
	if err != nil {
		return false, errors.New("native unit unavailable")
	}
	fields, err := parseUnitShowProperties(b)
	if err != nil {
		return false, err
	}
	return validateUnitProperties(fields)
}

func (d *Daemon) applyLifecycleAction(ctx context.Context, action, unit string) error {
	if action == "inspect" {
		return nil
	}
	verb := "enable"
	if action == "stop" {
		verb = "disable"
	}
	// Stop also disables next-boot start. Start restores it. No persistent Soda
	// desired-state copy or direct Podman stop competing with systemd Restart.
	if _, err := d.Exec.Run(ctx, nil, "/usr/bin/systemctl", verb, "--now", unit); err != nil {
		return errors.New("native lifecycle outcome unconfirmed")
	}
	return nil
}

func verifyLifecycleOutcome(action string, result LifecycleState) error {
	if action == "start" && (!result.Environment.Running || !result.BootEnabled) {
		return errors.New("native lifecycle outcome unconfirmed")
	}
	if action == "stop" && (result.Environment.Running || result.BootEnabled) {
		return errors.New("native lifecycle outcome unconfirmed")
	}
	return nil
}

func (d *Daemon) lifecycle(ctx context.Context, in Lifecycle) (LifecycleState, error) {
	var result LifecycleState
	if !validLifecycleAction(in.Action) {
		return result, errors.New("invalid lifecycle operation")
	}
	cid, err := d.projectContainer(ctx, in.Project, false)
	if err != nil {
		return result, err
	}
	unit := "soda-project@" + in.Project + ".service"
	if _, err = d.readProjectUnit(ctx, unit); err != nil {
		return result, err
	}
	if err = d.applyLifecycleAction(ctx, in.Action, unit); err != nil {
		return result, err
	}
	after, err := d.projectContainer(ctx, in.Project, false)
	if err != nil || after != cid {
		return result, errors.New("project identity changed during operation")
	}
	result.BootEnabled, err = d.readProjectUnit(ctx, unit)
	if err != nil {
		return result, err
	}
	result.Environment, _, err = d.inspect(ctx, in.Project)
	if err != nil {
		return result, err
	}
	return result, verifyLifecycleOutcome(in.Action, result)
}

func validAccessKeysRequest(in AccessKeys) bool {
	if !loginName.MatchString(in.Login) || in.Login == "root" || in.Identity <= 0 {
		return false
	}
	if in.Apply {
		return keyRevision.MatchString(in.Revision)
	}
	return in.Revision == "" && len(in.Keys) == 0
}

func (d *Daemon) previewAccessKeys(ctx context.Context, in AccessKeys) error {
	if !in.Apply {
		return nil
	}
	preview := in
	preview.Apply = false
	preview.Keys = nil
	preview.Revision = ""
	observed, e := d.accessKeys(ctx, preview)
	if e != nil || observed.Revision != in.Revision {
		return errors.New("native keys changed or are not managed canonical keys")
	}
	return nil
}

func decodeAccessKeyState(data []byte) (AccessKeyState, error) {
	var out AccessKeyState
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

func (d *Daemon) accessKeys(ctx context.Context, in AccessKeys) (AccessKeyState, error) {
	var out AccessKeyState
	if !validAccessKeysRequest(in) {
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
	if err = d.previewAccessKeys(ctx, in); err != nil {
		return out, err
	}
	// Load the existing fixed identity validator as an in-memory Python module.
	// No project file installation/import search, new image or duplicated validator.
	program := "import sys,types\nm=types.ModuleType('project_terminal')\nexec(" + strconv.Quote(projectTerminal) + ",m.__dict__)\nsys.modules['project_terminal']=m\n" + projectKeys
	body, _ := json.Marshal(map[string]any{"login": in.Login, "identity": in.Identity, "apply": in.Apply, "revision": in.Revision, "keys": keys})
	data, err := d.podman(ctx, body, "--remote=false", "exec", "--interactive", cid, "/usr/bin/python3", "-I", "-c", program)
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
