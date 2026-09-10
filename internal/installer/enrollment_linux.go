package installer

import (
	"context"
	"errors"
	"fmt"
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

// /dev/tty itself has major 5; TIOCGDEV obtains its actual controlling device.
// Only Linux virtual consoles are selected by this keyboard/monitor installer.
// SSH PTYs, tmux, serial consoles and a merely inherited SSH-free environment
// are insufficient to arm. Existing native root can of course administer the
// machine independently; this is not a security boundary against hostile root.
func enrollmentLocalConsole(tty *os.File) error {
	device, err := unix.IoctlGetInt(int(tty.Fd()), unix.TIOCGDEV)
	if err != nil || unix.Major(uint64(device)) != 4 || unix.Minor(uint64(device)) < 1 || unix.Minor(uint64(device)) > 63 || os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != "" {
		return errors.New("key enrollment must be armed after root login on the local keyboard/monitor console; remote and multiplexed terminals cannot arm it")
	}
	return nil
}

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
	fields := strings.Fields(string(data))
	if len(fields) != 3 {
		return selected, 0, errors.New("invalid enrollment arm state")
	}
	selected = enrollmentAddress{fields[0], fields[1]}
	if _, err := enrollmentPrivateAddress(selected.ip); err != nil {
		return selected, 0, err
	}
	until, err := strconv.ParseInt(fields[2], 10, 64)
	now, clockErr := enrollmentBootSeconds()
	if err != nil || clockErr != nil || until <= now || until-now > 300 {
		return selected, 0, errors.New("enrollment window expired")
	}
	return selected, time.Duration(until-now) * time.Second, nil
}

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
		if strings.EqualFold(value, "back") || strings.EqualFold(value, "cancel") {
			return enrollmentAddress{}, errors.New("key enrollment cancelled")
		}
		index, err := strconv.Atoi(value)
		if err == nil && index >= 1 && index <= len(addresses) {
			return addresses[index-1], nil
		}
		c.print("Enter a number from 1 to %d, or back/cancel.", len(addresses))
	}
}

