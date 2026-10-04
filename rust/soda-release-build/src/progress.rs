//! Native monotonic progress (`progress.go`): timing owner shared with
//! legacy callers, plus local command IO. No command arguments enter timing
//! records.

use crate::files::write_new;
use crate::{clock, io_error, look_path, Error};
use std::io::Write;
use std::path::Path;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex};
use std::time::Duration;

/// Monotonic build progress: phases, sections, and a retained timing log.
pub struct BuildProgress {
    title: String,
    label: String,
    path: String,
    origin: Duration,
    started: Duration,
    now: Box<dyn Fn() -> Duration + Send + Sync>,
    stderr: Box<dyn Write + Send>,
    finished: bool,
    phase: String,
    phase_started: Duration,
    reason: String,
}

impl BuildProgress {
    pub fn new_for_test(
        title: &str,
        now: Box<dyn Fn() -> Duration + Send + Sync>,
        stderr: Box<dyn Write + Send>,
    ) -> BuildProgress {
        BuildProgress {
            title: title.to_string(),
            label: String::new(),
            path: String::new(),
            origin: Duration::ZERO,
            started: Duration::ZERO,
            now,
            stderr,
            finished: false,
            phase: String::new(),
            phase_started: Duration::ZERO,
            reason: String::new(),
        }
    }

    /// Records the one-line cause attached to the next FAILED phase or
    /// section. Carries tool stderr only, never argv or environment. Empty
    /// reasons are ignored; Phase and Next clear it.
    pub fn note_reason(&mut self, s: &str) {
        let cut = match s.find(['\r', '\n']) {
            Some(i) => &s[..i],
            None => s,
        };
        let cleaned = cut.trim().replace('|', "/");
        let truncated: String = cleaned.chars().take(300).collect();
        if truncated.is_empty() {
            return;
        }
        self.reason = truncated;
    }

    fn emit(&mut self, kind: &str, text: &str) -> Result<(), Error> {
        let line = format!("{kind:<8} {text}\n");
        let mut first_err: Option<Error> = None;
        if let Err(e) = self.stderr.write_all(line.as_bytes()) {
            first_err = Some(Error::msg(e.to_string()));
        }
        if !self.path.is_empty() {
            let open = std::fs::OpenOptions::new()
                .append(true)
                .custom_flags(libc::O_NOFOLLOW)
                .open(&self.path);
            match open {
                Err(e) => {
                    let err = io_error("open", Path::new(&self.path), e);
                    return Err(join_errors(first_err, err));
                }
                Ok(mut f) => {
                    if let Err(e) = f.write_all(line.as_bytes()) {
                        let err = io_error("write", Path::new(&self.path), e);
                        let close = f
                            .sync_all()
                            .err()
                            .map(|e| io_error("write", Path::new(&self.path), e));
                        return Err(join3(first_err, Some(err), close));
                    }
                    if let Err(e) = f.sync_all() {
                        return Err(join_errors(
                            first_err,
                            io_error("write", Path::new(&self.path), e),
                        ));
                    }
                }
            }
        }
        match first_err {
            Some(e) => Err(e),
            None => Ok(()),
        }
    }

    pub fn create_log(&mut self, path: &str) -> Result<(), Error> {
        let inherited = std::env::var("SODA_BUILD_TIMING_LOG").unwrap_or_default();
        if !inherited.is_empty() && std::env::var("SODA_BUILD_CHILD").unwrap_or_default() == "1" {
            self.path = inherited;
            return Ok(());
        }
        write_new(Path::new(path), &[], 0o600)?;
        self.path = path.to_string();
        let path_owned = self.path.clone();
        self.emit("LOG", &path_owned)
    }

    pub fn phase(&mut self, label: &str) -> Result<(), Error> {
        if self.finished || label.contains(['\r', '\n']) {
            return Err(Error::msg("invalid progress transition"));
        }
        join_results(self.end(None), self.end_phase(None))?;
        self.phase = label.to_string();
        self.phase_started = (self.now)();
        self.reason.clear();
        let label_owned = self.phase.clone();
        self.emit("START", &label_owned)
    }

