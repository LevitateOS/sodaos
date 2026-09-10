package installer

import (
	"context"
	"errors"
	"fmt"
	"io"
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

type ptyDriver struct {
	t          *testing.T
	master     int
	transcript strings.Builder
	cursor     int
}

func (d *ptyDriver) sendAfter(prompt, input string) {
	d.t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var buf [4096]byte
		n, _ := unix.Read(d.master, buf[:])
		if n > 0 {
			d.transcript.Write(buf[:n])
		}
		text := d.transcript.String()
		if offset := strings.Index(text[d.cursor:], prompt); offset >= 0 {
			d.cursor += offset + len(prompt)
			if _, err := unix.Write(d.master, []byte(input+"\n")); err != nil {
				d.t.Fatal(err)
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	d.t.Fatalf("prompt %q not found in %q", prompt, d.transcript.String())
}

func (d *ptyDriver) drain() string {
	d.t.Helper()
	deadline := time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		var buf [4096]byte
		n, _ := unix.Read(d.master, buf[:])
		if n > 0 {
			d.transcript.Write(buf[:n])
			deadline = time.Now().Add(10 * time.Millisecond)
			continue
		}
		time.Sleep(time.Millisecond)
	}
	return d.transcript.String()
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
			driver := &ptyDriver{t: t, master: master}
			driver.sendAfter("Type keep, edit, back, restart, or cancel: ", "edit")
			if failed {
				driver.sendAfter("Type keep, edit, back, restart, or cancel: ", "keep")
			}
			driver.sendAfter("Type yes to use them, edit, back, restart, or cancel: ", "yes")
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("unexpected editor result: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("network review stuck")
			}
			output := driver.drain()
			editor := strings.Index(output, "\x1b[44mEDITOR")
			reset := strings.LastIndex(output, "\x1b[0m\x1b[2J\x1b[HSodaOS")
			if editor < 0 || reset <= editor {
				t.Fatal("editor terminal state was not reset")
			}
			review := strings.Index(output, "Type yes to use them")
			if review < reset {
				t.Fatal("network review missing after editor correction")
			}
			if failed && !strings.Contains(output, "NetworkManager editor failed") {
				t.Fatal("failed editor was not explained before correction")
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

func TestDiskChoicesCorrectInvalidInputWithoutPublicKey(t *testing.T) {
	master, slave := openTestPTY(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := console{tty: slave, ctx: ctx}
	password := "synthetic password only"
	hash := "$6$synthetic$" + strings.Repeat("a", 86)
	commands := []string{}
	run := func(_ context.Context, name string, args []string, input io.Reader) ([]byte, error) {
		commands = append(commands, name+" "+strings.Join(args, " "))
		switch {
		case name == "ip" && len(args) > 0 && args[0] == "-brief":
			return []byte("lo UNKNOWN 127.0.0.1/8\n"), nil
		case name == "ip" && len(args) > 0 && args[0] == "-json":
			return []byte(`[]`), nil
		case name == "openssl":
			provided, err := io.ReadAll(input)
			if err != nil || string(provided) != password+"\n" {
				t.Fatalf("unexpected password hashing input")
			}
			return []byte(hash + "\n"), nil
		default:
			t.Fatalf("unexpected command %s %v", name, args)
			return nil, errors.New("unreachable")
		}
	}
	done := make(chan struct {
		choices diskInstallChoices
		err     error
	}, 1)
	go func() {
		choices, err := collectDiskInstallChoices(ctx, c, run, func(context.Context, commandRunner) ([]Disk, error) {
			return []Disk{fixtureDisk()}, nil
		}, 42<<20)
		done <- struct {
			choices diskInstallChoices
			err     error
		}{choices, err}
	}()
	driver := &ptyDriver{t: t, master: master}
	driver.sendAfter("Type keep, edit, back, restart, or cancel: ", "invalid")
	driver.sendAfter("Type keep, edit, back, restart, or cancel: ", "keep")
	driver.sendAfter("Type yes to use them, edit, back, restart, or cancel: ", "not-yes")
	driver.sendAfter("Type yes to use them, edit, back, restart, or cancel: ", "yes")
	driver.sendAfter("Disk number, back, restart, or cancel: ", "9")
	driver.sendAfter("Disk number, back, restart, or cancel: ", "1")
	driver.sendAfter("Hostname [soda], back, restart, or cancel: ", "Soda")
	driver.sendAfter("Hostname [soda], back, restart, or cancel: ", "soda-fixed")
	driver.sendAfter("Password (at least 12 characters): ", "short")
	driver.sendAfter("Confirm password: ", "short")
	driver.sendAfter("Password (at least 12 characters): ", password)
	driver.sendAfter("Confirm password: ", "different password")
	driver.sendAfter("Password (at least 12 characters): ", password)
	driver.sendAfter("Confirm password: ", password)
	driver.sendAfter("Private project IPv4 subnet [10.89.0.0/24], back, restart, or cancel: ", "8.8.8.0/24")
	driver.sendAfter("Private project IPv4 subnet [10.89.0.0/24], back, restart, or cancel: ", "10.89.0.0/24")
	driver.sendAfter("Type exactly ERASE /dev/sda, back, restart, or cancel: ", "erase /dev/sda")
	driver.sendAfter("Type exactly ERASE /dev/sda, back, restart, or cancel: ", "ERASE /dev/sda")

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatal(got.err)
		}
		if got.choices.disk.Device.Name != "/dev/sda" || got.choices.hostname != "soda-fixed" || got.choices.passwordHash != hash || got.choices.subnet != "10.89.0.0/24" {
			t.Fatalf("unexpected choices: %+v", got.choices)
		}
	case <-ctx.Done():
		t.Fatal("corrected installer input stuck")
	}
	transcript := driver.drain()
	if strings.Contains(transcript, password) {
		t.Fatal("password leaked into a later prompt or transcript")
	}
	if strings.Contains(strings.ToLower(transcript), "public key") || strings.Contains(strings.ToLower(transcript), "fingerprint") {
		t.Fatal("manual disk flow asked for an SSH key")
	}
	for _, command := range commands {
		if strings.HasPrefix(command, "coreos-installer ") {
			t.Fatal("input collection started disk installation")
		}
	}
}

func TestDiskChoicesBackAndRestartBeforeWriting(t *testing.T) {
	master, slave := openTestPTY(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c := console{tty: slave, ctx: ctx}
	run := func(_ context.Context, name string, args []string, _ io.Reader) ([]byte, error) {
		if name == "ip" && len(args) > 0 && args[0] == "-brief" {
			return []byte("lo UNKNOWN 127.0.0.1/8\n"), nil
		}
		t.Fatalf("unexpected command before restart: %s %v", name, args)
		return nil, errors.New("unreachable")
	}
	done := make(chan error, 1)
	inspections := 0
	go func() {
		_, err := collectDiskInstallChoices(ctx, c, run, func(context.Context, commandRunner) ([]Disk, error) {
			inspections++
			return []Disk{fixtureDisk()}, nil
		}, 42<<20)
		done <- err
	}()
	driver := &ptyDriver{t: t, master: master}
	driver.sendAfter("Type keep, edit, back, restart, or cancel: ", "keep")
	driver.sendAfter("Type yes to use them, edit, back, restart, or cancel: ", "yes")
	driver.sendAfter("Disk number, back, restart, or cancel: ", "1")
	driver.sendAfter("Hostname [soda], back, restart, or cancel: ", "back")
	driver.sendAfter("Disk number, back, restart, or cancel: ", "restart")
	if err := <-done; !errors.Is(err, errRestart) {
		t.Fatalf("restart not returned: %v", err)
	}
	if inspections != 2 {
		t.Fatalf("Back did not return to disk selection: %d inspections", inspections)
	}
}

func TestDiskChoicesBackPreservesNonSecretDefaults(t *testing.T) {
	master, slave := openTestPTY(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := console{tty: slave, ctx: ctx}
	password := "synthetic password only"
	hash := "$6$synthetic$" + strings.Repeat("a", 86)
	hashCalls := 0
	run := func(_ context.Context, name string, args []string, input io.Reader) ([]byte, error) {
		switch {
		case name == "ip" && len(args) > 0 && args[0] == "-brief":
			return []byte("lo UNKNOWN 127.0.0.1/8\n"), nil
		case name == "ip" && len(args) > 0 && args[0] == "-json":
			return []byte(`[]`), nil
		case name == "openssl":
			hashCalls++
			provided, err := io.ReadAll(input)
			if err != nil || string(provided) != password+"\n" {
				t.Fatal("unexpected password hashing input")
			}
			return []byte(hash + "\n"), nil
		default:
			t.Fatalf("unexpected command %s %v", name, args)
			return nil, errors.New("unreachable")
		}
	}
	done := make(chan struct {
		choices diskInstallChoices
		err     error
	}, 1)
	go func() {
		choices, err := collectDiskInstallChoices(ctx, c, run, func(context.Context, commandRunner) ([]Disk, error) {
			return []Disk{fixtureDisk()}, nil
		}, 42<<20)
		done <- struct {
			choices diskInstallChoices
			err     error
		}{choices, err}
	}()
	driver := &ptyDriver{t: t, master: master}
	driver.sendAfter("Type keep, edit, back, restart, or cancel: ", "keep")
	driver.sendAfter("Type yes to use them, edit, back, restart, or cancel: ", "yes")
	driver.sendAfter("Disk number, back, restart, or cancel: ", "1")
	driver.sendAfter("Hostname [soda], back, restart, or cancel: ", "soda-original")
	driver.sendAfter("Password (at least 12 characters): ", password)
	driver.sendAfter("Confirm password: ", password)
	driver.sendAfter("Private project IPv4 subnet [10.89.0.0/24], back, restart, or cancel: ", "10.90.0.0/24")
	driver.sendAfter("Type exactly ERASE /dev/sda, back, restart, or cancel: ", "back")
	driver.sendAfter("Private project IPv4 subnet [10.90.0.0/24], back, restart, or cancel: ", "back")
	driver.sendAfter("Password (at least 12 characters): ", "back")
	driver.sendAfter("Hostname [soda-original], back, restart, or cancel: ", "soda-corrected")
	driver.sendAfter("Password (at least 12 characters): ", password)
	driver.sendAfter("Confirm password: ", password)
	driver.sendAfter("Private project IPv4 subnet [10.90.0.0/24], back, restart, or cancel: ", "")
	driver.sendAfter("Type exactly ERASE /dev/sda, back, restart, or cancel: ", "ERASE /dev/sda")

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatal(got.err)
		}
		if got.choices.hostname != "soda-corrected" || got.choices.subnet != "10.90.0.0/24" || got.choices.passwordHash != hash {
			t.Fatalf("non-secret defaults were not preserved: %+v", got.choices)
		}
		if hashCalls != 2 {
			t.Fatalf("password was reused instead of collected again: %d hash calls", hashCalls)
		}
	case <-ctx.Done():
		t.Fatal("Back navigation with prior defaults stuck")
	}
	if strings.Contains(driver.drain(), password) {
		t.Fatal("password leaked while navigating Back")
	}
}

func TestPreWriteCancellationCanRestartButMarkerCannot(t *testing.T) {
	t.Run("pre-write restart", func(t *testing.T) {
		master, slave := openTestPTY(t)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		marker := filepath.Join(t.TempDir(), "attempted")
		calls := 0
		done := make(chan error, 1)
		go func() {
			done <- retryDiskInstall(ctx, console{tty: slave, ctx: ctx}, marker, func(context.Context, console) error {
				calls++
				if calls == 1 {
					return context.Canceled
				}
				return nil
			})
		}()
		driver := &ptyDriver{t: t, master: master}
		driver.sendAfter("Type restart or quit: ", "restart")
		if err := <-done; err != nil || calls != 2 {
			t.Fatalf("pre-write restart failed: calls=%d err=%v", calls, err)
		}
	})

	t.Run("attempt marker refuses retry", func(t *testing.T) {
		_, slave := openTestPTY(t)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		marker := filepath.Join(t.TempDir(), "attempted")
		calls := 0
		err := retryDiskInstall(ctx, console{tty: slave, ctx: ctx}, marker, func(context.Context, console) error {
			calls++
			if err := os.WriteFile(marker, []byte("/dev/sda\n"), 0600); err != nil {
				t.Fatal(err)
			}
			return context.Canceled
		})
		if err == nil || calls != 1 {
			t.Fatalf("marked disk attempt retried: calls=%d err=%v", calls, err)
		}
	})

	t.Run("terminal lifecycle cancellation exits", func(t *testing.T) {
		_, slave := openTestPTY(t)
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		err := retryDiskInstall(ctx, console{tty: slave, ctx: ctx}, filepath.Join(t.TempDir(), "attempted"), func(context.Context, console) error {
			calls++
			cancel()
			return context.Canceled
		})
		if !errors.Is(err, context.Canceled) || calls != 1 {
			t.Fatalf("terminal lifecycle cancellation restarted: calls=%d err=%v", calls, err)
		}
	})
}
