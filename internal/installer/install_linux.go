package installer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/levitateos/sodaos/internal/installlayout"
	"github.com/levitateos/sodaos/internal/nativebuild"
	"golang.org/x/sys/unix"
)

const (
	dataDir           = "/usr/local/share/soda-installer"
	installerBinary   = installlayout.Libexec + "/soda-install"
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
	if m.Architecture != arch || m.Release == "" || m.Release != imageVersion || !nativebuild.Revision(m.Revision) || !m.validContent() {
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

func coreOSHost(live bool) error {
	// Display branding in /etc must not become a stale copy of base identity.
	data, err := os.ReadFile("/usr/lib/os-release")
	if err != nil {
		return err
	}
	values := osRelease(data)
	if values["ID"] != "fedora" || values["VARIANT_ID"] != "coreos" {
		return errors.New("upstream Fedora CoreOS required")
	}
	cmdline, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		return err
	}
	isLive := false
	for _, arg := range strings.Fields(string(cmdline)) {
		if arg == "coreos.liveiso" || strings.HasPrefix(arg, "coreos.liveiso=") || strings.HasPrefix(arg, "coreos.live.rootfs_url=") {
			isLive = true
		}
	}
	if live != isLive {
		return errors.New("disk action requires the live ISO; continuation requires the installed host")
	}
	if live {
		var media mediaIdentity
		if err := nativebuild.ReadJSON(filepath.Join(dataDir, "media.json"), &media); err != nil {
			return errors.New("missing media identity")
		}
		// Selected CoreOS reports the Fedora major in VERSION_ID (44), and
		// the exact image release in IMAGE_VERSION (44.20260817.3.2).
		if err := media.validate(values["IMAGE_VERSION"], architecture()); err != nil {
			return err
		}
		observed, err := command(context.Background(), "coreos-installer", []string{"--version"}, nil)
		if err != nil || strings.TrimSpace(string(observed)) != media.InstallerVersion {
			return errors.New("unreviewed CoreOS Installer version")
		}
	}
	enforcing, err := os.ReadFile("/sys/fs/selinux/enforce")
	if err != nil || strings.TrimSpace(string(enforcing)) != "1" {
		return errors.New("SELinux must remain enforcing")
	}
	return nil
}

func osRelease(data []byte) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[k] = strings.Trim(v, "\"'")
	}
	return values
}

func routes(ctx context.Context, run commandRunner) ([]string, error) {
	data, err := run(ctx, "ip", []string{"-json", "-4", "route", "show", "table", "all"}, nil)
	if err != nil {
		return nil, err
	}
	var entries []struct {
		Destination string `json:"dst"`
	}
	if err = json.Unmarshal(data, &entries); err != nil {
		return nil, errors.New("cannot inspect current routes")
	}
	var result []string
	for _, entry := range entries {
		result = append(result, entry.Destination)
	}
	return result, nil
}

func installDisk(ctx context.Context, c console) error {
	return installDiskAt(ctx, c, diskAttemptMarker)
}

