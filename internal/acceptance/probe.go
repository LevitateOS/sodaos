package acceptance

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func explicitGitSSHEndpoint(r Remote) error {
	if r.User != "git" || !validSSHPort(r.Port) || !filepath.IsAbs(r.KnownHosts) {
		return errors.New("explicit Git SSH endpoint and pin required")
	}
	return nil
}

func regularUnwritableFile(st os.FileInfo) bool {
	return st.Mode().IsRegular() && st.Mode().Perm()&0o022 == 0
}

func trustedKnownHostsFile(path string) error {
	st, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !regularUnwritableFile(st) {
		return errors.New("trusted regular known_hosts required")
	}
	return nil
}

type hostKeyProbe struct {
	verify      ssh.HostKeyCallback
	fingerprint string
}

func (p *hostKeyProbe) check(host string, remote net.Addr, key ssh.PublicKey) error {
	if err := p.verify(host, remote, key); err != nil {
		return err
	}
	p.fingerprint = ssh.FingerprintSHA256(key)
	return nil
}

func closeClientConn(client ssh.Conn) {
	if client != nil {
		_ = client.Close()
	}
}

func observedHostFingerprint(probe *hostKeyProbe, handshakeErr error) (string, error) {
	if probe.fingerprint == "" {
		return "", errors.Join(errors.New("pinned endpoint key was not observed"), handshakeErr)
	}
	return probe.fingerprint, nil
}

// ProbeSSHKey makes a direct client connection and verifies the supplied pin.
// No password, agent, private key, proxy, keyscan or Git authentication is used.
// Auth rejection after the key exchange is not interpreted as an account denial.
func (r Remote) ProbeSSHKey(ctx context.Context) (string, error) {
	if err := explicitGitSSHEndpoint(r); err != nil {
		return "", err
	}
	if err := trustedKnownHostsFile(r.KnownHosts); err != nil {
		return "", err
	}
	verify, err := knownhosts.New(r.KnownHosts)
	if err != nil {
		return "", err
	}
	phase, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	host := net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
	connection, err := (&net.Dialer{}).DialContext(phase, "tcp", host)
	if err != nil {
		return "", err
	}
	defer connection.Close()
	deadline, _ := phase.Deadline()
	if err = connection.SetDeadline(deadline); err != nil {
		return "", err
	}
	stop := context.AfterFunc(phase, func() { _ = connection.Close() })
	defer stop()
	probe := &hostKeyProbe{verify: verify}
	config := &ssh.ClientConfig{User: r.User, HostKeyCallback: probe.check}
	client, _, _, handshakeErr := ssh.NewClientConn(connection, host, config)
	closeClientConn(client)
	if err := phase.Err(); err != nil {
		return "", err
	}
	return observedHostFingerprint(probe, handshakeErr)
}
