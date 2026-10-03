package installer

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestEnrollmentPTYProceedsPastConsoleCheck(t *testing.T) {
	_, tty := openTestPTY(t)
	t.Setenv("SSH_CONNECTION", "synthetic")
	t.Setenv("SSH_TTY", "/dev/pts/99")
	run := func(context.Context, string, []string, io.Reader) ([]byte, error) { return nil, nil }
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := armEnrollment(ctx, console{tty: tty, ctx: ctx}, run)
	if err == nil || strings.Contains(err.Error(), "local keyboard/monitor") {
		t.Fatalf("SSH arming still refused at the console: %v", err)
	}
}

func TestEnrollmentDirectServerRefused(t *testing.T) {
	// Tests never launch the transient unit; this must stop at cgroup ownership
	// before reading arm state or opening any socket/listener.
	if err := ServeEnrollment(context.Background()); err == nil {
		t.Fatal("direct server action escaped fixed unit check")
	}
}

func TestEnrollmentCommandsRefusedBeforeState(t *testing.T) {
	for _, value := range []string{"sh", "scp -t /root", "internal-sftp", ":", "\n"} {
		t.Setenv("SSH_ORIGINAL_COMMAND", value)
		if err := ReceiveEnrollment(context.Background()); err == nil {
			t.Fatal("client command accepted")
		}
	}
	t.Setenv("SSH_ORIGINAL_COMMAND", "")
	t.Setenv("SSH_TTY", "/dev/pts/1")
	if err := ReceiveEnrollment(context.Background()); err == nil {
		t.Fatal("PTY enrollment accepted")
	}
}

func TestEnrollmentStatePublishesWholeAndPreservesExisting(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "result")
	value := strings.Repeat("synthetic state\n", 16384)
	done := make(chan error, 1)
	go func() { done <- enrollmentWrite(directory, "result", value) }()
	for {
		data, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if err == nil && string(data) != value {
			t.Fatal("reader observed a partial state receipt")
		}
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			if err := enrollmentWrite(directory, "result", "replacement"); err == nil {
				t.Fatal("existing state was overwritten")
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != value {
				t.Fatal("existing receipt changed")
			}
			entries, err := os.ReadDir(directory)
			if err != nil || len(entries) != 1 {
				t.Fatal("state publication left temporary files")
			}
			return
		default:
		}
	}
}

func enrollmentGuardRunner(t *testing.T, listOutput []byte, listErr error) commandRunner {
	t.Helper()
	return func(_ context.Context, name string, args []string, _ io.Reader) ([]byte, error) {
		if name != "systemctl" || len(args) == 0 {
			t.Fatalf("unexpected guard command %s %v", name, args)
		}
		switch args[0] {
		case "show":
			return []byte("not-found\n"), nil
		case "list-unit-files":
			return listOutput, listErr
		default:
			t.Fatalf("unexpected guard command %s %v", name, args)
			return nil, nil
		}
	}
}

func TestGuardExistingEnrollmentTemplateAbsent(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		listOutput []byte
		listErr    error
	}{
		{"older systemd empty listing", []byte(""), nil},
		{"systemd 259 no-match exit 1", nil, &commandExit{name: "systemctl", code: 1}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			err := guardExistingEnrollmentState(context.Background(), enrollmentGuardRunner(t, scenario.listOutput, scenario.listErr))
			if err == nil {
				// Privileged runs reserve real state; release what this call created.
				if removeErr := os.Remove(enrollmentDir); removeErr != nil {
					t.Fatal(removeErr)
				}
				return
			}
			if strings.Contains(err.Error(), "template") {
				t.Fatalf("absent template refused arming: %v", err)
			}
			if !strings.Contains(err.Error(), "cannot be reserved") {
				t.Fatalf("unexpected guard error: %v", err)
			}
		})
	}
}

func TestGuardExistingEnrollmentTemplateRefused(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		listOutput []byte
		listErr    error
	}{
		{"template listed", []byte("soda-key-enrollment@.service static -\n"), nil},
		{"inspection interrupted", nil, &commandExit{name: "systemctl", code: 1, interrupted: true}},
		{"inspection failed", nil, &commandExit{name: "systemctl", code: -1}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			err := guardExistingEnrollmentState(context.Background(), enrollmentGuardRunner(t, scenario.listOutput, scenario.listErr))
			if err == nil {
				_ = os.Remove(enrollmentDir)
				t.Fatal("guard accepted despite template presence or failed inspection")
			}
			if strings.Contains(err.Error(), "cannot be reserved") {
				t.Fatalf("guard proceeded past the template check: %v", err)
			}
		})
	}
}

func TestEnrollmentAddressSelectionRetriesTypos(t *testing.T) {
	for _, input := range []string{"typo\n0\n3\n2\n", "Back\n", "cancel\n"} {
		t.Run(strings.ReplaceAll(input, "\n", "-"), func(t *testing.T) {
			master, tty := openTestPTY(t)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			addresses := []enrollmentAddress{{"test0", "192.168.1.20"}, {"test1", "10.0.0.20"}}
			if _, err := unix.Write(master, []byte(input)); err != nil {
				t.Fatal(err)
			}
			selected, err := selectEnrollmentAddress(console{tty: tty, ctx: ctx}, addresses)
			if strings.HasPrefix(input, "typo") {
				if err != nil || selected != addresses[1] {
					t.Fatalf("corrected selection failed: %v", err)
				}
				var transcript strings.Builder
				var buffer [4096]byte
				for {
					n, _ := unix.Read(master, buffer[:])
					if n <= 0 {
						break
					}
					transcript.Write(buffer[:n])
				}
				if strings.Count(transcript.String(), "Enter a number from 1 to 2") != 3 {
					t.Fatal("invalid numbers did not receive clear retry feedback")
				}
			} else if err == nil || selected != (enrollmentAddress{}) {
				t.Fatal("explicit back/cancel selected an enrollment address")
			}
		})
	}
}