func armEnrollment(ctx context.Context, c console, run commandRunner) (result error) {
	if err := enrollmentLocalConsole(c.tty); err != nil {
		return err
	}
	if _, err := enrollmentRootHome(); err != nil {
		return err
	}
	addresses, err := enrollmentAddresses()
	if err != nil || len(addresses) == 0 {
		return errors.New("no active private IPv4 address: configure the host network and an actual laptop route before key enrollment")
	}
	c.print("Import one laptop public key using the native root password. Choose the host address your laptop can actually reach; an interface address alone does not prove a client route.")
	selected, err := selectEnrollmentAddress(c, addresses)
	if err != nil {
		return err
	}
	// Derive the public host fingerprint from the actual existing private host
	// key via native ssh-keygen; do not trust a possibly stale .pub companion.
	public, err := run(ctx, "/usr/bin/ssh-keygen", []string{"-y", "-f", "/etc/ssh/ssh_host_ed25519_key"}, nil)
	if err != nil {
		return errors.New("native Ed25519 SSH host key unavailable; no enrollment window opened")
	}
	key, _, _, _, err := ssh.ParseAuthorizedKey(public)
	if err != nil || key.Type() != ssh.KeyAlgoED25519 {
		return errors.New("cannot inspect native SSH host identity")
	}
	c.print("Native host fingerprint: %s", ssh.FingerprintSHA256(key))
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
	if err := ctx.Err(); err != nil {
		return err
	}
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
	if err := os.Mkdir(enrollmentDir, 0700); err != nil {
		return errors.New("enrollment state already exists or cannot be reserved; no replacement")
	}
	started := false
	var publishedUnits []string
	defer func() {
		stopped := !started
		if started {
			cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// Stopping the socket first waits for its BindsTo/After connection
			// instances. Only these exact owned names are stopped, never a glob.
			_, socketErr := run(cleanup, "systemctl", []string{"stop", enrollmentSocketUnit}, nil)
			_, stopErr := run(cleanup, "systemctl", []string{"stop", enrollmentUnit}, nil)
			if stopErr == nil {
				stopped = true
			} else {
				// --collect may already have unloaded a successful unit. Confirm
				// both manager absence and cgroup removal before treating that
				// stop error as an already completed close.
				load, inspectErr := run(cleanup, "systemctl", []string{"show", "--property=LoadState", "--value", enrollmentUnit}, nil)
				_, groupErr := os.Lstat("/sys/fs/cgroup/system.slice/" + enrollmentUnit)
				stopped = inspectErr == nil && strings.TrimSpace(string(load)) == "not-found" && errors.Is(groupErr, os.ErrNotExist)
			}
			stopped = stopped && socketErr == nil
			if !stopped {
				result = errors.New("enrollment close could not be confirmed; the five-minute native service limit still applies; preserve this attempt and inspect locally")
			}
		}
		if stopped {
			for _, name := range publishedUnits {
				if err := os.Remove(filepath.Join(enrollmentUnitDirectory, name)); err != nil {
					result = errors.New("owned enrollment unit cleanup failed; inspect locally before reopening")
				}
			}
			if len(publishedUnits) > 0 {
				cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if _, err := run(cleanup, "systemctl", []string{"daemon-reload"}, nil); err != nil {
					result = errors.New("owned enrollment unit removal could not be reloaded; inspect locally before reopening")
				}
			}
			// Exact run-owned files only. Never remove another tree or stale evidence.
			for _, name := range []string{"armed", "sshd_config", "receive.sock", "ready", "result"} {
				_ = os.Remove(filepath.Join(enrollmentDir, name))
			}
			_ = os.Remove(enrollmentDir)
		}
	}()
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
	unitDirectory, err := unix.Open(enrollmentUnitDirectory, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return errors.New("native runtime unit directory unavailable")
	}
	defer unix.Close(unitDirectory)
	if err := enrollmentSafeDirectory(unitDirectory, 0); err != nil {
		return err
	}
	socketConfig, err := enrollmentSocketUnitConfig(selected.ip)
	if err != nil {
		return err
	}
	for _, unit := range []struct{ name, contents string }{{enrollmentSocketUnit, socketConfig}, {enrollmentTemplateUnit, enrollmentTemplateUnitConfig()}} {
		if err := enrollmentWrite(enrollmentUnitDirectory, unit.name, unit.contents); err != nil {
			return errors.New("runtime enrollment unit already exists or cannot be published; no replacement")
		}
		publishedUnits = append(publishedUnits, unit.name)
	}
	if _, err := run(ctx, "systemctl", []string{"daemon-reload"}, nil); err != nil {
		return errors.New("native enrollment units could not be loaded")
	}
	// Mark before starting: a lost systemd-run reply is not permission to leave
	// a possibly started window running without the cleanup attempt.
	started = true
	if _, err := run(ctx, "systemd-run", enrollmentStartArgs(), nil); err != nil {
		return errors.New("temporary enrollment service could not start")
	}
	waitCtx, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()
	cancelConsole := console{tty: c.tty, ctx: waitCtx}
	ready := false
	lastStateCheck := time.Time{}
	for {
		if data, err := readRegular(enrollmentDir+"/result", 128); err == nil {
			if string(data) == "uncertain\n" {
				return errEnrollmentWriteUncertain
			}
			if string(data) != "imported\n" {
				return errors.New("key import failed; preserve existing access and inspect locally")
			}
			c.print("Public key installed. The import window is closing. Verify a NEW ordinary key-only SSH login from the laptop before continuing.")
			c.print("Use the laptop private-key path matching the public .pub file you imported:")
			c.print("ssh -i ~/.ssh/id_ed25519 -o IdentitiesOnly=yes -o PreferredAuthentications=publickey -o PasswordAuthentication=no -o KbdInteractiveAuthentication=no -o ControlMaster=no -o ControlPath=none root@%s", selected.ip)
			return nil
		}
		if err := waitCtx.Err(); err != nil {
			return errors.New("key enrollment closed without confirmed import")
		}
		if _, _, err := enrollmentState(); err != nil {
			return errors.New("key enrollment window expired; closing it now")
		}
		if !ready {
			if _, err := os.Lstat(enrollmentDir + "/ready"); err == nil {
				ready = true
				laptop, _ := enrollmentClientCommand(selected.ip)
				c.print("On the laptop, check the host fingerprint above, then run (adjust only your PUBLIC .pub file path):\n%s", laptop)
				c.print("Waiting for one public key. Press Enter to cancel; do not close this console while importing.")
			}
		}
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
		// Detect failures after binding too. A missing receipt is not proof that
		// no key write occurred; never retry a potentially completed append.
		if time.Since(lastStateCheck) >= time.Second {
			lastStateCheck = time.Now()
			state, err := run(waitCtx, "systemctl", []string{"show", "--property=ActiveState", "--value", enrollmentUnit}, nil)
			active := strings.TrimSpace(string(state))
			if err != nil || (active != "active" && active != "activating") {
				// The receipt may have appeared since the start of this loop.
				if _, err := os.Lstat(enrollmentDir + "/result"); err == nil {
					continue
				}
				return errors.New("enrollment service closed without a receipt; a key may have been imported; verify native access locally before trying another import")
			}
		}
	}
}

