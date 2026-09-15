// controller runs the admitted soda-build executable and relays its
// stderr event protocol to the progress display. It never admits workers,
// signs, or publishes; those authorities stay with the controller and the
// release configs.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

// controllerRun is a started controller child with its progress view.
type controllerRun struct {
	cmd      *exec.Cmd
	view     *renderer
	tty      bool
	relay    chan os.Signal
	childErr io.Reader
	feedErr  error
}

func controllerEnv(ns int64) []string {
	env := os.Environ()
	if os.Getenv("SODA_BUILD_START_NS") == "" {
		env = append(env, fmt.Sprintf("SODA_BUILD_START_NS=%d", ns))
	}
	return env
}

func controllerArgv(o options, ns int64) []string {
	argv := append([]string{o.controller}, controllerArgs(o)...)
	if os.Geteuid() != 0 {
		argv = append([]string{"sudo", fmt.Sprintf("SODA_BUILD_START_NS=%d", ns)}, argv...)
	}
	return argv
}

// relaySignals forwards terminal interrupts to the controller child.
func relaySignals(cmd *exec.Cmd, relay chan os.Signal) {
	for s := range relay {
		if sig, ok := s.(syscall.Signal); ok {
			_ = cmd.Process.Signal(sig)
		}
	}
}

func startControllerRun(o options, stdin, stdout, stderr *os.File) (*controllerRun, error) {
	ns, err := monotonicNS()
	if err != nil {
		return nil, err
	}
	argv := controllerArgv(o, ns)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir, _ = os.Getwd()
	cmd.Env = controllerEnv(ns)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	childErr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	tty := isTerminal(stderr) && !o.nonInteractive
	view := newRenderer(stderr, tty, termWidth(stderr))
	view.SetOutDir(o.out)
	if err := view.note("soda-candidate: " + describe(o)); err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	relay := make(chan os.Signal, 1)
	signal.Notify(relay, syscall.SIGINT, syscall.SIGTERM)
	go relaySignals(cmd, relay)
	if tty {
		view.startTicker()
	}
	return &controllerRun{cmd: cmd, view: view, tty: tty, relay: relay, childErr: childErr}, nil
}

// relayOutput feeds every child stderr line to the display.
func (r *controllerRun) relayOutput() error {
	sc := bufio.NewScanner(r.childErr)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		if err := r.view.feed(strings.TrimRight(sc.Text(), "\r")); err != nil && r.feedErr == nil {
			r.feedErr = err
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return r.feedErr
}

// waitExit reaps the child and closes the display with its exit code.
func (r *controllerRun) waitExit() (int, error) {
	waitErr := r.cmd.Wait()
	if r.tty {
		r.view.stopTicker()
	}
	code := 0
	var exit *exec.ExitError
	if errors.As(waitErr, &exit) {
		code = exit.ExitCode()
	} else if waitErr != nil {
		return 0, waitErr
	}
	if err := r.view.finish(code); err != nil {
		return 0, err
	}
	return code, nil
}

func (r *controllerRun) wait() (int, error) {
	defer signal.Stop(r.relay)
	if err := r.relayOutput(); err != nil {
		return 0, err
	}
	return r.waitExit()
}

// maybeServeFixture serves the loopback pickup address itself for
// development media: the installer needs a live address, and the operator
// should never hand-run a file server. Anything else returns a no-op stop.
func maybeServeFixture(o options) (func(), error) {
	if !fixtureWanted(o.mode, o.rootfsURL) {
		return func() {}, nil
	}
	addr, err := fixtureAddr(o.rootfsURL)
	if err != nil {
		return nil, err
	}
	stop, _, err := serveFixture(addr, o.rootfsDir)
	if err != nil {
		return nil, err
	}
	return stop, nil
}

func fileBuiltRootfs(o options, stderr *os.File) error {
	if !fixtureWanted(o.mode, o.rootfsURL) {
		return nil
	}
	name, err := copyBuiltRootfs(o.out, o.rootfsDir)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stderr, "soda-candidate: serving "+name+" from "+strings.TrimSuffix(o.rootfsURL, "/")+"/"+name)
	return err
}
