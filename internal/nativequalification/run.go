package nativequalification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/acceptance"
	rd "github.com/levitateos/sodaos/internal/releasedelivery"
	"golang.org/x/sys/unix"
)

// Run is the fixed protected worker entry, not a build-supplied scenario runner.
func Run(ctx context.Context, configPath string) (err error) {
	u, err := user.Lookup("soda-qualifier")
	if err != nil {
		return err
	}
	if strconv.Itoa(os.Geteuid()) != u.Uid {
		return errors.New("exact qualifier identity required")
	}
	if configPath != "/run/soda-p9-input/config.json" {
		return errors.New("controller-mounted qualification inputs required")
	}
	if err = unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0); err != nil {
		return err
	}
	var c Config
	if err = rd.ReadJSON(configPath, &c); err != nil {
		return err
	}
	if c.Work != "/run/soda-p9-work" {
		return errors.New("controller work mount required")
	}
	var a Artifact
	if err = rd.ReadJSON("/run/soda-p9-input/baseline.json", &a); err != nil {
		return err
	}
	b, err := ReadArtifact("/run/soda-p9-input/candidate")
	if err != nil {
		return err
	}
	if err = SameBaseScenario(a, b); err != nil {
		return err
	}
	if b.Payload.RepositoryPrefix != "ghcr.io/levitateos/sodaos" {
		return errors.New("selected local scenario repository required")
	}
	media, err := ReadMedia("/run/soda-p9-input/candidate", b)
	if err != nil {
		return err
	}
	e, err := acceptance.CreateEvidence(filepath.Join(c.Work, "evidence"), nil)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, e.Close()) }()
	f, err := newFixture(ctx, c, a, b, media, e)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.close()) }()
	receipt := Receipt{Scope: "native-install-upgrade-recovery", HostManifest: b.Candidate.Host.Manifest, PayloadSHA256: b.Candidate.PayloadSHA256}
	mark := func(name string) error {
		if len(receipt.Checks) >= len(requiredChecks) || name != requiredChecks[len(receipt.Checks)] {
			return errors.New("qualification step order differs")
		}
		receipt.Checks = append(receipt.Checks, name)
		return e.WriteJSON(name+".json", map[string]string{"check": name, "host": b.Candidate.Host.Manifest})
	}
	receipt.InstallCommit, err = installB(ctx, c, b, media, f)
	if err != nil {
		return err
	}
	if err = mark("b-install-media-free"); err != nil {
		return err
	}
	// The only update target is the dispatcher-created copy of populated A.
	ae, err := acceptance.CreateEvidence(filepath.Join(c.Work, "update-evidence"), nil)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, ae.Close()) }()
	m, err := newMachine(ctx, c, filepath.Join(c.Work, "a-vm"), filepath.Join(c.Work, "a.qcow2"), filepath.Join(c.Work, "a-vars.fd"), "", 32294, filepath.Join(c.Work, "a-key"), filepath.Join(c.Work, "a-known-hosts"), ae)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, m.vm.Close()) }()
	ready, cancel := context.WithTimeout(ctx, 10*time.Minute)
	err = m.remote.WaitReady(ready)
	cancel()
	if err != nil {
		return err
	}
	if err = m.driver(ctx, c.Executable); err != nil {
		return err
	}
	original, err := m.identity(ctx, a, true)
	if err != nil {
		return err
	}
	before, err := m.state(ctx, "snapshot", a.Payload.ID)
	if err != nil {
		return err
	}
	var admitted map[string]any
	if err = rd.ReadJSON(filepath.Join(c.Work, "a-state.json"), &admitted); err != nil {
		return err
	}
	if err = sameState(admitted, before); err != nil {
		return err
	}
	if err = mark("a-populated"); err != nil {
		return err
	}
	if err = f.push(ctx, "wrong"); err != nil {
		return err
	}
	if err = configureGuest(ctx, m, f); err != nil {
		return err
	}
	if err = refusal(ctx, m, f, b, "wrong-signer", ""); err != nil {
		return err
	}
	if err = mark("wrong-signer"); err != nil {
		return err
	}
	if err = f.push(ctx, "correct"); err != nil {
		return err
	}
	for _, failure := range [][2]string{{"missing-content", "missing"}, {"tampered-content", "tamper"}, {"interrupted-content", "interrupt"}} {
		if err = refusal(ctx, m, f, b, failure[0], failure[1]); err != nil {
			return err
		}
		if err = mark(failure[0]); err != nil {
			return err
		}
	}
	f.mu.Lock()
	f.mode = ""
	f.offer = true
	f.mu.Unlock()
	if err = maintenance(ctx, m, 6*time.Hour); err != nil {
		return err
	}
	if _, err = m.command(ctx, "enable-native-agent", "systemctl start zincati.service", nil); err != nil {
		return err
	}
	wait, cancel := context.WithTimeout(ctx, 40*time.Minute)
	err = poll(wait, func() (bool, error) {
		ds, e := m.status(wait)
		if e != nil {
			return false, e
		}
		for _, d := range ds {
			if d.Booted && d.Digest != a.Candidate.Host.Manifest {
				return false, errors.New("premature reboot")
			}
			if d.Staged && d.Digest == b.Candidate.Host.Manifest {
				return d.Locked, nil
			}
		}
		return false, nil
	})
	cancel()
	if err != nil {
		return err
	}
	// Check the native lock remains held outside the configured maintenance window.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Minute):
	}
	ds, err := m.status(ctx)
	if err != nil {
		return err
	}
	locked := false
	for _, d := range ds {
		if d.Booted && d.Checksum != original {
			return errors.New("rebooted outside maintenance")
		}
		if d.Staged && d.Digest == b.Candidate.Host.Manifest && d.Locked {
			locked = true
		}
	}
	if !locked {
		return errors.New("staged finalization lock not retained")
	}
	if err = mark("locked-outside-window"); err != nil {
		return err
	}
	if err = f.registry.Close(); err != nil {
		return err
	}
	f.registry = nil
	if _, err = f.run(ctx, "content-offline", "podman", "--remote=false", "stop", "--time=10", f.cid); err != nil {
		return err
	}
	f.cid = ""
	if err = maintenance(ctx, m, 2*time.Minute); err != nil {
		return err
	}
	if _, err = m.command(ctx, "reload-maintenance", "systemctl restart zincati.service", nil); err != nil {
		return err
	}
	wait, cancel = context.WithTimeout(ctx, 15*time.Minute)
	err = poll(wait, func() (bool, error) {
		ds, e := m.status(wait)
		if e != nil {
			return false, nil
		}
		for _, d := range ds {
			if d.Booted && d.Digest == b.Candidate.Host.Manifest {
				return true, nil
			}
		}
		return false, nil
	})
	cancel()
	if err != nil {
		return err
	}
	receipt.UpdatedCommit, err = m.identity(ctx, b, true)
	if err != nil {
		return err
	}
	if err = mark("offline-window-activation"); err != nil {
		return err
	}
	current, err := m.state(ctx, "snapshot", b.Payload.ID)
	if err != nil {
		return err
	}
	if err = sameState(before, current); err != nil {
		return err
	}
	if err = mark("b-state-preserved"); err != nil {
		return err
	}
	later, err := m.state(ctx, "later", b.Payload.ID)
	if err != nil {
		return err
	}
	if err = laterState(before, later); err != nil {
		return err
	}
	if err = mark("later-writes"); err != nil {
		return err
	}
	// Suppress reoffer before the native, schema-compatible rollback; no DB restore.
	f.mu.Lock()
	f.offer = false
	f.mu.Unlock()
	if _, err = m.command(ctx, "rollback-native", "set -eu; systemctl stop zincati.service; rpm-ostree rollback", nil); err != nil {
		return err
	}
	if err = m.vm.Restart(ctx); err != nil {
		return err
	}
	ready, cancel = context.WithTimeout(ctx, 10*time.Minute)
	err = m.remote.WaitReady(ready)
	cancel()
	if err != nil {
		return err
	}
	receipt.RecoveredCommit, err = m.identity(ctx, a, true)
	if err != nil {
		return err
	}
	if receipt.RecoveredCommit != original {
		return errors.New("native rollback commit differs")
	}
	if err = mark("compatible-native-rollback"); err != nil {
		return err
	}
	recovered, err := m.state(ctx, "snapshot", a.Payload.ID)
	if err != nil {
		return err
	}
	if err = sameState(later, recovered); err != nil {
		return err
	}
	if err = mark("both-generations-preserved"); err != nil {
		return err
	}
	if err = receipt.Validate(b, media.HostManifest); err != nil {
		return err
	}
	return e.WriteJSON("qualification.json", receipt)
}
func poll(ctx context.Context, check func() (bool, error)) error {
	for {
		ok, err := check()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}
func configureGuest(ctx context.Context, m *machine, f *fixture) error {
	if _, err := m.command(ctx, "fixture-directories", "set -eu; systemctl stop zincati.service; mkdir -p /etc/soda-qualification /etc/containers/registries.d /etc/containers/registries.conf.d /etc/zincati/config.d", nil); err != nil {
		return err
	}
	ca, err := os.ReadFile(filepath.Join(f.work, "ca.crt"))
	if err != nil {
		return err
	}
	key, err := os.ReadFile(filepath.Join(f.work, "correct.pub"))
	if err != nil {
		return err
	}
	policy := map[string]any{"default": []any{map[string]string{"type": "reject"}}, "transports": map[string]any{"docker": map[string]any{"ghcr.io/levitateos/sodaos-host": []any{map[string]any{"type": "sigstoreSigned", "keyPath": "/etc/soda-qualification/key.pem", "signedIdentity": map[string]string{"type": "exactRepository", "dockerRepository": "ghcr.io/levitateos/sodaos-host"}}}}}}
	p, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	for _, file := range []struct {
		path string
		data []byte
	}{
		{"/etc/pki/ca-trust/source/anchors/soda-p9.crt", ca}, {"/etc/soda-qualification/key.pem", key}, {"/etc/containers/policy.json", p},
		{"/etc/containers/registries.conf.d/99-soda-p9.conf", []byte("[[registry]]\nprefix = \"ghcr.io/levitateos/sodaos-host\"\nlocation = \"10.0.2.2:19443/levitateos/sodaos-host\"\n")},
		{"/etc/containers/registries.d/soda-p9.yaml", []byte("docker:\n  ghcr.io/levitateos/sodaos-host:\n    use-sigstore-attachments: true\n")},
	} {
		if err = m.put(ctx, "fixture-trust", file.path, file.data); err != nil {
			return err
		}
	}
	_, err = m.command(ctx, "native-ca-trust", "update-ca-trust", nil)
	return err
}
func maintenance(ctx context.Context, m *machine, after time.Duration) error {
	raw, err := m.command(ctx, "native-clock", "date +%s", nil)
	if err != nil {
		return err
	}
	seconds, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil {
		return err
	}
	start := time.Unix(seconds, 0).UTC().Add(after)
	config := fmt.Sprintf("[cincinnati]\nbase_url = \"https://10.0.2.2:19444\"\n[updates]\nenabled = true\nstrategy = \"periodic\"\n[[updates.periodic.window]]\ndays = [\"%s\"]\nstart_time = \"%s\"\nlength_minutes = 10\n", start.Format("Mon"), start.Format("15:04"))
	return m.put(ctx, "maintenance-policy", "/etc/zincati/config.d/99-soda-p9.toml", []byte(config))
}
func refusal(ctx context.Context, m *machine, f *fixture, b Artifact, label, mode string) error {
	f.mu.Lock()
	f.mode = mode
	f.hits = 0
	f.mu.Unlock()
	args, err := m.remote.Args()
	if err != nil {
		return err
	}
	attempt, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	result, err := acceptance.Execute(attempt, m.evidence, label, acceptance.Command{Name: "ssh", Args: append(args, "--", "RPMOSTREE_CLIENT_ID=zincati rpm-ostree deploy --lock-finalization --disallow-downgrade "+b.Candidate.Host.Manifest)})
	if err != nil {
		return err
	}
	if attempt.Err() != nil {
		return errors.New("native refusal did not complete")
	}
	if result.Err == nil || result.ExitCode == nil || *result.ExitCode == 0 {
		return errors.New("native invalid content was accepted")
	}
	diagnostics := strings.ToLower(string(result.Stdout) + string(result.Stderr))
	if mode == "" && !strings.Contains(diagnostics, "signature") {
		return errors.New("wrong signer did not fail at signature verification")
	}
	f.mu.Lock()
	hits := f.hits
	f.mu.Unlock()
	if mode != "" && hits == 0 {
		return errors.New("content refusal did not exercise selected fault")
	}
	ds, err := m.status(ctx)
	if err != nil {
		return err
	}
	booted := false
	for _, d := range ds {
		if d.Staged || d.Booted && d.Digest != f.a.Candidate.Host.Manifest {
			return errors.New("refusal changed native deployment")
		}
		if d.Booted {
			booted = true
		}
	}
	if !booted {
		return errors.New("refusal did not retain booted A")
	}
	return nil
}
