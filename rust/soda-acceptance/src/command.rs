//! Command execution with separated results, mirroring `command.go`.
//!
//! [`execute`] runs one labeled command with redacted capture: the result
//! carries execution failure and the return value carries evidence
//! retention failure, and both must be checked. [`Remote`] builds pinned
//! SSH invocations; nothing here performs trust refresh or proxying.

use std::net::IpAddr;
use std::str::FromStr;
use std::sync::{Arc, Mutex};
use std::time::Duration;

use soda_json::JsonValue;

use crate::error::Error;
use crate::evidence::{Evidence, RedactingWriter};
use crate::files;
use crate::jsonio;
use crate::process::{self, Phase, SharedWriter};

/// Standard input wiring for a spawned command.
#[derive(Debug, Clone)]
pub enum StdinSpec {
    /// Inherit the caller's stdin, like Go's `os.Stdin`.
    Inherit,
    /// Empty stdin.
    Null,
    /// Pumped input bytes.
    Bytes(Vec<u8>),
}

/// Command description, mirroring Go's `Command`.
#[derive(Debug, Clone)]
pub struct CommandSpec {
    /// Program name or path.
    pub name: String,
    /// Arguments.
    pub args: Vec<String>,
    /// Working directory, when set.
    pub dir: Option<String>,
    /// Standard input wiring.
    pub stdin: StdinSpec,
    /// Extra `KEY=value` environment entries.
    pub env: Vec<String>,
}

/// Separated command outcome, mirroring Go's `Result`.
pub struct CommandResult {
    /// Redacted captured stdout.
    pub stdout: Vec<u8>,
    /// Redacted captured stderr.
    pub stderr: Vec<u8>,
    /// Execution failure, if any.
    pub err: Option<Error>,
    /// True once the process started.
    pub started: bool,
    /// Exit code, or `-1` for signal termination.
    pub exit_code: Option<i32>,
}

impl Default for CommandResult {
    fn default() -> CommandResult {
        CommandResult {
            stdout: Vec::new(),
            stderr: Vec::new(),
            err: None,
            started: false,
            exit_code: None,
        }
    }
}

