package installer

import (
	"context"
	"errors"
	"os"
	"runtime"
	"syscall"

	"github.com/levitateos/sodaos/internal/platform"
	"github.com/levitateos/sodaos/internal/release/build"
	"golang.org/x/sys/unix"
)

const (
	dataDir           = "/usr/local/share/soda-installer"
	installerBinary   = platform.Libexec + "/soda-install"
	diskAttemptMarker = "/run/soda-installer-disk-started"
)

var (
	errBack    = errors.New("back requested")
	errRestart = errors.New("restart requested")
	errCancel  = errors.New("cancel requested")
)

type mediaIdentity struct {
	Format                                            int
	Architecture, Release, InstallerVersion, Revision string
	BundleSHA256                                      string
	HostManifest, PayloadSHA256, ConsoleSHA256        string
}

func (m mediaIdentity) validate(imageVersion, arch string) error {
	if m.Architecture != arch || m.Release == "" || m.Release != imageVersion || !build.Revision(m.Revision) || !m.validContent() {
		return errors.New("media release, architecture, or included-payload mismatch")
	}
	return nil
}

func architecture() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	}
	return "unsupported"
}

// Run requires a controlling terminal. The live service starts disk review;
// installed-host continuation is explicitly operator-started.
func runEnrollmentAction(ctx context.Context, action string) (bool, error) {
	switch action {
	case "enrollment-serve":
		if err := coreOSHost(false); err != nil {
			return true, err
		}
		return true, ServeEnrollment(ctx)
	case "enrollment-receive":
		if err := coreOSHost(false); err != nil {
			return true, err
		}
		return true, ReceiveEnrollment(ctx)
	}
	return false, nil
}

func lockInstaller() (*os.File, error) {
	lock, err := os.OpenFile("/run/soda-installer.lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, errors.New("cannot open installer lock")
	}
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		lock.Close()
		return nil, errors.New("another installer is active")
	}
	return lock, nil
}

func runLockedInstall(ctx context.Context, c console, action string) error {
	switch action {
	case "continue":
		return continueInstall(ctx, c, command)
	case "configure":
		return configureInstall(ctx, c, command)
	case "enroll-key":
		return armEnrollment(ctx, c, command)
	default:
		return installDisk(ctx, c)
	}
}

func validInstallAction(action string) bool {
	return action == "disk" || action == "continue" || action == "configure" || action == "enroll-key"
}

func Run(ctx context.Context, action string) error {
	if os.Geteuid() != 0 || runtime.GOOS != "linux" {
		return errors.New("native CoreOS root required")
	}
	if handled, err := runEnrollmentAction(ctx, action); handled {
		return err
	}
	if !validInstallAction(action) {
		return errors.New("usage: soda-install disk|continue|configure|enroll-key")
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return errors.New("interactive operator terminal required")
	}
	defer tty.Close()
	c := console{tty: tty, ctx: ctx}
	if err := coreOSHost(action == "disk"); err != nil {
		return err
	}
	// One local caller, including when different consoles are active. Lock file is
	// not a success marker and is never removed to pretend a partial attempt is new.
	lock, err := lockInstaller()
	if err != nil {
		return err
	}
	defer lock.Close()
	defer unix.Flock(int(lock.Fd()), unix.LOCK_UN)
	return runLockedInstall(ctx, c, action)
}
