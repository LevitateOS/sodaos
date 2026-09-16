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
	if media.Format == 2 {
		factory, err := readRegular("/usr/share/soda/defaults/host.example.json", 16384)
		if err != nil {
			return nil, err
		}
		return candidateDestination(template, factory, choices)
	}
	return Destination(template, choices.hostname, "", choices.passwordHash, choices.subnet)
}

func attachContinuation(media mediaIdentity, destination []byte) ([]byte, error) {
	if media.Format != 0 {
		return destination, nil
	}
	binary, err := readRegular(installerBinary, 64<<20)
	if err != nil {
		return nil, err
	}
	return addContinuation(destination, binary)
}

func printDiskComplete(c console, media mediaIdentity) {
	if media.Format == 2 {
		c.print("SodaOS disk installation completed with all five application images local.")
		c.print("Remove installation media and reboot explicitly; log in locally as root with your password.")
		c.print("Native startup imports the included images before starting their services. No reboot was performed.")
		c.print("SSH password access is enabled; log in as root over SSH with your password.")
		c.print("To go key-only later, run locally after reboot: %s enroll-key, then disable password logins yourself.", candidateInstallerBinary)
		c.print("Then complete browser setup from your SSH terminal: %s configure", candidateInstallerBinary)
		return
	}
	c.print("CoreOS disk installation completed; Soda setup is not complete.")
	c.print("Remove installation media and reboot explicitly.")
	c.print("On the installed system run: sudo %s continue", installerBinary)
	c.print("No reboot was performed.")
}

func collectDiskAttempt(ctx context.Context, c console) (mediaIdentity, diskInstallChoices, uint64, error) {
	var media mediaIdentity
	if err := build.ReadJSON(filepath.Join(dataDir, "media.json"), &media); err != nil {
		return media, diskInstallChoices{}, 0, errors.New("missing media identity")
	}
	payloadBytes, err := payloadRequirement(media)
	if err != nil {
		return media, diskInstallChoices{}, 0, err
	}
	choices, err := collectDiskInstallChoices(ctx, c, command, scanDisks, payloadBytes)
	return media, choices, payloadBytes, err
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
	c.print("Press Enter to begin. Ctrl-C cancels safely before disk writing.")
	_, err = c.line()
	return err
}

func finishDiskAttempt(ctx context.Context, c console, disk Disk, media mediaIdentity, payloadBytes uint64) error {
	if media.Format != 2 {
		if err := copyInstalledPayload(ctx, disk, media, payloadBytes, command); err != nil {
			return err
		}
	}
	printDiskComplete(c, media)
	return nil
}

func installDiskAttempt(ctx context.Context, c console, marker string) error {
	if err := beginDiskAttempt(c); err != nil {
		return err
	}
	media, choices, payloadBytes, err := collectDiskAttempt(ctx, c)
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
	destination, err = attachContinuation(media, destination)
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
	if err := executeAttemptDisk(ctx, choices.disk, ignition, marker); err != nil {
		return err
	}
	return finishDiskAttempt(ctx, c, choices.disk, media, payloadBytes)
}

func executeAttemptDisk(ctx context.Context, disk Disk, ignition, marker string) error {
	return executeDisk(ctx, disk, ignition, func() ([]Disk, error) { return scanDisks(ctx, command) }, func() error {
		return build.WriteNew(marker, []byte(disk.Device.Name+"\n"), 0o600)
	}, command)
}

func executeDisk(ctx context.Context, selected Disk, ignition string, inspect func() ([]Disk, error), mark func() error, run commandRunner) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	observed, err := inspect()
	if err != nil {
		return err
	}
	if err := sameDisk(selected, observed); err != nil {
		return err
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
