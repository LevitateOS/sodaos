package qualify

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
	"github.com/levitateos/sodaos/internal/release/deliver"
	"github.com/levitateos/sodaos/internal/release/image"
	"golang.org/x/sys/unix"
)

func requireQualifierMount(configPath string) error {
	u, err := user.Lookup("soda-qualifier")
	if err != nil {
		return err
	}
	if strconv.Itoa(os.Geteuid()) != u.Uid || configPath != "/run/soda-p9-input/config.json" {
		return errors.New("protected qualifier identity and mounted inputs required")
	}
	return unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0)
}

func loadRunInputs(configPath string) (Config, Artifact, Artifact, image.Media, error) {
	var c Config
	if err := deliver.ReadJSON(configPath, &c); err != nil {
		return c, Artifact{}, Artifact{}, image.Media{}, err
	}
	if c.Work != "/run/soda-p9-work" {
		return c, Artifact{}, Artifact{}, image.Media{}, errors.New("controller work mount required")
	}
	var a Artifact
	if err := deliver.ReadJSON("/run/soda-p9-input/baseline.json", &a); err != nil {
		return c, a, Artifact{}, image.Media{}, err
	}
	b, err := ReadArtifact("/run/soda-p9-input/candidate")
	if err != nil {
		return c, a, b, image.Media{}, err
	}
	if err = SameBaseScenario(a, b); err != nil {
		return c, a, b, image.Media{}, err
	}
	if b.Payload.RepositoryPrefix != "ghcr.io/levitateos/sodaos" {
		return c, a, b, image.Media{}, errors.New("selected local qualification repository required")
	}
	media, err := ReadMedia("/run/soda-p9-input/candidate", b)
	return c, a, b, media, err
}

func readAPassword(work string) ([]byte, error) {
	password, err := os.ReadFile(filepath.Join(work, "a-password"))
	if err != nil {
		return nil, err
	}
	return bytes.TrimSpace(password), nil
}

func (m *machine) admitGuest(ctx context.Context, executable string, a Artifact, configured bool) (string, error) {
	if err := m.remote.WaitReady(ctx); err != nil {
		return "", err
	}
	if err := m.driver(ctx, executable); err != nil {
		return "", err
	}
	return m.identity(ctx, a, configured)
}

func sameAdmittedState(work string, before map[string]any) error {
	var admitted map[string]any
	if err := deliver.ReadJSON(filepath.Join(work, "a-state.json"), &admitted); err != nil {
		return err
	}
	return sameState(admitted, before)
}

func observeBaseline(ctx context.Context, c Config, a Artifact, ae *acceptance.Evidence, receipt *Receipt) (m *machine, before map[string]any, forgejoBefore []byte, err error) {
	m, err = newMachine(ctx, c, filepath.Join(c.Work, "a-vm"), filepath.Join(c.Work, "a.qcow2"), filepath.Join(c.Work, "a-vars.fd"), "", 32294, filepath.Join(c.Work, "a-key"), filepath.Join(c.Work, "a-known-hosts"), ae)
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, m.vm.Close())
		}
	}()
	commit, err := m.admitGuest(ctx, c.Executable, a, true)
	if err != nil {
		return m, nil, nil, err
	}
	receipt.BaselineCommit = commit
	before, err = m.state(ctx, "snapshot", a.Payload.ID)
	if err != nil {
		return m, nil, nil, err
	}
	if err = sameAdmittedState(c.Work, before); err != nil {
		return m, nil, nil, err
	}
	forgejoBefore, err = m.command(ctx, "a-forgejo-version", "podman exec --user git soda-forgejo forgejo --version", nil)
	return m, before, forgejoBefore, err
}

func verifyRefusals(ctx context.Context, m *machine, f *fixture, a, b Artifact, before map[string]any) error {
	if err := configureGuest(ctx, m, f); err != nil {
		return err
	}
	if err := f.push(ctx, "wrong"); err != nil {
		return err
	}
	if err := refusal(ctx, m, f, b, true); err != nil {
		return err
	}
	if err := f.push(ctx, "correct"); err != nil {
		return err
	}
	if err := refusal(ctx, m, f, b, false); err != nil {
		return err
	}
	retained, err := m.state(ctx, "snapshot", a.Payload.ID)
	if err != nil {
		return err
	}
	return sameState(before, retained)
}

func stagedDeploymentReady(m *machine, ctx context.Context, baseline, wantDigest string, staged *deployment) (bool, error) {
	ds, err := m.status(ctx)
	if err != nil {
		return false, err
	}
	for _, d := range ds {
		if d.Booted && d.Checksum != baseline {
			return false, errors.New("premature activation")
		}
		if d.Staged && d.Digest == wantDigest {
			*staged = d
			return true, nil
		}
	}
	return false, nil
}