/// Shell-quote argv for one SSH remote command, byte-identical to Go's
/// `Quote`: single quotes with `'"'"'` splices.
pub fn quote(args: &[String]) -> String {
    args.iter()
        .map(|s| format!("'{}'", s.replace('\'', "'\"'\"'")))
        .collect::<Vec<_>>()
        .join(" ")
}

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
pub fn decode_remote(value: &JsonValue) -> Result<Remote, Error> {
    jsonio::check_no_unknown(value, &["User", "Host", "Port", "Key", "KnownHosts", "Timeout"])?;
    let port = jsonio::opt_integer(value, "Port")?;
    let port: i64 = port.try_into().map_err(|_| Error::msg("invalid Port: integer required"))?;
    Ok(Remote {
        user: jsonio::opt_string(value, "User")?,
        host: jsonio::opt_string(value, "Host")?,
        key: jsonio::opt_string(value, "Key")?,
        known_hosts: jsonio::opt_string(value, "KnownHosts")?,
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
        look_path("ssh")?;
        let args = self.args()?;
        loop {
            if ssh_true(&args) {
                return Ok(());
            }
            if let Err(e) = phase.check() {
                return Err(Error::wrap("pinned SSH readiness", e));
            }
            std::thread::sleep(Duration::from_secs(1));
        }
    }
}

pub(crate) fn look_path(name: &str) -> Result<(), Error> {
    if let Some(paths) = std::env::var_os("PATH") {
        for dir in std::env::split_paths(&paths) {
            let candidate = dir.join(name);
            if let Ok(meta) = std::fs::symlink_metadata(&candidate) {
                use std::os::unix::fs::MetadataExt;
                if meta.is_file() && meta.mode() & 0o111 != 0 {
                    return Ok(());
                }
            }
        }
    }
    Err(Error::msg(format!("exec: {name:?}: executable file not found in $PATH")))
}

fn ssh_true(args: &[String]) -> bool {
    let mut command = std::process::Command::new("ssh");
    command.args(args).arg("true");
    command.stdin(std::process::Stdio::null());
    command.stdout(std::process::Stdio::null());
    command.stderr(std::process::Stdio::null());
    let mut child = match command.spawn() {
        Ok(child) => child,
        Err(_) => return false,
    };
    let deadline = std::time::Instant::now() + Duration::from_secs(12);
    loop {
        match child.try_wait() {
            Ok(Some(status)) => return status.success(),
            Ok(None) => {
                if std::time::Instant::now() >= deadline {
                    let _ = child.kill();
                    let _ = child.wait();
                    return false;
                }
                std::thread::sleep(Duration::from_millis(50));
            }
            Err(_) => return false,
        }
    }
}

fn valid_command_label(label: &str) -> bool {
    !label.is_empty() && label.chars().all(|c| matches!(c, 'a'..='z' | 'A'..='Z' | '0'..='9' | '_' | '-'))
}

fn redact_opt(evidence: &Evidence, err: Option<Error>) -> Option<Error> {
    err.map(|e| evidence.redact_error(e))
}

fn close_shared(writer: &SharedWriter) -> Option<Error> {
    writer
        .lock()
        .unwrap_or_else(|e| e.into_inner())
        .close()
        .err()
}

/// Execute one labeled command with redacted capture. The result carries
/// execution failure; the return value carries evidence retention failure.
/// Both must be checked, including expected denials.
pub fn execute(
    phase: &Phase,
    evidence: &Evidence,
    label: &str,
    spec: &CommandSpec,
) -> (CommandResult, Option<Error>) {
    if !valid_command_label(label) {
        return (CommandResult::default(), Some(Error::msg("invalid command label")));
    }
    let out_file = match evidence.open_file(&format!("{label}.stdout")) {
        Ok(file) => file,
        Err(e) => return (CommandResult::default(), Some(e)),
    };
    let err_file = match evidence.open_file(&format!("{label}.stderr")) {
        Ok(file) => file,
        Err(e) => {
            drop(out_file);
            return (CommandResult::default(), Some(e));
        }
    };
    let out: SharedWriter = Arc::new(Mutex::new(RedactingWriter::tee(out_file, evidence.secrets().to_vec())));
    let err_writer: SharedWriter = Arc::new(Mutex::new(RedactingWriter::tee(err_file, evidence.secrets().to_vec())));
    let process = match process::start_process(phase, spec, out.clone(), err_writer.clone()) {
        Ok(process) => process,
        Err(run_err) => {
            let write_err = Error::join(vec![close_shared(&out), close_shared(&err_writer)]);
            let mut result = CommandResult::default();
            result.err = Some(evidence.redact_error(run_err));
            return (result, redact_opt(evidence, write_err));
        }
    };
    let mut run_err = process.wait(phase).err();
    if phase.check().is_err() {
        run_err = Error::join(vec![run_err, phase.check().err(), process.stop().err()]);
    }
    if !process.is_done() {
        let out_close = out.clone();
        let err_close = err_writer.clone();
        std::thread::spawn(move || {
            let _ = process.wait(&Phase::background());
            let _ = close_shared(&out_close);
            let _ = close_shared(&err_close);
        });
        let mut result = CommandResult::default();
        result.started = true;
        result.err = redact_opt(evidence, run_err);
        return (result, Some(Error::msg("evidence still owned by incomplete process cleanup")));
    }
    let pump_err = process.join_pumps();
    let close_err = Error::join(vec![close_shared(&out), close_shared(&err_writer)]);
    let outcome = process.outcome();
    let exit_code = outcome.as_ref().and_then(|o| o.exit_code);
    let (run_err, write_err) = match (&outcome, exit_code) {
        (Some(o), Some(0)) if pump_err.is_some() && phase.check().is_ok() => {
            // Copy failures after exit zero belong to retention, not a
            // native denial. Group cleanup failures remain execution failures.
            (
                o.cleanup_message.clone().map(Error::msg),
                Error::join(vec![close_err, pump_err.map(Error::msg)]),
            )
        }
        _ => (run_err, close_err),
    };
    let stdout = out.lock().unwrap_or_else(|e| e.into_inner()).buffer().to_vec();
    let stderr = err_writer.lock().unwrap_or_else(|e| e.into_inner()).buffer().to_vec();
    let mut result = CommandResult {
        stdout,
        stderr,
        err: None,
        started: true,
        exit_code,
    };
    if let Some(e) = run_err {
        result.err = Some(evidence.redact_error(Error::wrap("command execution", e)));
    }
    (result, redact_opt(evidence, write_err))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn fixture_evidence() -> (std::path::PathBuf, Evidence) {
        let mut dir = std::env::temp_dir();
        dir.push(format!("soda-command-{}-{}", std::process::id(), fresh_id()));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("evidence").to_string_lossy().into_owned();
        let evidence = crate::evidence::create_evidence(&path, &[]).unwrap();
        (dir, evidence)
    }

    fn fresh_id() -> u64 {
        use std::sync::atomic::{AtomicU64, Ordering};
        static NEXT: AtomicU64 = AtomicU64::new(0);
        NEXT.fetch_add(1, Ordering::SeqCst)
    }

    #[test]
    fn literal_remote_arguments() {
        let values = ["", "a b", "'quoted'", "a; b", "$(false)", "two\nlines"];
        let mut args = vec!["printf".to_string(), "%s\\0".to_string()];
        args.extend(values.iter().map(|s| s.to_string()));
        let output = std::process::Command::new("/bin/sh")
            .arg("-c")
            .arg(quote(&args))
            .output()
            .unwrap();
        assert!(output.status.success());
        let mut want = values.join("\x00");
        want.push('\x00');
        assert_eq!(String::from_utf8_lossy(&output.stdout), want);
    }

    #[test]
    fn pinned_ssh_options() {
        let mut dir = std::env::temp_dir();
        dir.push(format!("soda-remote-{}-{}", std::process::id(), fresh_id()));
        std::fs::create_dir_all(&dir).unwrap();
        let key = dir.join("key");
        let hosts = dir.join("known_hosts");
        std::fs::write(&key, b"synthetic identity, not used for authentication").unwrap();
        std::fs::write(&hosts, b"synthetic pin, not used for authentication").unwrap();
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&key, std::fs::Permissions::from_mode(0o600)).unwrap();
        let remote = Remote {
            user: "root".to_string(),
            host: "127.0.0.1".to_string(),
            key: key.to_string_lossy().into_owned(),
            known_hosts: hosts.to_string_lossy().into_owned(),
            port: 22222,
            timeout: Duration::ZERO,
        };
        let command = remote
            .command(&["printf".to_string(), "%s".to_string(), "a b".to_string()], StdinSpec::Bytes(b"input".to_vec()))
            .unwrap();
        let joined = command.args.join(" ");
        for required in [
            "StrictHostKeyChecking=yes",
            "IdentitiesOnly=yes",
            "GlobalKnownHostsFile=/dev/null",
            "timeout",
            "--kill-after=10s",
        ] {
            assert!(joined.contains(required), "{joined}");
        }
        assert!(!joined.contains("accept-new"), "{joined}");
        assert!(!joined.contains("ProxyJump"), "{joined}");
        std::fs::set_permissions(&key, std::fs::Permissions::from_mode(0o644)).unwrap();
        assert!(remote.args().is_err());
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    #[cfg(target_os = "linux")]
    fn command_and_evidence_failures_are_separate() {
        let (_dir, evidence) = secret_evidence();
        let spec = CommandSpec {
            name: "/bin/sh".to_string(),
            args: vec!["-c".to_string(), "read -r value; printf '%s\\n' \"$value\"; printf 'expected denial\\n' >&2; exit 23".to_string()],
            dir: None,
            stdin: StdinSpec::Bytes(b"synthetic-password\n".to_vec()),
            env: Vec::new(),
        };
        let (result, err) = execute(&Phase::background(), &evidence, "denied", &spec);
        assert!(err.is_none());
        assert!(result.started);
        assert_eq!(result.exit_code, Some(23));
        assert!(result.err.is_some());
        assert!(!String::from_utf8_lossy(&result.stdout).contains("synthetic-password"));

        evidence.write("collision.stdout", b"").unwrap();
        let idle = CommandSpec {
            name: "/bin/sh".to_string(),
            args: vec!["-c".to_string(), "exit 0".to_string()],
            dir: None,
            stdin: StdinSpec::Null,
            env: Vec::new(),
        };
        let (result, err) = execute(&Phase::background(), &evidence, "collision", &idle);
        assert!(err.is_some());
        assert!(!result.started);
        assert!(result.err.is_none());
    }

    #[test]
    #[cfg(target_os = "linux")]
    fn cancelled_command_is_not_denial_or_success() {
        let (_dir, evidence) = fixture_evidence_pair();
        let phase = Phase::background();
        phase.cancel();
        let spec = CommandSpec {
            name: "/bin/sh".to_string(),
            args: vec!["-c".to_string(), "exit 0".to_string()],
            dir: None,
            stdin: StdinSpec::Null,
            env: Vec::new(),
        };
        let (result, err) = execute(&phase, &evidence, "cancelled", &spec);
        assert!(err.is_none());
        assert!(!result.started);
        assert!(result.err.as_ref().is_some_and(|e| e.is_cancelled()));
    }

    fn secret_evidence() -> (std::path::PathBuf, Evidence) {
        let mut dir = std::env::temp_dir();
        dir.push(format!("soda-command-secret-{}-{}", std::process::id(), fresh_id()));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("evidence").to_string_lossy().into_owned();
        let evidence = crate::evidence::create_evidence(&path, &[b"synthetic-password".to_vec()]).unwrap();
        (dir, evidence)
    }

    fn fixture_evidence_pair() -> (std::path::PathBuf, Evidence) {
        fixture_evidence()
    }
}
