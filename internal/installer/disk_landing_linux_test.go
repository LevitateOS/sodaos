package installer

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
)

type powerRecorder struct {
	calls int
	names []string
	args  [][]string
	errs  []error
}

func (r *powerRecorder) run(_ context.Context, name string, args []string, _ io.Reader) ([]byte, error) {
	r.calls++
	r.names = append(r.names, name)
	r.args = append(r.args, args)
	if len(r.errs) == 0 {
		return nil, nil
	}
	err := r.errs[0]
	r.errs = r.errs[1:]
	return nil, err
}

func landingTestConsole(t *testing.T, input string, closeWrite bool) console {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	if input != "" {
		if _, err := w.WriteString(input); err != nil {
			t.Fatal(err)
		}
	}
	if closeWrite {
		w.Close()
	} else {
		t.Cleanup(func() { w.Close() })
	}
	return console{tty: r, ctx: context.Background()}
}

func TestLandingSuccessRebootsOnChoice(t *testing.T) {
	rec := &powerRecorder{}
	err := landDiagnosticConsole(context.Background(), landingTestConsole(t, "reboot\n", false), rec.run, nil)
	if err != nil {
		t.Fatalf("landDiagnosticConsole returned %v, want nil", err)
	}
	if rec.calls != 1 || rec.names[0] != "systemctl" || len(rec.args[0]) != 1 || rec.args[0][0] != "reboot" {
		t.Fatalf("runner got %d calls (%v %v), want one systemctl reboot", rec.calls, rec.names, rec.args)
	}
}

func TestLandingFailurePowersOffOnChoice(t *testing.T) {
	rec := &powerRecorder{}
	installErr := errors.New("disk write failed")
	err := landDiagnosticConsole(context.Background(), landingTestConsole(t, "poweroff\n", false), rec.run, installErr)
	if !errors.Is(err, installErr) {
		t.Fatalf("landDiagnosticConsole returned %v, want install error", err)
	}
	if rec.calls != 1 || rec.args[0][0] != "poweroff" {
		t.Fatalf("runner got %d calls (%v), want one systemctl poweroff", rec.calls, rec.args)
	}
}

func TestLandingCancelKeepsInstallError(t *testing.T) {
	rec := &powerRecorder{}
	installErr := errors.New("cancelled; no disk installation started")
	err := landDiagnosticConsole(context.Background(), landingTestConsole(t, "reboot\n", false), rec.run, installErr)
	if !errors.Is(err, installErr) {
		t.Fatalf("landDiagnosticConsole returned %v, want cancel error", err)
	}
	if rec.calls != 1 {
		t.Fatalf("runner got %d calls, want one reboot after cancel", rec.calls)
	}
}

func TestLandingUnknownChoiceReprompts(t *testing.T) {
	rec := &powerRecorder{}
	err := landDiagnosticConsole(context.Background(), landingTestConsole(t, "retry\nreboot\n", false), rec.run, nil)
	if err != nil {
		t.Fatalf("landDiagnosticConsole returned %v, want nil", err)
	}
	if rec.calls != 1 || rec.args[0][0] != "reboot" {
		t.Fatalf("runner got %d calls (%v), want reprompt then one reboot", rec.calls, rec.args)
	}
}

func TestLandingFailedPowerActionStays(t *testing.T) {
	bootErr := errors.New("systemctl reboot refused")
	rec := &powerRecorder{errs: []error{bootErr}}
	err := landDiagnosticConsole(context.Background(), landingTestConsole(t, "reboot\npoweroff\n", false), rec.run, nil)
	if err != nil {
		t.Fatalf("landDiagnosticConsole returned %v, want nil after poweroff succeeds", err)
	}
	if rec.calls != 2 || rec.args[0][0] != "reboot" || rec.args[1][0] != "poweroff" {
		t.Fatalf("runner got %d calls (%v), want failed reboot then poweroff", rec.calls, rec.args)
	}
}

func TestLandingDeadTerminalRebootsOnce(t *testing.T) {
	rec := &powerRecorder{}
	err := landDiagnosticConsole(context.Background(), landingTestConsole(t, "", true), rec.run, nil)
	if err != nil {
		t.Fatalf("landDiagnosticConsole returned %v, want nil", err)
	}
	if rec.calls != 1 || rec.args[0][0] != "reboot" {
		t.Fatalf("runner got %d calls (%v), want one fallback reboot on dead terminal", rec.calls, rec.args)
	}
}

func TestLandingDeadTerminalFailedRebootReports(t *testing.T) {
	bootErr := errors.New("systemctl reboot refused")
	rec := &powerRecorder{errs: []error{bootErr}}
	installErr := errors.New("disk write failed")
	err := landDiagnosticConsole(context.Background(), landingTestConsole(t, "", true), rec.run, installErr)
	if !errors.Is(err, installErr) {
		t.Fatalf("landDiagnosticConsole returned %v, want install error", err)
	}
}
