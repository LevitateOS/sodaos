package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func openTestPTY(t *testing.T) (int, *os.File) {
	t.Helper()
	master, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { unix.Close(master) })
	if err := unix.IoctlSetPointerInt(master, unix.TIOCSPTLCK, 0); err != nil {
		t.Fatal(err)
	}
	number, err := unix.IoctlGetInt(master, unix.TIOCGPTN)
	if err != nil {
		t.Fatal(err)
	}
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { slave.Close() })
	return master, slave
}

func TestNetworkEditorRestoresConsole(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(fmt.Sprint(failed), func(t *testing.T) {
			dir := t.TempDir()
			exit := 0
			if failed {
				exit = 1
			}
			for name, body := range map[string]string{
				"nmtui": fmt.Sprintf("#!/bin/sh\nprintf '\\033[44mEDITOR'\nexit %d\n", exit),
				"ip":    "#!/bin/sh\nprintf 'lo UNKNOWN 127.0.0.1/8\\n'\n",
			} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0700); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			master, slave := openTestPTY(t)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			c := console{tty: slave, ctx: ctx}
			done := make(chan error, 1)
			go func() { done <- c.network(ctx) }()
			if _, err := unix.Write(master, []byte("edit\nyes\n")); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if (err != nil) != failed {
					t.Fatalf("unexpected editor result: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("network review stuck")
			}
			var transcript strings.Builder
			var buf [4096]byte
			for {
				n, _ := unix.Read(master, buf[:])
				if n <= 0 {
					break
				}
				transcript.Write(buf[:n])
			}
			output := transcript.String()
			editor := strings.Index(output, "\x1b[44mEDITOR")
			reset := strings.Index(output, "\x1b[0m\x1b[2J\x1b[HSodaOS")
			if editor < 0 || reset < editor {
				t.Fatal("editor terminal state was not reset")
			}
			review := strings.Index(output, "Use these network settings")
			if failed && review >= 0 || !failed && review < reset {
				t.Fatal("unexpected review after editor exit")
			}
		})
	}
}

func TestPasswordTerminalEchoAndCancellation(t *testing.T) {
	for _, cancelInput := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelInput), func(t *testing.T) {
			master, slave := openTestPTY(t)
			original, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c := console{tty: slave, ctx: ctx}
			type result struct {
				value string
				err   error
			}
			done := make(chan result, 1)
			go func() { value, err := c.secret("Operator password"); done <- result{value, err} }()
			var transcript strings.Builder
			read := func() {
				var buf [2048]byte
				n, _ := unix.Read(master, buf[:])
				if n > 0 {
					transcript.Write(buf[:n])
				}
			}
			deadline := time.Now().Add(2 * time.Second)
			for !strings.Contains(transcript.String(), "Operator password: ") && time.Now().Before(deadline) {
				read()
				time.Sleep(time.Millisecond)
			}
			if !strings.Contains(transcript.String(), "Operator password: ") {
				t.Fatal("no prompt")
			}
			hidden, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
			if err != nil || hidden.Lflag&unix.ECHO != 0 {
				t.Fatal("password echo enabled")
			}
			if cancelInput {
				cancel()
			} else {
				if _, err := unix.Write(master, []byte("synthetic password only\n")); err != nil {
					t.Fatal(err)
				}
			}
			select {
			case got := <-done:
				if cancelInput && got.err == nil {
					t.Fatal("cancellation ignored")
				}
				if !cancelInput && (got.err != nil || got.value != "synthetic password only") {
					t.Fatal("password input failed")
				}
			case <-time.After(3 * time.Second):
				t.Fatal("password input stuck")
			}
			restored, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
			if err != nil || restored.Lflag != original.Lflag {
				t.Fatal("terminal echo not restored")
			}
			read()
			if strings.Contains(transcript.String(), "synthetic password only") {
				t.Fatal("password echoed to terminal")
			}
		})
	}
}
