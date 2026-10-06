use super::paths::{go_strerror, path_error};
use super::MUSE_NATIVE;
use std::ffi::CString;
use std::fs;
use std::io;
use std::process::{Command, Stdio};
use std::time::{Duration, Instant};

pub(crate) fn check_runtime(version: &str) -> Result<(), String> {
    let private = make_private_dir("soda-muse-check-")?;
    struct Cleanup(String);
    impl Drop for Cleanup {
        fn drop(&mut self) {
            let _ = fs::remove_dir_all(&self.0);
        }
    }
    let _cleanup = Cleanup(private.clone());
    let output = run_pristine(
        MUSE_NATIVE,
        &["--version".to_string()],
        &private,
        Duration::from_secs(5),
    )
    .map_err(|e| format!("muse native runtime prerequisite failed: {e}"))?;
    if output.trim() != format!("Muse Code 1.4.0 ({version})") {
        return Err(format!("muse version differs from pinned {version}"));
    }
    Ok(())
}

fn make_private_dir(prefix: &str) -> Result<String, String> {
    let pid = unsafe { libc::getpid() };
    for n in 0..1000 {
        let path = if n == 0 {
            format!("/tmp/{prefix}{pid}")
        } else {
            format!("/tmp/{prefix}{pid}-{n}")
        };
        let c = CString::new(path.clone()).unwrap();
        if unsafe { libc::mkdir(c.as_ptr(), 0o700) } == 0 {
            return Ok(path);
        }
        let e = io::Error::last_os_error();
        if e.kind() != io::ErrorKind::AlreadyExists {
            return Err(path_error("mkdir", "/tmp", e));
        }
    }
    Err(String::from("temporary directory unavailable"))
}

// run_pristine runs the native binary with the fixed clean environment,
// killing it after the deadline the way Go's CommandContext does.
fn run_pristine(
    program: &str,
    args: &[String],
    root: &str,
    timeout: Duration,
) -> Result<String, String> {
    let mut child = Command::new(program)
        .args(args)
        .env_clear()
        .env("PATH", "/usr/local/bin:/usr/bin:/bin")
        .env("LANG", "C.UTF-8")
        .env("HOME", root)
        .env("XDG_CONFIG_HOME", format!("{root}/config"))
        .env("XDG_STATE_HOME", format!("{root}/state"))
        .env("XDG_CACHE_HOME", format!("{root}/cache"))
        .env("XDG_DATA_HOME", format!("{root}/data"))
        .env("TMPDIR", root)
        .env("TBH_CREDENTIAL_BACKEND", "file")
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|e| format!("fork/exec {program}: {}", go_strerror(e.raw_os_error())))?;
    let deadline = Instant::now() + timeout;
    loop {
        match child.try_wait().map_err(|e| e.to_string())? {
            Some(status) => {
                let mut stdout = Vec::new();
                if let Some(mut out) = child.stdout.take() {
                    use std::io::Read;
                    let _ = out.read_to_end(&mut stdout);
                }
                if !status.success() {
                    if let Some(code) = status.code() {
                        return Err(format!("exit status {code}"));
                    }
                    return Err(String::from("signal: killed"));
                }
                return Ok(String::from_utf8_lossy(&stdout).into_owned());
            }
            None => {
                if Instant::now() >= deadline {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Err(String::from("signal: killed"));
                }
                std::thread::sleep(Duration::from_millis(10));
            }
        }
    }
}