    pub fn end_phase(&mut self, err: Option<&Error>) -> Result<(), Error> {
        if self.phase.is_empty() {
            return Ok(());
        }
        let mut kind = "DONE";
        if err.is_some() {
            kind = "FAILED";
        }
        if matches!(build_exit_code(err), 130 | 143) {
            kind = "CANCELLED";
        }
        let label = std::mem::take(&mut self.phase);
        let text = format!(
            "{label} | phase {} | total {}",
            duration((self.now)().saturating_sub(self.phase_started)),
            duration((self.now)().saturating_sub(self.origin))
        );
        let text = failed_text(&text, if kind == "FAILED" { &self.reason } else { "" });
        self.emit(kind, &text)
    }

    pub fn next(&mut self, label: &str) -> Result<(), Error> {
        if self.finished || label.contains(['\r', '\n']) {
            return Err(Error::msg("invalid progress transition"));
        }
        self.end(None)?;
        self.label = label.to_string();
        self.started = (self.now)();
        self.reason.clear();
        let label_owned = self.label.clone();
        self.emit("START", &label_owned)
    }

    pub fn end(&mut self, err: Option<&Error>) -> Result<(), Error> {
        if self.label.is_empty() {
            return Ok(());
        }
        let mut kind = "DONE";
        if err.is_some() {
            kind = "FAILED";
        }
        if matches!(build_exit_code(err), 130 | 143) {
            kind = "CANCELLED";
        }
        let label = std::mem::take(&mut self.label);
        let now = (self.now)();
        let text = format!(
            "{label} | section {} | total {}",
            duration(now.saturating_sub(self.started)),
            duration(now.saturating_sub(self.origin))
        );
        let text = failed_text(&text, if kind == "FAILED" { &self.reason } else { "" });
        self.emit(kind, &text)
    }

    fn write_section_summary(&mut self) -> Result<(), Error> {
        if self.path.is_empty() {
            return Ok(());
        }
        let data = std::fs::read(&self.path);
        let mut end: Option<Error> = None;
        if let Err(e) = writeln!(self.stderr, "\nSECTION SUMMARY") {
            end = Some(join_errors(end, Error::msg(e.to_string())));
        }
        let data = match data {
            Ok(data) => data,
            Err(e) => return Err(join_errors(end, io_error("open", Path::new(&self.path), e))),
        };
        // Go ranges over ReadFile errors first, then prints; a read failure
        // still prints the header before returning.
        for line in String::from_utf8_lossy(&data).split('\n') {
            if line.starts_with("DONE ")
                || line.starts_with("FAILED ")
                || line.starts_with("CANCELLED ")
            {
                if let Err(e) = writeln!(self.stderr, "  {line}") {
                    end = Some(join_errors(end, Error::msg(e.to_string())));
                }
            }
        }
        match end {
            Some(e) => Err(e),
            None => Ok(()),
        }
    }

    pub fn finish(&mut self, err: Option<&Error>) -> Result<(), Error> {
        if self.finished {
            return Ok(());
        }
        self.finished = true;
        let mut end = join_results(self.end(err), self.end_phase(err));
        if std::env::var("SODA_BUILD_CHILD").unwrap_or_default() == "1" {
            return end;
        }
        end = join_results(end, self.write_section_summary());
        let joined = join_option(err.cloned(), end.clone());
        let code = build_exit_code(joined.as_ref());
        let title = self.title.clone();
        let origin = self.origin;
        let total = duration((self.now)().saturating_sub(origin));
        let emit_result = self.emit(
            &finish_kind(code),
            &format!("{title} | total {total} | exit {code}"),
        );
        join_results(end, emit_result)
    }
}

fn join_errors(first: Option<Error>, second: Error) -> Error {
    match first {
        None => second,
        Some(first) => Error::msg(format!("{}\n{}", first.message(), second.message())),
    }
}

fn join3(first: Option<Error>, second: Option<Error>, third: Option<Error>) -> Error {
    let mut parts = Vec::new();
    for err in [first, second, third].into_iter().flatten() {
        parts.push(err.message().to_string());
    }
    Error::msg(parts.join("\n"))
}