// ServeEnrollment is an internal fixed systemd service action, not an arming
// entrypoint. Run dispatches it without requesting a terminal, after root and
// installed-CoreOS checks. The cgroup check prevents accidental direct execution.
func ServeEnrollment(ctx context.Context) error {
	group, err := os.ReadFile("/proc/self/cgroup")
	if err != nil || strings.TrimSpace(string(group)) != "0::/system.slice/"+enrollmentUnit {
		return errors.New("enrollment server requires its fixed native transient service")
	}
	selected, remaining, err := enrollmentState()
	if err != nil {
		return err
	}
	if !enrollmentLiveAddress(selected) {
		return errors.New("selected private address is no longer live")
	}
	phase, cancel := context.WithTimeout(ctx, remaining)
	defer cancel()
	broker, err := net.ListenUnix("unix", &net.UnixAddr{Name: enrollmentSocket, Net: "unix"})
	if err != nil {
		return err
	}
	defer broker.Close()
	if err := os.Chmod(enrollmentSocket, 0600); err != nil {
		return err
	}
	if _, err := command(phase, "systemctl", []string{"start", enrollmentSocketUnit}, nil); err != nil {
		return errors.New("native key-import socket could not start")
	}
	if err := enrollmentWrite(enrollmentDir, "ready", "ready\n"); err != nil {
		return err
	}
	requests := make(chan *net.UnixConn)
	go func() {
		for {
			connection, err := broker.AcceptUnix()
			if err != nil {
				return
			}
			select {
			case requests <- connection:
			case <-phase.Done():
				connection.Close()
				return
			}
		}
	}()
	// CLOCK_BOOTTIME state also expires across suspend, unlike Go's ordinary
	// monotonic timers. A resumed machine cannot extend an old arm window.
	clockCheck := time.NewTicker(time.Second)
	defer clockCheck.Stop()
	for {
		select {
		case <-phase.Done():
			return errors.New("enrollment window closed")
		case <-clockCheck.C:
			if _, _, err := enrollmentState(); err != nil {
				return err
			}
		case connection := <-requests:
			// The receiver has already authenticated as root through stock sshd. The
			// local broker also checks kernel peer credentials and accepts one bounded
			// key only. No user-selected path/account/command reaches this writer.
			deadline := time.Now().Add(15 * time.Second)
			if window, ok := phase.Deadline(); ok && window.Before(deadline) {
				deadline = window
			}
			connection.SetDeadline(deadline)
			if unit, err := enrollmentConnectionUnit(connection); err != nil || !enrollmentReceiverUnit(unit) {
				connection.Close()
				continue
			}
			key, err := enrollmentPublicKey(connection)
			if err != nil {
				io.WriteString(connection, "refused\n")
				connection.Close()
				continue
			}
			if phase.Err() != nil {
				connection.Close()
				return phase.Err()
			}
			if _, _, err := enrollmentState(); err != nil {
				connection.Close()
				return err
			}
			// One validated import consumes the broker immediately. On service
			// exit, native BindsTo/After ordering closes the TCP socket and
			// all connection service cgroups, including sshd session children.
			broker.Close()
			home, err := enrollmentRootHome()
			if err == nil {
				err = appendEnrollmentKey(phase, home, key, 0)
			}
			if err == nil {
				if _, labelErr := command(phase, "/usr/sbin/restorecon", []string{"--", home + "/.ssh", home + "/.ssh/authorized_keys"}, nil); labelErr != nil {
					err = errEnrollmentWriteUncertain
				}
			}
			// A valid commit attempt consumes the window even on an uncertain write or
			// labeling failure. Never retry a possible completed append automatically.
			status := "failed\n"
			if err == nil {
				status = "imported\n"
			} else if errors.Is(err, errEnrollmentWriteUncertain) {
				status = "uncertain\n"
			}
			if writeErr := enrollmentWrite(enrollmentDir, "result", status); writeErr != nil {
				err = writeErr
			}
			io.WriteString(connection, status)
			connection.Close()
			return err
		}
	}
}

