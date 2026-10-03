package installer

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/levitateos/sodaos/internal/release/build"
)

func installDisk(ctx context.Context, c console) error {
	return installDiskAt(ctx, c, diskAttemptMarker)
}

func installDiskAt(ctx context.Context, c console, marker string) error {
	return retryDiskInstall(ctx, c, marker, func(attemptCtx context.Context, attemptConsole console) error {
		return installDiskAttempt(attemptCtx, attemptConsole, marker)
	})
}

func askRestartOrQuit(c console) (restart bool, err error) {
	c.page("Installation cancelled before disk writing")
	c.print("No disk installation was started.")
	c.print("Restart reuses this already loaded installer executable.")
	for {
		choice, askErr := c.ask("Type restart or quit")
		if askErr != nil {
			return false, askErr
		}
		switch strings.ToLower(choice) {
		case "restart":
			return true, nil
		case "quit", "cancel":
			return false, errors.New("cancelled; no disk installation started")
		default:
			c.print("Choose restart or quit.")
		}
	}
}

func afterFailedDiskAttempt(c console, marker string, err error, interrupted bool) (retry bool, out error) {
	started, markerErr := diskInstallationStarted(marker)
	if markerErr != nil {
		return false, markerErr
	}
	if started || (!errors.Is(err, errRestart) && !errors.Is(err, errCancel) && !interrupted) {
		return false, err
	}
	if errors.Is(err, errRestart) {
		return true, nil
	}
	return askRestartOrQuit(c)
}

func retryDiskInstall(ctx context.Context, c console, marker string, attempt func(context.Context, console) error) error {
	for {
		started, err := diskInstallationStarted(marker)
		if err != nil {
			return err
		}
		if started {
			return errors.New("disk installation was already attempted this boot; inspect the result, do not replay")
		}
		attemptCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT)
		err = attempt(attemptCtx, console{tty: c.tty, ctx: attemptCtx})
		interrupted := ctx.Err() == nil && (attemptCtx.Err() != nil || errors.Is(err, context.Canceled))
		stop()
		if err == nil {
			return nil
		}
		retry, err := afterFailedDiskAttempt(c, marker, err, interrupted)
		if err != nil || !retry {
			return err
		}
	}
}

func diskInstallationStarted(marker string) (bool, error) {
	_, err := os.Lstat(marker)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, errors.New("cannot verify disk installation attempt marker")
	}
}

func destinationForMedia(media mediaIdentity, template []byte, choices diskInstallChoices) ([]byte, error) {
	if err := media.validate(media.Release, architecture()); err != nil {
		return nil, err
	}
	factory, err := readRegular("/usr/share/soda/defaults/host.example.json", 16384)
	if err != nil {
		return nil, err
	}
	return candidateDestination(template, factory, choices)
}

func printDiskComplete(c console) {
	c.print("SodaOS disk installation completed with all five application images local.")
	c.print("Remove installation media, then confirm the reboot prompt below; log in locally as root with your password.")
	c.print("Native startup imports the included images before starting their services.")
	c.print("SSH password access is enabled; log in as root over SSH with your password.")
	c.print("To go key-only later, run locally after reboot: %s enroll-key, then disable password logins yourself.", candidateInstallerBinary)
	c.print("Then complete browser setup from your SSH terminal: %s configure", candidateInstallerBinary)
}

// rebootAfterInstall closes the live-console installer session. The console
// service runs with Restart=no and the getty masked, so merely returning
// strands the operator on a frozen screen with no way forward; every terminal
// outcome (success, cancel, or failure) therefore ends in a prompted reboot.
// The prompt preserves the completion screen for reading; the operator
// confirms after removing the installation media. A read error (Ctrl-C/EOF)
// still reboots: nothing is unsaved at this point and a frozen console helps
// nobody.
func rebootAfterInstall(ctx context.Context, c console, run commandRunner, installErr error) error {
	if installErr != nil {
		// Post-failure landing: no automatic retry or reboot happened. The
		// prompt below is the inspection window: examine this live boot from
		// another terminal before confirming, because rebooting discards it.
		c.print("Installation did not complete: %v.", installErr)
		c.print("No automatic retry or reboot was performed. Inspect this live boot from another terminal before confirming the reboot below; rebooting discards live-boot inspection state.")
	}
	c.print("Press Enter to reboot the machine.")
	_, _ = c.line()
	c.print("Rebooting...")
	if _, err := run(ctx, "systemctl", []string{"reboot"}, nil); err != nil {
		if installErr != nil {
			return installErr
		}
		return err
	}
	return installErr
}