fn join_option(err: Option<Error>, end: Result<(), Error>) -> Option<Error> {
    match (err, end) {
        (None, Ok(())) => None,
        (Some(e), Ok(())) => Some(e),
        (None, Err(e)) => Some(e),
        (Some(a), Err(b)) => Some(Error::msg(format!("{}\n{}", a.message(), b.message()))),
    }
}

fn join_results(first: Result<(), Error>, second: Result<(), Error>) -> Result<(), Error> {
    match (first, second) {
        (Ok(()), Ok(())) => Ok(()),
        (Err(e), Ok(())) | (Ok(()), Err(e)) => Err(e),
        (Err(a), Err(b)) => Err(Error::msg(format!("{}\n{}", a.message(), b.message()))),
    }
}

fn failed_text(label: &str, reason: &str) -> String {
    if reason.is_empty() {
        label.to_string()
    } else {
        format!("{label} | reason {reason}")
    }
}

fn duration(d: Duration) -> String {
    let s = d.as_secs() as i64;
    let s = s.max(0);
    format!("{:02}:{:02}:{:02}", s / 3600, s / 60 % 60, s % 60)
}

fn finish_kind(code: i32) -> String {
    if code == 0 {
        "SUCCESS".to_string()
    } else if code == 130 || code == 143 {
        "CANCELLED".to_string()
    } else {
        "FAILED".to_string()
    }
}

/// Creates progress sharing the inherited (or fresh) timing origin.
/// `source` is unused, matching the Go owner's signature.
pub fn new_build_progress(_source: &str, title: &str) -> Result<BuildProgress, Error> {
    let stderr: Box<dyn Write + Send> = Box::new(std::io::stderr());
    let mut progress = BuildProgress {
        title: title.to_string(),
        label: String::new(),
        path: String::new(),
        origin: clock::monotonic(),
        started: Duration::ZERO,
        now: Box::new(clock::monotonic),
        stderr,
        finished: false,
        phase: String::new(),
        phase_started: Duration::ZERO,
        reason: String::new(),
    };
    let raw = std::env::var("SODA_BUILD_START_NS").unwrap_or_default();
    if !raw.is_empty() {
        let parsed: Result<i64, _> = raw.parse();
        match parsed {
            Ok(n) if n >= 0 && Duration::from_nanos(n as u64) <= progress.origin => {
                progress.origin = Duration::from_nanos(n as u64);
            }
            _ => return Err(Error::msg("invalid inherited timing origin")),
        }
    } else {
        std::env::set_var(
            "SODA_BUILD_START_NS",
            progress.origin.as_nanos().to_string(),
        );
    }
    Ok(progress)
}

/// Exit code for a finished error: command codes, 128+signal, 130 for
/// cancellation, else 1.
pub fn build_exit_code(err: Option<&Error>) -> i32 {
    match err {
        None => 0,
        Some(err) => {
            if let Some(code) = err.exit_code() {
                if code > 0 {
                    return code;
                }
            }
            if let Some(signal) = err.signal() {
                return 128 + signal;
            }
            if err.is_cancelled() {
                return 130;
            }
            1
        }
    }
}

/// Shared in-memory writer for tests and log fan-out.
#[derive(Debug, Clone, Default)]
pub struct SharedBuffer {
    inner: Arc<Mutex<Vec<u8>>>,
}

impl SharedBuffer {
    pub fn new() -> SharedBuffer {
        SharedBuffer::default()
    }

    pub fn contents(&self) -> Vec<u8> {
        self.inner.lock().unwrap().clone()
    }

    pub fn text(&self) -> String {
        String::from_utf8_lossy(&self.contents()).into_owned()
    }
}

impl Write for SharedBuffer {
    fn write(&mut self, buf: &[u8]) -> std::io::Result<usize> {
        self.inner.lock().unwrap().extend_from_slice(buf);
        Ok(buf.len())
    }

    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}

/// Local command IO/environment only; not a job runner. `cancelled`
/// replaces Go's `context.Context`: when set, commands fail fast with a
/// 130-class error.
pub struct BuildExecution {
    pub cancelled: Arc<AtomicBool>,
    pub log: Box<dyn Write + Send>,
    pub output: Option<Box<dyn Write + Send>>,
}

impl Default for BuildExecution {
    fn default() -> BuildExecution {
        BuildExecution {
            cancelled: Arc::new(AtomicBool::new(false)),
            log: Box::new(std::io::sink()),
            output: None,
        }
    }
}

