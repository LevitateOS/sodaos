package qualify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/acceptance"
	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
)

type Receipt struct {
	Scope, HostManifest, PayloadSHA256, BaselineHostManifest      string
	BaselineCommit, InstallCommit, UpdatedCommit, RecoveredCommit string
}

func receiptBindingsMatch(r Receipt, a, b Artifact, host string) bool {
	return r.Scope == "native-install-upgrade-recovery" && r.HostManifest == host && r.HostManifest == b.Candidate.Host.Manifest && r.BaselineHostManifest == a.Candidate.Host.Manifest && r.PayloadSHA256 == b.Candidate.PayloadSHA256 && r.BaselineCommit == r.RecoveredCommit
}

func receiptDigests(r Receipt) bool {
	return build.Digest(r.BaselineCommit) && build.Digest(r.InstallCommit) && build.Digest(r.UpdatedCommit)
}

func (r Receipt) Validate(a, b Artifact, host string) error {
	if !receiptBindingsMatch(r, a, b, host) || !receiptDigests(r) {
		return errors.New("incomplete protected qualification receipt")
	}
	return nil
}

type deployment struct {
	Booted, Staged   bool
	Checksum, Osname string
	Serial           int
	Digest           string `json:"container-image-reference-digest"`
}
type machine struct {
	remote   acceptance.Remote
	vm       *acceptance.VM
	evidence *acceptance.Evidence
	sequence int
}

func (m *machine) command(ctx context.Context, label, command string, input []byte) ([]byte, error) {
	m.sequence++
	args, err := m.remote.Args()
	if err != nil {
		return nil, err
	}
	result, err := acceptance.Execute(ctx, m.evidence, fmt.Sprintf("%03d-%s", m.sequence, label), acceptance.Command{Name: "ssh", Args: append(args, "--", command), Stdin: bytes.NewReader(input)})
	if err = errors.Join(err, result.Err); err != nil {
		return result.Stdout, fmt.Errorf("%s: %w", label, err)
	}
	return result.Stdout, nil
}

func (m *machine) status(ctx context.Context) ([]deployment, error) {
	b, err := m.command(ctx, "status", "rpm-ostree status --json", nil)
	if err != nil {
		return nil, err
	}
	var s struct{ Deployments []deployment }
	err = json.Unmarshal(b, &s)
	return s.Deployments, err
}

func bootedCommit(ds []deployment, manifest string) (string, error) {
	commit := ""
	for _, d := range ds {
		if d.Booted {
			if d.Digest != manifest {
				return "", errors.New("booted host OCI differs")
			}
			commit = d.Checksum
		}
	}
	if !build.Digest(commit) {
		return "", errors.New("no exact native deployment")
	}
	return commit, nil
}

func loadGuestPayload(raw []byte, want deliver.Payload) (deliver.Payload, error) {
	var p deliver.Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, err
	}
	if !reflect.DeepEqual(p, want) {
		return p, errors.New("installed payload differs")
	}
	return p, nil
}

func startupProbe(configured bool) string {
	units := "soda-image-import.service forgejo.service soda-host.socket"
	if configured {
		units += " soda-dashboard.service soda-proxy.service"
	}
	// Fresh install keeps Forgejo unlocked; /api/v1/version 404s until fixture setup.
	startup := "systemctl is-active --quiet " + units
	if configured {
		startup += " && curl --silent --show-error --fail --max-time 2 http://127.0.0.1:3000/api/v1/version"
	}
	return startup
}

func healthScript(p deliver.Payload, configured bool) string {
	script := "set -eu; systemctl is-active soda-image-import.service forgejo.service soda-host.socket; test \"$(getenforce)\" = Enforcing; test \"$(sshd -T | grep '^passwordauthentication ')\" = 'passwordauthentication no'; test \"$(sshd -T | grep '^permitrootlogin ')\" = 'permitrootlogin prohibit-password'; "
	if configured {
		script += "systemctl is-active soda-dashboard.service soda-proxy.service; "
	}
	services := []string{"forgejo"}
	if configured {
		services = append(services, "dashboard", "proxy")
	}
	for _, name := range services {
		script += "test \"$(podman inspect --format '{{.Image}}' soda-" + name + ")\" = " + strings.TrimPrefix(p.Images[name].Config, "sha256:") + "; "
	}
	for _, name := range deliver.Names {
		im := p.Images[name]
		script += "test \"$(podman image inspect --format '{{.Id}}' " + im.Config + ")\" = " + strings.TrimPrefix(im.Config, "sha256:") + "; "
	}
	return script
}

func (m *machine) waitStartup(ctx context.Context, startup string) error {
	ready, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	return poll(ready, func() (bool, error) {
		_, e := m.command(ready, "startup", startup, nil)
		return e == nil, nil
	})
}

func (m *machine) verifyNativeContent(ctx context.Context, a Artifact, p deliver.Payload, configured bool) error {
	if _, err := m.command(ctx, "native-content", "/var/tmp/soda-p9-driver --qualification-state content --expected-payload "+a.Payload.ID, nil); err != nil {
		return err
	}
	if _, err := m.command(ctx, "native-health", healthScript(p, configured), nil); err != nil {
		return err
	}
	_, err := m.command(ctx, "native-resources", "free -b; df -B1 /sysroot /var; lsblk -b -o NAME,SIZE,TYPE", nil)
	return err
}

