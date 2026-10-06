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

fn drain_pipe<R: std::io::Read + Send + 'static>(
    pipe: Option<R>,
) -> Option<std::thread::JoinHandle<Vec<u8>>> {
    pipe.map(|mut pipe| {
        std::thread::spawn(move || {
            let mut buf = Vec::new();
            let _ = pipe.read_to_end(&mut buf);
            buf
        })
    })
}

fn join_drains(
    out: &mut dyn std::io::Write,
    err_out: &mut dyn std::io::Write,
    out_drain: Option<std::thread::JoinHandle<Vec<u8>>>,
    err_drain: Option<std::thread::JoinHandle<Vec<u8>>>,
) -> Result<(), String> {
    let stdout = out_drain
        .map(|t| t.join().unwrap_or_default())
        .unwrap_or_default();
    let stderr = err_drain
        .map(|t| t.join().unwrap_or_default())
        .unwrap_or_default();
    out.write_all(&stdout).map_err(|e| e.to_string())?;
    err_out.write_all(&stderr).map_err(|e| e.to_string())?;
    let _ = out.flush();
    let _ = err_out.flush();
    Ok(())
}

/// Typed worker result: cancellation (with exact-unit stop/reap evidence)
/// is distinct from failure so the CLI boundary maps exits without
/// inferring cancellation from string text.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum WorkerError {
    Failed(String),
    Cancelled(String),
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
/// stopped with a 30s timeout and the joined failure is reported.
/// Output is forwarded after the unit exits rather than streamed.
pub fn run_worker(
    w: &Worker,
    out: &mut dyn std::io::Write,
    err_out: &mut dyn std::io::Write,
    cancelled: &dyn Fn() -> bool,
) -> Result<(), WorkerError> {
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
    let mut child = Command::new("/usr/bin/systemd-run")
        .args(&args)
        .env_clear()
        .env("PATH", "/usr/sbin:/usr/bin:/sbin:/bin")
        .env("LANG", "C.UTF-8")
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|e| e.to_string())?;
    let out_drain = drain_pipe(child.stdout.take());
    let err_drain = drain_pipe(child.stderr.take());
    loop {
        if cancelled() {
            break;
        }
        match child.try_wait().map_err(|e| e.to_string())? {
            Some(status) => {
                join_drains(out, err_out, out_drain, err_drain)?;
                if status.success() {
                    return Ok(());
                }
                return Err(WorkerError::Failed(format!(
                    "worker {} failed: {status}",
                    w.name
                )));
            }
            None => std::thread::sleep(Duration::from_millis(10)),
        }
    }
    // Stopping only systemd-run does not stop its service. Own the exact unit
    // even when the controller's request has already been cancelled.
    let deadline = Instant::now() + Duration::from_secs(30);
    let stop_err: Option<String> = match Command::new("/usr/bin/systemctl")
        .args(["stop", &unit])
        .env_clear()
        .env("PATH", "/usr/sbin:/usr/bin:/sbin:/bin")
        .env("LANG", "C.UTF-8")
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn()
    {
        Ok(mut stop) => loop {
            match stop.try_wait().map_err(|e| e.to_string())? {
                Some(status) if status.success() => break None,
                Some(status) => break Some(format!("systemctl stop {unit}: {status}")),
                None if Instant::now() >= deadline => {
                    let _ = stop.kill();
                    let _ = stop.wait();
                    break Some(format!("systemctl stop {unit}: timed out"));
                }
                None => std::thread::sleep(Duration::from_millis(10)),
            }
        },
        Err(e) => Some(e.to_string()),
    };
    let _ = child.kill();
    let child_err = child.wait().err().map(|e| e.to_string());
    join_drains(out, err_out, out_drain, err_drain).ok();
    let mut parts = vec!["cancelled".to_owned()];
    parts.extend(stop_err);
    parts.extend(child_err);
    Err(WorkerError::Cancelled(format!(
        "worker {} failed: {}",
        w.name,
        parts.join("\n")
    )))
}