fn go_toolchain_pin() -> Option<String> {
    use std::sync::OnceLock;
    static PIN: OnceLock<Option<String>> = OnceLock::new();
    PIN.get_or_init(|| {
        std::process::Command::new("go")
            .arg("version")
            .output()
            .ok()
            .and_then(|out| {
                if !out.status.success() {
                    return None;
                }
                let text = String::from_utf8_lossy(&out.stdout);
                let mut parts = text.split_whitespace();
                parts.next();
                parts.next();
                parts.next().map(str::to_string)
            })
    })
    .clone()
}

fn resolve_build_tool(name: &str) -> String {
    if name == "go" {
        if let Ok(path) = look_path("go") {
            return path;
        }
    }
    name.to_string()
}

fn signal_name(signal: i32) -> &'static str {
    match signal {
        1 => "hangup",
        2 => "interrupt",
        3 => "quit",
        6 => "aborted",
        9 => "killed",
        13 => "broken pipe",
        14 => "alarm clock",
        15 => "terminated",
        _ => "signal",
    }
}

impl BuildExecution {
    fn spawn(&self, dir: &str, name: &str, args: &[String]) -> Result<std::process::Child, Error> {
        let executable = resolve_build_tool(name);
        let mut cmd = std::process::Command::new(&executable);
        cmd.args(args);
        cmd.current_dir(dir);
        cmd.env("GOWORK", "off");
        cmd.env("GOFLAGS", "-mod=readonly");
        cmd.env("CGO_ENABLED", "0");
        if let Some(pin) = go_toolchain_pin() {
            cmd.env("GOTOOLCHAIN", pin);
        }
        cmd.stdin(std::process::Stdio::null());
        cmd.stdout(std::process::Stdio::piped());
        cmd.stderr(std::process::Stdio::piped());
        cmd.spawn().map_err(|e| {
            if e.kind() == std::io::ErrorKind::NotFound {
                Error::msg(format!(
                    "exec: {name:?}: executable file not found in $PATH"
                ))
            } else {
                Error::msg(e.to_string())
            }
        })
    }

    fn command_failed(&self, name: &str, detail: &str) -> Error {
        if self.cancelled.load(Ordering::SeqCst) {
            return Error::cancelled(format!(
                "{name} failed; retain attempt and inspect build.log: context canceled"
            ));
        }
        Error::msg(format!(
            "{name} failed; retain attempt and inspect build.log: {detail}"
        ))
    }

    fn observation_failed(&self, name: &str, detail: &str) -> Error {
        if self.cancelled.load(Ordering::SeqCst) {
            return Error::cancelled(format!(
                "{name} observation failed; retain attempt and inspect build.log: context canceled"
            ));
        }
        Error::msg(format!(
            "{name} observation failed; retain attempt and inspect build.log: {detail}"
        ))
    }

