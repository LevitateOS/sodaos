//! Build command execution substrate: cancellation, the recall-ring command
//! runner with log attach, and the current child environment.
use std::cell::RefCell;
use std::fs;
use std::io::Write;
use std::rc::Rc;
use std::sync::atomic::{AtomicBool, Ordering};

use crate::build::PINNED_GO_VERSION;
use crate::error::{join_close, Error};
use crate::recall;
use crate::sys;

/// Cancellation flag standing in for the Go build context.
#[derive(Debug, Default, Clone)]
pub struct Cancel {
    pub(crate) flag: std::sync::Arc<AtomicBool>,
}

impl Cancel {
    pub fn new() -> Cancel {
        Cancel::default()
    }

    pub fn cancel(&self) {
        self.flag.store(true, Ordering::SeqCst);
    }

    pub fn is_cancelled(&self) -> bool {
        self.flag.load(Ordering::SeqCst)
    }
}

/// Shared build-log file: the recall ring writes through it while the
/// closer still owns it.
#[derive(Clone)]
pub struct SharedFile(Rc<RefCell<fs::File>>);

impl SharedFile {
    pub fn wrap(file: fs::File) -> SharedFile {
        SharedFile(Rc::new(RefCell::new(file)))
    }
}

impl Write for SharedFile {
    fn write(&mut self, data: &[u8]) -> std::io::Result<usize> {
        self.0.borrow_mut().write(data)
    }

    fn flush(&mut self) -> std::io::Result<()> {
        self.0.borrow_mut().flush()
    }
}

/// Command runner wired to one recall ring: the `execute`/`capture` pair the
/// Go `Build` constructor closes over, plus log attach and failure reason.
#[derive(Clone)]
pub struct Runner {
    cancel: Rc<Cancel>,
    pub(crate) log: Rc<RefCell<recall::RecallLog>>,
}

impl Runner {
    pub fn new(cancel: Rc<Cancel>) -> Runner {
        Runner {
            cancel,
            log: Rc::new(RefCell::new(recall::RecallLog::new())),
        }
    }

    pub fn execute(&self, dir: &str, name: &str, args: &[String]) -> Result<(), Error> {
        let mut log = self.log.borrow_mut();
        let stdout = std::io::stdout();
        let mut locked = stdout.lock();
        run_build_command(&self.cancel, &mut *log, Some(&mut locked), dir, name, args)?;
        Ok(())
    }

    pub fn capture(&self, dir: &str, name: &str, args: &[String]) -> Result<String, Error> {
        let mut log = self.log.borrow_mut();
        run_build_command(&self.cancel, &mut *log, None, dir, name, args)
    }

    pub fn open_log(&self, path: &str, wants_media: bool) -> Result<LogCloser, Error> {
        let file = fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .open(path)
            .map_err(|e| Error::msg(e.to_string()))?;
        use std::os::unix::fs::PermissionsExt;
        file.set_permissions(fs::Permissions::from_mode(0o600))?;
        let shared = SharedFile::wrap(file);
        if !wants_media {
            self.log.borrow_mut().attach(Box::new(shared.clone()))?;
            return Ok(LogCloser {
                files: vec![shared],
            });
        }
        let events_path = sys::join(&[&sys::dir_name(path), "media-events.jsonl"]);
        let events = fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .open(&events_path);
        match events {
            Ok(events) => {
                events.set_permissions(fs::Permissions::from_mode(0o600))?;
                let shared_events = SharedFile::wrap(events);
                self.log
                    .borrow_mut()
                    .attach(Box::new(crate::events::MediaEventWriter::new(
                        shared.clone(),
                        shared_events.clone(),
                    )))?;
                Ok(LogCloser {
                    files: vec![shared, shared_events],
                })
            }
            Err(e) => {
                let closer = LogCloser {
                    files: vec![shared],
                };
                let _ = closer.close();
                Err(Error::msg(e.to_string()))
            }
        }
    }

    pub fn reason(&self) -> String {
        self.log.borrow().reason()
    }
}

