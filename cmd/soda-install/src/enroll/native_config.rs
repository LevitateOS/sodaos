use crate::errors::Error;
use crate::netip;

use super::{
    ENROLLMENT_CONFIG_PATH, ENROLLMENT_KEY_LIMIT, ENROLLMENT_PORT, ENROLLMENT_SOCKET_UNIT,
    ENROLLMENT_UNIT,
};

pub(crate) fn enrollment_binary() -> &'static str {
    crate::candidate::CANDIDATE_INSTALLER_BINARY
}

// This is an independent inetd-mode configuration; it never includes or edits
// ordinary sshd policy. OpenSSH verifies the native shadow password itself.
// UsePAM=no is deliberate: pam_systemd may move authenticated children out of
// the temporary service's cgroup, defeating its bounded lifetime. PAM-specific
// extra authentication/account/session policies therefore are not inherited.
// See upstream portable auth-passwd.c, auth.c and sshd_config(5).
pub fn enrollment_config() -> String {
    format!(
        "HostKey /etc/ssh/ssh_host_ed25519_key
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
ForceCommand {} enrollment-receive
",
        enrollment_binary()
    )
}

// EOF is required, rather than accepting the first line and ignoring extra keys
// or protocol input. The transport imposes a deadline independently of this bound.
pub fn enrollment_public_key(data: &[u8]) -> Result<String, Error> {
    if data.len() > ENROLLMENT_KEY_LIMIT {
        return Err(Error::msg("one bounded SSH public-key file required"));
    }
    let value = data.strip_suffix(b"\n").unwrap_or(data);
    // Go converts arbitrary bytes to string; lossy decoding keeps that
    // acceptance: valid keys are ASCII, anything else fails key parsing.
    let text = String::from_utf8_lossy(value);
    crate::sshkey::public_key(&text).map_err(Error::msg)
}

/// Bounded stream read for the broker: up to `limit + 1` bytes to detect
/// overflow, like `io.ReadAll(io.LimitReader(...))`.
pub fn read_bounded<R: std::io::Read>(reader: &mut R, limit: usize) -> Result<Vec<u8>, Error> {
    let mut data = Vec::new();
    let mut chunk = [0u8; 4096];
    loop {
        if data.len() > limit {
            return Err(Error::msg("one bounded SSH public-key file required"));
        }
        match reader.read(&mut chunk) {
            Ok(0) => return Ok(data),
            Ok(n) => data.extend_from_slice(&chunk[..n]),
            Err(_) => return Err(Error::msg("one bounded SSH public-key file required")),
        }
    }
}

pub fn enrollment_private_address(value: &str) -> Result<netip::Addr, Error> {
    // The manual installer currently configures IPv4; do not invent a scope-zone
    // or route for link-local IPv6, wildcard, loopback or public interfaces.
    match netip::parse_addr(value) {
        Ok(address)
            if address.is4()
                && address.is_private()
                && address.to_string_go() == value.as_bytes() =>
        {
            Ok(address)
        }
        _ => Err(Error::msg("selected live RFC1918 IPv4 address required")),
    }
}

pub fn enrollment_client_command(address: &str) -> Result<String, Error> {
    enrollment_private_address(address)?;
    Ok(format!("ssh -T -p {ENROLLMENT_PORT} -o StrictHostKeyChecking=ask -o PreferredAuthentications=password -o PubkeyAuthentication=no -o KbdInteractiveAuthentication=no -o ControlMaster=no -o ControlPath=none -o RequestTTY=no -o ClearAllForwardings=yes root@{address} < ~/.ssh/id_ed25519.pub"))
}

pub fn enrollment_start_args() -> Vec<String> {
    [
        "--quiet",
        "--collect",
        "--unit=soda-key-enrollment.service",
        "--service-type=exec",
        "--property=RuntimeMaxSec=300s",
        "--property=TimeoutStopSec=2s",
        "--property=KillMode=control-group",
        "--property=SendSIGKILL=yes",
        "--property=Restart=no",
        "--property=UMask=0077",
        "--property=StandardInput=null",
        "--property=StandardOutput=null",
        "--property=StandardError=null",
        "--",
        enrollment_binary(),
        "enrollment-serve",
    ]
    .iter()
    .map(|s| s.to_string())
    .collect()
}

// Native systemd owns TCP accept, the two-connection limit, inetd descriptors,
// child supervision and stop ordering. Soda owns only the bounded key transaction.
pub fn enrollment_socket_unit_config(address: &str) -> Result<String, Error> {
    enrollment_private_address(address)?;
    Ok(format!(
        "[Unit]
Description=Soda temporary public-key import socket
BindsTo={ENROLLMENT_UNIT}
After={ENROLLMENT_UNIT}

[Socket]
ListenStream={address}:{ENROLLMENT_PORT}
Accept=yes
MaxConnections=2
"
    ))
}

pub fn enrollment_template_unit_config() -> String {
    format!(
        "[Unit]
Description=Soda temporary public-key import connection
BindsTo={ENROLLMENT_UNIT} {ENROLLMENT_SOCKET_UNIT}
After={ENROLLMENT_UNIT} {ENROLLMENT_SOCKET_UNIT}
CollectMode=inactive-or-failed

[Service]
Type=exec
ExecStart=/usr/sbin/sshd -i -e -f {ENROLLMENT_CONFIG_PATH}
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
"
    )
}
