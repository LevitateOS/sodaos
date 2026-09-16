package installer

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/sys/unix"
)

// Key enrollment arms from any interactive root terminal, local or remote:
// an operator who already holds root can administer keys directly, so the
// terminal type was never a security boundary, only an inconvenience.
// Explicit typed intent and the expiring password-only window remain.
type enrollmentAddress struct{ name, ip string }

func enrollmentAddresses() ([]enrollmentAddress, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var result []enrollmentAddress
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, address := range addresses {
			ip, _, err := net.ParseCIDR(address.String())
			if err != nil {
				continue
			}
			if _, err := enrollmentPrivateAddress(ip.String()); err == nil {
				result = append(result, enrollmentAddress{iface.Name, ip.String()})
			}
		}
	}
	return result, nil
}

func enrollmentLiveAddress(selected enrollmentAddress) bool {
	addresses, err := enrollmentAddresses()
	if err != nil {
		return false
	}
	for _, candidate := range addresses {
		if candidate == selected {
			return true
		}
	}
	return false
}

func enrollmentBootSeconds() (int64, error) {
	var now unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_BOOTTIME, &now); err != nil {
		return 0, err
	}
	return now.Sec, nil
}

func enrollmentWrite(directory, name, value string) error {
	f, err := os.CreateTemp(directory, "."+name+"-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err := io.WriteString(f, value); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	// Readers must never observe a just-created empty success receipt. Linking
	// the complete private temporary file publishes atomically without replacing
	// any existing state; only this call's exact temporary path is removed.
	return os.Link(f.Name(), filepath.Join(directory, name))
}

func enrollmentWindowRemaining(until int64, parseErr, clockErr error, now int64) (time.Duration, error) {
	if parseErr != nil || clockErr != nil || until <= now || until-now > 300 {
		return 0, errors.New("enrollment window expired")
	}
	return time.Duration(until-now) * time.Second, nil
}

func parseArmedEnrollment(data []byte) (enrollmentAddress, int64, error) {
	fields := strings.Fields(string(data))
	if len(fields) != 3 {
		return enrollmentAddress{}, 0, errors.New("invalid enrollment arm state")
	}
	selected := enrollmentAddress{fields[0], fields[1]}
	if _, err := enrollmentPrivateAddress(selected.ip); err != nil {
		return selected, 0, err
	}
	until, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return selected, 0, errors.New("enrollment window expired")
	}
	return selected, until, nil
}

func enrollmentState() (enrollmentAddress, time.Duration, error) {
	var selected enrollmentAddress
	fd, err := unix.Open(enrollmentDir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return selected, 0, err
	}
	defer unix.Close(fd)
	if err := enrollmentSafeDirectory(fd, 0); err != nil {
		return selected, 0, err
	}
	data, err := readRegular(enrollmentDir+"/armed", 512)
	if err != nil {
		return selected, 0, err
	}
	selected, until, err := parseArmedEnrollment(data)
	if err != nil {
		return selected, 0, err
	}
	now, clockErr := enrollmentBootSeconds()
	remaining, err := enrollmentWindowRemaining(until, nil, clockErr, now)
	return selected, remaining, err
}

func parseEnrollmentChoice(value string, n int) (int, error) {
	if strings.EqualFold(value, "back") || strings.EqualFold(value, "cancel") {
		return 0, errors.New("key enrollment cancelled")
	}
	index, err := strconv.Atoi(value)
	if err != nil || index < 1 || index > n {
		return 0, errEnrollmentChoice
	}
	return index, nil
}

var errEnrollmentChoice = errors.New("enter a listed enrollment address")

func selectEnrollmentAddress(c console, addresses []enrollmentAddress) (enrollmentAddress, error) {
	if len(addresses) == 0 {
		return enrollmentAddress{}, errors.New("no private address available for key enrollment")
	}
	for i, address := range addresses {
		c.print("%d. %q: %s", i+1, address.name, address.ip)
	}
	for {
		value, err := c.ask("Private interface/address number (back or cancel exits)")
		if err != nil {
			return enrollmentAddress{}, err
		}
		index, err := parseEnrollmentChoice(value, len(addresses))
		if errors.Is(err, errEnrollmentChoice) {
			c.print("Enter a number from 1 to %d, or back/cancel.", len(addresses))
			continue
		}
		if err != nil {
			return enrollmentAddress{}, err
		}
		return addresses[index-1], nil
	}
}

