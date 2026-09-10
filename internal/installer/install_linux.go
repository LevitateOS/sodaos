package installer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/levitateos/sodaos/internal/nativebuild"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sys/unix"
)

const dataDir = "/usr/local/share/soda-installer"
const installerBinary = "/usr/local/libexec/soda/soda-install"

type mediaIdentity struct{ Architecture, Release, InstallerVersion, Revision string }

func (m mediaIdentity) validate(imageVersion, arch string) error {
	if m.Architecture != arch || m.Release == "" || m.Release != imageVersion || !nativebuild.Revision(m.Revision) {
		return errors.New("media release/architecture mismatch")
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
func Run(ctx context.Context, action string) error {
	if os.Geteuid() != 0 || runtime.GOOS != "linux" {
		return errors.New("native CoreOS root required")
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return errors.New("interactive operator terminal required")
	}
	defer tty.Close()
	c := console{tty: tty, ctx: ctx}
	if action != "disk" && action != "continue" {
		return errors.New("usage: soda-install disk|continue")
	}
	if err := coreOSHost(action == "disk"); err != nil {
		return err
	}
	// One local caller, including when different consoles are active. Lock file is
	// not a success marker and is never removed to pretend a partial attempt is new.
	lock, err := os.OpenFile("/run/soda-installer.lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return errors.New("cannot open installer lock")
	}
	defer lock.Close()
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return errors.New("another installer is active")
	}
	defer unix.Flock(int(lock.Fd()), unix.LOCK_UN)
	if action == "continue" {
		return continueInstall(ctx, c, command)
	}
	return installDisk(ctx, c)
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
	if _, err := os.Lstat("/run/soda-installer-disk-started"); !errors.Is(err, os.ErrNotExist) {
		return errors.New("disk installation was already attempted this boot; inspect the result, do not replay")
	}
	welcome, err := readRegular("/etc/motd", 16384)
	if err != nil {
		return errors.New("cannot read installer welcome text")
	}
	c.print("\x1b[2J\x1b[H%s", string(welcome))
	c.print("SodaOS installation")
	c.print("This is a network-assisted fresh installation, not an upgrade. No disk writes occur before the final ERASE confirmation. Ctrl-C cancels input; interruption after writing starts can leave a partial installation.")
	if err := c.network(ctx); err != nil {
		return err
	}
	disks, err := scanDisks(ctx, command)
	if err != nil {
		return err
	}
	for i, disk := range disks {
		c.print("%d. %s", i+1, diskSummary(disk.Device))
		for _, child := range disk.Device.Children {
			c.print("   partition %q: %.1f GiB, filesystem %q, UUID %q", child.Name, float64(child.Size)/(1<<30), child.FSType, child.UUID)
		}
		if disk.Blocked != "" {
			c.print("   Unavailable: %s", disk.Blocked)
		}
	}
	selected, err := c.ask("Disk number (no default; anything else cancels)")
	if err != nil {
		return err
	}
	index, err := strconv.Atoi(selected)
	if err != nil || index < 1 || index > len(disks) {
		return errors.New("disk selection cancelled")
	}
	disk := disks[index-1]
	if disk.Blocked != "" {
		return errors.New("selected disk is unavailable")
	}
	hostname, err := c.ask("Hostname [soda]")
	if err != nil {
		return err
	}
	if hostname == "" {
		hostname = "soda"
	}
	if !Hostname(hostname) {
		return errors.New("invalid hostname")
	}
	key, err := c.ask("Paste operator SSH public key, or @/absolute/public-key-file")
	if err != nil {
		return err
	}
	if strings.HasPrefix(key, "@") {
		path := strings.TrimPrefix(key, "@")
		if !filepath.IsAbs(path) {
			return errors.New("absolute public-key path required")
		}
		data, e := readRegular(path, 16384)
		if e != nil {
			return errors.New("cannot read bounded public-key file")
		}
		key = strings.TrimSpace(string(data))
	}
	key, err = PublicKey(key)
	if err != nil {
		return err
	}
	password, err := c.secret("Native operator/root password (at least 12 characters)")
	if err != nil {
		return err
	}
	confirmation, err := c.secret("Confirm operator password")
	if err != nil {
		return err
	}
	if !utf8.ValidString(password) || utf8.RuneCountInString(password) < 12 || password != confirmation {
		return errors.New("passwords must match and contain at least 12 characters")
	}
	hash, err := command(ctx, "openssl", []string{"passwd", "-6", "-stdin"}, strings.NewReader(password+"\n"))
	password, confirmation = "", ""
	if err != nil {
		return errors.New("password hashing failed")
	}
	subnet, err := c.ask("Private project IPv4 subnet [10.89.0.0/24]")
	if err != nil {
		return err
	}
	if subnet == "" {
		subnet = "10.89.0.0/24"
	}
	observedRoutes, err := routes(ctx, command)
	if err != nil {
		return err
	}
	if err := ProjectSubnet(subnet, observedRoutes); err != nil {
		return err
	}
	template, err := readRegular(filepath.Join(dataDir, "destination.ign"), 4<<20)
	if err != nil {
		return err
	}
	destination, err := Destination(template, hostname, key, strings.TrimSpace(string(hash)), subnet)
	if err != nil {
		return err
	}
	// Deliver this media's already trusted executable for explicit post-boot
	// continuation. No remote runtime lookup or executable from the untrusted bundle.
	binary, err := readRegular(installerBinary, 64<<20)
	if err != nil {
		return err
	}
	destination, err = addContinuation(destination, binary)
	if err != nil {
		return err
	}
	public, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key))
	if err != nil {
		return errors.New("cannot review operator public key")
	}
	c.print("Review: ERASE ALL DATA on %s", diskSummary(disk.Device))
	c.print("Operator SSH public-key fingerprint: %s", ssh.FingerprintSHA256(public))
	c.print("Hostname: %s; project subnet: %s; operator: native root (not a Forgejo account).", hostname, subnet)
	c.print("Use CoreOS's standard disk layout. Network settings will be copied. First boot installs network-fetched RPM dependencies; an activation reboot and verified Soda bundle are still required. Client routing is separate.")
	phrase := "ERASE " + disk.Device.Name
	answer, err := c.ask("Type exactly " + phrase + " to install; anything else cancels")
	if err != nil {
		return err
	}
	if answer != phrase {
		return errors.New("cancelled; no disk installation started")
	}
	work, err := os.MkdirTemp("/run", "soda-installer-")
	if err != nil {
		return err
	}
	// Private inputs remain in this boot's tmpfs, including on failure. No logs
	// contain them; no cleanup touches unrelated or retained evidence.
	ignition := filepath.Join(work, "destination.ign")
	if err := nativebuild.WriteNew(ignition, destination, 0600); err != nil {
		return err
	}
	c.print("Installing the confirmed disk. Do not disconnect it. Raw installer diagnostics are suppressed to protect provisioning inputs.")
	err = executeDisk(ctx, disk, ignition, func() ([]Disk, error) { return scanDisks(ctx, command) }, func() error {
		return nativebuild.WriteNew("/run/soda-installer-disk-started", []byte(disk.Device.Name+"\n"), 0600)
	}, command)
	if err != nil {
		return err
	}
	c.print("CoreOS disk installation completed, not Soda readiness. Remove installation media and reboot explicitly. On the installed system run sudo %s continue. No reboot was performed.", installerBinary)
	return nil
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