/// Build-log closer: flushing + dropping closes the exclusive log files.
pub struct LogCloser {
    files: Vec<SharedFile>,
}

impl LogCloser {
    pub fn close(self) -> Result<(), Error> {
        let mut result: Result<(), Error> = Ok(());
        for mut file in self.files {
            result = join_close(result, file.flush().map_err(Error::from));
        }
        result
    }
}

/// Only the build's tool/cache environment is inherited. In particular,
/// provider, installed-test, signing and private-token variables cannot
/// activate extra work.
pub fn build_environment_pairs() -> Vec<(String, String)> {
    let mut env = Vec::new();
    for key in [
        "HOME",
        "PATH",
        "TMPDIR",
        "XDG_RUNTIME_DIR",
        "XDG_CACHE_HOME",
        "GOCACHE",
        "GOMODCACHE",
        "PLAYWRIGHT_BROWSERS_PATH",
        "HTTP_PROXY",
        "HTTPS_PROXY",
        "NO_PROXY",
    ] {
        if let Ok(value) = std::env::var(key) {
            env.push((key.to_string(), value));
        }
    }
    // A -trimpath controller can have no embedded GOROOT. Use Go's pinned
    // upstream toolchain selection, not a relative bin/go or the ambient version.
    env.push(("GOTOOLCHAIN".to_string(), PINNED_GO_VERSION.to_string()));
    env.push(("GOWORK".to_string(), "off".to_string()));
    env.push(("GOFLAGS".to_string(), "-mod=readonly".to_string()));
    env.push(("CGO_ENABLED".to_string(), "0".to_string()));
    env
}

pub fn run_build_command(
    cancel: &Cancel,
    log: &mut dyn Write,
    output: Option<&mut dyn Write>,
    dir: &str,
    name: &str,
    args: &[String],
) -> Result<String, Error> {
    let budget = bounded_operation_deadline(&sys::base_name(name), args);
    run_build_command_with_budget(cancel, log, output, dir, name, args, budget)
}

