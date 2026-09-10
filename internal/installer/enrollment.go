package installer

import (
	"errors"
	"fmt"
	"io"
	"net/netip"
	"strings"
)

const enrollmentDir = "/run/soda-key-enrollment"
const enrollmentUnit = "soda-key-enrollment.service"
const enrollmentSocketUnit = "soda-key-enrollment.socket"
const enrollmentTemplateUnit = "soda-key-enrollment@.service"
const enrollmentUnitDirectory = "/run/systemd/system"
const enrollmentPort = "22222"
const enrollmentBinary = "/usr/local/libexec/soda/soda-install"
const enrollmentSocket = enrollmentDir + "/receive.sock"
const enrollmentConfigPath = enrollmentDir + "/sshd_config"
const enrollmentKeyLimit = 16384

// This is an independent inetd-mode configuration; it never includes or edits
// ordinary sshd policy. OpenSSH verifies the native shadow password itself.
// UsePAM=no is deliberate: pam_systemd may move authenticated children out of
// the temporary service's cgroup, defeating its bounded lifetime. PAM-specific
// extra authentication/account/session policies therefore are not inherited.
// See upstream portable auth-passwd.c, auth.c and sshd_config(5).
func enrollmentConfig() string {
	return `HostKey /etc/ssh/ssh_host_ed25519_key
HostKeyAlgorithms ssh-ed25519
AllowUsers root
PermitRootLogin yes
AuthenticationMethods password
PasswordAuthentication yes
PermitEmptyPasswords no
UsePAM no
KbdInteractiveAuthentication no
PubkeyAuthentication no
HostbasedAuthentication no
GSSAPIAuthentication no
KerberosAuthentication no
AuthorizedKeysFile none
AuthorizedKeysCommand none
TrustedUserCAKeys none
DisableForwarding yes
AllowTcpForwarding no
AllowStreamLocalForwarding no
AllowAgentForwarding no
X11Forwarding no
PermitTunnel no
PermitTTY no
PermitUserRC no
PermitUserEnvironment no
MaxSessions 1
MaxAuthTries 3
LoginGraceTime 30
ClientAliveInterval 10
ClientAliveCountMax 2
PrintMotd no
PrintLastLog no
UseDNS no
LogLevel QUIET
ForceCommand ` + enrollmentBinary + ` enrollment-receive
`
}

// EOF is required, rather than accepting the first line and ignoring extra keys
// or protocol input. The transport imposes a deadline independently of this bound.
func enrollmentPublicKey(input io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(input, enrollmentKeyLimit+1))
	if err != nil || len(data) > enrollmentKeyLimit {
		return "", errors.New("one bounded SSH public-key file required")
	}
	value := strings.TrimSuffix(string(data), "\n")
	return PublicKey(value)
}

func enrollmentPrivateAddress(value string) (netip.Addr, error) {
	address, err := netip.ParseAddr(value)
	// The manual installer currently configures IPv4; do not invent a scope-zone
	// or route for link-local IPv6, wildcard, loopback or public interfaces.
	if err != nil || !address.Is4() || !address.IsPrivate() || address.String() != value {
		return netip.Addr{}, errors.New("selected live RFC1918 IPv4 address required")
	}
	return address, nil
}

func enrollmentClientCommand(address string) (string, error) {
	if _, err := enrollmentPrivateAddress(address); err != nil {
		return "", err
	}
	return fmt.Sprintf("ssh -T -p %s -o StrictHostKeyChecking=ask -o PreferredAuthentications=password -o PubkeyAuthentication=no -o KbdInteractiveAuthentication=no -o ControlMaster=no -o ControlPath=none -o RequestTTY=no -o ClearAllForwardings=yes root@%s < ~/.ssh/id_ed25519.pub", enrollmentPort, address), nil
}

func enrollmentStartArgs() []string {
	return []string{"--quiet", "--collect", "--unit=" + enrollmentUnit, "--service-type=exec",
		"--property=RuntimeMaxSec=300s", "--property=TimeoutStopSec=2s",
		"--property=KillMode=control-group", "--property=SendSIGKILL=yes", "--property=Restart=no",
		"--property=UMask=0077", "--property=StandardInput=null", "--property=StandardOutput=null", "--property=StandardError=null",
		"--", enrollmentBinary, "enrollment-serve"}
}

// Native systemd owns TCP accept, the two-connection limit, inetd descriptors,
// child supervision and stop ordering. Soda owns only the bounded key transaction.
func enrollmentSocketUnitConfig(address string) (string, error) {
	if _, err := enrollmentPrivateAddress(address); err != nil {
		return "", err
	}
	return `[Unit]
Description=Soda temporary public-key import socket
BindsTo=` + enrollmentUnit + `
After=` + enrollmentUnit + `

[Socket]
ListenStream=` + address + `:` + enrollmentPort + `
Accept=yes
MaxConnections=2
`, nil
}

func enrollmentTemplateUnitConfig() string {
	return `[Unit]
Description=Soda temporary public-key import connection
BindsTo=` + enrollmentUnit + ` ` + enrollmentSocketUnit + `
After=` + enrollmentUnit + ` ` + enrollmentSocketUnit + `
CollectMode=inactive-or-failed

[Service]
Type=exec
ExecStart=/usr/sbin/sshd -i -e -f ` + enrollmentConfigPath + `
StandardInput=socket
StandardOutput=socket
StandardError=null
Environment=PATH=/usr/sbin:/usr/bin:/sbin:/bin LANG=C
Slice=system.slice
UMask=0077
RuntimeMaxSec=300s
TimeoutStopSec=2s
KillMode=control-group
SendSIGKILL=yes
Restart=no
`
}

func enrollmentPeerUnit(cgroup string) (string, error) {
	const prefix = "0::/system.slice/"
	value := strings.TrimSuffix(cgroup, "\n")
	if !strings.HasPrefix(value, prefix) || strings.ContainsAny(value, "\r\n") {
		return "", errors.New("dedicated enrollment service peer required")
	}
	unit := strings.TrimPrefix(value, prefix)
	if strings.Contains(unit, "/") {
		return "", errors.New("dedicated enrollment service peer required")
	}
	return unit, nil
}

func enrollmentReceiverUnit(unit string) bool {
	const prefix = "soda-key-enrollment@"
	const suffix = ".service"
	return strings.HasPrefix(unit, prefix) && strings.HasSuffix(unit, suffix) && len(unit) > len(prefix)+len(suffix) && !strings.ContainsAny(unit, "/\r\n \t")
}
