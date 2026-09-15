package installer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

type enrollmentSession struct {
	run            commandRunner
	started        bool
	publishedUnits []string
}

func (s *enrollmentSession) stopUnits() (bool, error) {
	if !s.started {
		return true, nil
	}
	cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Stopping the socket first waits for its BindsTo/After connection
	// instances. Only these exact owned names are stopped, never a glob.
	_, socketErr := s.run(cleanup, "systemctl", []string{"stop", enrollmentSocketUnit}, nil)
	_, stopErr := s.run(cleanup, "systemctl", []string{"stop", enrollmentUnit}, nil)
	stopped := stopErr == nil
	if !stopped {
		// --collect may already have unloaded a successful unit. Confirm
		// both manager absence and cgroup removal before treating that
		// stop error as an already completed close.
		load, inspectErr := s.run(cleanup, "systemctl", []string{"show", "--property=LoadState", "--value", enrollmentUnit}, nil)
		_, groupErr := os.Lstat("/sys/fs/cgroup/system.slice/" + enrollmentUnit)
		stopped = inspectErr == nil && strings.TrimSpace(string(load)) == "not-found" && errors.Is(groupErr, os.ErrNotExist)
	}
	stopped = stopped && socketErr == nil
	if !stopped {
		return false, errors.New("enrollment close could not be confirmed; the five-minute native service limit still applies; preserve this attempt and inspect locally")
	}
	return true, nil
}

func (s *enrollmentSession) cleanupUnits() error {
	var result error
	for _, name := range s.publishedUnits {
		if err := os.Remove(filepath.Join(enrollmentUnitDirectory, name)); err != nil {
			result = errors.New("owned enrollment unit cleanup failed; inspect locally before reopening")
		}
	}
	if len(s.publishedUnits) > 0 {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := s.run(cleanup, "systemctl", []string{"daemon-reload"}, nil); err != nil {
			result = errors.New("owned enrollment unit removal could not be reloaded; inspect locally before reopening")
		}
	}
	// Exact run-owned files only. Never remove another tree or stale evidence.
	for _, name := range []string{"armed", "sshd_config", "receive.sock", "ready", "result"} {
		_ = os.Remove(filepath.Join(enrollmentDir, name))
	}
	_ = os.Remove(enrollmentDir)
	return result
}

func (s *enrollmentSession) close() error {
	stopped, err := s.stopUnits()
	if stopped {
		if cleanErr := s.cleanupUnits(); cleanErr != nil {
			return cleanErr
		}
	}
	return err
}

func writeEnrollmentConfig(ctx context.Context, run commandRunner, selected enrollmentAddress) error {
	now, err := enrollmentBootSeconds()
	if err != nil {
		return err
	}
	if err := enrollmentWrite(enrollmentDir, "armed", fmt.Sprintf("%s %s %d\n", selected.name, selected.ip, now+300)); err != nil {
		return err
	}
	if err := enrollmentWrite(enrollmentDir, "sshd_config", enrollmentConfig()); err != nil {
		return err
	}
	if _, err := run(ctx, "/usr/sbin/sshd", []string{"-t", "-f", enrollmentConfigPath}, nil); err != nil {
		return errors.New("native SSH configuration check failed; no listener opened")
	}
	return nil
}

func publishEnrollmentUnits(ctx context.Context, session *enrollmentSession, ip string) error {
	unitDirectory, err := unix.Open(enrollmentUnitDirectory, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return errors.New("native runtime unit directory unavailable")
	}
	defer unix.Close(unitDirectory)
	if err := enrollmentSafeDirectory(unitDirectory, 0); err != nil {
		return err
	}
	socketConfig, err := enrollmentSocketUnitConfig(ip)
	if err != nil {
		return err
	}
	for _, unit := range []struct{ name, contents string }{{enrollmentSocketUnit, socketConfig}, {enrollmentTemplateUnit, enrollmentTemplateUnitConfig()}} {
		if err := enrollmentWrite(enrollmentUnitDirectory, unit.name, unit.contents); err != nil {
			return errors.New("runtime enrollment unit already exists or cannot be published; no replacement")
		}
		session.publishedUnits = append(session.publishedUnits, unit.name)
	}
	if _, err := session.run(ctx, "systemctl", []string{"daemon-reload"}, nil); err != nil {
		return errors.New("native enrollment units could not be loaded")
	}
	return nil
}

func startEnrollmentService(ctx context.Context, session *enrollmentSession) error {
	// Mark before starting: a lost systemd-run reply is not permission to leave
	// a possibly started window running without the cleanup attempt.
	session.started = true
	if _, err := session.run(ctx, "systemd-run", enrollmentStartArgs(), nil); err != nil {
		return errors.New("temporary enrollment service could not start")
	}
	return nil
}

