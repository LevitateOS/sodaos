#[cfg(target_os = "linux")]
use std::io::{Read, Write};
#[cfg(target_os = "linux")]
use std::os::unix::process::CommandExt;
#[cfg(target_os = "linux")]
use std::process::{Command, Stdio};
use std::sync::{Arc, Mutex};
use std::thread::JoinHandle;

use super::owned_process::{reaper_main, Process};
use super::{Phase, SharedWriter};
use crate::command::CommandSpec;
use crate::error::Error;
use crate::evidence::RedactingWriter;

/// Byte sink for pump threads. Redacting writers keep their exact
/// failure strings; raw sinks never fail (they bound instead).
pub(super) trait PumpSink {
    fn pump_write(&mut self, bytes: &[u8]) -> Result<(), String>;
}

impl PumpSink for RedactingWriter {
    fn pump_write(&mut self, bytes: &[u8]) -> Result<(), String> {
        self.write_bytes(bytes)
            .map(|_| ())
            .map_err(|e| e.to_string())
    }
}

/// Start an owned process, like `StartProcess`. Stdout/stderr pump into the
/// shared redacting writers; stdin is inherited, null, or pumped bytes.
pub fn start_process(
    phase: &Phase,
    spec: &CommandSpec,
    out: SharedWriter,
    err: SharedWriter,
) -> Result<Arc<Process>, Error> {
    phase.check()?;
    start_inner(spec, out, err)
}

#[cfg(target_os = "linux")]
pub(super) fn start_inner<S: PumpSink + Send + 'static>(
    spec: &CommandSpec,
    out: Arc<Mutex<S>>,
    err: Arc<Mutex<S>>,
) -> Result<Arc<Process>, Error> {
    start_inner_linux(spec, out, err)
}

#[cfg(not(target_os = "linux"))]
pub(super) fn start_inner<S>(
    _spec: &CommandSpec,
    _out: Arc<Mutex<S>>,
    _err: Arc<Mutex<S>>,
) -> Result<Arc<Process>, Error> {
    Err(Error::msg(
        "safe owned process execution requires Linux non-reaping wait support",
    ))
}

#[cfg(target_os = "linux")]
fn start_inner_linux<S: PumpSink + Send + 'static>(
    spec: &CommandSpec,
    out: Arc<Mutex<S>>,
    err: Arc<Mutex<S>>,
) -> Result<Arc<Process>, Error> {
    use crate::command::StdinSpec;

    let mut command = Command::new(&spec.name);
    command.args(&spec.args);
    if let Some(dir) = &spec.dir {
        command.current_dir(dir);
    }
    for entry in &spec.env {
        if let Some((key, value)) = entry.split_once('=') {
            command.env(key, value);
        }
    }
    let stdin_bytes = match &spec.stdin {
        StdinSpec::Inherit => {
            command.stdin(Stdio::inherit());
            None
        }
        StdinSpec::Null => {
            command.stdin(Stdio::null());
            None
        }
        StdinSpec::Bytes(data) => {
            command.stdin(Stdio::piped());
            Some(data.clone())
        }
    };
    command.stdout(Stdio::piped()).stderr(Stdio::piped());
    unsafe {
        command.pre_exec(|| {
            if libc::setpgid(0, 0) == 0 {
                Ok(())
            } else {
                Err(std::io::Error::last_os_error())
            }
        });
    }
    let mut child = command.spawn()?;
    let pid = child.id() as i32;
    if let (Some(bytes), Some(mut stdin)) = (stdin_bytes, child.stdin.take()) {
        std::thread::spawn(move || {
            // A child that exits before reading stdin reports its exit
            // status, not the broken pipe; Go surfaces the same outcome.
            let _ = stdin.write_all(&bytes);
            let _ = stdin.flush();
        });
    }
    let process = Arc::new(Process::new(pid));
    let stdout = child.stdout.take();
    let stderr = child.stderr.take();
    let pump_out = spawn_pump(stdout, out, process.pump_error.clone());
    let pump_err = spawn_pump(stderr, err, process.pump_error.clone());
    *process.pumps.lock().unwrap_or_else(|e| e.into_inner()) = Some((pump_out, pump_err));
    let reaper = process.clone();
    std::thread::spawn(move || reaper_main(reaper, pid));
    Ok(process)
}

pub(super) fn lock<T>(mutex: &Mutex<T>) -> std::sync::MutexGuard<'_, T> {
    mutex.lock().unwrap_or_else(|e| e.into_inner())
}

#[cfg(target_os = "linux")]
fn spawn_pump<S: PumpSink + Send + 'static>(
    stream: Option<impl Read + Send + 'static>,
    writer: Arc<Mutex<S>>,
    pump_error: Arc<Mutex<Option<String>>>,
) -> JoinHandle<()> {
    std::thread::spawn(move || {
        let mut stream = match stream {
            Some(stream) => stream,
            None => return,
        };
        let mut buf = [0u8; 32768];
        loop {
            match stream.read(&mut buf) {
                Ok(0) => break,
                Ok(n) => {
                    if let Err(message) = lock(&writer).pump_write(&buf[..n]) {
                        *lock(&pump_error) = Some(message);
                        break;
                    }
                }
                Err(e) => {
                    *lock(&pump_error) = Some(e.to_string());
                    break;
                }
            }
        }
    })
}
