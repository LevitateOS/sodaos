package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParseControllerEvents(t *testing.T) {
	e, ok := parseEvent("START P1 / Build runtime")
	if !ok || e.kind != "START" || e.label != "P1 / Build runtime" {
		t.Fatalf("START not parsed: %+v %v", e, ok)
	}
	e, ok = parseEvent("DONE P1 / Build runtime | phase 00:05:30 | total 00:05:30")
	if !ok || e.kind != "DONE" || e.phaseDur != "00:05:30" || e.totalDur != "00:05:30" {
		t.Fatalf("DONE not parsed: %+v %v", e, ok)
	}
	e, ok = parseEvent("FAILED P8 / Media | section 00:18:35 | total 00:21:24")
	if !ok || e.kind != "FAILED" || e.phaseDur != "00:18:35" {
		t.Fatalf("section duration not parsed: %+v %v", e, ok)
	}
	e, ok = parseEvent("CANDIDATE /out/artifacts/candidate.json")
	if !ok || e.kind != "CANDIDATE" || e.path == "" {
		t.Fatalf("CANDIDATE not parsed: %+v %v", e, ok)
	}
	if _, ok := parseEvent("some builder log line"); ok {
		t.Fatal("log traffic must not parse as an event")
	}
	if _, ok := parseEvent("START"); ok {
		t.Fatal("bare kind must not parse")
	}
}

func baseOptions() options {
	return options{
		controller:   "/admitted/soda-build",
		workerConfig: "/restricted/worker.json",
		arch:         "x86_64",
		out:          "/source/.artifacts/releases/isolated/test",
		mode:         "media",
		rootfsURL:    "http://fixture:8080",
		repoPrefix:   "ghcr.io/levitateos/sodaos",
	}
}

func TestValidateResolvedBoundaries(t *testing.T) {
	o := baseOptions()
	if err := validateResolved(&o); err != nil {
		t.Fatal(err)
	}
	o.mode, o.rootfsURL = "candidate", "http://fixture:8080"
	if err := validateResolved(&o); err == nil {
		t.Fatal("candidate must refuse media-only inputs")
	}
	o = baseOptions()
	o.rootfsURL = ""
	if err := validateResolved(&o); err == nil {
		t.Fatal("media requires the rootfs base URL")
	}
	o = baseOptions()
	o.mode = "production"
	if err := validateResolved(&o); err == nil {
		t.Fatal("production requires the qualification config")
	}
	o.qualConfig = "/restricted/qualification.json"
	if err := validateResolved(&o); err != nil {
		t.Fatal(err)
	}
	o.signConfig = "/restricted/signing.json"
	args := controllerArgs(o)
	joined := strings.Join(args, " ")
	for _, want := range []string{"--qualification-config", "--signing-config"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("production args miss %s: %s", want, joined)
		}
	}
	if strings.Contains(joined, "--development") {
		t.Fatalf("production must not pass development flags: %s", joined)
	}
}

func scriptedOverview(t *testing.T, o options, input string) (options, string) {
	t.Helper()
	var out bytes.Buffer
	p := newPrompter(strings.NewReader(input), &out)
	err := p.overview(&o, func() string { return "/suggested/out" })
	return o, out.String() + "\x00" + errString(err)
}

func errString(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}

func validScriptedOptions(tmp string) options {
	o := baseOptions()
	o.out = tmp + "/fresh"
	return o
}

func TestOverviewStartsWhenValid(t *testing.T) {
	o, screen := scriptedOverview(t, validScriptedOptions(t.TempDir()), "go\n")
	if !strings.HasSuffix(screen, "<nil>") {
		t.Fatalf("overview refused a valid screen: %s", screen)
	}
	if err := validateResolved(&o); err != nil {
		t.Fatalf("overview answers invalid: %v", err)
	}
}

func TestOverviewEditsField(t *testing.T) {
	o := validScriptedOptions(t.TempDir())
	o.rootfsURL = "http://old.invalid"
	// Field 5 is the media rootfs URL; then start.
	o, _ = scriptedOverview(t, o, "5\nhttp://new.invalid\n\ngo\n")
	if o.rootfsURL != "http://new.invalid" {
		t.Fatalf("field edit not applied: %+v", o)
	}
}

