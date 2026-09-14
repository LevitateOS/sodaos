package nativequalification

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
	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/nativebuild"
)

var requiredChecks = []string{"b-install-media-free", "a-populated", "wrong-signer", "missing-content", "tampered-content", "interrupted-content", "locked-outside-window", "offline-window-activation", "b-state-preserved", "later-writes", "compatible-native-rollback", "both-generations-preserved"}

type Receipt struct {
	Scope, HostManifest, PayloadSHA256            string
	Checks                                        []string
	InstallCommit, UpdatedCommit, RecoveredCommit string
}

func (r Receipt) Validate(b Artifact, host string) error {
	if r.Scope != "native-install-upgrade-recovery" || r.HostManifest != host || r.PayloadSHA256 != b.Candidate.PayloadSHA256 || !reflect.DeepEqual(r.Checks, requiredChecks) || !nativebuild.Digest(r.InstallCommit) || !nativebuild.Digest(r.UpdatedCommit) || !nativebuild.Digest(r.RecoveredCommit) {
		return errors.New("incomplete protected qualification receipt")
	}
	return nil
}

type deployment struct {
	Booted, Staged bool
	Checksum       string
	Digest         string `json:"container-image-reference-digest"`
	Locked         bool   `json:"staged-finalization-locked"`
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
	return result.Stdout, errors.Join(err, result.Err)
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
func (m *machine) identity(ctx context.Context, a Artifact, configured bool) (string, error) {
	ds, err := m.status(ctx)
	if err != nil {
		return "", err
	}
	commit := ""
	for _, d := range ds {
		if d.Booted {
			if d.Digest != a.Candidate.Host.Manifest {
				return "", errors.New("booted host OCI differs")
			}
			commit = d.Checksum
		}
	}
	if !nativebuild.Digest(commit) {
		return "", errors.New("no exact native deployment")
	}
	raw, err := m.command(ctx, "payload", "cat /usr/share/soda/release.json", nil)
	if err != nil {
		return "", err
	}
	var p appliancerelease.Payload
	if err = json.Unmarshal(raw, &p); err != nil {
		return "", err
	}
	if !reflect.DeepEqual(p, a.Payload) {
		return "", errors.New("installed payload differs")
	}
	units := "soda-image-import.service forgejo.service soda-host.socket"
	if configured {
		units += " soda-dashboard.service soda-proxy.service"
	}
	ready, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	if err = poll(ready, func() (bool, error) {
		_, e := m.command(ready, "startup", "systemctl is-active --quiet "+units, nil)
		return e == nil, nil
	}); err != nil {
		return "", err
	}
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
	for _, name := range appliancerelease.Names {
		im := p.Images[name]
		script += "test \"$(podman image inspect --format '{{.Id}}' " + im.Config + ")\" = " + strings.TrimPrefix(im.Config, "sha256:") + "; "
	}
	if _, err = m.command(ctx, "native-content", "/var/tmp/soda-p9-driver --qualification-state content --expected-payload "+a.Payload.ID, nil); err != nil {
		return "", err
	}
	if _, err = m.command(ctx, "native-health", script, nil); err != nil {
		return "", err
	}
	if _, err = m.command(ctx, "native-resources", "free -b; df -B1 /sysroot /var; lsblk -b -o NAME,SIZE,TYPE", nil); err != nil {
		return "", err
	}
	return commit, nil
}
func (m *machine) driver(ctx context.Context, executable string) error {
	b, err := os.ReadFile(executable)
	if err != nil {
		return err
	}
	_, err = m.command(ctx, "admit-driver", "set -eu; umask 077; test ! -e /var/tmp/soda-p9-driver; cat > /var/tmp/soda-p9-driver; chmod 700 /var/tmp/soda-p9-driver", b)
	return err
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
func laterState(before, after map[string]any) error {
	for _, key := range []string{"project", "forgejo_repository", "machine_settings_public_keys", "schema"} {
		if !reflect.DeepEqual(before[key], after[key]) {
			return fmt.Errorf("later write changed protected fixture %s", key)
		}
	}
	a, ok := before["project_files"].(map[string]any)
	if !ok {
		return errors.New("missing initial project hashes")
	}
	b, ok := after["project_files"].(map[string]any)
	if !ok || len(b) != 2 || b["a.txt"] != a["a.txt"] || b["b.txt"] == nil {
		return errors.New("project generations not preserved")
	}
	if reflect.DeepEqual(before["forgejo_ref"], after["forgejo_ref"]) || reflect.DeepEqual(before["user"], after["user"]) {
		return errors.New("later Git/database writes not observed")
	}
	return nil
}
func (m *machine) put(ctx context.Context, name, path string, b []byte) error {
	_, err := m.command(ctx, name, "set -eu; umask 022; cat > "+path+"; chmod 644 "+path, b)
	return err
}
func newMachine(ctx context.Context, c Config, work, disk, variables, iso string, port int, key, known string, e *acceptance.Evidence) (*machine, error) {
	if err := os.Mkdir(work, 0700); err != nil {
		return nil, err
	}
	r := acceptance.Remote{Host: "127.0.0.1", Port: port, User: "root", Key: key, KnownHosts: known}
	cfg := acceptance.VMConfig{Name: "soda-native-p9-" + filepath.Base(work), Architecture: "x86_64", QEMU: c.QEMU, Firmware: c.Firmware, Work: work, SSH: r}
	vm, err := acceptance.LaunchDiskVM(ctx, cfg, disk, variables, iso, e)
	return &machine{remote: r, vm: vm, evidence: e}, err
}
