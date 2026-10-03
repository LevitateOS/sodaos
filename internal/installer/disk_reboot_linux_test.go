package installer

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
)

type rebootRecorder struct {
	calls int
	name  string
	args  []string
	err   error
}

func (r *rebootRecorder) run(_ context.Context, name string, args []string, _ io.Reader) ([]byte, error) {
	r.calls++
	r.name, r.args = name, args
	return nil, r.err
}

func rebootTestConsole(t *testing.T, input string, closeWrite bool) console {
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

func TestRebootAfterInstallSuccess(t *testing.T) {
	rec := &rebootRecorder{}
	err := rebootAfterInstall(context.Background(), rebootTestConsole(t, "\n", false), rec.run, nil)
	if err != nil {
		t.Fatalf("rebootAfterInstall returned %v, want nil", err)
	}
	if rec.calls != 1 || rec.name != "systemctl" || len(rec.args) != 1 || rec.args[0] != "reboot" {
		t.Fatalf("runner got %d calls (%s %v), want one systemctl reboot", rec.calls, rec.name, rec.args)
	}
}

func TestRebootAfterInstallFailureStillReboots(t *testing.T) {
	rec := &rebootRecorder{}
	installErr := errors.New("cancelled; no disk installation started")
	err := rebootAfterInstall(context.Background(), rebootTestConsole(t, "\n", false), rec.run, installErr)
	if !errors.Is(err, installErr) {
		t.Fatalf("rebootAfterInstall returned %v, want install error", err)
	}
	if rec.calls != 1 {
		t.Fatalf("runner got %d calls, want one reboot even on failure", rec.calls)
	}
}

func TestRebootAfterInstallLineErrorStillReboots(t *testing.T) {
	rec := &rebootRecorder{}
	err := rebootAfterInstall(context.Background(), rebootTestConsole(t, "", true), rec.run, nil)
	if err != nil {
		t.Fatalf("rebootAfterInstall returned %v, want nil", err)
	}
	if rec.calls != 1 {
		t.Fatalf("runner got %d calls, want one reboot even when input fails", rec.calls)
	}
}

func TestRebootAfterInstallRebootFailure(t *testing.T) {
	bootErr := errors.New("systemctl reboot refused")
	rec := &rebootRecorder{err: bootErr}
	if err := rebootAfterInstall(context.Background(), rebootTestConsole(t, "\n", false), rec.run, nil); !errors.Is(err, bootErr) {
		t.Fatalf("clean install with failed reboot returned %v, want reboot error", err)
	}
	rec = &rebootRecorder{err: bootErr}
	installErr := errors.New("disk write failed")
	if err := rebootAfterInstall(context.Background(), rebootTestConsole(t, "\n", false), rec.run, installErr); !errors.Is(err, installErr) {
		t.Fatalf("failed install with failed reboot returned %v, want install error", err)
	}
}
