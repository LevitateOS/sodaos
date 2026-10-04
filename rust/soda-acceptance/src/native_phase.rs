//! One explicitly requested native phase, mirroring
//! `internal/acceptance/remote_executor.py`.
//!
//! The payload runs on the target, delivered as compiled bytes over SSH
//! (never installed): it validates an exact-source request, admits a fresh
//! or retained run directory, verifies the unchanged clean checkout around
//! one phase, and retains failed work. Failure taxonomy mirrors the Python
//! owner: `ValueError` for validation, `OSError` shapes for filesystem
//! failures, `SubprocessError` for command failures.

use std::collections::HashMap;
use std::io::ErrorKind;
use std::os::unix::ffi::OsStrExt;

use soda_json::JsonValue;

use crate::jsonio;

/// Maximum request size, like the Python stdin bound.
pub const REQUEST_LIMIT: usize = 16384;
/// Pinned source repository, like the Python owner.
pub const SOURCE_URL: &str = "https://github.com/LevitateOS/sodaos.git";

/// Payload failure with the Python owner's exception taxonomy.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum PayloadFailure {
    /// Validation failure: `ValueError`.
    Value(String),
    /// Filesystem failure: `OSError` shapes by kind.
    Os {
        /// I/O kind, selecting the exception shape.
        kind: ErrorKind,
        /// Detail message.
        message: String,
    },
    /// Command failure: `SubprocessError`.
    Subprocess(String),
}

impl PayloadFailure {
    /// Exception name, like Python's `type(exc).__name__`.
    pub fn kind_name(&self) -> &'static str {
        match self {
            PayloadFailure::Value(_) => "ValueError",
            PayloadFailure::Subprocess(_) => "SubprocessError",
            PayloadFailure::Os { kind, .. } => match kind {
                ErrorKind::AlreadyExists => "FileExistsError",
                ErrorKind::NotFound => "FileNotFoundError",
                ErrorKind::PermissionDenied => "PermissionError",
                _ => "OSError",
            },
        }
    }

    /// Detail message.
    pub fn message(&self) -> &str {
        match self {
            PayloadFailure::Value(message) => message,
            PayloadFailure::Os { message, .. } => message,
            PayloadFailure::Subprocess(message) => message,
        }
    }

    fn os(error: std::io::Error) -> PayloadFailure {
        PayloadFailure::Os {
            kind: error.kind(),
            message: error.to_string(),
        }
    }
}

impl std::fmt::Display for PayloadFailure {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}: {}", self.kind_name(), self.message())
    }
}

impl std::error::Error for PayloadFailure {}

/// Command runner, abstracted for tests. The system runner applies the
/// payload's Git environment (`GIT_TERMINAL_PROMPT=0`, `GOTOOLCHAIN=local`).
pub trait Runner {
    /// Run to successful exit.
    fn status(&self, argv: &[String], cwd: Option<&str>) -> Result<(), PayloadFailure>;
    /// Capture stdout, requiring successful exit.
    fn output(&self, argv: &[String], cwd: Option<&str>) -> Result<Vec<u8>, PayloadFailure>;
}

/// Real command runner.
pub struct SystemRunner;

impl Runner for SystemRunner {
    fn status(&self, argv: &[String], cwd: Option<&str>) -> Result<(), PayloadFailure> {
        let mut command = std::process::Command::new(&argv[0]);
        command.args(&argv[1..]);
        if let Some(dir) = cwd {
            command.current_dir(dir);
        }
        command.env("GIT_TERMINAL_PROMPT", "0").env("GOTOOLCHAIN", "local");
        match command.status() {
            Ok(status) if status.success() => Ok(()),
            Ok(_) => Err(PayloadFailure::Subprocess("command failed".to_string())),
            Err(e) => Err(PayloadFailure::os(e)),
        }
    }

    fn output(&self, argv: &[String], cwd: Option<&str>) -> Result<Vec<u8>, PayloadFailure> {
        let mut command = std::process::Command::new(&argv[0]);
        command.args(&argv[1..]);
        if let Some(dir) = cwd {
            command.current_dir(dir);
        }
        command.env("GIT_TERMINAL_PROMPT", "0").env("GOTOOLCHAIN", "local");
        match command.output() {
            Ok(output) if output.status.success() => Ok(output.stdout),
            Ok(_) => Err(PayloadFailure::Subprocess("command failed".to_string())),
            Err(e) => Err(PayloadFailure::os(e)),
        }
    }
}