func waitStagedDeployment(ctx context.Context, m *machine, baseline, wantDigest string) (deployment, error) {
	stageCtx, stop := context.WithTimeout(ctx, 4*time.Minute)
	defer stop()
	var staged deployment
	err := poll(stageCtx, func() (bool, error) {
		return stagedDeploymentReady(m, stageCtx, baseline, wantDigest, &staged)
	})
	return staged, err
}

func holdFinalizationLock(ctx context.Context, m *machine, staged deployment) error {
	if err := m.locked(ctx, staged); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(20 * time.Second):
	}
	return m.locked(ctx, staged)
}

func confirmStagingWindow(ctx context.Context, m *machine, f *fixture, window time.Time) (int, error) {
	clock, err := nativeClock(ctx, m)
	if err != nil {
		return 0, err
	}
	if !clock.Before(window) {
		return 0, errors.New("staging overran the selected maintenance window")
	}
	f.mu.Lock()
	requests := f.graphRequests
	f.mu.Unlock()
	if requests == 0 {
		return 0, errors.New("native agent did not consult the actual A/B graph")
	}
	return requests, nil
}

func stageCandidate(ctx context.Context, m *machine, f *fixture, b Artifact, baselineCommit string) (deployment, time.Time, int, error) {
	f.mu.Lock()
	f.blocked = false
	f.offer = true
	f.mu.Unlock()
	window, err := maintenance(ctx, m)
	if err != nil {
		return deployment{}, time.Time{}, 0, err
	}
	if _, err = m.command(ctx, "enable-native-agent", "systemctl start zincati.service", nil); err != nil {
		return deployment{}, time.Time{}, 0, err
	}
	staged, err := waitStagedDeployment(ctx, m, baselineCommit, b.Candidate.Host.Manifest)
	if err != nil {
		return deployment{}, time.Time{}, 0, err
	}
	if err = holdFinalizationLock(ctx, m, staged); err != nil {
		return deployment{}, time.Time{}, 0, err
	}
	requests, err := confirmStagingWindow(ctx, m, f, window)
	return staged, window, requests, err
}

func nativeBooted(m *machine, ctx context.Context, digest, commit string) (bool, error) {
	ds, err := m.status(ctx)
	if err != nil {
		return false, nil
	}
	for _, d := range ds {
		if d.Booted && d.Digest == digest && d.Checksum == commit {
			return true, nil
		}
	}
	return false, nil
}

func waitNativeBoot(ctx context.Context, m *machine, digest, commit string) error {
	ready, stop := context.WithTimeout(ctx, 8*time.Minute)
	defer stop()
	return poll(ready, func() (bool, error) {
		return nativeBooted(m, ready, digest, commit)
	})
}

func activateOffline(ctx context.Context, m *machine, f *fixture, b Artifact, staged deployment) error {
	if _, err := f.run(ctx, "content-offline", "podman", "--remote=false", "stop", "--time=10", f.cid); err != nil {
		return err
	}
	f.cid = ""
	f.mu.Lock()
	payloadBefore := f.payloadRequests
	f.mu.Unlock()
	// The same native agent retains its staged-update state. Restarting it here
	// would restage and require registry metadata rather than test offline activation.
	if err := waitNativeBoot(ctx, m, b.Candidate.Host.Manifest, staged.Checksum); err != nil {
		return err
	}
	f.mu.Lock()
	payloadAfter := f.payloadRequests
	f.mu.Unlock()
	if payloadAfter != payloadBefore {
		return errors.New("payload requested during offline activation")
	}
	return nil
}

func verifyUpdatedGuest(ctx context.Context, m *machine, b Artifact, before map[string]any, forgejoBefore []byte, e *acceptance.Evidence, receipt *Receipt, window time.Time, requests int) error {
	var err error
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
	return e.WriteJSON("native-update.json", map[string]any{"a_commit": receipt.BaselineCommit, "b_commit": receipt.UpdatedCommit, "window": window, "graph_requests": requests, "payload_requests_during_activation": 0, "soda_schema": b.Payload.Schema, "forgejo_version": strings.TrimSpace(string(forgejoAfter))})
}

func activateAndVerify(ctx context.Context, m *machine, f *fixture, b Artifact, before map[string]any, forgejoBefore []byte, e *acceptance.Evidence, receipt *Receipt, staged deployment, window time.Time, requests int) error {
	if err := activateOffline(ctx, m, f, b, staged); err != nil {
		return err
	}
	return verifyUpdatedGuest(ctx, m, b, before, forgejoBefore, e, receipt, window, requests)
}

