package nativequalification

import (
	"bytes"
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

// Run executes the single native scenario after protected admission. There is no
// release resume, selectable scenario graph, or builder-supplied pass assertion.
func Run(parent context.Context, configPath string) (err error) {
	ctx, cancel := context.WithTimeout(parent, 45*time.Minute)
	defer cancel()
	u, err := user.Lookup("soda-qualifier")
	if err != nil {
		return err
	}
	if strconv.Itoa(os.Geteuid()) != u.Uid || configPath != "/run/soda-p9-input/config.json" {
		return errors.New("protected qualifier identity and mounted inputs required")
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
		return errors.New("selected local qualification repository required")
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
	receipt := Receipt{Scope: "native-install-upgrade-recovery", HostManifest: b.Candidate.Host.Manifest, PayloadSHA256: b.Candidate.PayloadSHA256, BaselineHostManifest: a.Candidate.Host.Manifest}
	receipt.InstallCommit, err = installB(ctx, c, b, media, f)
	if err != nil {
		return err
	}
	fmt.Println("P9: actual ISO installation and media-free B verified")

	password, err := os.ReadFile(filepath.Join(c.Work, "a-password"))
	if err != nil {
		return err
	}
	password = bytes.TrimSpace(password)
	ae, err := acceptance.CreateEvidence(filepath.Join(c.Work, "update-evidence"), [][]byte{password})
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, ae.Close()) }()
	m, err := newMachine(ctx, c, filepath.Join(c.Work, "a-vm"), filepath.Join(c.Work, "a.qcow2"), filepath.Join(c.Work, "a-vars.fd"), "", 32294, filepath.Join(c.Work, "a-key"), filepath.Join(c.Work, "a-known-hosts"), ae)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, m.vm.Close()) }()
	if err = m.remote.WaitReady(ctx); err != nil {
		return err
	}
	if err = m.driver(ctx, c.Executable); err != nil {
		return err
	}
	receipt.BaselineCommit, err = m.identity(ctx, a, true)
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
	forgejoBefore, err := m.command(ctx, "a-forgejo-version", "podman exec --user git soda-forgejo forgejo --version", nil)
	if err != nil {
		return err
	}
	if err = configureGuest(ctx, m, f); err != nil {
		return err
	}
	if err = f.push(ctx, "wrong"); err != nil {
		return err
	}
	if err = refusal(ctx, m, f, b, true); err != nil {
		return err
	}
	if err = f.push(ctx, "correct"); err != nil {
		return err
	}
	if err = refusal(ctx, m, f, b, false); err != nil {
		return err
	}
	retained, err := m.state(ctx, "snapshot", a.Payload.ID)
	if err != nil {
		return err
	}
	if err = sameState(before, retained); err != nil {
		return err
	}
	fmt.Println("P9: wrong-key and required-content refusals verified; populated A unchanged")

	f.mu.Lock()
	f.blocked = false
	f.offer = true
	f.mu.Unlock()
	window, err := maintenance(ctx, m)
	if err != nil {
		return err
	}
	if _, err = m.command(ctx, "enable-native-agent", "systemctl start zincati.service", nil); err != nil {
		return err
	}
	stageCtx, stop := context.WithTimeout(ctx, 4*time.Minute)
	var staged deployment
	err = poll(stageCtx, func() (bool, error) {
		ds, err := m.status(stageCtx)
		if err != nil {
			return false, err
		}
		for _, d := range ds {
			if d.Booted && d.Checksum != receipt.BaselineCommit {
				return false, errors.New("premature activation")
			}
			if d.Staged && d.Digest == b.Candidate.Host.Manifest {
				staged = d
				return true, nil
			}
		}
		return false, nil
	})
	stop()
	if err != nil {
		return err
	}
	if err = m.locked(ctx, staged); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(20 * time.Second):
	}
	if err = m.locked(ctx, staged); err != nil {
		return err
	}
	clock, err := nativeClock(ctx, m)
	if err != nil {
		return err
	}
	if !clock.Before(window) {
		return errors.New("staging overran the selected maintenance window")
	}
	f.mu.Lock()
	requests := f.graphRequests
	f.mu.Unlock()
	if requests == 0 {
		return errors.New("native agent did not consult the actual A/B graph")
	}
	if _, err = f.run(ctx, "content-offline", "podman", "--remote=false", "stop", "--time=10", f.cid); err != nil {
		return err
	}
	f.cid = ""
	f.mu.Lock()
	payloadBefore := f.payloadRequests
	f.mu.Unlock()
	// The same native agent retains its staged-update state. Restarting it here
	// would restage and require registry metadata rather than test offline activation.
	waitBoot := func(digest, commit string) error {
		ready, stop := context.WithTimeout(ctx, 8*time.Minute)
		defer stop()
		return poll(ready, func() (bool, error) {
			ds, err := m.status(ready)
			if err != nil {
				return false, nil
			}
			for _, d := range ds {
				if d.Booted && d.Digest == digest && d.Checksum == commit {
					return true, nil
				}
			}
			return false, nil
		})
	}
	if err = waitBoot(b.Candidate.Host.Manifest, staged.Checksum); err != nil {
		return err
	}
	f.mu.Lock()
	payloadAfter := f.payloadRequests
	f.mu.Unlock()
	if payloadAfter != payloadBefore {
		return errors.New("payload requested during offline activation")
	}
	receipt.UpdatedCommit, err = m.identity(ctx, b, true)
	if err != nil {
		return err
	}
	current, err := m.state(ctx, "snapshot", b.Payload.ID)
	if err != nil {
		return err
	}
	if err = sameState(before, current); err != nil {
		return err
	}
	forgejoAfter, err := m.command(ctx, "b-forgejo-version", "podman exec --user git soda-forgejo forgejo --version", nil)
	if err != nil {
		return err
	}
	if !bytes.Equal(bytes.TrimSpace(forgejoBefore), bytes.TrimSpace(forgejoAfter)) {
		return errors.New("native Forgejo version changed; compatible rollback not established")
	}
	if err = e.WriteJSON("native-update.json", map[string]any{"a_commit": receipt.BaselineCommit, "b_commit": receipt.UpdatedCommit, "window": window, "graph_requests": requests, "payload_requests_during_activation": payloadAfter - payloadBefore, "soda_schema": b.Payload.Schema, "forgejo_version": strings.TrimSpace(string(forgejoAfter))}); err != nil {
		return err
	}
	fmt.Println("P9: native maintenance/offline B activation and populated-state preservation verified")

	later, err := m.state(ctx, "later", b.Payload.ID)
	if err != nil {
		return err
	}
	if err = laterState(before, later); err != nil {
		return err
	}
	f.mu.Lock()
	f.offer = false
	f.mu.Unlock()
	if _, err = m.command(ctx, "native-rollback", "set -eu; systemctl stop zincati.service; rpm-ostree rollback", nil); err != nil {
		return err
	}
	if _, err = m.command(ctx, "reboot-to-a", "systemctl reboot", nil); err != nil {
		return err
	}
	if err = waitBoot(a.Candidate.Host.Manifest, receipt.BaselineCommit); err != nil {
		return err
	}
	receipt.RecoveredCommit, err = m.identity(ctx, a, true)
	if err != nil {
		return err
	}
	recovered, err := m.state(ctx, "snapshot", a.Payload.ID)
	if err != nil {
		return err
	}
	if err = sameState(later, recovered); err != nil {
		return err
	}
	if err = m.recoveryConsole(ctx, c.Work, password); err != nil {
		return err
	}
	if err = receipt.Validate(a, b, media.HostManifest); err != nil {
		return err
	}
	fmt.Println("P9: native rollback, console access and both generations of state verified")
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
	if _, err := m.command(ctx, "fixture-directories", "set -eu; systemctl stop zincati.service; chmod 0755 /run/containers; mkdir -p /etc/soda-qualification /etc/containers/registries.d /etc/containers/registries.conf.d /etc/zincati/config.d /etc/systemd/system/rpm-ostreed.service.d", nil); err != nil {
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
		{"/etc/pki/ca-trust/source/anchors/soda-p9.crt", ca}, {"/etc/soda-qualification/key.pem", key}, {"/etc/soda-qualification/policy.json", p},
		{"/etc/systemd/system/rpm-ostreed.service.d/99-soda-p9.conf", []byte("[Service]\nBindReadOnlyPaths=\nBindReadOnlyPaths=/etc/soda-qualification/policy.json:/etc/containers/policy.json\n")},
		{"/etc/containers/registries.conf.d/99-soda-p9.conf", []byte("[[registry]]\nprefix = \"ghcr.io/levitateos/sodaos-host\"\nlocation = \"10.0.2.2:19443/levitateos/sodaos-host\"\n")},
		{"/etc/containers/registries.d/soda-p9.yaml", []byte("docker:\n  \"10.0.2.2:19443/levitateos/sodaos-host\":\n    use-sigstore-attachments: true\n")},
	} {
		if err = m.put(ctx, "fixture-trust", file.path, file.data); err != nil {
			return err
		}
	}
	_, err = m.command(ctx, "native-update-trust", "set -eu; update-ca-trust; systemctl daemon-reload; systemctl restart rpm-ostreed.service; systemctl show rpm-ostreed.service --property=BindReadOnlyPaths", nil)
	return err
}
func nativeClock(ctx context.Context, m *machine) (time.Time, error) {
	raw, err := m.command(ctx, "native-clock", "date +%s", nil)
	if err != nil {
		return time.Time{}, err
	}
	seconds, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	return time.Unix(seconds, 0).UTC(), err
}
func maintenance(ctx context.Context, m *machine) (time.Time, error) {
	clock, err := nativeClock(ctx, m)
	if err != nil {
		return time.Time{}, err
	}
	start := clock.Add(4 * time.Minute).Truncate(time.Minute)
	config := fmt.Sprintf("[cincinnati]\nbase_url = \"https://10.0.2.2:19444\"\n[updates]\nenabled = true\nstrategy = \"periodic\"\n[[updates.periodic.window]]\ndays = [\"%s\"]\nstart_time = \"%s\"\nlength_minutes = 10\n", start.Format("Mon"), start.Format("15:04"))
	return start, m.put(ctx, "maintenance-policy", "/etc/zincati/config.d/99-soda-p9.toml", []byte(config))
}
func refusal(ctx context.Context, m *machine, f *fixture, b Artifact, wrongKey bool) error {
	label := "required-content-refusal"
	if wrongKey {
		label = "wrong-key-refusal"
	}
	f.mu.Lock()
	f.blocked = !wrongKey
	f.hits = 0
	f.mu.Unlock()
	args, err := m.remote.Args()
	if err != nil {
		return err
	}
	attempt, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	result, err := acceptance.Execute(attempt, m.evidence, label, acceptance.Command{Name: "ssh", Args: append(args, "--", "RPMOSTREE_CLIENT_ID=zincati rpm-ostree deploy --lock-finalization --disallow-downgrade "+b.Candidate.Host.Manifest)})
	if err != nil {
		return err
	}
	if attempt.Err() != nil || result.Err == nil || result.ExitCode == nil || *result.ExitCode == 0 {
		return errors.New("native refusal did not complete")
	}
	diagnostics := strings.ToLower(string(result.Stderr))
	if wrongKey && !strings.Contains(diagnostics, "cryptographic signature verification failed") {
		return errors.New("wrong-key cryptographic refusal not established")
	}
	f.mu.Lock()
	hits := f.hits
	f.mu.Unlock()
	if !wrongKey && (hits == 0 || !strings.Contains(diagnostics, "503")) {
		return errors.New("native required-content refusal not established")
	}
	ds, err := m.status(ctx)
	if err != nil {
		return err
	}
	booted := false
	for _, d := range ds {
		if d.Staged || (d.Booted && d.Digest != f.a.Candidate.Host.Manifest) {
			return errors.New("refusal changed native deployment")
		}
		booted = booted || d.Booted
	}
	if !booted {
		return errors.New("refusal did not retain A")
	}
	return nil
}