    /// Runs a command with stdout/stderr streaming to the log.
    pub fn execute(&mut self, dir: &str, name: &str, args: &[String]) -> Result<(), Error> {
        // Build arguments are public; raw provider outputs stay out of this
        // producer. Tool logs stay separate from the timing log.
        let _ = writeln!(self.log, "\n$ {name} {}", args.join(" "));
        if self.cancelled.load(Ordering::SeqCst) {
            return Err(self.command_failed(name, "context canceled"));
        }
        let mut child = self.spawn(dir, name, args).map_err(|e| {
            let detail = e.to_string();
            self.command_failed(name, &detail)
        })?;
        let stdout = child.stdout.take();
        let stderr = child.stderr.take();
        // Drain stderr to the log on a helper thread while stdout streams.
        let log_stderr = std::mem::replace(&mut self.log, Box::new(std::io::sink()));
        let log_stderr = Arc::new(Mutex::new(log_stderr));
        let stderr_thread = stderr.map(|mut err| {
            let log = log_stderr.clone();
            std::thread::spawn(move || {
                let mut buf = [0u8; 32 << 10];
                loop {
                    match std::io::Read::read(&mut err, &mut buf) {
                        Ok(0) => break,
                        Ok(n) => {
                            let _ = log.lock().unwrap().write_all(&buf[..n]);
                        }
                        Err(_) => break,
                    }
                }
            })
        });
        let log_main = log_stderr.clone();
        if let Some(mut out) = stdout {
            let mut buf = [0u8; 32 << 10];
            loop {
                // Honor cancellation mid-run like CommandContext.
                if self.cancelled.load(Ordering::SeqCst) {
                    let _ = child.kill();
                    break;
                }
                match std::io::Read::read(&mut out, &mut buf) {
                    Ok(0) => break,
                    Ok(n) => {
                        let _ = log_main.lock().unwrap().write_all(&buf[..n]);
                        if let Some(output) = self.output.as_mut() {
                            let _ = output.write_all(&buf[..n]);
                        }
                    }
                    Err(_) => break,
                }
            }
        }
        let status = child.wait().map_err(|e| {
            let detail = e.to_string();
            self.command_failed(name, &detail)
        })?;
        if let Some(thread) = stderr_thread {
            let _ = thread.join();
        }
        let log_restored = Arc::try_unwrap(log_stderr)
            .map(|mutex| mutex.into_inner().unwrap())
            .unwrap_or_else(|_| Box::new(std::io::sink()) as Box<dyn Write + Send>);
        self.log = log_restored;
        if self.cancelled.load(Ordering::SeqCst) {
            return Err(self.command_failed(name, "context canceled"));
        }
        use std::os::unix::process::ExitStatusExt;
        if let Some(signal) = status.signal() {
            let detail = format!("signal: {}", signal_name(signal));
            return Err(self.command_failed(name, &detail).with_signal(signal));
        }
        match status.code() {
            Some(0) => Ok(()),
            Some(code) => {
                let detail = format!("exit status {code}");
                Err(self.command_failed(name, &detail).with_exit_code(code))
            }
            None => Err(self
                .command_failed(name, "exit status -1")
                .with_exit_code(-1)),
        }
    }

    /// Runs a command, capturing trimmed stdout; stderr streams to the log.
    pub fn capture(&mut self, dir: &str, name: &str, args: &[String]) -> Result<String, Error> {
        let _ = writeln!(self.log, "\n$ {name} {}", args.join(" "));
        if self.cancelled.load(Ordering::SeqCst) {
            return Err(self.observation_failed(name, "context canceled"));
        }
        let mut child = self.spawn(dir, name, args).map_err(|e| {
            let detail = e.to_string();
            self.observation_failed(name, &detail)
        })?;
        let stdout = child.stdout.take();
        let stderr = child.stderr.take();
        let log_stderr = std::mem::replace(&mut self.log, Box::new(std::io::sink()));
        let log_stderr = Arc::new(Mutex::new(log_stderr));
        let stderr_thread = stderr.map(|mut err| {
            let log = log_stderr.clone();
            std::thread::spawn(move || {
                let mut buf = [0u8; 32 << 10];
                loop {
                    match std::io::Read::read(&mut err, &mut buf) {
                        Ok(0) => break,
                        Ok(n) => {
                            let _ = log.lock().unwrap().write_all(&buf[..n]);
                        }
                        Err(_) => break,
                    }
                }
            })
        });
        let mut collected = Vec::new();
        if let Some(mut out) = stdout {
            let mut buf = [0u8; 32 << 10];
            loop {
                if self.cancelled.load(Ordering::SeqCst) {
                    let _ = child.kill();
                    break;
                }
                match std::io::Read::read(&mut out, &mut buf) {
                    Ok(0) => break,
                    Ok(n) => collected.extend_from_slice(&buf[..n]),
                    Err(_) => break,
                }
            }
        }
        let status = child.wait().map_err(|e| {
            let detail = e.to_string();
            self.observation_failed(name, &detail)
        })?;
        if let Some(thread) = stderr_thread {
            let _ = thread.join();
        }
        let log_restored = Arc::try_unwrap(log_stderr)
            .map(|mutex| mutex.into_inner().unwrap())
            .unwrap_or_else(|_| Box::new(std::io::sink()) as Box<dyn Write + Send>);
        self.log = log_restored;
        if self.cancelled.load(Ordering::SeqCst) {
            return Err(self.observation_failed(name, "context canceled"));
        }
        use std::os::unix::process::ExitStatusExt;
        if let Some(signal) = status.signal() {
            let detail = format!("signal: {}", signal_name(signal));
            return Err(self.observation_failed(name, &detail).with_signal(signal));
        }
        match status.code() {
            Some(0) => Ok(String::from_utf8_lossy(&collected).trim().to_string()),
            Some(code) => {
                let detail = format!("exit status {code}");
                Err(self.observation_failed(name, &detail).with_exit_code(code))
            }
            None => Err(self
                .observation_failed(name, "exit status -1")
                .with_exit_code(-1)),
        }
    }
}

