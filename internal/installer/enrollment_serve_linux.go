package installer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func validateEnrollmentServerEnvironment() (time.Duration, error) {
	group, err := os.ReadFile("/proc/self/cgroup")
	if err != nil || strings.TrimSpace(string(group)) != "0::/system.slice/"+enrollmentUnit {
		return 0, errors.New("enrollment server requires its fixed native transient service")
	}
	selected, remaining, err := enrollmentState()
	if err != nil {
		return 0, err
	}
	if !enrollmentLiveAddress(selected) {
		return 0, errors.New("selected private address is no longer live")
	}
	return remaining, nil
}

func startEnrollmentBroker(phase context.Context) (*net.UnixListener, error) {
	broker, err := net.ListenUnix("unix", &net.UnixAddr{Name: enrollmentSocket, Net: "unix"})
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(enrollmentSocket, 0o600); err != nil {
		broker.Close()
		return nil, err
	}
	if _, err := command(phase, "systemctl", []string{"start", enrollmentSocketUnit}, nil); err != nil {
		broker.Close()
		return nil, errors.New("native key-import socket could not start")
	}
	if err := enrollmentWrite(enrollmentDir, "ready", "ready\n"); err != nil {
		broker.Close()
		return nil, err
	}
	return broker, nil
}

func acceptEnrollmentRequests(phase context.Context, broker *net.UnixListener) <-chan *net.UnixConn {
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
	return requests
}

func setEnrollmentConnectionDeadline(phase context.Context, connection *net.UnixConn) {
	deadline := time.Now().Add(15 * time.Second)
	if window, ok := phase.Deadline(); ok && window.Before(deadline) {
		deadline = window
	}
	connection.SetDeadline(deadline)
}

func readEnrollmentKeyFromConnection(phase context.Context, connection *net.UnixConn) (string, bool, error) {
	setEnrollmentConnectionDeadline(phase, connection)
	unit, err := enrollmentConnectionUnit(connection)
	if err != nil || !enrollmentReceiverUnit(unit) {
		connection.Close()
		return "", false, nil
	}
	key, err := enrollmentPublicKey(connection)
	if err != nil {
		io.WriteString(connection, "refused\n")
		connection.Close()
		return "", false, nil
	}
	if phase.Err() != nil {
		connection.Close()
		return "", false, phase.Err()
	}
	if _, _, err := enrollmentState(); err != nil {
		connection.Close()
		return "", false, err
	}
	return key, true, nil
}

func enrollmentCommitStatus(err error) string {
	if err == nil {
		return "imported\n"
	}
	if errors.Is(err, errEnrollmentWriteUncertain) {
		return "uncertain\n"
	}
	return "failed\n"
}

func commitEnrollmentKey(phase context.Context, connection *net.UnixConn, key string) error {
	home, err := enrollmentRootHome()
	if err == nil {
		err = appendEnrollmentKey(phase, home, key, 0)
	}
	if err == nil {
		if _, labelErr := command(phase, "/usr/sbin/restorecon", []string{"--", home + "/.ssh", home + "/.ssh/authorized_keys"}, nil); labelErr != nil {
			err = errEnrollmentWriteUncertain
		}
	}
	status := enrollmentCommitStatus(err)
	if writeErr := enrollmentWrite(enrollmentDir, "result", status); writeErr != nil {
		err = writeErr
	}
	io.WriteString(connection, status)
	connection.Close()
	return err
}

func serveEnrollmentLoop(phase context.Context, broker *net.UnixListener, requests <-chan *net.UnixConn) error {
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
			key, accepted, err := readEnrollmentKeyFromConnection(phase, connection)
			if err != nil {
				return err
			}
			if !accepted {
				continue
			}
			broker.Close()
			return commitEnrollmentKey(phase, connection, key)
		}
	}
}

// ServeEnrollment is an internal fixed systemd service action, not an arming
// entrypoint. Run dispatches it without requesting a terminal, after root and
// installed-CoreOS checks. The cgroup check prevents accidental direct execution.
func ServeEnrollment(ctx context.Context) error {
	remaining, err := validateEnrollmentServerEnvironment()
	if err != nil {
		return err
	}
	phase, cancel := context.WithTimeout(ctx, remaining)
	defer cancel()

	broker, err := startEnrollmentBroker(phase)
	if err != nil {
		return err
	}
	defer broker.Close()

	requests := acceptEnrollmentRequests(phase, broker)
	return serveEnrollmentLoop(phase, broker, requests)
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

func admitReceiveEnrollment() (time.Duration, error) {
	if os.Geteuid() != 0 || os.Getenv("SSH_ORIGINAL_COMMAND") != "" || os.Getenv("SSH_TTY") != "" {
		return 0, errors.New("only public-key stdin enrollment is allowed")
	}
	selected, remaining, err := enrollmentState()
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(os.Getenv("SSH_CONNECTION"))
	if len(fields) != 4 || fields[2] != selected.ip || fields[3] != enrollmentPort {
		return 0, errors.New("dedicated key-import SSH connection required")
	}
	if remaining > 30*time.Second {
		remaining = 30 * time.Second
	}
	return remaining, nil
}

func confirmEnrollmentImport(result []byte, err error) error {
	if err == nil && string(result) == "uncertain\n" {
		return errEnrollmentWriteUncertain
	}
	if err != nil || string(result) != "imported\n" {
		return errors.New("key import was not confirmed; inspect the local console before retrying")
	}
	return nil
}

func submitEnrollmentKey(ctx context.Context, key string) error {
	connection, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: enrollmentSocket, Net: "unix"})
	if err != nil {
		return errors.New("key-import window is closed")
	}
	defer connection.Close()
	if deadline, ok := ctx.Deadline(); ok {
		connection.SetDeadline(deadline)
	}
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
	return confirmEnrollmentImport(result, err)
}

// ReceiveEnrollment is the sole ForceCommand. Client commands/subsystems are
// refused, not interpreted. It never handles the native password or a private key.
func ReceiveEnrollment(ctx context.Context) error {
	remaining, err := admitReceiveEnrollment()
	if err != nil {
		return err
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
	if err := submitEnrollmentKey(phase, key); err != nil {
		return err
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