func inspectHostFingerprint(ctx context.Context, run commandRunner) (string, error) {
	// Derive the public host fingerprint from the actual existing private host
	// key via native ssh-keygen; do not trust a possibly stale .pub companion.
	public, err := run(ctx, "/usr/bin/ssh-keygen", []string{"-y", "-f", "/etc/ssh/ssh_host_ed25519_key"}, nil)
	if err != nil {
		return "", errors.New("native Ed25519 SSH host key unavailable; no enrollment window opened")
	}
	key, _, _, _, err := ssh.ParseAuthorizedKey(public)
	if err != nil || key.Type() != ssh.KeyAlgoED25519 {
		return "", errors.New("cannot inspect native SSH host identity")
	}
	return ssh.FingerprintSHA256(key), nil
}

func confirmEnrollmentIntent(c console, selected enrollmentAddress, fingerprint string) error {
	c.print("Native host fingerprint: %s", fingerprint)
	c.print("A temporary password-only key-import connection will listen on %s:%s for at most five minutes. It uses OpenSSH's native password verification; additional PAM policies are not inherited. Ordinary SSH policy is unchanged. Press Enter or Ctrl-C to close it early.", selected.ip, enrollmentPort)
	answer, err := c.ask("Type ARM KEY IMPORT to open this window")
	if err != nil {
		return err
	}
	if answer != "ARM KEY IMPORT" {
		return errors.New("key enrollment cancelled")
	}
	if !enrollmentLiveAddress(selected) {
		return errors.New("selected interface address changed; no window opened")
	}
	return nil
}

func selectEnrollmentTarget(ctx context.Context, c console, run commandRunner) (enrollmentAddress, error) {
	if _, err := enrollmentRootHome(); err != nil {
		return enrollmentAddress{}, err
	}
	addresses, err := enrollmentAddresses()
	if err != nil || len(addresses) == 0 {
		return enrollmentAddress{}, errors.New("no active private IPv4 address: configure the host network and an actual laptop route before key enrollment")
	}
	c.print("Import one laptop public key using the native root password. Choose the host address your laptop can actually reach; an interface address alone does not prove a client route.")
	selected, err := selectEnrollmentAddress(c, addresses)
	if err != nil {
		return enrollmentAddress{}, err
	}
	fingerprint, err := inspectHostFingerprint(ctx, run)
	if err != nil {
		return enrollmentAddress{}, err
	}
	if err = confirmEnrollmentIntent(c, selected, fingerprint); err != nil {
		return enrollmentAddress{}, err
	}
	return selected, ctx.Err()
}

func guardExistingEnrollmentState(ctx context.Context, run commandRunner) error {
	// An existing unit or state is never replaced or adopted. No reopen after a
	// crash is inferred; runtime expiry/reboot bounds abandoned attempts.
	for _, name := range []string{enrollmentUnit, enrollmentSocketUnit} {
		unit, inspectErr := run(ctx, "systemctl", []string{"show", "--property=LoadState", "--value", name}, nil)
		if inspectErr != nil || strings.TrimSpace(string(unit)) != "not-found" {
			return errors.New("existing enrollment service/state must finish; no replacement")
		}
	}
	// A template without an instance is a unit file, not a loadable service.
	// Inspect the manager's unit-file search path rather than trying to start or
	// load the abstract @.service name, and never replace a vendor/operator file.
	templates, inspectErr := run(ctx, "systemctl", []string{"list-unit-files", "--no-legend", "--no-pager", enrollmentTemplateUnit}, nil)
	if inspectErr != nil || strings.TrimSpace(string(templates)) != "" {
		return errors.New("existing enrollment service template must be preserved; no replacement")
	}
	if err := os.Mkdir(enrollmentDir, 0o700); err != nil {
		return errors.New("enrollment state already exists or cannot be reserved; no replacement")
	}
	return nil
}