func installDiskAt(ctx context.Context, c console, marker string) error {
	return retryDiskInstall(ctx, c, marker, func(attemptCtx context.Context, attemptConsole console) error {
		return installDiskAttempt(attemptCtx, attemptConsole, marker)
	})
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
		started, markerErr := diskInstallationStarted(marker)
		if markerErr != nil {
			return markerErr
		}
		if started {
			return err
		}
		if errors.Is(err, errRestart) {
			continue
		}
		if !errors.Is(err, errCancel) && !interrupted {
			return err
		}

		c.page("Installation cancelled before disk writing")
		c.print("No disk installation was started.")
		c.print("Restart reuses this already loaded installer executable.")
		for {
			choice, askErr := c.ask("Type restart or quit")
			if askErr != nil {
				return askErr
			}
			switch strings.ToLower(choice) {
			case "restart":
				// Retry via the outer loop; err is re-read from the marker above.
			case "quit", "cancel":
				return errors.New("cancelled; no disk installation started")
			default:
				c.print("Choose restart or quit.")
				continue
			}
			break
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

type diskInstallChoices struct {
	disk         Disk
	hostname     string
	passwordHash string
	subnet       string
}

func installDiskAttempt(ctx context.Context, c console, marker string) error {
	welcome, err := readRegular("/etc/motd", 16384)
	if err != nil {
		return errors.New("cannot read installer welcome text")
	}
	c.print("\x1b[0m\x1b[2J\x1b[H%s", string(welcome))
	c.print("Press Enter to begin. Ctrl-C cancels safely before disk writing.")
	if _, err := c.line(); err != nil {
		return err
	}

	var media mediaIdentity
	if err := nativebuild.ReadJSON(filepath.Join(dataDir, "media.json"), &media); err != nil {
		return errors.New("missing media identity")
	}
	payloadBytes, err := payloadRequirement(media)
	if err != nil {
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
	var destination []byte
	if media.Format == 2 {
		factory, readErr := readRegular("/usr/share/soda/defaults/host.example.json", 16384)
		if readErr != nil {
			return readErr
		}
		destination, err = candidateDestination(template, factory, choices)
	} else {
		destination, err = Destination(template, choices.hostname, "", choices.passwordHash, choices.subnet)
	}
	choices.passwordHash = ""
	if err != nil {
		return err
	}
	// Deliver this media's already trusted executable for explicit post-boot
	// continuation. No remote runtime lookup or executable from the untrusted bundle.
	if media.Format == 0 {
		binary, err := readRegular(installerBinary, 64<<20)
		if err != nil {
			return err
		}
		destination, err = addContinuation(destination, binary)
		if err != nil {
			return err
		}
	}
	work, err := os.MkdirTemp("/run", "soda-installer-")
	if err != nil {
		return err
	}
	// Private inputs remain in this boot's tmpfs, including on failure. No logs
	// contain them; no cleanup touches unrelated or retained evidence.
	ignition := filepath.Join(work, "destination.ign")
	if err := nativebuild.WriteNew(ignition, destination, 0o600); err != nil {
		return err
	}
	c.page("Installing CoreOS")
	c.print("Writing the confirmed disk. Do not disconnect it.")
	c.print("Raw diagnostics are suppressed to protect provisioning inputs.")
	err = executeDisk(ctx, choices.disk, ignition, func() ([]Disk, error) { return scanDisks(ctx, command) }, func() error {
		return nativebuild.WriteNew(marker, []byte(choices.disk.Device.Name+"\n"), 0o600)
	}, command)
	if err != nil {
		return err
	}
	if media.Format == 2 {
		c.print("SodaOS disk installation completed with all five application images local.")
		c.print("Remove installation media and reboot explicitly; log in locally as root with your password.")
		c.print("Native startup imports the included images before starting their services. No reboot was performed.")
		c.print("For key-only SSH access, run locally after reboot: %s enroll-key", candidateInstallerBinary)
		c.print("Then complete browser setup from your SSH terminal: %s configure", candidateInstallerBinary)
		return nil
	}
	if err := copyInstalledPayload(ctx, choices.disk, media, payloadBytes, command); err != nil {
		return err
	}
	c.print("CoreOS disk installation completed; Soda setup is not complete.")
	c.print("Remove installation media and reboot explicitly.")
	c.print("On the installed system run: sudo %s continue", installerBinary)
	c.print("No reboot was performed.")
	return nil
}

func checkNav(value string) error {
	switch strings.ToLower(value) {
	case "back":
		return errBack
	case "restart":
		return errRestart
	case "cancel":
		return errCancel
	default:
		return nil
	}
}

func askNav(c console, prompt string) (string, error) {
	val, err := c.ask(prompt)
	if err != nil {
		return "", err
	}
	if navErr := checkNav(val); navErr != nil {
		return "", navErr
	}
	return val, nil
}

func askSecretNav(c console, prompt string) (string, error) {
	val, err := c.secret(prompt)
	if err != nil {
		return "", err
	}
	if navErr := checkNav(val); navErr != nil {
		return "", navErr
	}
	return val, nil
}

func stepNetwork(ctx context.Context, c console, run commandRunner) error {
	err := c.networkWith(ctx, run)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, errBack), errors.Is(err, errCancel):
		return errCancel
	default:
		return err
	}
}

func handleDiskInspectFailure(c console) error {
	c.page("Step 2 of 5 — Installation disk")
	c.print("Could not inspect disks. No disk installation started.")
	_, err := askNav(c, "Type retry, back, restart, or cancel")
	return err
}

func printDiskList(c console, disks []Disk, feedback string) {
	c.page("Step 2 of 5 — Installation disk")
	if feedback != "" {
		c.print("%s", feedback)
		c.print("")
	}
	for i, disk := range disks {
		c.print("%d. %s", i+1, diskSummary(disk.Device))
		for _, child := range disk.Device.Children {
			c.print("   %q: %.1f GiB, filesystem %q", child.Name, float64(child.Size)/(1<<30), child.FSType)
		}
		if disk.Blocked != "" {
			c.print("   Unavailable: %s", disk.Blocked)
		}
	}
}

func selectDisk(disks []Disk, selected string) (Disk, string, bool) {
	index, err := strconv.Atoi(selected)
	if err != nil || index < 1 || index > len(disks) {
		return Disk{}, "Choose an available disk number.", false
	}
	disk := disks[index-1]
	if disk.Blocked != "" {
		return Disk{}, "That disk is unavailable; choose another disk.", false
	}
	return disk, "", true
}

func stepDisk(ctx context.Context, c console, run commandRunner, inspect func(context.Context, commandRunner) ([]Disk, error)) (Disk, error) {
	feedback := ""
	for {
		disks, err := inspect(ctx, run)
		if err != nil {
			if navErr := handleDiskInspectFailure(c); navErr != nil {
				return Disk{}, navErr
			}
			continue
		}
		printDiskList(c, disks, feedback)
		feedback = ""
		selected, err := askNav(c, "Disk number, back, restart, or cancel")
		if err != nil {
			return Disk{}, err
		}
		disk, errFeedback, ok := selectDisk(disks, selected)
		if !ok {
			feedback = errFeedback
			continue
		}
		return disk, nil
	}
}

func stepHostname(c console, selectedDisk Disk, currentHostname string) (string, error) {
	feedback := ""
	for {
		c.page("Step 3 of 5 — Hostname")
		if feedback != "" {
			c.print("%s", feedback)
			c.print("")
			feedback = ""
		}
		c.print("Selected disk: %s", diskSummary(selectedDisk.Device))
		hostnameDefault := currentHostname
		if hostnameDefault == "" {
			hostnameDefault = "soda"
		}
		value, err := askNav(c, "Hostname ["+hostnameDefault+"], back, restart, or cancel")
		if err != nil {
			return "", err
		}
		if value == "" {
			value = hostnameDefault
		}
		if !Hostname(value) {
			feedback = "Use lowercase letters, digits, dots, and interior hyphens."
			continue
		}
		return value, nil
	}
}

func validPassword(password, confirmation string) bool {
	return utf8.ValidString(password) && utf8.RuneCountInString(password) >= 12 && password == confirmation
}

func hashPassword(ctx context.Context, run commandRunner, password string) (string, error) {
	hash, err := run(ctx, "openssl", []string{"passwd", "-6", "-stdin"}, strings.NewReader(password+"\n"))
	if err != nil {
		return "", errors.New("password hashing failed")
	}
	return strings.TrimSpace(string(hash)), nil
}

func stepPassword(ctx context.Context, c console, run commandRunner) (string, error) {
	feedback := ""
	for {
		c.page("Step 4 of 5 — Native operator password")
		if feedback != "" {
			c.print("%s", feedback)
			c.print("")
			feedback = ""
		}
		c.print("This password is for local root login after reboot.")
		c.print("It does not enable ordinary root-password SSH.")
		c.print("Type back, restart, or cancel in a password field to navigate.")
		password, err := askSecretNav(c, "Password (at least 12 characters)")
		if err != nil {
			return "", err
		}
		confirmation, err := askSecretNav(c, "Confirm password")
		if err != nil {
			return "", err
		}
		if !validPassword(password, confirmation) {
			feedback = "Passwords must match and contain at least 12 characters."
			continue
		}
		return hashPassword(ctx, run, password)
	}
}

func stepSubnet(ctx context.Context, c console, run commandRunner, currentSubnet string) (string, error) {
	feedback := ""
	for {
		c.page("Step 5 of 5 — Project network and review")
		if feedback != "" {
			c.print("%s", feedback)
			c.print("")
			feedback = ""
		}
		c.print("Developer client routing is configured separately.")
		subnetDefault := currentSubnet
		if subnetDefault == "" {
			subnetDefault = "10.89.0.0/24"
		}
		subnet, err := askNav(c, "Private project IPv4 subnet ["+subnetDefault+"], back, restart, or cancel")
		if err != nil {
			return "", err
		}
		if subnet == "" {
			subnet = subnetDefault
		}
		observedRoutes, routeErr := routes(ctx, run)
		if routeErr != nil {
			feedback = "Could not inspect current IPv4 routes. Correct networking or retry."
			continue
		}
		if err := ProjectSubnet(subnet, observedRoutes); err != nil {
			feedback = "Invalid subnet: " + err.Error()
			continue
		}
		return subnet, nil
	}
}

func printFinalReview(c console, choices diskInstallChoices, payloadBytes uint64) {
	c.page("Final review")
	c.print("ERASE ALL DATA on:")
	c.print("  %s", diskSummary(choices.disk.Device))
	c.print("Hostname: %s", choices.hostname)
	c.print("Project subnet: %s", choices.subnet)
	c.print("Operator access: local native root password")
	c.print("Included Soda payload: %.1f MiB verified", float64(payloadBytes)/(1<<20))
	c.print("Network settings will be copied to the installed system.")
	c.print("After writing, follow the completion screen for media removal and next steps.")
}

func confirmFinalReview(c console, diskName string) error {
	phrase := "ERASE " + diskName
	for {
		answer, err := askNav(c, "Type exactly "+phrase+", back, restart, or cancel")
		if err != nil {
			return err
		}
		if answer == phrase {
			return nil
		}
		c.print("Confirmation did not match. No disk writing started.")
	}
}

func stepSubnetAndReview(ctx context.Context, c console, run commandRunner, choices *diskInstallChoices, payloadBytes uint64) error {
	for {
		subnet, err := stepSubnet(ctx, c, run, choices.subnet)
		if err != nil {
			return err
		}
		choices.subnet = subnet
		printFinalReview(c, *choices, payloadBytes)
		err = confirmFinalReview(c, choices.disk.Device.Name)
		if errors.Is(err, errBack) {
			continue
		}
		return err
	}
}

func dispatchInstallStep(ctx context.Context, c console, run commandRunner, inspect func(context.Context, commandRunner) ([]Disk, error), payloadBytes uint64, step int, result *diskInstallChoices) error {
	switch step {
	case 0:
		return stepNetwork(ctx, c, run)
	case 1:
		var err error
		result.disk, err = stepDisk(ctx, c, run, inspect)
		return err
	case 2:
		var err error
		result.hostname, err = stepHostname(c, result.disk, result.hostname)
		return err
	case 3:
		var err error
		result.passwordHash, err = stepPassword(ctx, c, run)
		return err
	case 4:
		err := stepSubnetAndReview(ctx, c, run, result, payloadBytes)
		if err != nil {
			result.passwordHash = ""
		}
		return err
	default:
		return nil
	}
}

func collectDiskInstallChoices(ctx context.Context, c console, run commandRunner, inspect func(context.Context, commandRunner) ([]Disk, error), payloadBytes uint64) (diskInstallChoices, error) {
	var result diskInstallChoices
	step := 0
	for {
		err := dispatchInstallStep(ctx, c, run, inspect, payloadBytes, step, &result)
		if errors.Is(err, errBack) {
			step--
		} else if err != nil {
			return result, err
		} else if step == 4 {
			return result, nil
		} else {
			step++
		}
	}
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