/// Observed platform identity, abstracted for tests.
pub struct Platform {
    /// Operating system name.
    pub os: String,
    /// Machine architecture.
    pub arch: String,
    /// Actual hostname.
    pub hostname: String,
}

/// Read the native platform identity.
pub fn native_platform() -> Result<Platform, PayloadFailure> {
    let mut name = [0 as libc::c_char; 256];
    let rc = unsafe { libc::gethostname(name.as_mut_ptr(), name.len()) };
    if rc != 0 {
        return Err(PayloadFailure::os(std::io::Error::last_os_error()));
    }
    let hostname = unsafe { std::ffi::CStr::from_ptr(name.as_ptr()) }
        .to_string_lossy()
        .into_owned();
    Ok(Platform {
        os: std::env::consts::OS.to_string(),
        arch: std::env::consts::ARCH.to_string(),
        hostname,
    })
}

/// Restrictive creation mask, like the Python owner's `os.umask(0o077)`.
pub fn restrict_umask() {
    unsafe {
        libc::umask(0o077);
    }
}

/// Validated native request.
pub struct PhaseRequest {
    /// Full source revision.
    pub revision: String,
    /// Target architecture.
    pub architecture: String,
    /// Actual target hostname.
    pub target: String,
    /// Fresh work path.
    pub work: String,
    /// Requested phase.
    pub phase: String,
}

fn value_error(message: impl Into<String>) -> PayloadFailure {
    PayloadFailure::Value(message.into())
}

/// Validate an exact-source request, like `phase_request`.
pub fn phase_request(raw: &str, platform: &Platform) -> Result<(PhaseRequest, HashMap<String, String>), PayloadFailure> {
    if raw.len() > REQUEST_LIMIT {
        return Err(value_error("request too large"));
    }
    let parsed = JsonValue::parse(raw).map_err(|_| value_error("unexpected request fields"))?;
    let JsonValue::Object(entries) = &parsed else {
        return Err(value_error("unexpected request fields"));
    };
    let mut fields = HashMap::new();
    for (key, item) in entries {
        let text = item.as_str().ok_or_else(|| value_error("request values must be strings"))?;
        fields.insert(key.clone(), text.to_string());
    }
    let keys: std::collections::HashSet<&str> = fields.keys().map(|k| k.as_str()).collect();
    if keys != ["Revision", "Architecture", "Target", "Work", "Phase"].into_iter().collect() {
        return Err(value_error("unexpected request fields"));
    }
    let get = |name: &str| fields.get(name).cloned().unwrap_or_default();
    let request = PhaseRequest {
        revision: get("Revision"),
        architecture: get("Architecture"),
        target: get("Target"),
        work: get("Work"),
        phase: get("Phase"),
    };
    if !soda_build_tools::reader::is_revision(&request.revision) {
        return Err(value_error("full revision required"));
    }
    if request.architecture != "x86_64" || platform.os != "Linux" || platform.arch != request.architecture {
        return Err(value_error("matching-native Linux required"));
    }
    if platform.hostname != request.target {
        return Err(value_error("actual native hostname does not match target"));
    }
    if !["prepare", "build", "check"].contains(&request.phase.as_str()) {
        return Err(value_error("unknown phase; no install/VM/provider/release phases"));
    }
    if !request.work.starts_with('/') {
        return Err(value_error("absolute fresh work path with existing real parent required"));
    }
    let parent = std::path::Path::new(&request.work).parent().unwrap_or(std::path::Path::new("/"));
    let resolved = std::fs::canonicalize(parent).map_err(|_| value_error("absolute fresh work path with existing real parent required"))?;
    if resolved.to_string_lossy() != crate::files::lexical_clean(&parent.to_string_lossy()) || !parent.is_dir() {
        return Err(value_error("absolute fresh work path with existing real parent required"));
    }
    let mut receipt = fields;
    receipt.remove("Phase");
    Ok((request, receipt))
}

fn git_head(runner: &dyn Runner, checkout: &str) -> Result<String, PayloadFailure> {
    let out = runner.output(&["git".to_string(), "rev-parse".to_string(), "HEAD".to_string()], Some(checkout))?;
    Ok(String::from_utf8_lossy(&out).trim().to_string())
}