func TestOverviewBlocksBadStartWithoutLosingAnswers(t *testing.T) {
	o := validScriptedOptions(t.TempDir())
	o.mode, o.rootfsURL = "production", "" // no qualification config either
	_, screen := scriptedOverview(t, o, "go\nquit\n")
	if !strings.Contains(screen, "Cannot start:") {
		t.Fatalf("blocked start unexplained:\n%s", screen)
	}
	if !strings.Contains(screen, "aborted by operator") {
		t.Fatalf("quit did not abort:\n%s", screen)
	}
}

func TestOverviewPrefillsStandardPaths(t *testing.T) {
	realExists := fileExists
	fileExists = func(path string) bool {
		return path == "/usr/local/lib/soda/soda-build" || path == "/var/lib/soda-candidate-authority/worker.json"
	}
	defer func() { fileExists = realExists }()
	o := options{arch: "x86_64", mode: "media", out: t.TempDir() + "/fresh", rootfsURL: "http://fixture:8080"}
	got, screen := scriptedOverview(t, o, "go\n")
	if !strings.HasSuffix(screen, "<nil>") {
		t.Fatalf("overview with standard paths refused to start:\n%s", screen)
	}
	if got.controller != "/usr/local/lib/soda/soda-build" {
		t.Fatalf("controller not prefilled: %+v", got)
	}
	if got.workerConfig != "/var/lib/soda-candidate-authority/worker.json" {
		t.Fatalf("worker config not prefilled: %+v", got)
	}
}

func TestOverviewDefaultsFixtureRootfsURL(t *testing.T) {
	o := options{arch: "x86_64", out: t.TempDir() + "/fresh", controller: "/a", workerConfig: "/b", repoPrefix: "x"}
	got, _ := scriptedOverview(t, o, "quit\n")
	if got.mode != "media" {
		t.Fatalf("mode not defaulted to media: %+v", got)
	}
	if got.rootfsURL != fixtureRootfsURL {
		t.Fatalf("fixture pickup address not defaulted: %+v", got)
	}
}

func TestModeSwitchKeepsFixtureURLHonest(t *testing.T) {
	base := options{arch: "x86_64", out: t.TempDir() + "/fresh", controller: "/a", workerConfig: "/b", repoPrefix: "x", qualConfig: "/q"}
	media := base
	media.mode, media.rootfsURL = "media", fixtureRootfsURL
	got, _ := scriptedOverview(t, media, "1\n3\nquit\n")
	if got.mode != "production" || got.rootfsURL != "" {
		t.Fatalf("fixture URL leaked into production: %+v", got)
	}
	prod := base
	prod.mode = "production"
	got, _ = scriptedOverview(t, prod, "1\n2\nquit\n")
	if got.mode != "media" || got.rootfsURL != fixtureRootfsURL {
		t.Fatalf("fixture URL not restored for media: %+v", got)
	}
}

func TestOutLeafFollowsWorkerNameRule(t *testing.T) {
	for _, leaf := range []string{"20260915t212541z", "manual-01", "a"} {
		if !validOutLeaf(leaf) {
			t.Fatalf("valid leaf refused: %q", leaf)
		}
	}
	for _, leaf := range []string{"", "20260915T212541Z", "has space", "UPPER", "under_score", strings.Repeat("a", 49)} {
		if validOutLeaf(leaf) {
			t.Fatalf("invalid leaf accepted: %q", leaf)
		}
	}
	// The exact reported failure: an uppercase timestamp must be refused
	// with a plain message before anything privileged runs.
	o := baseOptions()
	o.out = "/source/.artifacts/releases/isolated/20260915T212541Z"
	if err := validateResolved(&o); err == nil || !strings.Contains(err.Error(), "lowercase") {
		t.Fatalf("uppercase output not refused plainly, got: %v", err)
	}
	if got := filepath.Base(suggestOut()); !validOutLeaf(got) {
		t.Fatalf("suggested output violates the worker rule: %q", got)
	}
}

