#[cfg(target_os = "linux")]
use std::io::{Read, Write};
#[cfg(target_os = "linux")]
use std::os::unix::process::CommandExt;
#[cfg(target_os = "linux")]
use std::process::{Command, Stdio};
use std::sync::{Arc, Condvar, Mutex};
use std::thread::JoinHandle;

use super::{reaper_main, Phase, Process, ProcessState, SharedWriter};
use crate::command::CommandSpec;
use crate::error::Error;

/// Start an owned process, like `StartProcess`. Stdout/stderr pump into the
/// shared redacting writers; stdin is inherited, null, or pumped bytes.
pub fn start_process(
    phase: &Phase,
    spec: &CommandSpec,
    out: SharedWriter,
    err: SharedWriter,
) -> Result<Arc<Process>, Error> {
    phase.check()?;
    start_process_inner(spec, out, err)
}

#[cfg(target_os = "linux")]
fn start_process_inner(
    spec: &CommandSpec,
    out: SharedWriter,
    err: SharedWriter,
) -> Result<Arc<Process>, Error> {
    start_process_linux(spec, out, err)
}

#[cfg(not(target_os = "linux"))]
fn start_process_inner(
    _spec: &CommandSpec,
    _out: SharedWriter,
    _err: SharedWriter,
) -> Result<Arc<Process>, Error> {
    Err(Error::msg(
        "safe owned process execution requires Linux non-reaping wait support",
    ))
}

#[cfg(target_os = "linux")]
fn start_process_linux(
    spec: &CommandSpec,
    out: SharedWriter,
    err: SharedWriter,
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
    let process = Arc::new(Process {
        pid,
        state: Mutex::new(ProcessState {
            finished: false,
            outcome: None,
        }),
        done: Condvar::new(),
        sealed: Mutex::new(false),
        stop_result: Mutex::new(None),
        pumps: Mutex::new(None),
        pump_error: Arc::new(Mutex::new(None)),
    });
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
fn spawn_pump(
    stream: Option<impl Read + Send + 'static>,
    writer: SharedWriter,
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
                    if let Err(e) = lock(&writer).write_bytes(&buf[..n]) {
                        *lock(&pump_error) = Some(e.to_string());
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
