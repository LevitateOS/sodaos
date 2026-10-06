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
#[derive(Debug, Default)]
pub struct Cancel {
    pub(crate) flag: AtomicBool,
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
        self.log.borrow_mut().attach(Box::new(shared.clone()));
        if !wants_media {
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
                    )));
                Ok(LogCloser {
                    files: vec![shared, shared_events],
                })
            }
            Err(e) => {
                let closer = LogCloser {
                    files: vec![shared],
                };
                let close_result = closer.close();
                let _ = close_result;
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

/// resolveBuildTool resolves a build tool at run time. The GOTOOLCHAIN pin in
/// buildEnvironment forces the exact compiler version; PATH decides which
/// installation provides it. A build-time GOROOT would answer a run-time
/// question with a stale path once the binary moves machines.
pub fn resolve_build_tool(name: &str) -> String {
    if name == "go" {
        if let Some(path) = look_path("go") {
            return path;
        }
    }
    name.to_string()
}

fn look_path(name: &str) -> Option<String> {
    let path = std::env::var_os("PATH")?;
    for dir in std::env::split_paths(&path) {
        let candidate = dir.join(name);
        if let Ok(meta) = fs::metadata(&candidate) {
            use std::os::unix::fs::PermissionsExt;
            if meta.is_file() && meta.permissions().mode() & 0o111 != 0 {
                return Some(candidate.to_string_lossy().into_owned());
            }
        }
    }
    None
}

pub fn run_build_command(
    cancel: &Cancel,
    log: &mut dyn Write,
    output: Option<&mut dyn Write>,
    dir: &str,
    name: &str,
    args: &[String],
) -> Result<String, Error> {
    let resolved = resolve_build_tool(name);
    let base = sys::base_name(&resolved);
    let mut command = std::process::Command::new(&resolved);
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
    let mut child = command.spawn().map_err(|e| Error::msg(e.to_string()))?;
    // D03-F1: drain both pipes concurrently with the exit poll. A child
    // filling the pipe buffer would otherwise block forever while the
    // parent waits for exit. Same shape as the worker drain threads.
    let stdout_drain = drain_pipe(child.stdout.take());
    let stderr_drain = drain_pipe(child.stderr.take());
    let status = loop {
        match child.try_wait().map_err(|e| Error::msg(e.to_string()))? {
            Some(status) => break status,
            None if cancel.is_cancelled() => {
                let _ = child.kill();
                let _ = child.wait();
                let _ = stdout_drain.map(|t| t.join());
                let _ = stderr_drain.map(|t| t.join());
                return Err(Error::msg(format!(
                    "{base} failed; retain attempt and inspect build.log: build cancelled"
                )));
            }
            None => std::thread::sleep(std::time::Duration::from_millis(10)),
        }
    };
    let stdout_bytes = stdout_drain
        .map(|t| t.join().unwrap_or_default())
        .unwrap_or_default();
    let stderr_bytes = stderr_drain
        .map(|t| t.join().unwrap_or_default())
        .unwrap_or_default();
    log.write_all(&stderr_bytes).map_err(Error::from)?;
    let text = String::from_utf8_lossy(&stdout_bytes).into_owned();
    match output {
        None => {}
        Some(out) => {
            log.write_all(&stdout_bytes).map_err(Error::from)?;
            out.write_all(&stdout_bytes).map_err(Error::from)?;
        }
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

#[cfg(test)]
mod tests;
