//! Worker execution: argv construction, process run, and output drains.
use std::process::{Command, Stdio};
use std::time::{Duration, Instant};

use super::config::{euid, trusted_executable, Worker};
use super::runtime::valid_worker_name;

// Isolated-worker execution boundary (Go `internal/acceptance`
// `worker_linux.go` `arguments` + `Run`, `tools/soda-build`
// `runBuildWorker` + `resolveWorkerLiveInputs`). Identity and
// executable admission reuse `valid_worker_name` (which matches the Go
// `^soda-(build|qualify)-[a-z0-9-]{1,48}$` rule exactly) and
// `trusted_executable` (root-owned, non-writable, symlink-free chain
// with a regular executable leaf, as in Go).

fn valid_worker_identity(w: &Worker) -> Result<(), String> {
    if !valid_worker_name(&w.name) {
        return Err("exact task worker name required".to_owned());
    }
    if w.user != "soda-build-worker" && w.user != "soda-qualifier" {
        return Err("separate approved worker identity required".to_owned());
    }
    if !w.directory.starts_with('/')
        || w.directory
            .chars()
            .any(|c| c == '\n' || c == '\r' || c == ':')
    {
        return Err("absolute worker directory required".to_owned());
    }
    trusted_executable(&w.executable)
}

fn append_bind_paths(
    mut args: Vec<String>,
    property: &str,
    paths: &[String],
) -> Result<Vec<String>, String> {
    for pair in paths {
        let parts: Vec<&str> = pair.split(':').collect();
        let ok = parts.len() == 2
            && parts[0].starts_with('/')
            && parts[1].starts_with('/')
            && !pair
                .chars()
                .any(|c| c == '\n' || c == '\r' || c == '\t' || c == ' ' || c == '%');
        if !ok {
            return Err("explicit absolute worker bind pair required".to_owned());
        }
        args.push(format!("--property={property}={pair}"));
    }
    Ok(args)
}

fn allowed_worker_env_key(key: &str) -> bool {
    matches!(
        key,
        "HOME"
            | "PATH"
            | "XDG_RUNTIME_DIR"
            | "GOTOOLCHAIN"
            | "GOCACHE"
            | "GOMODCACHE"
            | "CARGO_HOME"
            | "CARGO_TARGET_DIR"
            | "CARGO_NET_OFFLINE"
            | "BUN_INSTALL_CACHE_DIR"
            | "PLAYWRIGHT_BROWSERS_PATH"
            | "SODA_BUILD_START_NS"
    )
}

fn append_worker_env(mut args: Vec<String>, environment: &[String]) -> Result<Vec<String>, String> {
    for env in environment {
        let (key, ok) = match env.split_once('=') {
            Some((key, _)) => (key, true),
            None => (env.as_str(), false),
        };
        if !allowed_worker_env_key(key) {
            return Err("worker environment key refused".to_owned());
        }
        if !ok || env.chars().any(|c| c == '\n' || c == '\r' || c == '\0') {
            return Err("invalid worker environment".to_owned());
        }
        args.push(format!("--setenv={env}"));
    }
    Ok(args)
}

/// `acceptance.Worker.arguments`: the exact `systemd-run` transient-unit
/// argv for one admitted worker dispatch.
pub fn worker_argv(w: &Worker) -> Result<Vec<String>, String> {
    valid_worker_identity(w)?;
    let mut args = vec![
        "--quiet".to_owned(),
        "--wait".to_owned(),
        "--pipe".to_owned(),
        "--collect".to_owned(),
        "--service-type=exec".to_owned(),
        format!("--unit={}", w.name),
        format!("--property=User={}", w.user),
        format!("--property=Group={}", w.user),
        format!("--property=WorkingDirectory={}", w.directory),
        "--property=ProtectHome=tmpfs".to_owned(),
        "--property=ProtectSystem=strict".to_owned(),
        "--property=PrivateTmp=yes".to_owned(),
        "--property=PrivateMounts=yes".to_owned(),
        "--property=Delegate=yes".to_owned(),
        "--property=CPUQuota=400%".to_owned(),
        "--property=MemoryMax=16G".to_owned(),
        "--property=CPUAffinity=0 1 2 3".to_owned(),
        "--property=KillMode=control-group".to_owned(),
        "--property=TimeoutStopSec=20s".to_owned(),
        "--property=UMask=0077".to_owned(),
        "--property=InaccessiblePaths=-/var/lib/soda-release -/root".to_owned(),
    ];
    args = append_bind_paths(args, "BindReadOnlyPaths", &w.read_only)?;
    args = append_bind_paths(args, "BindPaths", &w.writable)?;
    args = append_worker_env(args, &w.environment)?;
    args.push("--".to_owned());
    args.push(w.executable.clone());
    args.extend(w.arguments.iter().cloned());
    Ok(args)
}