func TestAskOutRejectsThenAccepts(t *testing.T) {
	tmp := t.TempDir()
	var out bytes.Buffer
	p := newPrompter(strings.NewReader("rel\n"+tmp+"/fresh\n"), &out)
	got, err := p.askOut("Fresh output directory", "rel")
	if err != nil {
		t.Fatal(err)
	}
	if got != tmp+"/fresh" {
		t.Fatalf("askOut returned %q", got)
	}
	if !strings.Contains(out.String(), "Absolute path required.") {
		t.Fatalf("rejection unexplained:\n%s", out.String())
	}
}

func TestFixtureWanted(t *testing.T) {
	for _, u := range []string{"http://127.0.0.1:8080", "http://localhost:8080", "http://[::1]:8080"} {
		if !fixtureWanted("media", u) {
			t.Fatalf("loopback media not self-served: %s", u)
		}
	}
	if fixtureWanted("media", "https://example.invalid/rootfs") {
		t.Fatal("public URL must stay operator-managed")
	}
	if fixtureWanted("candidate", "http://127.0.0.1:8080") {
		t.Fatal("candidate needs no pickup server")
	}
	if fixtureWanted("production", "http://127.0.0.1:8080") {
		t.Fatal("production must stay operator-managed")
	}
	if fixtureWanted("media", "://bogus") {
		t.Fatal("unparseable URL accepted")
	}
}

func TestFixtureAddr(t *testing.T) {
	addr, err := fixtureAddr("http://localhost:8080")
	if err != nil || addr != "127.0.0.1:8080" {
		t.Fatalf("addr = %q, %v", addr, err)
	}
	if _, err := fixtureAddr("http://127.0.0.1/"); err == nil {
		t.Fatal("portless URL accepted")
	}
}

