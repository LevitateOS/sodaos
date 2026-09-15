package build

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func progressFixture(t *testing.T) (*BuildProgress, string, *bytes.Buffer) {
	t.Helper()
	for _, key := range []string{"SODA_BUILD_START_NS", "SODA_BUILD_TIMING_LOG", "SODA_BUILD_CHILD"} {
		t.Setenv(key, "")
	}
	source, e := filepath.Abs("../../..")
	if e != nil {
		t.Fatal(e)
	}
	p, e := NewBuildProgress(source, "Release fixture")
	if e != nil {
		t.Fatal(e)
	}
	var output bytes.Buffer
	p.Stderr = &output
	path := filepath.Join(t.TempDir(), "timing.log")
	if e = p.CreateLog(path); e != nil {
		t.Fatal(e)
	}
	return p, path, &output
}

func TestGoProductionUsesNativeTimingOwner(t *testing.T) {
	p, path, output := progressFixture(t)
	if e := p.Next("Build fixture"); e != nil {
		t.Fatal(e)
	}
	execution := BuildExecution{Context: context.Background(), Log: io.Discard}
	got, e := execution.Capture(t.TempDir(), "sh", "-c", "printf native-image-id")
	if e != nil || got != "native-image-id" {
		t.Fatal(got, e)
	}
	if e = p.Next("Failing fixture"); e != nil {
		t.Fatal(e)
	}
	e = execution.Execute(t.TempDir(), "sh", "-c", "exit 7", "private-sentinel-not-in-timings")
	if BuildExitCode(e) != 7 {
		t.Fatal(e)
	}
	if e = p.Finish(e); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	text := string(b)
	for _, s := range []string{"DONE     Build fixture", "FAILED   Failing fixture", "section ", "total ", "exit 7"} {
		if !strings.Contains(text, s) {
			t.Fatalf("missing %s: %s", s, text)
		}
	}
	if strings.Contains(text, "private-sentinel") || strings.Contains(text, "SUCCESS") {
		t.Fatal(text)
	}
	if !strings.Contains(output.String(), "SECTION SUMMARY") {
		t.Fatal(output.String())
	}
	st, e := os.Stat(path)
	if e != nil || st.Mode().Perm() != 0o600 {
		t.Fatal(st, e)
	}
}

func TestNativePhaseClockAndFailedOutput(t *testing.T) {
	var output bytes.Buffer
	now := time.Duration(0)
	p := &BuildProgress{title: "fixture", Now: func() time.Duration { return now }, Stderr: &output}
	if err := p.Phase("P3"); err != nil {
		t.Fatal(err)
	}
	now = 2 * time.Second
	if err := p.Next("Compile once"); err != nil {
		t.Fatal(err)
	}
	now = 5 * time.Second
	if err := p.Phase("P4"); err != nil {
		t.Fatal(err)
	}
	now = 9 * time.Second
	if err := p.Finish(errors.New("failure")); err != nil {
		t.Fatal(err)
	}
	if err := p.Finish(nil); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"section 00:00:03", "P3 | phase 00:00:05", "P4 | phase 00:00:04", "exit 1"} {
		if !strings.Contains(output.String(), text) {
			t.Fatal(output.String())
		}
	}
	if strings.Contains(output.String(), "SUCCESS") || strings.Count(output.String(), "exit 1") != 1 {
		t.Fatal(output.String())
	}
}

func TestGoProgressRetainsOccupiedLog(t *testing.T) {
	p, path, _ := progressFixture(t)
	t.Setenv("SODA_BUILD_TIMING_LOG", "")
	before, _ := os.ReadFile(path)
	if e := p.CreateLog(path); e == nil {
		t.Fatal("occupied log accepted")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("retained log overwritten")
	}
}

func TestGoProgressSharesParentTotalAndSuppressesChildSummary(t *testing.T) {
	p, path, output := progressFixture(t)
	origin := os.Getenv("SODA_BUILD_START_NS")
	t.Setenv("SODA_BUILD_CHILD", "1")
	if e := p.Next("Child production"); e != nil {
		t.Fatal(e)
	}
	if e := p.Finish(nil); e != nil {
		t.Fatal(e)
	}
	if os.Getenv("SODA_BUILD_START_NS") != origin || strings.Contains(output.String(), "SUCCESS") || strings.Contains(output.String(), "SECTION SUMMARY") {
		t.Fatal(output.String())
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "DONE     Child production") {
		t.Fatal(string(b))
	}
}

func TestGoProgressCancellationAndPinnedCompiler(t *testing.T) {
	p, path, _ := progressFixture(t)
	if e := p.Next("Cancelled production"); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	execution := BuildExecution{Context: ctx, Log: io.Discard}
	err := execution.Execute(t.TempDir(), "sh", "-c", "exit 0")
	if BuildExitCode(err) != 130 {
		t.Fatal(err)
	}
	if e := p.Finish(err); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "CANCELLED Cancelled production") {
		t.Fatal(string(b))
	}
	cmd := execution.command(t.TempDir(), "go", "version")
	pinned, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no Go on PATH; nothing to pin against")
	}
	if cmd.Path != pinned && cmd.Path != "go" {
		t.Fatal("unexpected Go resolved", cmd.Path)
	}
}