enum PipeRead {
    Data(Vec<u8>),
    Pending,
    Eof,
}

struct ChildGuard(std::process::Child);

impl std::ops::Deref for ChildGuard {
    type Target = std::process::Child;
    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl std::ops::DerefMut for ChildGuard {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.0
    }
}

impl Drop for ChildGuard {
    fn drop(&mut self) {
        let _ = self.0.kill();
        let _ = self.0.wait();
    }
}

fn set_nonblocking(fd: Option<std::os::fd::RawFd>) -> Result<(), String> {
    let Some(fd) = fd else { return Ok(()) };
    let flags = unsafe { libc::fcntl(fd, libc::F_GETFL) };
    if flags < 0 || unsafe { libc::fcntl(fd, libc::F_SETFL, flags | libc::O_NONBLOCK) } < 0 {
        return Err(std::io::Error::last_os_error().to_string());
    }
    Ok(())
}

fn read_available<R: std::io::Read>(pipe: &mut Option<R>) -> Result<PipeRead, String> {
    let Some(reader) = pipe.as_mut() else {
        return Ok(PipeRead::Eof);
    };
    let mut buf = [0; 8192];
    match reader.read(&mut buf) {
        Ok(0) => {
            pipe.take();
            Ok(PipeRead::Eof)
        }
        Ok(n) => Ok(PipeRead::Data(buf[..n].to_vec())),
        Err(error) if error.kind() == std::io::ErrorKind::WouldBlock => Ok(PipeRead::Pending),
        Err(error) if error.kind() == std::io::ErrorKind::Interrupted => Ok(PipeRead::Pending),
        Err(error) => Err(error.to_string()),
    }
}

fn stop_worker_unit(unit: &str) -> Option<String> {
    let deadline = Instant::now() + Duration::from_secs(30);
    match Command::new("/usr/bin/systemctl")
        .args(["stop", unit])
        .env_clear()
        .env("PATH", "/usr/sbin:/usr/bin:/sbin:/bin")
        .env("LANG", "C.UTF-8")
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn()
    {
        Ok(mut stop) => loop {
            match stop.try_wait() {
                Ok(Some(status)) if status.success() => break None,
                Ok(Some(status)) => break Some(format!("systemctl stop {unit}: {status}")),
                Err(error) => {
                    let _ = stop.kill();
                    let _ = stop.wait();
                    break Some(format!("systemctl stop {unit}: {error}"));
                }
                Ok(None) if Instant::now() >= deadline => {
                    let _ = stop.kill();
                    let _ = stop.wait();
                    break Some(format!("systemctl stop {unit}: timed out"));
                }
                Ok(None) => std::thread::sleep(Duration::from_millis(10)),
            }
        },
        Err(error) => Some(format!("systemctl stop {unit}: {error}")),
    }
}

fn systemctl_unit_terminal(unit: &str) -> Result<bool, String> {
    let mut child = Command::new("/usr/bin/systemctl")
        .args([
            "show",
            unit,
            "--property=LoadState",
            "--property=ActiveState",
            "--value",
        ])
        .env_clear()
        .env("PATH", "/usr/sbin:/usr/bin:/sbin:/bin")
        .env("LANG", "C.UTF-8")
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::null())
        .spawn()
        .map_err(|error| format!("systemctl show {unit}: {error}"))?;
    let deadline = Instant::now() + Duration::from_secs(5);
    let status = loop {
        match child.try_wait() {
            Ok(Some(status)) => break status,
            Ok(None) if Instant::now() < deadline => std::thread::sleep(Duration::from_millis(10)),
            Ok(None) => {
                let _ = child.kill();
                let _ = child.wait();
                return Err(format!("systemctl show {unit}: timed out"));
            }
            Err(error) => {
                let _ = child.kill();
                let _ = child.wait();
                return Err(format!("systemctl show {unit}: {error}"));
            }
        }
    };
    let mut output = Vec::new();
    use std::io::Read;
    child
        .stdout
        .take()
        .ok_or_else(|| format!("systemctl show {unit}: stdout unavailable"))?
        .take(257)
        .read_to_end(&mut output)
        .map_err(|error| format!("systemctl show {unit}: {error}"))?;
    if !status.success() || output.len() > 256 {
        return Err(format!("systemctl show {unit}: invalid result"));
    }
    let text = std::str::from_utf8(&output)
        .map_err(|_| format!("systemctl show {unit}: invalid state encoding"))?;
    let mut lines = text.lines();
    let load = lines.next().unwrap_or_default();
    let active = lines.next().unwrap_or_default();
    let terminal = unit_state_is_terminal(load, active, lines.next().is_some())
        .map_err(|()| format!("systemctl show {unit}: unrecognized unit state"))?;
    Ok(terminal)
}