// verifyDiskMedia hashes the full media payload (gigabytes on slow drives).
// It runs before any prompt so the operator never waits on a silent screen.
func verifyDiskMedia() (mediaIdentity, uint64, error) {
	var media mediaIdentity
	if err := build.ReadJSON(filepath.Join(dataDir, "media.json"), &media); err != nil {
		return media, 0, errors.New("missing media identity")
	}
	payloadBytes, err := candidateRequirement(media, "/")
	if err != nil {
		return media, 0, err
	}
	return media, payloadBytes, nil
}

func writeAttemptIgnition(destination []byte) (string, error) {
	work, err := os.MkdirTemp("/run", "soda-installer-")
	if err != nil {
		return "", err
	}
	ignition := filepath.Join(work, "destination.ign")
	if err := build.WriteNew(ignition, destination, 0o600); err != nil {
		return "", err
	}
	return ignition, nil
}

func beginDiskAttempt(c console) error {
	welcome, err := readRegular("/etc/motd", 16384)
	if err != nil {
		return errors.New("cannot read installer welcome text")
	}
	c.print("\x1b[0m\x1b[2J\x1b[H%s", string(welcome))
	c.print("Checking installation media. Please wait; this hashes gigabytes and takes a while on slow drives.")
	return nil
}

// promptDiskAttempt runs only after media verification, so Enter leads
// straight into the first step with no further silent work.
func promptDiskAttempt(c console) error {
	c.print("Media verified. Press Enter to begin. Ctrl-C cancels safely before disk writing.")
	_, err := c.line()
	return err
}

func finishDiskAttempt(c console) error {
	printDiskComplete(c)
	return nil
}

func installDiskAttempt(ctx context.Context, c console, marker string) error {
	if err := beginDiskAttempt(c); err != nil {
		return err
	}
	media, payloadBytes, err := verifyDiskMedia()
	if err != nil {
		return err
	}
	if err := promptDiskAttempt(c); err != nil {
		return err
	}
	choices, err := collectDiskInstallChoices(ctx, c, command, scanDisks, payloadBytes)
	if err != nil {
		return err
	}
	template, err := readRegular(filepath.Join(dataDir, "destination.ign"), 4<<20)
	if err != nil {
		return err
	}
	destination, err := destinationForMedia(media, template, choices)
	choices.passwordHash = ""
	if err != nil {
		return err
	}
	ignition, err := writeAttemptIgnition(destination)
	if err != nil {
		return err
	}
	c.page("Installing CoreOS")
	c.print("Writing the confirmed disk. Do not disconnect it.")
	c.print("Raw diagnostics are suppressed to protect provisioning inputs.")
	if err := executeAttemptDisk(ctx, choices.disk, ignition, marker, choices.removableOK); err != nil {
		return err
	}
	return finishDiskAttempt(c)
}

func executeAttemptDisk(ctx context.Context, disk Disk, ignition, marker string, removableConfirmed bool) error {
	return executeDisk(ctx, disk, ignition, func() ([]Disk, error) { return scanDisks(ctx, command) }, func() error {
		return build.WriteNew(marker, []byte(disk.Device.Name+"\n"), 0o600)
	}, command, removableConfirmed)
}

func executeDisk(ctx context.Context, selected Disk, ignition string, inspect func() ([]Disk, error), mark func() error, run commandRunner, removableConfirmed bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Blocked or unavailable selections never reach a write, even when the
	// fresh inventory matches them exactly.
	if selected.Blocked != "" {
		return errors.New("selected disk is unavailable (" + selected.Blocked + "); no installation started")
	}
	observed, err := inspect()
	if err != nil {
		return err
	}
	if err := sameDisk(selected, observed); err != nil {
		return err
	}
	for _, current := range observed {
		if current.Device.Name == selected.Device.Name && current.Blocked != "" {
			return errors.New("selected disk became unavailable (" + current.Blocked + "); no installation started")
		}
	}
	// Removable targets write only after the explicit removable
	// confirmation travelled with this attempt.
	if selected.Removable && !removableConfirmed {
		return errors.New("removable disk " + selected.Device.Name + " needs explicit intentional confirmation; no installation started")
	}
	if err := mark(); err != nil {
		return errors.New("cannot reserve disk installation attempt")
	}
	_, err = run(ctx, "coreos-installer", []string{"install", "--offline", "--ignition-file", ignition, "--copy-network", selected.Device.Name}, nil)
	if err != nil {
		return errors.New("CoreOS installation failed or was interrupted; disk may be partially written. No retry or reboot was performed; preserve this boot for operator inspection. " + failureSummary(err))
	}
	return nil
}

func readRegular(path string, limit int64) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > limit {
		return nil, errors.New("bounded regular file required")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("file exceeds size limit")
	}
	return bytes.Clone(data), nil
}
