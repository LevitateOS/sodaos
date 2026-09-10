package installer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type commandRunner func(context.Context, string, []string, io.Reader) ([]byte, error)

// Command diagnostics can contain private values (especially provisioning). Only
// bounded stdout explicitly consumed by the caller is returned; raw stderr is not
// copied to the terminal, journal or evidence.
func command(ctx context.Context, name string, args []string, input io.Reader) ([]byte, error) {
	phase, cancel := context.WithTimeout(ctx, 2*time.Hour)
	defer cancel()
	cmd := exec.CommandContext(phase, name, args...)
	cmd.Stdin = input
	cmd.WaitDelay = 2 * time.Second
	output := &boundedOutput{}
	cmd.Stdout = output
	if err := cmd.Run(); err != nil {
		code := -1
		if cmd.ProcessState != nil {
			code = cmd.ProcessState.ExitCode()
		}
		return nil, &commandExit{name: name, code: code, interrupted: phase.Err() != nil}
	}
	return output.data, nil
}

type commandExit struct {
	name        string
	code        int
	interrupted bool
}

func (e *commandExit) Error() string {
	return fmt.Sprintf("%s failed (exit %d, interrupted %t); raw diagnostics suppressed", e.name, e.code, e.interrupted)
}
func failureSummary(err error) string {
	var result *commandExit
	if errors.As(err, &result) {
		return result.Error()
	}
	return "native command failed; raw diagnostics suppressed"
}

type boundedOutput struct{ data []byte }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if len(b.data)+len(p) > 8<<20 {
		return 0, errors.New("command output exceeded bound")
	}
	b.data = append(b.data, p...)
	return len(p), nil
}

type console struct {
	tty *os.File
	ctx context.Context
}

func (c console) print(format string, args ...interface{}) { fmt.Fprintf(c.tty, format+"\n", args...) }
func (c console) ask(prompt string) (string, error) {
	fmt.Fprint(c.tty, prompt+": ")
	return c.line()
}
func (c console) line() (string, error) {
	var data []byte
	var b [1]byte
	for len(data) <= 16384 {
		if err := c.ctx.Err(); err != nil {
			return "", err
		}
		events := []unix.PollFd{{Fd: int32(c.tty.Fd()), Events: unix.POLLIN}}
		if _, err := unix.Poll(events, 100); err != nil && err != unix.EINTR {
			return "", err
		}
		if events[0].Revents&(unix.POLLHUP|unix.POLLERR|unix.POLLNVAL) != 0 {
			return "", errors.New("terminal disconnected")
		}
		if events[0].Revents&unix.POLLIN == 0 {
			continue
		}
		n, err := c.tty.Read(b[:])
		if err != nil {
			return "", err
		}
		if n == 0 {
			return "", io.EOF
		}
		if b[0] == '\n' {
			return strings.TrimSpace(string(data)), nil
		}
		if b[0] < 32 && b[0] != '\t' {
			return "", errors.New("control character refused")
		}
		data = append(data, b[0])
	}
	return "", errors.New("input exceeds limit")
}
func (c console) secret(prompt string) (string, error) {
	fd := int(c.tty.Fd())
	state, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return "", errors.New("password entry requires a terminal")
	}
	hidden := *state
	hidden.Lflag &^= unix.ECHO | unix.ECHONL
	// Signals terminate input through context handling after echo is restored.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)
	if err = unix.IoctlSetTermios(fd, unix.TCSETS, &hidden); err != nil {
		return "", err
	}
	// Flush queued private input before restoring echo, including overlong entry
	// and cancellation. It must not become visible input to a later prompt.
	defer unix.IoctlSetTermios(fd, unix.TCSETSF, state)
	fmt.Fprint(c.tty, prompt+": ")
	// Poll rather than leaving a password-reading goroutine behind on cancellation.
	var data []byte
	for len(data) <= 1024 {
		if err := c.ctx.Err(); err != nil {
			return "", err
		}
		select {
		case <-signals:
			c.print("")
			return "", errors.New("password entry cancelled")
		default:
		}
		events := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		if _, e := unix.Poll(events, 100); e != nil && e != unix.EINTR {
			return "", e
		}
		if events[0].Revents&(unix.POLLHUP|unix.POLLERR|unix.POLLNVAL) != 0 {
			return "", errors.New("terminal disconnected")
		}
		if events[0].Revents&unix.POLLIN == 0 {
			continue
		}
		var b [1]byte
		n, e := c.tty.Read(b[:])
		if e != nil || n == 0 {
			return "", errors.New("password input ended")
		}
		if b[0] == '\n' {
			c.print("")
			return string(data), nil
		}
		if b[0] < 32 || b[0] == 127 {
			return "", errors.New("password contains control characters")
		}
		data = append(data, b[0])
	}
	return "", errors.New("password exceeds limit")
}

func (c console) network(ctx context.Context) error {
	c.print("Networking uses NetworkManager. DHCP is the default. Static address/prefix, gateway and DNS can be edited in nmtui. This changes only the live system until disk installation copies it.")
	choice, err := c.ask("Enter edit to open nmtui, or keep to keep current networking")
	if err != nil {
		return err
	}
	switch choice {
	case "edit":
		cmd := exec.CommandContext(ctx, "nmtui")
		cmd.Stdin, cmd.Stdout, cmd.Stderr = c.tty, c.tty, c.tty
		err := cmd.Run()
		// NEWT leaves its background/cursor position behind when it exits.
		// Restore our page on both success and failure, before any next prompt.
		c.print("\x1b[0m\x1b[2J\x1b[HSodaOS installation — Network settings")
		if err != nil {
			return errors.New("NetworkManager editor failed; no disk installation started")
		}
	case "keep":
	default:
		return errors.New("network configuration cancelled")
	}
	data, err := command(ctx, "ip", []string{"-brief", "address"}, nil)
	if err != nil {
		return err
	}
	// Quote native output so a configured interface name cannot inject terminal controls.
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		c.print("%q", line)
	}
	answer, err := c.ask("Use these network settings for the installed system? Type yes")
	if err != nil {
		return err
	}
	if answer != "yes" {
		return errors.New("network confirmation cancelled")
	}
	return nil
}