pub(super) fn unit_state_is_terminal(load: &str, active: &str, extra: bool) -> Result<bool, ()> {
    if extra {
        return Err(());
    }
    if load == "not-found" {
        return if active.is_empty() { Ok(true) } else { Err(()) };
    }
    if !matches!(load, "loaded" | "masked" | "error") {
        return Err(());
    }
    if !matches!(
        active,
        "active" | "reloading" | "inactive" | "failed" | "activating" | "deactivating"
    ) {
        return Err(());
    }
    Ok(matches!(active, "inactive" | "failed"))
}

fn settle_worker_unit(unit: &str) -> (WorkerUnitCustody, Option<String>) {
    match systemctl_unit_terminal(unit) {
        Ok(true) => (WorkerUnitCustody::ExactUnitTerminal, None),
        initial => {
            let initial_error = match initial {
                Err(error) => Some(error),
                _ => None,
            };
            let stop_error = stop_worker_unit(unit);
            if stop_error.is_none() {
                return (WorkerUnitCustody::ExactUnitTerminal, None);
            }
            let mut errors = Vec::new();
            errors.extend(initial_error);
            errors.extend(stop_error);
            match systemctl_unit_terminal(unit) {
                Ok(true) => (WorkerUnitCustody::ExactUnitTerminal, None),
                Ok(false) => (
                    WorkerUnitCustody::ExactUnitUnconfirmed,
                    Some(errors.join("; ")),
                ),
                Err(observation_error) => {
                    errors.push(format!(
                        "terminal-state check also failed: {observation_error}"
                    ));
                    (
                        WorkerUnitCustody::ExactUnitUnconfirmed,
                        Some(errors.join("; ")),
                    )
                }
            }
        }
    }
}

/// Typed worker result: cancellation (with exact-unit stop/reap evidence)
/// is distinct from failure so the CLI boundary maps exits without
/// inferring cancellation from string text.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum WorkerError {
    Failed(String),
    Cancelled(String),
}

/// What the controller knows about the exact transient unit owned by this
/// dispatch. A launcher process exit is not itself terminal-unit evidence.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum WorkerUnitCustody {
    NeverDispatched,
    ExactUnitTerminal,
    ExactUnitUnconfirmed,
}

impl From<String> for WorkerError {
    fn from(message: String) -> WorkerError {
        WorkerError::Failed(message)
    }
}