func publishEnrollmentState(ctx context.Context, session *enrollmentSession, selected enrollmentAddress) error {
	if err := writeEnrollmentConfig(ctx, session.run, selected); err != nil {
		return err
	}
	if err := publishEnrollmentUnits(ctx, session, selected.ip); err != nil {
		return err
	}
	return startEnrollmentService(ctx, session)
}

func checkEnrollmentResult(c console, ip string) (bool, error) {
	data, err := readRegular(enrollmentDir+"/result", 128)
	if err != nil {
		return false, nil
	}
	if string(data) == "uncertain\n" {
		return true, errEnrollmentWriteUncertain
	}
	if string(data) != "imported\n" {
		return true, errors.New("key import failed; preserve existing access and inspect locally")
	}
	c.print("Public key installed. The import window is closing. Verify a NEW ordinary key-only SSH login from the laptop before continuing.")
	c.print("Use the laptop private-key path matching the public .pub file you imported:")
	c.print("ssh -i ~/.ssh/id_ed25519 -o IdentitiesOnly=yes -o PreferredAuthentications=publickey -o PasswordAuthentication=no -o KbdInteractiveAuthentication=no -o ControlMaster=no -o ControlPath=none root@%s", ip)
	return true, nil
}

func notifyEnrollmentReady(c console, selected enrollmentAddress, ready *bool) {
	if !*ready {
		if _, err := os.Lstat(enrollmentDir + "/ready"); err == nil {
			*ready = true
			laptop, _ := enrollmentClientCommand(selected.ip)
			c.print("On the laptop, check the host fingerprint above, then run (adjust only your PUBLIC .pub file path):\n%s", laptop)
			c.print("Waiting for one public key. Press Enter to cancel; do not close this console while importing.")
		}
	}
}

func pollConsoleCancel(c console, cancelConsole console) error {
	// A short poll keeps cancellation responsive without abandoning a reader
	// goroutine that could consume a subsequent wizard answer.
	events := []unix.PollFd{{Fd: int32(c.tty.Fd()), Events: unix.POLLIN}}
	if _, err := unix.Poll(events, 200); err != nil && err != unix.EINTR {
		return err
	}
	if events[0].Revents&(unix.POLLHUP|unix.POLLERR|unix.POLLNVAL) != 0 {
		return errors.New("local console disconnected; closing enrollment")
	}
	if events[0].Revents&unix.POLLIN != 0 {
		_, _ = cancelConsole.line()
		return errors.New("key enrollment cancelled locally")
	}
	return nil
}

func checkEnrollmentServiceActive(ctx context.Context, run commandRunner) error {
	state, err := run(ctx, "systemctl", []string{"show", "--property=ActiveState", "--value", enrollmentUnit}, nil)
	active := strings.TrimSpace(string(state))
	if err != nil || (active != "active" && active != "activating") {
		// The receipt may have appeared since the start of this loop.
		if _, err := os.Lstat(enrollmentDir + "/result"); err == nil {
			return nil
		}
		return errors.New("enrollment service closed without a receipt; a key may have been imported; verify native access locally before trying another import")
	}
	return nil
}

func waitEnrollmentResult(ctx context.Context, c console, run commandRunner, selected enrollmentAddress) error {
	waitCtx, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()
	cancelConsole := console{tty: c.tty, ctx: waitCtx}
	ready := false
	lastStateCheck := time.Time{}
	for {
		if done, err := checkEnrollmentResult(c, selected.ip); done {
			return err
		}
		if err := waitCtx.Err(); err != nil {
			return errors.New("key enrollment closed without confirmed import")
		}
		if _, _, err := enrollmentState(); err != nil {
			return errors.New("key enrollment window expired; closing it now")
		}
		notifyEnrollmentReady(c, selected, &ready)
		if err := pollConsoleCancel(c, cancelConsole); err != nil {
			return err
		}
		if time.Since(lastStateCheck) >= time.Second {
			lastStateCheck = time.Now()
			if err := checkEnrollmentServiceActive(waitCtx, run); err != nil {
				return err
			}
		}
	}
}

func armEnrollment(ctx context.Context, c console, run commandRunner) (result error) {
	selected, err := selectEnrollmentTarget(ctx, c, run)
	if err != nil {
		return err
	}
	if err = guardExistingEnrollmentState(ctx, run); err != nil {
		return err
	}
	session := enrollmentSession{run: run}
	defer func() {
		if closeErr := session.close(); closeErr != nil {
			result = closeErr
		}
	}()
	if err = publishEnrollmentState(ctx, &session, selected); err != nil {
		return err
	}
	return waitEnrollmentResult(ctx, c, run, selected)
}
