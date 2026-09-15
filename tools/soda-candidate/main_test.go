package main

import (
	"bytes"
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
	o.rootfsURL = "" // media cannot start without it
	_, screen := scriptedOverview(t, o, "go\nquit\n")
	if !strings.Contains(screen, "Cannot start:") {
		t.Fatalf("blocked start unexplained:\n%s", screen)
	}
	if !strings.Contains(screen, "aborted by operator") {
		t.Fatalf("quit did not abort:\n%s", screen)
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