/// `acceptance.Worker.Run`: dispatch one admitted worker under its own
/// transient systemd unit, forwarding unit output to the given writers.
/// `cancelled` is polled while the unit runs (no async runtime in this
/// crate, so `try_wait` polling); on cancellation the exact unit is
/// stopped with a 30s timeout. Bounded pipe readers forward output as it
/// arrives and finish under a separate drain deadline.
pub(super) fn run_worker(
    w: &Worker,
    out: &mut dyn std::io::Write,
    err_out: &mut dyn std::io::Write,
    cancelled: &dyn Fn() -> bool,
    custody: &mut WorkerUnitCustody,
) -> Result<(), WorkerError> {
    *custody = WorkerUnitCustody::NeverDispatched;
    if euid() != 0 {
        return Err(WorkerError::Failed(
            "trusted root controller required for worker dispatch".to_owned(),
        ));
    }
    let args = worker_argv(w)?;
    // Refuse an existing unit instead of adopting or replacing its processes.
    let unit = format!("{}.service", w.name);
    let probed = Command::new("/usr/bin/systemctl")
        .args(["show", &unit, "--property=LoadState", "--value"])
        .output();
    match probed {
        Ok(output)
            if output.status.success()
                && String::from_utf8_lossy(&output.stdout).trim() == "not-found" => {}
        _ => {
            return Err(WorkerError::Failed(
                "worker unit is already present or could not be checked".to_owned(),
            ));
        }
    }
    // Give systemd anonymous pipes, not caller-owned log file descriptors.
    let child = Command::new("/usr/bin/systemd-run")
        .args(&args)
        .env_clear()
        .env("PATH", "/usr/sbin:/usr/bin:/sbin:/bin")
        .env("LANG", "C.UTF-8")
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|e| e.to_string())?;
    let mut child = ChildGuard(child);
    *custody = WorkerUnitCustody::ExactUnitUnconfirmed;
    let mut stdout = child.stdout.take();
    let mut stderr = child.stderr.take();
    if let Err(error) = set_nonblocking(stdout.as_ref().map(std::os::fd::AsRawFd::as_raw_fd))
        .and_then(|()| set_nonblocking(stderr.as_ref().map(std::os::fd::AsRawFd::as_raw_fd)))
    {
        let (settled, stop_error) = settle_worker_unit(&unit);
        *custody = settled;
        let child_error = child
            .kill()
            .err()
            .map(|error| format!("launcher kill: {error}"))
            .into_iter()
            .chain(
                child
                    .wait()
                    .err()
                    .map(|error| format!("launcher wait: {error}")),
            );
        let mut details = vec![error];
        details.extend(stop_error);
        details.extend(child_error);
        return Err(WorkerError::Failed(details.join("\n")));
    }
    let mut output_error = None;
    let mut status = None;
    let mut drain_deadline = None;
    let mut cancel_seen = false;
    loop {
        if cancelled() {
            cancel_seen = true;
        }
        if cancel_seen && status.is_none() {
            let _ = child.kill();
            status = child.wait().ok();
            drain_deadline = Some(Instant::now() + Duration::from_secs(2));
        }
        if status.is_none() {
            match child.try_wait() {
                Ok(found) => {
                    if found.is_some() {
                        status = found;
                        drain_deadline = Some(Instant::now() + Duration::from_secs(2));
                    }
                }
                Err(error) => {
                    output_error = Some(error.to_string());
                    break;
                }
            }
        }
        let mut progressed = false;
        match read_available(&mut stdout) {
            Ok(PipeRead::Data(bytes)) => {
                progressed = true;
                if let Err(error) = out.write_all(&bytes) {
                    output_error = Some(error.to_string());
                }
            }
            Ok(PipeRead::Eof) => progressed = true,
            Ok(PipeRead::Pending) => {}
            Err(error) => output_error = Some(error),
        }
        if output_error.is_some() {
            break;
        }
        match read_available(&mut stderr) {
            Ok(PipeRead::Data(bytes)) => {
                progressed = true;
                if let Err(error) = err_out.write_all(&bytes) {
                    output_error = Some(error.to_string());
                }
            }
            Ok(PipeRead::Eof) => progressed = true,
            Ok(PipeRead::Pending) => {}
            Err(error) => output_error = Some(error),
        }
        if output_error.is_some() {
            break;
        }
        if drain_deadline.is_some_and(|deadline| Instant::now() >= deadline)
            && (stdout.is_some() || stderr.is_some())
        {
            output_error = Some("worker output pipes remained open after launcher exit".to_owned());
            break;
        }
        if status.is_some() && stdout.is_none() && stderr.is_none() {
            break;
        }
        if !progressed {
            std::thread::sleep(Duration::from_millis(10));
        }
    }
    if output_error.is_some() || cancel_seen {
        let (settled, stop_error) = settle_worker_unit(&unit);
        *custody = settled;
        let _ = child.kill();
        let child_error = child.wait().err().map(|e| e.to_string());
        // Close only our pipe descriptors. The exact unit stop above owns
        // descendant cleanup; no process group is inferred from the launcher.
        stdout.take();
        stderr.take();
        let mut details = if cancel_seen && output_error.is_none() {
            vec!["cancelled".to_owned()]
        } else {
            Vec::new()
        };
        details.extend(output_error.clone());
        details.extend(stop_error);
        details.extend(child_error);
        if output_error.is_some() {
            return Err(WorkerError::Failed(format!(
                "worker {} failed: {}",
                w.name,
                details.join("\n")
            )));
        }
        return Err(WorkerError::Cancelled(format!(
            "worker {} failed: {}",
            w.name,
            details.join("\n")
        )));
    }
    let flush_error = out.flush().err().or_else(|| err_out.flush().err());
    if let Some(error) = flush_error {
        let (settled, stop_error) = settle_worker_unit(&unit);
        *custody = settled;
        let mut details = vec![error.to_string()];
        details.extend(stop_error);
        return Err(WorkerError::Failed(format!(
            "worker {} failed: {}",
            w.name,
            details.join("\n")
        )));
    }
    let Some(status) = status else {
        let (settled, stop_error) = settle_worker_unit(&unit);
        *custody = settled;
        let mut details = vec!["worker launcher status unavailable".to_owned()];
        details.extend(stop_error);
        return Err(WorkerError::Failed(details.join("\n")));
    };
    let (settled, stop_error) = settle_worker_unit(&unit);
    *custody = settled;
    if settled == WorkerUnitCustody::ExactUnitUnconfirmed {
        let reason = stop_error.unwrap_or_else(|| "unit termination unconfirmed".to_owned());
        return Err(WorkerError::Failed(format!(
            "worker {} failed: exact unit termination unconfirmed: {reason}",
            w.name
        )));
    }
    if status.success() {
        Ok(())
    } else {
        Err(WorkerError::Failed(format!(
            "worker {} failed: {status}",
            w.name
        )))
    }
}
