use std::fs;
use std::io::Read;
use std::os::unix::io::FromRawFd;
use std::process::{Command, Stdio};
use std::time::{Duration, Instant};

pub(crate) fn podman(args: &[&str], deadline: Instant) -> Result<Vec<u8>, String> {
    let mut cmd = Command::new("podman");
    cmd.arg("--remote=false");
    for a in args {
        cmd.arg(a);
    }
    cmd.stdin(Stdio::null());
    cmd.stdout(Stdio::piped());
    cmd.stderr(Stdio::null());
    let child = cmd
        .spawn()
        .map_err(|_| String::from("project maintenance command failed"))?;
    wait_output(child, deadline)
}

// podman_streamed feeds stdin from a writer thread through a pipe while
// podman consumes it, mirroring Go's io.Pipe archive delivery. A podman
// failure wins over the archive result exactly like stagePublicTools.
pub(crate) fn podman_streamed(
    feed: impl FnOnce(fs::File) -> Result<(), String> + Send + 'static,
    args: &[&str],
    deadline: Instant,
) -> Result<(), String> {
    let mut fds = [0; 2];
    // O_CLOEXEC is load-bearing: without it the spawned child inherits
    // the stdin write end and its stdin reader never sees EOF.
    if unsafe { libc::pipe2(fds.as_mut_ptr(), libc::O_CLOEXEC) } != 0 {
        return Err(String::from("project maintenance command failed"));
    }
    let writer = unsafe { fs::File::from_raw_fd(fds[1]) };
    let reader = unsafe { fs::File::from_raw_fd(fds[0]) };
    let feeder = std::thread::spawn(move || feed(writer));
    let mut cmd = Command::new("podman");
    cmd.arg("--remote=false");
    for a in args {
        cmd.arg(a);
    }
    cmd.stdin(Stdio::from(reader));
    cmd.stdout(Stdio::piped());
    cmd.stderr(Stdio::null());
    let child = match cmd.spawn() {
        Ok(child) => child,
        Err(_) => {
            let _ = feeder.join();
            return Err(String::from("project maintenance command failed"));
        }
    };
    let output = wait_output(child, deadline);
    let fed = feeder
        .join()
        .map_err(|_| String::from("project maintenance command failed"))?;
    output?;
    fed
}

pub(crate) fn wait_output(
    mut child: std::process::Child,
    deadline: Instant,
) -> Result<Vec<u8>, String> {
    // Stdout drains on its own thread like exec.Output, so large output
    // never deadlocks against the exit wait. The child handle stays here
    // for the deadline kill.
    let failed = String::from("project maintenance command failed");
    let stdout = child.stdout.take();
    let reader = std::thread::spawn(move || {
        let mut out = Vec::new();
        match stdout {
            Some(mut pipe) => match pipe.read_to_end(&mut out) {
                Ok(_) => Some(out),
                Err(_) => None,
            },
            None => Some(out),
        }
    });
    loop {
        match child.try_wait() {
            Ok(Some(status)) => {
                let out = reader
                    .join()
                    .map_err(|_| failed.clone())?
                    .ok_or(failed.clone())?;
                if !status.success() {
                    return Err(failed);
                }
                return Ok(out);
            }
            Ok(None) => {
                if Instant::now() >= deadline {
                    let _ = child.kill();
                    let _ = child.wait();
                    let _ = reader.join();
                    return Err(failed);
                }
                std::thread::sleep(Duration::from_millis(10));
            }
            Err(_) => {
                let _ = reader.join();
                return Err(failed);
            }
        }
    }
}