func (m *machine) identity(ctx context.Context, a Artifact, configured bool) (string, error) {
	ds, err := m.status(ctx)
	if err != nil {
		return "", err
	}
	commit, err := bootedCommit(ds, a.Candidate.Host.Manifest)
	if err != nil {
		return "", err
	}
	raw, err := m.command(ctx, "payload", "cat /usr/share/soda/release.json", nil)
	if err != nil {
		return "", err
	}
	p, err := loadGuestPayload(raw, a.Payload)
	if err != nil {
		return "", err
	}
	if err = m.waitStartup(ctx, startupProbe(configured)); err != nil {
		return "", err
	}
	return commit, m.verifyNativeContent(ctx, a, p, configured)
}

func (m *machine) driver(ctx context.Context, executable string) error {
	b, err := os.ReadFile(executable)
	if err != nil {
		return err
	}
	result, err := m.command(ctx, "admit-driver", "set -eu; umask 077; test ! -e /var/tmp/soda-p9-driver; cat > /var/tmp/soda-p9-driver; chmod 700 /var/tmp/soda-p9-driver; sha256sum /var/tmp/soda-p9-driver", b)
	if err != nil {
		return err
	}
	hash, err := build.HashFile(executable)
	if err != nil || !strings.HasPrefix(string(result), hash+" ") {
		return errors.New("guest driver readback differs")
	}
	return nil
}

// Native rpm-ostree 2026.2 JSON does not expose this lock. libostree's CLI does.
func finalizationLocked(text string, d deployment) bool {
	wanted := fmt.Sprintf("%s %s.%d (finalization locked)", d.Osname, d.Checksum, d.Serial)
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == wanted {
			return true
		}
	}
	return false
}

func (m *machine) locked(ctx context.Context, d deployment) error {
	text, err := m.command(ctx, "native-finalization-lock", "LC_ALL=C ostree admin status", nil)
	if err != nil {
		return err
	}
	if !finalizationLocked(string(text), d) {
		return errors.New("exact native finalization lock absent")
	}
	return nil
}

func (m *machine) state(ctx context.Context, action, payload string) (map[string]any, error) {
	if action != "snapshot" && action != "later" {
		return nil, errors.New("baseline reseeding refused")
	}
	raw, err := m.command(ctx, "state-"+action, "/var/tmp/soda-p9-driver --qualification-state "+action+" --expected-payload "+payload, nil)
	if err != nil {
		return nil, err
	}
	var s map[string]any
	err = json.Unmarshal(raw, &s)
	return s, err
}

func sameState(want, got map[string]any) error {
	if !reflect.DeepEqual(want, got) {
		return errors.New("persistent fixture state differs")
	}
	return nil
}

func laterProtectedUnchanged(before, after map[string]any) error {
	for _, key := range []string{"project", "forgejo_repository", "machine_settings_public_keys", "schema"} {
		if !reflect.DeepEqual(before[key], after[key]) {
			return fmt.Errorf("later write changed protected fixture %s", key)
		}
	}
	return nil
}

func laterFilesPreserved(before, after map[string]any) error {
	a, ok := before["project_files"].(map[string]any)
	if !ok {
		return errors.New("missing initial project hashes")
	}
	b, ok := after["project_files"].(map[string]any)
	if !ok || len(b) != 2 || b["a.txt"] != a["a.txt"] || b["b.txt"] == nil {
		return errors.New("project generations not preserved")
	}
	return nil
}

func laterWritesObserved(before, after map[string]any) bool {
	return !reflect.DeepEqual(before["forgejo_ref"], after["forgejo_ref"]) && !reflect.DeepEqual(before["user"], after["user"])
}

func laterState(before, after map[string]any) error {
	if err := laterProtectedUnchanged(before, after); err != nil {
		return err
	}
	if err := laterFilesPreserved(before, after); err != nil {
		return err
	}
	if !laterWritesObserved(before, after) {
		return errors.New("later Git/database writes not observed")
	}
	return nil
}

func (m *machine) put(ctx context.Context, name, path string, b []byte) error {
	_, err := m.command(ctx, name, "set -eu; umask 022; cat > "+path+"; chmod 644 "+path, b)
	return err
}

func newMachine(ctx context.Context, c Config, work, disk, variables, iso string, port int, key, known string, e *acceptance.Evidence) (*machine, error) {
	if err := os.Mkdir(work, 0o700); err != nil {
		return nil, err
	}
	r := acceptance.Remote{Host: "127.0.0.1", Port: port, User: "root", Key: key, KnownHosts: known}
	cfg := acceptance.VMConfig{Name: "soda-native-p9-" + filepath.Base(work), Architecture: "x86_64", QEMU: c.QEMU, Firmware: c.Firmware, Work: work, SSH: r}
	vm, err := acceptance.LaunchDiskVM(ctx, cfg, disk, variables, iso, e)
	return &machine{remote: r, vm: vm, evidence: e}, err
}