fn git_dirty(runner: &dyn Runner, checkout: &str) -> Result<Vec<u8>, PayloadFailure> {
    runner.output(
        &[
            "git".to_string(),
            "status".to_string(),
            "--porcelain".to_string(),
            "--untracked-files=normal".to_string(),
        ],
        Some(checkout),
    )
}

fn require_checkout(runner: &dyn Runner, checkout: &str, revision: &str) -> Result<String, PayloadFailure> {
    let meta = std::fs::symlink_metadata(checkout).map_err(PayloadFailure::os)?;
    if meta.is_symlink() || !meta.is_dir() {
        return Err(value_error("real checkout required"));
    }
    let head = git_head(runner, checkout)?;
    if head != revision || !git_dirty(runner, checkout)?.is_empty() {
        return Err(value_error("checkout revision/content changed; use a fresh run"));
    }
    Ok(head)
}

fn write_receipt(path: &std::path::Path, receipt: &HashMap<String, String>) -> Result<(), PayloadFailure> {
    let mut entries: Vec<(String, JsonValue)> = receipt.iter().map(|(k, v)| (k.clone(), JsonValue::Str(v.clone()))).collect();
    entries.sort_by(|a, b| a.0.cmp(&b.0));
    let mut text = String::new();
    jsonio::write_compact(&mut text, &JsonValue::Object(entries));
    crate::files::write_new(&path.to_string_lossy(), text.as_bytes(), 0o666).map_err(|e| match e {
        crate::error::Error::Io(io) => PayloadFailure::os(io),
        other => value_error(other.to_string()),
    })
}

fn prepare_checkout(
    runner: &dyn Runner,
    request: &PhaseRequest,
    work: &std::path::Path,
    checkout: &str,
    receipt: &HashMap<String, String>,
) -> Result<(), PayloadFailure> {
    let raw = std::ffi::CString::new(work.as_os_str().as_bytes()).map_err(|_| value_error("absolute fresh work path with existing real parent required"))?;
    let rc = unsafe { libc::mkdir(raw.as_ptr(), 0o700) };
    if rc != 0 {
        return Err(PayloadFailure::os(std::io::Error::last_os_error()));
    }
    write_receipt(&work.join("request.json"), receipt)?;
    runner.status(
        &[
            "git".to_string(),
            "-c".to_string(),
            "core.hooksPath=/dev/null".to_string(),
            "clone".to_string(),
            "--no-checkout".to_string(),
            "--".to_string(),
            SOURCE_URL.to_string(),
            checkout.to_string(),
        ],
        None,
    )?;
    runner.status(
        &[
            "git".to_string(),
            "-c".to_string(),
            "core.hooksPath=/dev/null".to_string(),
            "checkout".to_string(),
            "--detach".to_string(),
            request.revision.clone(),
        ],
        Some(checkout),
    )
}

fn admit_existing_run(work: &std::path::Path, receipt: &HashMap<String, String>) -> Result<(), PayloadFailure> {
    let meta = std::fs::symlink_metadata(work).map_err(PayloadFailure::os)?;
    use std::os::unix::fs::MetadataExt;
    if !meta.is_dir() || meta.uid() != unsafe { libc::getuid() } || meta.mode() & 0o077 != 0 {
        return Err(value_error("private owned run directory required"));
    }
    let raw = std::fs::read_to_string(work.join("request.json")).map_err(PayloadFailure::os)?;
    let parsed = JsonValue::parse(&raw).map_err(|_| value_error("phase does not belong to this source/target/run"))?;
    let JsonValue::Object(entries) = &parsed else {
        return Err(value_error("phase does not belong to this source/target/run"));
    };
    let mut stored = HashMap::new();
    for (key, item) in entries {
        stored.insert(key.clone(), item.as_str().unwrap_or_default().to_string());
    }
    if stored != *receipt {
        return Err(value_error("phase does not belong to this source/target/run"));
    }
    if !work.join("prepare.completed").is_file() {
        return Err(value_error("prepare did not complete"));
    }
    Ok(())
}

fn phase_command(request: &PhaseRequest, work: &std::path::Path) -> Result<Option<Vec<String>>, PayloadFailure> {
    let candidate = work.join("candidate");
    match request.phase.as_str() {
        "build" => {
            if !candidate.is_dir() || !candidate.join("payload.json").is_file() || !candidate.join("candidate.json").is_file() {
                return Err(value_error(
                    "build admits an existing soda-build candidate at work/candidate; legacy build-native is retired",
                ));
            }
            Ok(None)
        }
        "check" => {
            if !work.join("build.completed").is_file() {
                return Err(value_error("required earlier phase did not complete"));
            }
            if !candidate.is_dir() {
                return Err(value_error("check requires work/candidate pointing at soda-build artifacts"));
            }
            Ok(Some(vec![
                "bash".to_string(),
                "scripts/check-native.sh".to_string(),
                request.architecture.clone(),
                candidate.to_string_lossy().into_owned(),
            ]))
        }
        _ => Ok(None),
    }
}