func recoverBaseline(ctx context.Context, m *machine, f *fixture, c Config, a Artifact, later map[string]any, password []byte, receipt *Receipt) error {
	f.mu.Lock()
	f.offer = false
	f.mu.Unlock()
	if _, err := m.command(ctx, "native-rollback", "set -eu; systemctl stop zincati.service; rpm-ostree rollback", nil); err != nil {
		return err
	}
	if _, err := m.command(ctx, "reboot-to-a", "systemctl reboot", nil); err != nil {
		return err
	}
	if err := waitNativeBoot(ctx, m, a.Candidate.Host.Manifest, receipt.BaselineCommit); err != nil {
		return err
	}
	var err error
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
	return m.recoveryConsole(ctx, c.Work, password)
}

func recoverQualifiedState(ctx context.Context, m *machine, f *fixture, c Config, a, b Artifact, media image.Media, before map[string]any, password []byte, receipt *Receipt) error {
	later, err := m.state(ctx, "later", b.Payload.ID)
	if err != nil {
		return err
	}
	if err = laterState(before, later); err != nil {
		return err
	}
	if err = recoverBaseline(ctx, m, f, c, a, later, password, receipt); err != nil {
		return err
	}
	if err = receipt.Validate(a, b, media.HostManifest); err != nil {
		return err
	}
	fmt.Println("P9: native rollback, console access and both generations of state verified")
	return nil
}

func runNativeScenario(ctx context.Context, c Config, a, b Artifact, media image.Media, e *acceptance.Evidence, f *fixture, receipt *Receipt) (err error) {
	fmt.Println("P9: actual ISO installation and media-free B verified")
	password, err := readAPassword(c.Work)
	if err != nil {
		return err
	}
	ae, err := acceptance.CreateEvidence(filepath.Join(c.Work, "update-evidence"), [][]byte{password})
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, ae.Close()) }()
	m, before, forgejoBefore, err := observeBaseline(ctx, c, a, ae, receipt)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, m.vm.Close()) }()
	if err = verifyRefusals(ctx, m, f, a, b, before); err != nil {
		return err
	}
	fmt.Println("P9: wrong-key and required-content refusals verified; populated A unchanged")
	staged, window, requests, err := stageCandidate(ctx, m, f, b, receipt.BaselineCommit)
	if err != nil {
		return err
	}
	if err = activateAndVerify(ctx, m, f, b, before, forgejoBefore, e, receipt, staged, window, requests); err != nil {
		return err
	}
	fmt.Println("P9: native maintenance/offline B activation and populated-state preservation verified")
	return recoverQualifiedState(ctx, m, f, c, a, b, media, before, password, receipt)
}

// Run executes the single native scenario after protected admission. There is no
// release resume, selectable scenario graph, or builder-supplied pass assertion.
func Run(parent context.Context, configPath string) (err error) {
	ctx, cancel := context.WithTimeout(parent, 45*time.Minute)
	defer cancel()
	if err = requireQualifierMount(configPath); err != nil {
		return err
	}
	c, a, b, media, err := loadRunInputs(configPath)
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
	if err = runNativeScenario(ctx, c, a, b, media, e, f, &receipt); err != nil {
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
		{"/etc/pki/ca-trust/source/anchors/soda-p9.crt", ca},
		{"/etc/soda-qualification/key.pem", key},
		{"/etc/soda-qualification/policy.json", p},
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

func refusalCompleted(attempt context.Context, result acceptance.Result) bool {
	return attempt.Err() == nil && result.Err != nil && result.ExitCode != nil && *result.ExitCode != 0
}

func refusalDiagnostics(wrongKey bool, diagnostics string, hits int) error {
	if wrongKey && !strings.Contains(diagnostics, "cryptographic signature verification failed") {
		return errors.New("wrong-key cryptographic refusal not established")
	}
	if !wrongKey && (hits == 0 || !strings.Contains(diagnostics, "503")) {
		return errors.New("native required-content refusal not established")
	}
	return nil
}

func refusalKeptA(ds []deployment, aManifest string) error {
	booted := false
	for _, d := range ds {
		if d.Staged || (d.Booted && d.Digest != aManifest) {
			return errors.New("refusal changed native deployment")
		}
		booted = booted || d.Booted
	}
	if !booted {
		return errors.New("refusal did not retain A")
	}
	return nil
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
	if !refusalCompleted(attempt, result) {
		return errors.New("native refusal did not complete")
	}
	f.mu.Lock()
	hits := f.hits
	f.mu.Unlock()
	if err = refusalDiagnostics(wrongKey, strings.ToLower(string(result.Stderr)), hits); err != nil {
		return err
	}
	ds, err := m.status(ctx)
	if err != nil {
		return err
	}
	return refusalKeptA(ds, f.a.Candidate.Host.Manifest)
}
