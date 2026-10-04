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

func TestEnrollmentPortLabeled(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		listing string
		want    bool
	}{
		{"absent", "ssh_port_t tcp 22\n", false},
		{"labeled with 22", "ssh_port_t tcp 22222, 22\n", true},
		{"labeled alone", "ssh_port_t                     tcp      22222\n", true},
		{"other type", "unreserved_port_t tcp 22222\n", false},
		{"other protocol", "ssh_port_t udp 22222\n", false},
		{"prefix port", "ssh_port_t tcp 2222\n", false},
		{"empty", "", false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			if got := enrollmentPortLabeled([]byte(scenario.listing)); got != scenario.want {
				t.Fatalf("got %t", got)
			}
		})
	}
}

func TestEnsureEnrollmentPortLabel(t *testing.T) {
	ctx := context.Background()
	t.Run("permissive skips without running semanage", func(t *testing.T) {
		run := func(context.Context, string, []string, io.Reader) ([]byte, error) {
			t.Fatal("semanage must not run without enforcement")
			return nil, nil
		}
		if err := ensureEnrollmentPortLabel(ctx, run, false); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("labeled port needs no change", func(t *testing.T) {
		var calls []string
		run := func(_ context.Context, name string, args []string, _ io.Reader) ([]byte, error) {
			calls = append(calls, name+" "+strings.Join(args, " "))
			return []byte("ssh_port_t tcp 22222, 22\n"), nil
		}
		if err := ensureEnrollmentPortLabel(ctx, run, true); err != nil {
			t.Fatal(err)
		}
		if len(calls) != 1 {
			t.Fatalf("ran %d commands, want list only", len(calls))
		}
	})
	t.Run("missing label is added", func(t *testing.T) {
		var calls []string
		run := func(_ context.Context, name string, args []string, _ io.Reader) ([]byte, error) {
			calls = append(calls, name+" "+strings.Join(args, " "))
			if len(args) > 0 && args[0] == "port" {
				return []byte("ssh_port_t tcp 22\n"), nil
			}
			return nil, nil
		}
		if err := ensureEnrollmentPortLabel(ctx, run, true); err != nil {
			t.Fatal(err)
		}
		if len(calls) != 2 || !strings.Contains(calls[1], "port -a -t ssh_port_t -p tcp 22222") {
			t.Fatalf("calls %v", calls)
		}
	})
	t.Run("missing semanage stays loud when enforcing", func(t *testing.T) {
		run := func(context.Context, string, []string, io.Reader) ([]byte, error) {
			return nil, &commandExit{name: "semanage", code: 127}
		}
		if err := ensureEnrollmentPortLabel(ctx, run, true); err == nil {
			t.Fatal("uninspectable labeling accepted")
		}
	})
}

func TestLabelEnrollmentConfig(t *testing.T) {
	ctx := context.Background()
	run := func(_ context.Context, name string, args []string, _ io.Reader) ([]byte, error) {
		if name != "chcon" {
			t.Fatalf("unexpected command %s", name)
		}
		want := []string{"-t", "etc_t", enrollmentConfigPath}
		if strings.Join(args, " ") != strings.Join(want, " ") {
			t.Fatalf("args %v", args)
		}
		return nil, nil
	}
	if err := labelEnrollmentConfig(ctx, run, true); err != nil {
		t.Fatal(err)
	}
	failing := func(context.Context, string, []string, io.Reader) ([]byte, error) {
		return nil, &commandExit{name: "chcon", code: 1}
	}
	if err := labelEnrollmentConfig(ctx, failing, true); err == nil {
		t.Fatal("failed relabel accepted while enforcing")
	}
	if err := labelEnrollmentConfig(ctx, failing, false); err != nil {
		t.Fatalf("failed relabel refused without enforcement: %v", err)
	}
}
