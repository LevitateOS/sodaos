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

// ProbeSSHKey makes a direct client connection and verifies the supplied pin.
// No password, agent, private key, proxy, keyscan or Git authentication is used.
// Auth rejection after the key exchange is not interpreted as an account denial.
func (r Remote) ProbeSSHKey(ctx context.Context) (string, error) {
	if r.User != "git" || r.Port < 1 || r.Port > 65535 || !filepath.IsAbs(r.KnownHosts) {
		return "", errors.New("explicit Git SSH endpoint and pin required")
	}
	st, err := os.Lstat(r.KnownHosts)
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() || st.Mode().Perm()&0022 != 0 {
		return "", errors.New("trusted regular known_hosts required")
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
	fingerprint := ""
	config := &ssh.ClientConfig{User: r.User, HostKeyCallback: func(host string, remote net.Addr, key ssh.PublicKey) error {
		if err := verify(host, remote, key); err != nil {
			return err
		}
		fingerprint = ssh.FingerprintSHA256(key)
		return nil
	}}
	client, _, _, handshakeErr := ssh.NewClientConn(connection, host, config)
	if client != nil {
		_ = client.Close()
	}
	if err := phase.Err(); err != nil {
		return "", err
	}
	if fingerprint == "" {
		return "", errors.Join(errors.New("pinned endpoint key was not observed"), handshakeErr)
	}
	return fingerprint, nil
}