// Kernel credentials and the peer's native service cgroup distinguish the fixed
// broker and socket-activated receiver instances from ordinary root SSH shells.
func enrollmentConnectionUnit(connection *net.UnixConn) (string, error) {
	raw, err := connection.SyscallConn()
	if err != nil {
		return "", err
	}
	var credentials *unix.Ucred
	var inner error
	if err := raw.Control(func(fd uintptr) {
		credentials, inner = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil {
		return "", err
	}
	if inner != nil || credentials == nil || credentials.Uid != 0 || credentials.Pid <= 0 {
		return "", errors.New("dedicated native root enrollment peer required")
	}
	group, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", credentials.Pid))
	if err != nil {
		return "", err
	}
	return enrollmentPeerUnit(string(group))
}

// ReceiveEnrollment is the sole ForceCommand. Client commands/subsystems are
// refused, not interpreted. It never handles the native password or a private key.
func ReceiveEnrollment(ctx context.Context) error {
	if os.Geteuid() != 0 || os.Getenv("SSH_ORIGINAL_COMMAND") != "" || os.Getenv("SSH_TTY") != "" {
		return errors.New("only public-key stdin enrollment is allowed")
	}
	selected, remaining, err := enrollmentState()
	if err != nil {
		return err
	}
	fields := strings.Fields(os.Getenv("SSH_CONNECTION"))
	if len(fields) != 4 || fields[2] != selected.ip || fields[3] != enrollmentPort {
		return errors.New("dedicated key-import SSH connection required")
	}
	if remaining > 30*time.Second {
		remaining = 30 * time.Second
	}
	phase, cancel := context.WithTimeout(ctx, remaining)
	defer cancel()
	data, err := enrollmentStdin(phase)
	if err != nil {
		return err
	}
	key, err := enrollmentPublicKey(strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	connection, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: enrollmentSocket, Net: "unix"})
	if err != nil {
		return errors.New("key-import window is closed")
	}
	defer connection.Close()
	deadline, _ := phase.Deadline()
	connection.SetDeadline(deadline)
	if unit, err := enrollmentConnectionUnit(connection); err != nil || unit != enrollmentUnit {
		return errors.New("dedicated enrollment broker required")
	}
	if _, err := io.WriteString(connection, key+"\n"); err != nil {
		return err
	}
	if err := connection.CloseWrite(); err != nil {
		return err
	}
	result, err := io.ReadAll(io.LimitReader(connection, 129))
	if err == nil && string(result) == "uncertain\n" {
		return errEnrollmentWriteUncertain
	}
	if err != nil || string(result) != "imported\n" {
		return errors.New("key import was not confirmed; inspect the local console before retrying")
	}
	fmt.Fprintln(os.Stdout, "Public key imported. Verify a fresh ordinary key-only SSH login.")
	return nil
}

func enrollmentStdin(ctx context.Context) ([]byte, error) {
	var data []byte
	for len(data) <= enrollmentKeyLimit {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		events := []unix.PollFd{{Fd: 0, Events: unix.POLLIN}}
		if _, err := unix.Poll(events, 100); err != nil && err != unix.EINTR {
			return nil, err
		}
		if events[0].Revents&(unix.POLLERR|unix.POLLNVAL) != 0 {
			return nil, errors.New("public-key input failed")
		}
		if events[0].Revents&(unix.POLLIN|unix.POLLHUP) == 0 {
			continue
		}
		var buffer [1024]byte
		n, err := os.Stdin.Read(buffer[:])
		data = append(data, buffer[:n]...)
		if err == io.EOF {
			return data, nil
		}
		if err != nil {
			return nil, err
		}
	}
	return nil, errors.New("public-key input exceeds limit")
}