func TestServeAndFileRootfs(t *testing.T) {
	serveDir := t.TempDir()
	stop, listen, err := serveFixture("127.0.0.1:0", serveDir)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	out := t.TempDir()
	media := filepath.Join(out, "artifacts", "media")
	if err := os.MkdirAll(media, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(media, "a-rootfs.img"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	name, err := copyBuiltRootfs(out, serveDir)
	if err != nil {
		t.Fatal(err)
	}
	if name != "a-rootfs.img" {
		t.Fatalf("filed %q", name)
	}
	resp, err := http.Get("http://" + listen + "/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "payload" {
		t.Fatalf("served %q", body)
	}
}

func TestRunningPhaseShowsLiveElapsed(t *testing.T) {
	var b bytes.Buffer
	r := newRenderer(&b, true, 80)
	if err := r.feed("START P1 / Build runtime"); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	if !strings.Contains(got, "P1 / Build runtime") || !strings.Contains(got, "(live ") {
		t.Fatalf("running row misses live elapsed:\n%s", got)
	}
	if err := r.feed("DONE P1 / Build runtime | phase 00:05:30 | total 00:05:30"); err != nil {
		t.Fatal(err)
	}
	if got = b.String(); !strings.Contains(got, "[ok]") || !strings.Contains(got, "00:05:30") {
		t.Fatalf("finished row misses controller duration:\n%s", got)
	}
}

func TestFailedRunPrintsWhyPanelWithHostPaths(t *testing.T) {
	for _, tty := range []bool{true, false} {
		var b bytes.Buffer
		r := newRenderer(&b, tty, 80)
		r.SetOutDir("/home/op/sodaos/.artifacts/releases/isolated/run-01")
		for _, line := range []string{
			"START P3 / Compile shipping programs",
			"LOG /run/soda-build-source/.artifacts/releases/isolated/run-01/logs/timing.log",
			"FAILED Compile soda-dashboard | section 00:00:00 | total 00:00:00 | reason open /run/go/src/a.go: permission denied",
			"FAILED P3 / Compile shipping programs | phase 00:00:00 | total 00:00:00",
		} {
			if err := r.feed(line); err != nil {
				t.Fatal(err)
			}
		}
		if err := r.finish(1); err != nil {
			t.Fatal(err)
		}
		got := b.String()
		for _, want := range []string{
			"why: Compile soda-dashboard",
			"cause: open /run/go/src/a.go: permission denied",
			"log: /home/op/sodaos/.artifacts/releases/isolated/run-01/logs/build.log",
		} {
			if !strings.Contains(got, want) {
				t.Fatalf("tty=%v panel misses %q:\n%s", tty, want, got)
			}
		}
		if strings.Contains(got, "/run/soda-build-source/.artifacts/releases/isolated/run-01/logs/timing.log") {
			t.Fatalf("tty=%v panel leaks sandbox path:\n%s", tty, got)
		}
	}
}

func TestFailedRunWithoutReasonFallsBackToLog(t *testing.T) {
	var b bytes.Buffer
	r := newRenderer(&b, false, 80)
	r.SetOutDir("/out/run-02")
	if err := r.feed("FAILED P1 / Admit | phase 00:00:01 | total 00:00:01"); err != nil {
		t.Fatal(err)
	}
	if err := r.finish(1); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	if !strings.Contains(got, "why: P1 / Admit") || !strings.Contains(got, "cause: see the build log") {
		t.Fatalf("fallback panel missing:\n%s", got)
	}
}

func TestHostArtifactPathLeavesForeignPathsAlone(t *testing.T) {
	out := "/home/op/sodaos/.artifacts/releases/isolated/run-01"
	if got := hostArtifactPath(out, "/run/soda-build-source/.artifacts/releases/isolated/run-01/artifacts/candidate.json"); got != "/home/op/sodaos/.artifacts/releases/isolated/run-01/artifacts/candidate.json" {
		t.Fatalf("not translated: %q", got)
	}
	for _, tc := range []struct{ out, sandbox string }{
		{"", "/run/soda-build-source/x"},
		{out, "/somewhere/else/media.json"},
		{out, "relative/path.json"},
	} {
		if got := hostArtifactPath(tc.out, tc.sandbox); got != tc.sandbox {
			t.Fatalf("changed foreign path: %q", got)
		}
	}
}

func writeWorkerJSON(t *testing.T, runtimeDir string) string {
	t.Helper()
	tools := t.TempDir()
	path := t.TempDir() + "/worker.json"
	raw := `{"OutputParent": "/out", "Runtime": "` + runtimeDir + `", "Tools": "` + tools + `"}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// stubProbeExec answers every environment probe positively.
func stubProbeExec(t *testing.T) {
	t.Helper()
	real := execRunner
	execRunner = func(name string, args ...string) (string, error) {
		switch name {
		case "stat":
			if len(args) > 0 && args[len(args)-1] == workerGoCache {
				return "system_u:object_r:soda_build_cache_t:s0", nil
			}
			return "system_u:object_r:lib_t:s0\n", nil
		case "/usr/bin/git":
			return "/run/soda-build-source\n", nil
		default:
			return "", nil
		}
	}
	t.Cleanup(func() { execRunner = real })
}

func prepareOpts(config string) options {
	return options{arch: "x86_64", mode: "candidate", out: "/out/fresh-01", workerConfig: config, repoPrefix: "x"}
}

// chdirRepoRoot runs a probe test from the checkout root, where go.mod lives.
func chdirRepoRoot(t *testing.T) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatal(err)
		}
	})
}

func TestPrepareRuntimeClearsIdleState(t *testing.T) {
	chdirRepoRoot(t)
	stubProbeExec(t)
	runtimeDir := t.TempDir()
	if err := os.WriteFile(runtimeDir+"/stale.lock", []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	quiet := func() (string, error) { return "", nil }
	if err := prepareRuntime(prepareOpts(writeWorkerJSON(t, runtimeDir)), quiet); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(runtimeDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("stale runtime state survived: %v", entries)
	}
}

func TestPrepareRuntimeRefusesConcurrentBuild(t *testing.T) {
	chdirRepoRoot(t)
	stubProbeExec(t)
	runtimeDir := t.TempDir()
	busy := func() (string, error) { return "soda-build-manual-01.service loaded active running\n", nil }
	err := prepareRuntime(prepareOpts(writeWorkerJSON(t, runtimeDir)), busy)
	if err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("concurrent build not refused, got: %v", err)
	}
}

func TestPrepareRuntimeNamesSetupDrift(t *testing.T) {
	stubProbeExec(t)
	quiet := func() (string, error) { return "", nil }
	if err := prepareRuntime(prepareOpts(t.TempDir()+"/absent.json"), quiet); err == nil ||
		!strings.Contains(err.Error(), "setup script") {
		t.Fatalf("missing config unexplained, got: %v", err)
	}
	bad, err := os.CreateTemp(t.TempDir(), "worker*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bad.WriteString("{nope"); err != nil {
		t.Fatal(err)
	}
	if err := bad.Close(); err != nil {
		t.Fatal(err)
	}
	if err := prepareRuntime(prepareOpts(bad.Name()), quiet); err == nil {
		t.Fatal("invalid worker config accepted")
	}
}

func TestControllerToolchainMismatchNamed(t *testing.T) {
	chdirRepoRoot(t)
	stubProbeExec(t)
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	o := prepareOpts(writeWorkerJSON(t, t.TempDir()))
	o.controller = self // test binary: never the pinned toolchain
	quiet := func() (string, error) { return "", nil }
	err = prepareRuntime(o, quiet)
	if err == nil || !strings.Contains(err.Error(), "rebuild with the pinned toolchain") {
		t.Fatalf("toolchain downgrade not named, got: %v", err)
	}
}

func TestModuleCacheLabelRefused(t *testing.T) {
	chdirRepoRoot(t)
	real := execRunner
	execRunner = func(name string, args ...string) (string, error) {
		if name == "stat" {
			return "unconfined_u:object_r:cache_home_t:s0\n", nil
		}
		return "", nil
	}
	t.Cleanup(func() { execRunner = real })
	o := prepareOpts(writeWorkerJSON(t, t.TempDir()))
	quiet := func() (string, error) { return "", nil }
	err := prepareRuntime(o, quiet)
	if err == nil || !strings.Contains(err.Error(), "want lib_t") {
		t.Fatalf("foreign label not refused, got: %v", err)
	}
}

func TestWorkerCacheLabelRefused(t *testing.T) {
	chdirRepoRoot(t)
	real := execRunner
	execRunner = func(name string, args ...string) (string, error) {
		if name == "stat" {
			if len(args) > 0 && args[len(args)-1] == workerGoCache {
				return "unconfined_u:object_r:var_lib_t:s0", nil
			}
			return "system_u:object_r:lib_t:s0", nil
		}
		return "", nil
	}
	t.Cleanup(func() { execRunner = real })
	o := prepareOpts(writeWorkerJSON(t, t.TempDir()))
	quiet := func() (string, error) { return "", nil }
	err := prepareRuntime(o, quiet)
	if err == nil || !strings.Contains(err.Error(), "want soda_build_cache_t") {
		t.Fatalf("foreign cache label not refused, got: %v", err)
	}
}

func TestSetpgidDenialExplained(t *testing.T) {
	chdirRepoRoot(t)
	real := execRunner
	execRunner = func(name string, args ...string) (string, error) {
		switch name {
		case "stat":
			if len(args) > 0 && args[len(args)-1] == workerGoCache {
				return "system_u:object_r:soda_build_cache_t:s0", nil
			}
			return "system_u:object_r:lib_t:s0\n", nil
		case "/usr/bin/git":
			return "/run/soda-build-source\n", nil
		default:
			return "", errors.New("exit status 1")
		}
	}
	t.Cleanup(func() { execRunner = real })
	o := prepareOpts(writeWorkerJSON(t, t.TempDir()))
	quiet := func() (string, error) { return "", nil }
	err := prepareRuntime(o, quiet)
	if err == nil || !strings.Contains(err.Error(), "SELinux module") {
		t.Fatalf("setpgid denial unexplained, got: %v", err)
	}
}

func TestPreflightRefusesOutsideCheckout(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatal(err)
		}
	}()
	err = preflight(baseOptions())
	if err == nil || !strings.Contains(err.Error(), "checkout root") {
		t.Fatalf("preflight must refuse a non-checkout directory first, got: %v", err)
	}
}

func TestPipeRendererSummarizes(t *testing.T) {
	var b bytes.Buffer
	r := newRenderer(&b, false, 80)
	for _, line := range []string{
		"START P1 / Build runtime",
		"DONE P1 / Build runtime | phase 00:05:30 | total 00:05:30",
		"CANDIDATE /out/artifacts/candidate.json",
	} {
		if err := r.feed(line); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.finish(0); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	for _, want := range []string{"P1 / Build runtime", "CANDIDATE", "exit 0"} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary misses %q:\n%s", want, got)
		}
	}
}
