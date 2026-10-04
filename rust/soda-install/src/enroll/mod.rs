//! Temporary password-window public-key enrollment: the fixed sshd
//! configuration, bounded key parsing, private-address admission, the
//! transient systemd units, and peer provenance.

pub mod arm;
pub mod keys;
pub mod receive;
pub mod selinux;
pub mod serve;
pub mod session;



use crate::errors::Error;
use crate::netip;

pub use session::arm_enrollment;
pub use serve::serve_enrollment;
pub use receive::receive_enrollment;

pub const ENROLLMENT_DIR: &str = "/run/soda-key-enrollment";
pub const ENROLLMENT_UNIT: &str = "soda-key-enrollment.service";
pub const ENROLLMENT_SOCKET_UNIT: &str = "soda-key-enrollment.socket";
pub const ENROLLMENT_TEMPLATE_UNIT: &str = "soda-key-enrollment@.service";
pub const ENROLLMENT_UNIT_DIRECTORY: &str = "/run/systemd/system";
pub const ENROLLMENT_PORT: &str = "22222";
pub const ENROLLMENT_SOCKET: &str = "/run/soda-key-enrollment/receive.sock";
pub const ENROLLMENT_CONFIG_PATH: &str = "/run/soda-key-enrollment/sshd_config";
pub const ENROLLMENT_KEY_LIMIT: usize = 16384;

fn enrollment_binary() -> &'static str {
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

pub fn enrollment_peer_unit(cgroup: &[u8]) -> Result<Vec<u8>, Error> {
    const PREFIX: &[u8] = b"0::/system.slice/";
    let value = cgroup.strip_suffix(b"\n").unwrap_or(cgroup);
    if !value.starts_with(PREFIX) || value.iter().any(|b| *b == b'\r' || *b == b'\n') {
        return Err(Error::msg("dedicated enrollment service peer required"));
    }
    let unit = &value[PREFIX.len()..];
    if unit.contains(&b'/') {
        return Err(Error::msg("dedicated enrollment service peer required"));
    }
    Ok(unit.to_vec())
}

pub fn enrollment_receiver_unit(unit: &[u8]) -> bool {
    const PREFIX: &[u8] = b"soda-key-enrollment@";
    const SUFFIX: &[u8] = b".service";
    unit.len() > PREFIX.len() + SUFFIX.len()
        && unit.starts_with(PREFIX)
        && unit.ends_with(SUFFIX)
        && !unit.iter().any(|b| matches!(b, b'/' | b'\r' | b'\n' | b' ' | b'\t'))
}