fn run_build_command_with_budget(
    cancel: &Cancel,
    log: &mut dyn Write,
    mut output: Option<&mut dyn Write>,
    dir: &str,
    name: &str,
    args: &[String],
    budget: Option<std::time::Duration>,
) -> Result<String, Error> {
    let operation_deadline = budget.map(|duration| std::time::Instant::now() + duration);
    let base = sys::base_name(name);
    let mut command = std::process::Command::new(name);
    command
        .args(args)
        .current_dir(dir)
        .env_clear()
        .envs(build_environment_pairs());
    command
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::piped());
    // Never copy raw argv, environment or stdin into the progress/evidence stream.
    writeln!(log, "\nCOMMAND {base}").map_err(Error::from)?;
    let child = command.spawn().map_err(|e| Error::msg(e.to_string()))?;
    let mut child = ChildGuard(child);
    const CAPTURE_LIMIT: usize = 16 * 1024 * 1024;
    const DRAIN_GRACE: std::time::Duration = std::time::Duration::from_secs(2);
    let mut stdout = child.stdout.take();
    let mut stderr = child.stderr.take();
    set_nonblocking(stdout.as_ref().map(std::os::fd::AsRawFd::as_raw_fd))?;
    set_nonblocking(stderr.as_ref().map(std::os::fd::AsRawFd::as_raw_fd))?;
    let mut captured = Vec::new();
    let mut status = None;
    let mut drain_deadline = None;
    let mut timed_out = false;
    while status.is_none() || stdout.is_some() || stderr.is_some() {
        if !timed_out
            && status.is_none()
            && operation_deadline.is_some_and(|deadline| std::time::Instant::now() >= deadline)
        {
            timed_out = true;
            let _ = child.kill();
            status = child.wait().ok();
            drain_deadline = Some(std::time::Instant::now() + DRAIN_GRACE);
        }
        if cancel.is_cancelled() && status.is_none() {
            let _ = child.kill();
            status = child.wait().ok();
            drain_deadline = Some(std::time::Instant::now() + DRAIN_GRACE);
        }
        if status.is_none() {
            if let Some(child_status) = child.try_wait().map_err(|e| Error::msg(e.to_string()))? {
                status = Some(child_status);
                drain_deadline = Some(std::time::Instant::now() + DRAIN_GRACE);
            }
        }
        let mut progressed = false;
        match read_available(&mut stdout)? {
            PipeRead::Data(bytes) => {
                progressed = true;
                if let Some(out) = output.as_deref_mut() {
                    log.write_all(&bytes).map_err(Error::from)?;
                    out.write_all(&bytes).map_err(Error::from)?;
                } else if captured.len().saturating_add(bytes.len()) <= CAPTURE_LIMIT {
                    captured.extend_from_slice(&bytes);
                } else {
                    return Err(Error::msg(format!(
                        "{base} failed; capture exceeded {CAPTURE_LIMIT} bytes"
                    )));
                }
            }
            PipeRead::Eof => progressed = true,
            PipeRead::Pending => {}
        }
        match read_available(&mut stderr)? {
            PipeRead::Data(bytes) => {
                progressed = true;
                log.write_all(&bytes).map_err(Error::from)?;
            }
            PipeRead::Eof => progressed = true,
            PipeRead::Pending => {}
        }
        if drain_deadline.is_some_and(|deadline| std::time::Instant::now() >= deadline)
            && (stdout.is_some() || stderr.is_some())
        {
            // Close our pipe ends; descendants outside direct-child authority
            // may still hold their copies.
            stdout.take();
            stderr.take();
            return Err(Error::msg(format!(
                "{base} failed; retain attempt and inspect build.log: output pipe remained open after child exit"
            )));
        }
        if !progressed {
            std::thread::sleep(std::time::Duration::from_millis(10));
        }
    }
    let status =
        status.ok_or_else(|| Error::msg(format!("{base} failed; child status unavailable")))?;
    let text = String::from_utf8_lossy(&captured).into_owned();
    if timed_out {
        return Err(Error::msg(format!(
            "{base} failed; retain attempt and inspect build.log: operation deadline exceeded"
        )));
    }
    if cancel.is_cancelled() {
        return Err(Error::msg(format!(
            "{base} failed; retain attempt and inspect build.log: build cancelled"
        )));
    }
    if !status.success() {
        let reason = match status.code() {
            Some(code) => format!("exit status {code}"),
            None => "terminated by signal".to_string(),
        };
        return Err(Error::msg(format!(
            "{base} failed; retain attempt and inspect build.log: {reason}"
        )));
    }
    Ok(text.trim().to_string())
}

fn bounded_operation_deadline(base: &str, args: &[String]) -> Option<std::time::Duration> {
    let metadata = base == "cargo" && args.first().is_some_and(|arg| arg == "metadata");
    let installer_version_probe = base == "podman"
        && args.iter().any(|arg| arg == "run")
        && args
            .iter()
            .any(|arg| arg == "--entrypoint=/usr/bin/coreos-installer")
        && args.last().is_some_and(|arg| arg == "--version");
    (metadata || installer_version_probe).then_some(std::time::Duration::from_secs(120))
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

fn set_nonblocking(fd: Option<std::os::fd::RawFd>) -> Result<(), Error> {
    let Some(fd) = fd else { return Ok(()) };
    let flags = unsafe { libc::fcntl(fd, libc::F_GETFL) };
    if flags < 0 || unsafe { libc::fcntl(fd, libc::F_SETFL, flags | libc::O_NONBLOCK) } < 0 {
        return Err(Error::msg(std::io::Error::last_os_error().to_string()));
    }
    Ok(())
}

#[derive(Debug)]
enum PipeRead {
    Data(Vec<u8>),
    Pending,
    Eof,
}

fn read_available<R: std::io::Read>(pipe: &mut Option<R>) -> Result<PipeRead, Error> {
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
        Err(error) => Err(Error::msg(error.to_string())),
    }
}

#[cfg(test)]
mod tests;
