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
func (c console) page(title string) {
	c.print("\x1b[0m\x1b[2J\x1b[HSodaOS installation")
	c.print("")
	c.print(title)
	c.print("")
}

func (c console) ask(prompt string) (string, error) {
	fmt.Fprint(c.tty, prompt+": ")
	return c.line()
}

func pollTTY(fd int, timeout int) (readable bool, err error) {
	events := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	if _, err := unix.Poll(events, timeout); err != nil && err != unix.EINTR {
		return false, err
	}
	if events[0].Revents&(unix.POLLHUP|unix.POLLERR|unix.POLLNVAL) != 0 {
		return false, errors.New("terminal disconnected")
	}
	return events[0].Revents&unix.POLLIN != 0, nil
}

func appendConsoleByte(data []byte, b byte) ([]byte, bool, error) {
	if b == '\n' {
		return data, true, nil
	}
	if b < 32 && b != '\t' {
		return nil, false, errors.New("control character refused")
	}
	return append(data, b), false, nil
}

func (c console) line() (string, error) {
	var data []byte
	var b [1]byte
	for len(data) <= 16384 {
		if err := c.ctx.Err(); err != nil {
			return "", err
		}
		readable, err := pollTTY(int(c.tty.Fd()), 100)
		if err != nil {
			return "", err
		}
		if !readable {
			continue
		}
		n, err := c.tty.Read(b[:])
		if err != nil {
			return "", err
		}
		if n == 0 {
			return "", io.EOF
		}
		var done bool
		data, done, err = appendConsoleByte(data, b[0])
		if err != nil {
			return "", err
		}
		if done {
			return strings.TrimSpace(string(data)), nil
		}
	}
	return "", errors.New("input exceeds limit")
}

func hideTerminalEcho(fd int) (*unix.Termios, error) {
	state, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, errors.New("password entry requires a terminal")
	}
	hidden := *state
	hidden.Lflag &^= unix.ECHO | unix.ECHONL
	if err = unix.IoctlSetTermios(fd, unix.TCSETS, &hidden); err != nil {
		return nil, err
	}
	return state, nil
}

func secretInterrupted(signals <-chan os.Signal) error {
	select {
	case sig := <-signals:
		if sig == syscall.SIGINT {
			return context.Canceled
		}
		return errors.New("password entry terminated")
	default:
		return nil
	}
}

func (c console) consumeSecretByte(data []byte) ([]byte, bool, error) {
	var b [1]byte
	n, e := c.tty.Read(b[:])
	if e != nil || n == 0 {
		return nil, false, errors.New("password input ended")
	}
	if b[0] == '\n' {
		c.print("")
		return data, true, nil
	}
	if b[0] < 32 || b[0] == 127 {
		return nil, false, errors.New("password contains control characters")
	}
	return append(data, b[0]), false, nil
}

func (c console) secret(prompt string) (string, error) {
	fd := int(c.tty.Fd())
	state, err := hideTerminalEcho(fd)
	if err != nil {
		return "", err
	}
	// Signals terminate input through context handling after echo is restored.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)
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
		if err := secretInterrupted(signals); err != nil {
			c.print("")
			return "", err
		}
		readable, e := pollTTY(fd, 100)
		if e != nil {
			return "", e
		}
		if !readable {
			continue
		}
		var done bool
		data, done, err = c.consumeSecretByte(data)
		if err != nil {
			return "", err
		}
		if done {
			return string(data), nil
		}
	}
	return "", errors.New("password exceeds limit")
}

func (c console) network(ctx context.Context) error { return c.networkWith(ctx, command) }

func (c console) chooseNetworkAction(openEditor bool) (string, error) {
	if openEditor {
		return "edit", nil
	}
	return c.ask("Type keep, edit, back, restart, or cancel")
}

func (c console) runNetworkEditor(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "nmtui")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = c.tty, c.tty, c.tty
	err := cmd.Run()
	// NEWT leaves its background/cursor position behind when it exits.
	// Restore our page on both success and failure, before any next prompt.
	c.page("Step 1 of 5 — Network")
	if err != nil {
		return "NetworkManager editor failed. No disk installation started."
	}
	return ""
}

func (c console) confirmLiveAddresses() (edit bool, err error) {
	for {
		answer, err := c.ask("Type yes to use them, edit, back, restart, or cancel")
		if err != nil {
			return false, err
		}
		switch strings.ToLower(answer) {
		case "yes":
			return false, nil
		case "edit":
			return true, nil
		case "back":
			return false, errBack
		case "restart":
			return false, errRestart
		case "cancel":
			return false, errCancel
		default:
			c.print("Choose yes, edit, back, restart, or cancel.")
		}
	}
}

func (c console) printLiveAddresses(data []byte) {
	c.print("")
	c.print("Current live addresses:")
	// Quote native output so a configured interface name cannot inject terminal controls.
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		c.print("  %q", line)
	}
}

func (c console) applyNetworkChoice(ctx context.Context, choice string) (feedback string, retry bool, err error) {
	switch strings.ToLower(choice) {
	case "back":
		return "", false, errBack
	case "restart":
		return "", false, errRestart
	case "cancel":
		return "", false, errCancel
	case "edit":
		if feedback = c.runNetworkEditor(ctx); feedback != "" {
			return feedback, true, nil
		}
		return "", false, nil
	case "keep":
		return "", false, nil
	default:
		return "Choose keep, edit, back, restart, or cancel.", true, nil
	}
}

func (c console) inspectAndConfirmNetwork(ctx context.Context, run commandRunner) (edit bool, feedback string, err error) {
	data, err := run(ctx, "ip", []string{"-brief", "address"}, nil)
	if err != nil {
		return false, "Could not inspect live network addresses.", nil
	}
	c.printLiveAddresses(data)
	edit, err = c.confirmLiveAddresses()
	return edit, "", err
}

func (c console) networkWith(ctx context.Context, run commandRunner) error {
	feedback := ""
	openEditor := false
	for {
		c.page("Step 1 of 5 — Network")
		if feedback != "" {
			c.print("%s", feedback)
			c.print("")
			feedback = ""
		}
		c.print("DHCP is ready by default.")
		c.print("Use nmtui to set a static address, gateway, or DNS.")
		c.print("The installed system will receive the reviewed live settings.")
		choice, err := c.chooseNetworkAction(openEditor)
		if err != nil {
			return err
		}
		openEditor = false
		retry := false
		feedback, retry, err = c.applyNetworkChoice(ctx, choice)
		if err != nil {
			return err
		}
		if retry {
			continue
		}
		edit, inspectFeedback, err := c.inspectAndConfirmNetwork(ctx, run)
		if err != nil {
			return err
		}
		if inspectFeedback != "" {
			feedback = inspectFeedback
			continue
		}
		if !edit {
			return nil
		}
		openEditor = true
	}
}