use std::os::unix::fs::OpenOptionsExt;

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::fs::PermissionsExt;
    use std::sync::atomic::AtomicU64;

    /// Caller must hold `test_env_lock` for the whole test body.
    fn progress_fixture(title: &str) -> (BuildProgress, String, SharedBuffer) {
        for key in [
            "SODA_BUILD_START_NS",
            "SODA_BUILD_TIMING_LOG",
            "SODA_BUILD_CHILD",
        ] {
            std::env::set_var(key, "");
        }
        let mut progress = new_build_progress("/repo", title).unwrap();
        let output = SharedBuffer::new();
        progress.stderr = Box::new(output.clone());
        let dir = std::env::temp_dir().join(format!(
            "soda-progress-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("timing.log").to_string_lossy().into_owned();
        progress.create_log(&path).unwrap();
        (progress, path, output)
    }

    fn fake_clock(now_ms: &Arc<AtomicU64>) -> Box<dyn Fn() -> Duration + Send + Sync> {
        let now_ms = now_ms.clone();
        Box::new(move || Duration::from_millis(now_ms.load(Ordering::SeqCst)))
    }

    #[test]
    fn oracle_native_timing_owner() {
        // Oracle: TestGoProductionUsesNativeTimingOwner.
        let _env = crate::test_env_lock();
        let (mut progress, path, output) = progress_fixture("Release fixture");
        progress.next("Build fixture").unwrap();
        let dir = std::env::temp_dir().to_string_lossy().into_owned();
        let mut execution = BuildExecution::default();
        let got = execution
            .capture(
                &dir,
                "sh",
                &["-c".to_string(), "printf native-image-id".to_string()],
            )
            .unwrap();
        assert_eq!(got, "native-image-id");
        progress.next("Failing fixture").unwrap();
        let err = execution
            .execute(
                &dir,
                "sh",
                &[
                    "-c".to_string(),
                    "exit 7".to_string(),
                    "private-sentinel-not-in-timings".to_string(),
                ],
            )
            .unwrap_err();
        assert_eq!(build_exit_code(Some(&err)), 7);
        progress.finish(Some(&err)).unwrap();
        let text = std::fs::read_to_string(&path).unwrap();
        for s in [
            "DONE     Build fixture",
            "FAILED   Failing fixture",
            "section ",
            "total ",
            "exit 7",
        ] {
            assert!(text.contains(s), "missing {s}: {text}");
        }
        assert!(!text.contains("private-sentinel") && !text.contains("SUCCESS"));
        assert!(output.text().contains("SECTION SUMMARY"));
        let mode = std::fs::metadata(&path).unwrap().permissions().mode() & 0o777;
        assert_eq!(mode, 0o600);
    }

    #[test]
    fn oracle_phase_clock_and_failed_output() {
        // Oracle: TestNativePhaseClockAndFailedOutput.
        let now_ms = Arc::new(AtomicU64::new(0));
        let output = SharedBuffer::new();
        let mut progress =
            BuildProgress::new_for_test("fixture", fake_clock(&now_ms), Box::new(output.clone()));
        progress.phase("P3").unwrap();
        now_ms.store(2000, Ordering::SeqCst);
        progress.next("Compile once").unwrap();
        now_ms.store(5000, Ordering::SeqCst);
        progress.phase("P4").unwrap();
        now_ms.store(9000, Ordering::SeqCst);
        let failure = Error::msg("failure");
        progress.finish(Some(&failure)).unwrap();
        progress.finish(None).unwrap();
        let text = output.text();
        for s in [
            "section 00:00:03",
            "P3 | phase 00:00:05",
            "P4 | phase 00:00:04",
            "exit 1",
        ] {
            assert!(text.contains(s), "missing {s}: {text}");
        }
        assert!(!text.contains("SUCCESS") && text.matches("exit 1").count() == 1);
    }

    #[test]
    fn oracle_note_reason_vectors() {
        // Oracle: TestNoteReasonAttachesToFailedOutput + SanitizedAndScoped.
        let now_ms = Arc::new(AtomicU64::new(0));
        let output = SharedBuffer::new();
        let mut progress =
            BuildProgress::new_for_test("fixture", fake_clock(&now_ms), Box::new(output.clone()));
        progress.phase("P1").unwrap();
        progress.note_reason("open /run/go/src/a.go: permission denied");
        now_ms.store(1000, Ordering::SeqCst);
        let failure = Error::msg("failure");
        progress.finish(Some(&failure)).unwrap();
        assert!(output.text().contains(
            "FAILED   P1 | phase 00:00:01 | total 00:00:01 | reason open /run/go/src/a.go: permission denied"
        ));

        let output = SharedBuffer::new();
        let mut progress = BuildProgress::new_for_test(
            "fixture",
            Box::new(|| Duration::ZERO),
            Box::new(output.clone()),
        );
        progress.note_reason("");
        progress.note_reason("   ");
        assert!(progress.reason.is_empty());
        progress.note_reason("a | b\nsecond");
        assert_eq!(progress.reason, "a / b");
        progress.note_reason(&"x".repeat(400));
        assert_eq!(progress.reason.chars().count(), 300);
        progress.phase("P1").unwrap();
        assert!(progress.reason.is_empty());
        progress.note_reason("stale");
        progress.next("step").unwrap();
        assert!(progress.reason.is_empty());
    }

    #[test]
    fn oracle_log_occupied_retained() {
        // Oracle: TestGoProgressRetainsOccupiedLog.
        let _env = crate::test_env_lock();
        let (mut progress, path, _output) = progress_fixture("fixture");
        std::env::set_var("SODA_BUILD_TIMING_LOG", "");
        let before = std::fs::read(&path).unwrap();
        assert!(progress.create_log(&path).is_err());
        assert_eq!(std::fs::read(&path).unwrap(), before);
    }

    #[test]
    fn oracle_child_shares_parent_total() {
        // Oracle: TestGoProgressSharesParentTotalAndSuppressesChildSummary.
        let _env = crate::test_env_lock();
        let (mut progress, path, output) = progress_fixture("fixture");
        let origin = std::env::var("SODA_BUILD_START_NS").unwrap_or_default();
        std::env::set_var("SODA_BUILD_CHILD", "1");
        progress.next("Child production").unwrap();
        progress.finish(None).unwrap();
        assert_eq!(
            std::env::var("SODA_BUILD_START_NS").unwrap_or_default(),
            origin
        );
        assert!(!output.text().contains("SUCCESS"));
        assert!(!output.text().contains("SECTION SUMMARY"));
        let body = std::fs::read_to_string(&path).unwrap();
        assert!(body.contains("DONE     Child production"));
    }

    #[test]
    fn oracle_cancellation_and_pinned_compiler() {
        // Oracle: TestGoProgressCancellationAndPinnedCompiler.
        let _env = crate::test_env_lock();
        let (mut progress, path, _output) = progress_fixture("fixture");
        progress.next("Cancelled production").unwrap();
        let mut execution = BuildExecution::default();
        execution.cancelled.store(true, Ordering::SeqCst);
        let dir = std::env::temp_dir().to_string_lossy().into_owned();
        let err = execution
            .execute(&dir, "sh", &["-c".to_string(), "exit 0".to_string()])
            .unwrap_err();
        assert_eq!(build_exit_code(Some(&err)), 130);
        progress.finish(Some(&err)).unwrap();
        let body = std::fs::read_to_string(&path).unwrap();
        assert!(body.contains("CANCELLED Cancelled production"));
        // Pinned compiler resolution.
        let resolved = resolve_build_tool("go");
        match look_path("go") {
            Ok(pinned) => assert!(resolved == pinned || resolved == "go", "{resolved}"),
            Err(_) => assert_eq!(resolved, "go"),
        }
    }
}
