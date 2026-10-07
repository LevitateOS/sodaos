use std::net::IpAddr;
use std::path::Path;
use std::process::{Child, Command, Stdio};
use std::str::FromStr;
use std::time::{Duration, Instant};

use serde_json::value::RawValue;
use std::collections::BTreeMap;

use crate::error::Error;
use crate::files;
use crate::process::Phase;

use super::{CommandSpec, StdinSpec, look_path, quote};

/// Pinned SSH endpoint. Keys are file references, never inline contents.
pub struct Remote {
    /// Login user.
    pub user: String,
    /// Host IP or DNS name.
    pub host: String,
    /// Private identity file.
    pub key: String,
    /// Pinned known_hosts file.
    pub known_hosts: String,
    /// SSH port.
    pub port: i64,
    /// Remote deadline. Never decoded from JSON, like Go's `json:"-"`.
    pub timeout: Duration,
}

/// Decode connection JSON with Go field names and no unknown fields.
/// `Timeout` is admitted and ignored, like Go's `json:"-"` under
/// `DisallowUnknownFields`; the driver always sets the deadline.
pub fn decode_remote(value: &RawValue) -> Result<Remote, Error> {
    let fields = match serde_json::from_str::<BTreeMap<String, Box<RawValue>>>(value.get()) {
        Ok(fields) => fields,
        Err(_) if !value.get().trim_start().starts_with('{') => BTreeMap::new(),
        Err(_) => return Err(Error::msg("invalid SSH configuration")),
    };
    for key in fields.keys() {
        if !["User", "Host", "Port", "Key", "KnownHosts", "Timeout"].contains(&key.as_str()) {
            return Err(Error::msg("unknown JSON field"));
        }
    }
    let string_field = |name: &str| -> Result<String, Error> {
        match fields.get(name) {
            None => Ok(String::new()),
            Some(raw) => serde_json::from_str::<Option<String>>(raw.get())
                .map(|v| v.unwrap_or_default())
                .map_err(|_| Error::msg("invalid JSON string field")),
        }
    };
    let port = match fields.get("Port") {
        None => 0,
        Some(raw) if raw.get() == "null" => 0,
        Some(raw) => raw
            .get()
            .parse::<i128>()
            .map_err(|_| Error::msg("invalid Port: integer required"))?,
    };
    let port: i64 = port
        .try_into()
        .map_err(|_| Error::msg("invalid Port: integer required"))?;
    Ok(Remote {
        user: string_field("User")?,
        host: string_field("Host")?,
        key: string_field("Key")?,
        known_hosts: string_field("KnownHosts")?,
        port,
        timeout: Duration::ZERO,
    })
}

fn valid_ssh_user(user: &str) -> bool {
    let mut chars = user.chars();
    match chars.next() {
        Some('a'..='z') | Some('_') => {}
        _ => return false,
    }
    chars.all(|c| matches!(c, 'a'..='z' | '0'..='9' | '_' | '-'))
}

fn valid_ssh_host(host: &str) -> bool {
    if IpAddr::from_str(host).is_ok() {
        return true;
    }
    let mut chars = host.chars();
    match chars.next() {
        Some('A'..='Z') | Some('a'..='z') | Some('0'..='9') => {}
        _ => return false,
    }
    chars.all(|c| matches!(c, 'A'..='Z' | 'a'..='z' | '0'..='9' | '.' | '-'))
}

fn valid_ssh_port(port: i64) -> bool {
    (1..=65535).contains(&port)
}

fn trusted_known_hosts(path: &str) -> Result<(), Error> {
    if !path.starts_with('/') {
        return Err(Error::msg("absolute pinned known_hosts required"));
    }
    let meta = std::fs::symlink_metadata(path)?;
    use std::os::unix::fs::MetadataExt;
    if !meta.is_file() || meta.mode() & 0o022 != 0 || meta.size() == 0 {
        return Err(Error::msg("trusted regular known_hosts required"));
    }
    Ok(())
}

impl Remote {
    /// Pinned SSH options, byte-identical to Go's `Remote.Args`.
    pub fn args(&self) -> Result<Vec<String>, Error> {
        if !valid_ssh_user(&self.user) || !valid_ssh_port(self.port) {
            return Err(Error::msg("invalid SSH user/port"));
        }
        if !valid_ssh_host(&self.host) {
            return Err(Error::msg("invalid SSH host"));
        }
        files::private_file(&self.key)?;
        trusted_known_hosts(&self.known_hosts)?;
        Ok(vec![
            "-F".to_string(),
            "/dev/null".to_string(),
            "-T".to_string(),
            "-o".to_string(),
            "BatchMode=yes".to_string(),
            "-o".to_string(),
            "IdentitiesOnly=yes".to_string(),
            "-o".to_string(),
            "StrictHostKeyChecking=yes".to_string(),
            "-o".to_string(),
            "GlobalKnownHostsFile=/dev/null".to_string(),
            "-o".to_string(),
            format!("UserKnownHostsFile={}", self.known_hosts),
            "-o".to_string(),
            "ConnectTimeout=10".to_string(),
            "-o".to_string(),
            "ServerAliveInterval=15".to_string(),
            "-o".to_string(),
            "ServerAliveCountMax=2".to_string(),
            "-i".to_string(),
            self.key.clone(),
            "-p".to_string(),
            self.port.to_string(),
            format!("{}@{}", self.user, self.host),
        ])
    }