// Kernel credentials and the peer's native service cgroup distinguish the fixed
// broker and socket-activated receiver instances from ordinary root SSH shells.
pub fn enrollment_connection_unit(stream: &std::os::unix::net::UnixStream) -> Result<Vec<u8>, Error> {
    use std::os::unix::io::AsRawFd;
    let mut ucred: libc::ucred = unsafe { std::mem::zeroed() };
    let mut len = std::mem::size_of::<libc::ucred>() as libc::socklen_t;
    if unsafe {
        libc::getsockopt(
            stream.as_raw_fd(),
            libc::SOL_SOCKET,
            libc::SO_PEERCRED,
            &mut ucred as *mut libc::ucred as *mut libc::c_void,
            &mut len,
        )
    } != 0
    {
        let errno = unsafe { *libc::__errno_location() };
        return Err(crate::errors::os_error(std::io::Error::from_raw_os_error(errno)));
    }
    if ucred.uid != 0 || ucred.pid <= 0 {
        return Err(Error::msg("dedicated native root enrollment peer required"));
    }
    let group = std::fs::read(format!("/proc/{}/cgroup", ucred.pid)).map_err(crate::errors::os_error)?;
    enrollment_peer_unit(&group)
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;

    pub const TEST_KEY: &str = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKQ0MsA1tWa7risZNVfq58qNB9BByJfSJUQpWI9KvglV";
    pub const TEST_KEY_2: &str = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCoDZAt7ewKwwXBa7tCvC9/+p//wMupqSjGVnTOvoYOUt1UNbromMK5hBZGq2xIlqJQ0rRZoTEtsE7w5BUgkNzvsioTnnjD48UeqMvJhBfSrJYTVMZeM/ttm1jmlNhbjs6nt98R/KmdsyK+2a+z5BQ+KRlr4kbKfd/UDOPj8XuA2/vW4K2301UUdDk9Jh2r/bcjRnrIyHUX1Rmga608tAWRZtJQRo+8/28JqnjQM5s4qcu1d1N2Y823P4YGaLYhRLoKhV1/gCMRRhD9ZTZpn58sVmiEGQ90YeHE/8vETm+Q2IkjZvx2vobzhvdsA3LGs3B1EN2kdbCGJRqaltrJLK+t";
    pub const TEST_KEY_3: &str = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBLtQL7Kq7o6tng5YEyRN3ICkkd2BErzxr+p3zkns1Apc0BJ7BDcTZqzWuusUsWLZxRtnOVtz2FT2vd0GlCz20RI=";

    /// Serializes tests that mutate process environment, which Rust runs
    /// in parallel threads unlike Go's sequential test binary.
    pub static ENV_LOCK: std::sync::Mutex<()> = std::sync::Mutex::new(());

    pub struct TempDir {
        pub path: String,
    }

    impl Drop for TempDir {
        fn drop(&mut self) {
            let _ = std::fs::remove_dir_all(&self.path);
        }
    }

    /// Go `t.TempDir`: fresh 0700 directory, removed on drop.
    pub fn temp_dir() -> TempDir {
        let template = std::ffi::CString::new("/tmp/soda-enroll-test-XXXXXX").unwrap();
        let raw = template.into_raw();
        let ok = unsafe { libc::mkdtemp(raw) };
        let template = unsafe { std::ffi::CString::from_raw(raw) };
        assert!(!ok.is_null());
        TempDir { path: template.to_string_lossy().into_owned() }
    }

    pub fn test_uid() -> u32 {
        unsafe { libc::geteuid() }
    }

    pub struct TestPty {
        pub master: std::fs::File,
        pub slave_path: String,
    }

    pub fn open_test_pty() -> TestPty {
        use std::os::unix::io::FromRawFd;
        let mut master = 0;
        let mut slave = 0;
        assert_eq!(
            unsafe {
                libc::openpty(
                    &mut master,
                    &mut slave,
                    std::ptr::null_mut(),
                    std::ptr::null_mut(),
                    std::ptr::null_mut(),
                )
            },
            0
        );
        // ptsname_r: ptsname's static buffer races across test threads.
        let mut name = [0 as libc::c_char; 64];
        assert_eq!(unsafe { libc::ptsname_r(master, name.as_mut_ptr(), name.len()) }, 0);
        let slave_path =
            unsafe { std::ffi::CStr::from_ptr(name.as_ptr()).to_string_lossy().into_owned() };
        unsafe { libc::close(slave) };
        TestPty { master: unsafe { std::fs::File::from_raw_fd(master) }, slave_path }
    }

    /// Drain whatever the console already wrote, stopping after 300 ms idle.
    pub fn drain_available(master: &mut std::fs::File) -> Vec<u8> {
        use std::io::Read;
        use std::os::unix::io::AsRawFd;
        let mut transcript = Vec::new();
        loop {
            let mut fd = libc::pollfd { fd: master.as_raw_fd(), events: libc::POLLIN, revents: 0 };
            if unsafe { libc::poll(&mut fd, 1, 300) } <= 0 || fd.revents & libc::POLLIN == 0 {
                break;
            }
            let mut buffer = [0u8; 4096];
            match master.read(&mut buffer) {
                Ok(0) => break,
                Ok(n) => transcript.extend_from_slice(&buffer[..n]),
                Err(_) => break,
            }
        }
        transcript
    }

    #[test]
    fn enrollment_input_matrix() {
        for value in [TEST_KEY.to_string(), format!("{TEST_KEY}\n"), format!("{TEST_KEY} laptop\n")] {
            assert_eq!(enrollment_public_key(value.as_bytes()).unwrap(), TEST_KEY, "{value:?}");
        }
        let big = "x".repeat(ENROLLMENT_KEY_LIMIT + 1);
        for value in [
            String::new(),
            format!("{TEST_KEY}\n\n"),
            format!("{TEST_KEY}\n{TEST_KEY}\n"),
            format!("command=\"sh\" {TEST_KEY}"),
            "-----BEGIN OPENSSH PRIVATE KEY-----".to_string(),
            big,
            format!("{TEST_KEY}\r\n"),
            format!("{TEST_KEY}\x00"),
        ] {
            assert!(enrollment_public_key(value.as_bytes()).is_err(), "{value:?}");
        }
    }

    #[test]
    fn enrollment_private_address_and_client() {
        for address in ["10.0.0.8", "172.16.3.4", "192.168.1.20"] {
            let command = enrollment_client_command(address).unwrap();
            assert!(command.contains(&format!("root@{address} < ~/.ssh/id_ed25519.pub")), "{command}");
            assert!(command.contains("ControlPath=none"), "{command}");
            assert!(command.contains("ClearAllForwardings=yes"), "{command}");
        }
        for address in [
            "",
            "0.0.0.0",
            "127.0.0.1",
            "169.254.1.1",
            "8.8.8.8",
            "100.64.0.1",
            "::1",
            "fd00::1",
            "192.168.1.1\nPermitTTY yes",
            "192.168.1.01",
        ] {
            assert!(enrollment_client_command(address).is_err(), "{address:?}");
        }
    }

    #[test]
    fn enrollment_native_policy() {
        let config = enrollment_config();
        for line in [
            "PermitRootLogin yes",
            "AuthenticationMethods password",
            "PasswordAuthentication yes",
            "UsePAM no",
            "KbdInteractiveAuthentication no",
            "PubkeyAuthentication no",
            "PermitEmptyPasswords no",
            "DisableForwarding yes",
            "PermitTunnel no",
            "PermitTTY no",
            "PermitUserRC no",
            "PermitUserEnvironment no",
            "MaxSessions 1",
            "MaxAuthTries 3",
            "AuthorizedKeysFile none",
            &format!("ForceCommand {} enrollment-receive", enrollment_binary()),
        ] {
            assert!(config.contains(&format!("\n{line}\n")), "missing {line}");
        }
        for forbidden in ["Include ", "AcceptEnv ", "Subsystem ", "ListenAddress ", "Match ", "SetEnv "] {
            assert!(!config.contains(forbidden), "unexpected {forbidden:?}");
        }
        let args = enrollment_start_args().join(" ");
        for required in [
            "--unit=soda-key-enrollment.service",
            "--property=RuntimeMaxSec=300s",
            "--property=KillMode=control-group",
            "--property=SendSIGKILL=yes",
            "--property=Restart=no",
            "--property=TimeoutStopSec=2s",
        ] {
            assert!(args.contains(required), "missing {required}");
        }
    }

    #[test]
    fn enrollment_native_socket_ownership() {
        let config = enrollment_socket_unit_config("192.168.1.20").unwrap();
        for line in [
            "BindsTo=soda-key-enrollment.service",
            "After=soda-key-enrollment.service",
            "ListenStream=192.168.1.20:22222",
            "Accept=yes",
            "MaxConnections=2",
        ] {
            assert!(config.contains(&format!("\n{line}\n")), "missing {line}");
        }
        assert!(enrollment_socket_unit_config("0.0.0.0\nAccept=no").is_err());
        let connection = enrollment_template_unit_config();
        for line in [
            "BindsTo=soda-key-enrollment.service soda-key-enrollment.socket",
            "After=soda-key-enrollment.service soda-key-enrollment.socket",
            "StandardInput=socket",
            "StandardOutput=socket",
            "Slice=system.slice",
            "RuntimeMaxSec=300s",
            "KillMode=control-group",
            "TimeoutStopSec=2s",
            "SendSIGKILL=yes",
            "Restart=no",
            "ExecStart=/usr/sbin/sshd -i -e -f /run/soda-key-enrollment/sshd_config",
        ] {
            assert!(connection.contains(&format!("\n{line}\n")), "missing {line}");
        }
        assert!(!(config + &connection).contains("[Install]"));
    }

    #[test]
    fn enrollment_peer_service_provenance() {
        let unit = enrollment_peer_unit(b"0::/system.slice/soda-key-enrollment.service\n").unwrap();
        assert_eq!(unit, b"soda-key-enrollment.service");
        assert!(!enrollment_receiver_unit(&unit));
        let unit = enrollment_peer_unit(b"0::/system.slice/soda-key-enrollment@0-192.168.1.20:22222-192.168.1.30:43123.service\n").unwrap();
        assert!(enrollment_receiver_unit(&unit));
        for group in [
            "0::/user.slice/user-0.slice/session-1.scope\n",
            "0::/system.slice/soda-key-enrollment@x.service/subgroup\n",
            "0::/system.slice/soda-key-enrollment@x.service\n0::/system.slice/sshd.service\n",
        ] {
            assert!(enrollment_peer_unit(group.as_bytes()).is_err(), "{group:?}");
        }
        for unit in [
            "soda-key-enrollment@.service",
            "sshd.service",
            "soda-key-enrollment-other@x.service",
            "soda-key-enrollment@x.service/child",
            "soda-key-enrollment@x.service\n",
        ] {
            assert!(!enrollment_receiver_unit(unit.as_bytes()), "{unit:?}");
        }
    }
}