/// Run one native phase, like the Python `main`.
pub fn run_phase(raw_request: &str, platform: &Platform, runner: &dyn Runner) -> Result<(), PayloadFailure> {
    let (request, receipt) = phase_request(raw_request, platform)?;
    let work = std::path::PathBuf::from(&request.work);
    let checkout = work.join("source").to_string_lossy().into_owned();
    if request.phase == "prepare" {
        prepare_checkout(runner, &request, &work, &checkout, &receipt)?;
    } else {
        admit_existing_run(&work, &receipt)?;
    }
    let head = require_checkout(runner, &checkout, &request.revision)?;
    write_receipt(&work.join(format!("{}.started", request.phase)), &receipt)?;
    if let Some(command) = phase_command(&request, &work)? {
        runner.status(&command, Some(&checkout))?;
    }
    if git_head(runner, &checkout)? != head || !git_dirty(runner, &checkout)?.is_empty() {
        return Err(value_error("source changed during phase"));
    }
    write_receipt(&work.join(format!("{}.completed", request.phase)), &receipt)?;
    println!("Native phase completed; no later phase was requested or implied.");
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::cell::RefCell;

    const REVISION: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";

    struct FakeRunner {
        calls: RefCell<Vec<Vec<String>>>,
        revision: String,
        dirty: RefCell<Vec<u8>>,
    }

    impl Runner for FakeRunner {
        fn status(&self, argv: &[String], _cwd: Option<&str>) -> Result<(), PayloadFailure> {
            self.calls.borrow_mut().push(argv.to_vec());
            if argv.iter().any(|a| a == "clone") {
                std::fs::create_dir_all(argv.last().unwrap()).unwrap();
            }
            Ok(())
        }

        fn output(&self, argv: &[String], _cwd: Option<&str>) -> Result<Vec<u8>, PayloadFailure> {
            if argv.get(1).is_some_and(|a| a == "rev-parse") {
                Ok(format!("{}\n", self.revision).into_bytes())
            } else {
                Ok(self.dirty.borrow().clone())
            }
        }
    }

    struct Harness {
        dir: std::path::PathBuf,
        work: std::path::PathBuf,
        platform: Platform,
        runner: FakeRunner,
    }

    impl Harness {
        fn new() -> Harness {
            let mut dir = std::env::temp_dir();
            dir.push(format!("soda-native-{}-{}", std::process::id(), fresh_id()));
            std::fs::create_dir_all(&dir).unwrap();
            // Realistic restrictive–but-passing parent for workdir admit checks.
            let work = dir.join("run");
            Harness {
                dir,
                work,
                platform: Platform {
                    os: "Linux".to_string(),
                    arch: "x86_64".to_string(),
                    hostname: "synthetic-builder".to_string(),
                },
                runner: FakeRunner {
                    calls: RefCell::new(Vec::new()),
                    revision: REVISION.to_string(),
                    dirty: RefCell::new(Vec::new()),
                },
            }
        }

        fn request(&self, phase: &str, changes: &[(&str, &str)]) -> String {
            let mut fields = HashMap::from([
                ("Revision".to_string(), REVISION.to_string()),
                ("Architecture".to_string(), "x86_64".to_string()),
                ("Target".to_string(), "synthetic-builder".to_string()),
                ("Work".to_string(), self.work.to_string_lossy().into_owned()),
                ("Phase".to_string(), phase.to_string()),
            ]);
            for (key, value) in changes {
                fields.insert(key.to_string(), value.to_string());
            }
            let mut text = String::new();
            let entries: Vec<(String, JsonValue)> =
                fields.into_iter().map(|(k, v)| (k, JsonValue::Str(v))).collect();
            jsonio::write_compact(&mut text, &JsonValue::Object(entries));
            text
        }

        fn invoke(&self, phase: &str, changes: &[(&str, &str)]) -> Result<(), PayloadFailure> {
            run_phase(&self.request(phase, changes), &self.platform, &self.runner)
        }

        fn calls(&self) -> usize {
            self.runner.calls.borrow().len()
        }
    }

    impl Drop for Harness {
        fn drop(&mut self) {
            let _ = std::fs::remove_dir_all(&self.dir);
        }
    }

    fn fresh_id() -> u64 {
        use std::sync::atomic::{AtomicU64, Ordering};
        static NEXT: AtomicU64 = AtomicU64::new(0);
        NEXT.fetch_add(1, Ordering::SeqCst)
    }

    fn seed_candidate(work: &std::path::Path) {
        let candidate = work.join("candidate");
        std::fs::create_dir_all(&candidate).unwrap();
        std::fs::write(candidate.join("payload.json"), b"{}").unwrap();
        std::fs::write(candidate.join("candidate.json"), b"{}").unwrap();
    }

    #[test]
    fn explicit_phase_order_without_automatic_work() {
        let harness = Harness::new();
        harness.invoke("prepare", &[]).unwrap();
        assert_eq!(harness.calls(), 2);
        assert!(!harness.work.join("build.started").exists());
        seed_candidate(&harness.work);
        let before = harness.calls();
        harness.invoke("build", &[]).unwrap();
        assert_eq!(harness.calls(), before, "build admits candidate bytes; it does not run a producer");
        assert!(!harness.work.join("check.started").exists());
        harness.invoke("check", &[]).unwrap();
        let last = harness.runner.calls.borrow().last().cloned().unwrap();
        assert_eq!(last[..3], ["bash".to_string(), "scripts/check-native.sh".to_string(), "x86_64".to_string()]);
        assert!(last[3].ends_with("/candidate"));
        for call in harness.runner.calls.borrow().iter() {
            for part in call {
                assert!(!part.contains("install-native.sh"));
                assert!(!part.contains("build-native.sh"));
            }
        }
    }

    #[test]
    fn unknown_phase_target_arch_and_revision_fail_before_creation() {
        for (phase, changes) in [
            ("install", vec![]),
            ("bundle", vec![]),
            ("prepare", vec![("Target", "other")]),
            ("prepare", vec![("Architecture", "aarch64")]),
            ("prepare", vec![("Revision", "main")]),
        ] {
            let harness = Harness::new();
            let err = harness.invoke(phase, &changes).unwrap_err();
            assert_eq!(err.kind_name(), "ValueError", "{phase} {changes:?}");
            assert!(!harness.work.exists());
            assert_eq!(harness.calls(), 0);
        }
    }

    #[test]
    fn occupied_prepare_and_phase_replay_do_not_run_commands() {
        let harness = Harness::new();
        harness.invoke("prepare", &[]).unwrap();
        let before = harness.calls();
        let err = harness.invoke("prepare", &[]).unwrap_err();
        assert_eq!(err.kind_name(), "FileExistsError");
        assert_eq!(harness.calls(), before);
        seed_candidate(&harness.work);
        harness.invoke("build", &[]).unwrap();
        let before = harness.calls();
        let err = harness.invoke("build", &[]).unwrap_err();
        assert_eq!(err.kind_name(), "FileExistsError");
        assert_eq!(harness.calls(), before);
    }

    #[test]
    fn missing_prerequisite_stale_binding_and_dirty_source() {
        let harness = Harness::new();
        harness.invoke("prepare", &[]).unwrap();
        let before = harness.calls();
        assert!(harness.invoke("check", &[]).is_err());
        assert!(harness.invoke("build", &[("Revision", &"b".repeat(40))]).is_err());
        *harness.runner.dirty.borrow_mut() = b" M source.go\n".to_vec();
        assert!(harness.invoke("build", &[]).is_err());
        assert_eq!(harness.calls(), before);
        assert!(!harness.work.join("build.started").exists());
    }

    #[test]
    fn build_without_candidate_retains_started_without_completion() {
        let harness = Harness::new();
        harness.invoke("prepare", &[]).unwrap();
        assert!(harness.invoke("build", &[]).is_err());
        assert!(harness.work.join("build.started").is_file());
        assert!(!harness.work.join("build.completed").exists());
    }

    #[test]
    fn remote_dispatch_has_no_runtime_or_publication_phases() {
        let harness = Harness::new();
        let err = harness.invoke("publish", &[]).unwrap_err();
        assert_eq!(err.kind_name(), "ValueError");
        assert_eq!(harness.calls(), 0);
    }
}