    /// SSH command running `args` under a bounded remote deadline.
    pub fn command(&self, args: &[String], stdin: StdinSpec) -> Result<CommandSpec, Error> {
        let base = self.args()?;
        let duration = if self.timeout.is_zero() {
            Duration::from_secs(30 * 60)
        } else {
            self.timeout
        };
        if duration.is_zero() || duration > Duration::from_secs(24 * 3600) {
            return Err(Error::msg("bounded remote deadline required"));
        }
        let mut bounded = vec![
            "timeout".to_string(),
            "--signal=TERM".to_string(),
            "--kill-after=10s".to_string(),
            format!("{:.3}s", duration.as_secs_f64()),
        ];
        bounded.extend(args.iter().cloned());
        let mut ssh_args = base;
        ssh_args.push(quote(&bounded));
        Ok(CommandSpec {
            name: "ssh".to_string(),
            args: ssh_args,
            dir: None,
            stdin,
            env: Vec::new(),
        })
    }

    /// Wait for pinned SSH readiness, retrying `ssh true` until the phase
    /// ends, like Go's `Remote.WaitReady`.
    pub fn wait_ready(&self, phase: &Phase) -> Result<(), Error> {
        phase.check()?;
        look_path("ssh")?;
        let args = self.args()?;
        wait_ready_with(phase, Path::new("ssh"), &args, Duration::from_secs(12))
    }
}

pub(super) fn wait_ready_with(
    phase: &Phase,
    ssh_program: &Path,
    args: &[String],
    attempt_limit: Duration,
) -> Result<(), Error> {
    loop {
        phase.check()?;
        if ssh_true_with(ssh_program, args, phase, attempt_limit)? {
            phase.check()?;
            return Ok(());
        }
        phase.check()?;
        let sleep = phase
            .deadline()
            .map(|deadline| {
                Duration::from_secs(1).min(deadline.saturating_duration_since(Instant::now()))
            })
            .unwrap_or(Duration::from_secs(1));
        if sleep.is_zero() {
            phase.check()?;
        }
        std::thread::sleep(sleep);
    }
}

pub(super) fn ssh_true_with(
    ssh_program: &Path,
    args: &[String],
    phase: &Phase,
    attempt_limit: Duration,
) -> Result<bool, Error> {
    phase.check()?;
    let now = Instant::now();
    let attempt_deadline =
        (now + attempt_limit).min(phase.deadline().unwrap_or(now + attempt_limit));
    phase.check()?;

    let mut command = Command::new(ssh_program);
    command.args(args).arg("true");
    command.stdin(Stdio::null());
    command.stdout(Stdio::null());
    command.stderr(Stdio::null());
    let mut child = match command.spawn() {
        Ok(child) => child,
        Err(_) => {
            phase.check()?;
            return Ok(false);
        }
    };
    loop {
        if let Err(primary) = phase.check() {
            return Err(
                Error::join(vec![Some(primary), stop_ssh_child(&mut child).err()]).unwrap(),
            );
        }
        match child.try_wait() {
            Ok(Some(status)) => {
                // The process may have exited successfully after the phase
                // or attempt ended but before this thread observed it.
                phase.check()?;
                return Ok(status.success() && Instant::now() < attempt_deadline);
            }
            Ok(None) if Instant::now() >= attempt_deadline => {
                stop_ssh_child(&mut child)?;
                return Ok(false);
            }
            Ok(None) => {
                let remaining = attempt_deadline.saturating_duration_since(Instant::now());
                std::thread::sleep(Duration::from_millis(25).min(remaining));
            }
            Err(wait_error) => {
                let wait_error = Error::wrap("SSH readiness client wait failed", wait_error.into());
                return Err(
                    Error::join(vec![Some(wait_error), stop_ssh_child(&mut child).err()]).unwrap(),
                );
            }
        }
    }
}

fn stop_ssh_child(child: &mut Child) -> Result<(), Error> {
    if matches!(child.try_wait(), Ok(Some(_))) {
        return Ok(());
    }
    let kill_error = child.kill().err().map(Error::from);
    match child.wait() {
        Ok(_) => Ok(()),
        Err(wait_error) => Err(Error::join(vec![kill_error, Some(wait_error.into())]).unwrap()),
    }
}

#[cfg(test)]
mod json_tests {
    use super::decode_remote;
    use serde_json::value::RawValue;

    #[test]
    fn remote_record_uses_last_exact_fields_before_type_decoding() {
        let raw = RawValue::from_string(
            r#"{"User":false,"User":"root","Port":1e400,"Port":22222,"Timeout":{"ignored":true}}"#
                .to_string(),
        )
        .unwrap();
        let decoded = decode_remote(&raw).unwrap();
        assert_eq!(decoded.user, "root");
        assert_eq!(decoded.port, 22222);
        let fractional = RawValue::from_string(r#"{"Port":2.5}"#.to_string()).unwrap();
        assert!(decode_remote(&fractional).is_err());
        let unknown = RawValue::from_string(r#"{"Other":null}"#.to_string()).unwrap();
        assert!(decode_remote(&unknown).is_err());
    }
}
