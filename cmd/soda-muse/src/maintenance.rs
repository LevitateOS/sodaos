use super::env::go_strerror;
use super::json_string;
use super::MUSE_NATIVE;
use std::ffi::{CStr, CString};
use std::fs;
use std::io;
use std::os::unix::fs::MetadataExt;
use std::process::{Command, Stdio};
use std::time::{Duration, Instant};

pub(crate) fn account_for(actor: &str) -> Result<(), String> {
    if unsafe { libc::geteuid() } != 0 {
        return Err(String::from("project root required"));
    }
    let parsed: i64 = actor
        .parse()
        .map_err(|_| String::from("invalid account identity"))?;
    if parsed <= 0 {
        return Err(String::from("invalid account identity"));
    }
    let directory = "/var/lib/soda/accounts";
    for path in ["/var", "/var/lib", "/var/lib/soda", directory] {
        account_node(path, true)?;
    }
    let mut names: Vec<String> = Vec::new();
    let entries = fs::read_dir(directory).map_err(|e| path_error("open", directory, e))?;
    for entry in entries {
        let entry = entry.map_err(|e| path_error("open", directory, e))?;
        names.push(entry.file_name().to_string_lossy().into_owned());
    }
    names.sort();
    for name in &names {
        if account_entry(directory, name, actor)? {
            return Ok(());
        }
    }
    Err(String::from("provisioned account missing"))
}

pub(crate) fn path_error(op: &str, path: &str, e: io::Error) -> String {
    format!("{op} {path}: {}", go_strerror(e.raw_os_error()))
}

fn account_entry(directory: &str, name: &str, actor: &str) -> Result<bool, String> {
    let marker = format!("{directory}/{name}");
    if account_node(&marker, false).is_err() {
        return Ok(false);
    }
    let body = fs::read(&marker).map_err(|e| path_error("open", &marker, e))?;
    if String::from_utf8_lossy(&body).trim() != actor {
        return Ok(false);
    }
    let (uid, gid, gecos, dir) = lookup_user(name)?;
    let mut out = String::from("{\"Uid\":");
    out.push_str(&json_string(&uid.to_string()));
    out.push_str(",\"Gid\":");
    out.push_str(&json_string(&gid.to_string()));
    out.push_str(",\"Username\":");
    out.push_str(&json_string(name));
    out.push_str(",\"Name\":");
    out.push_str(&json_string(&gecos));
    out.push_str(",\"HomeDir\":");
    out.push_str(&json_string(&dir));
    out.push_str("}\n");
    print!("{out}");
    use std::io::Write;
    io::stdout().flush().map_err(|e| e.to_string())?;
    Ok(true)
}

pub(crate) fn account_node(path: &str, directory: bool) -> Result<(), String> {
    let info = fs::symlink_metadata(path).map_err(|e| path_error("lstat", path, e))?;
    if info.uid() != 0 || info.gid() != 0 || info.mode() & 0o022 != 0 {
        return Err(String::from("unsafe account record"));
    }
    if directory {
        if !info.is_dir() {
            return Err(String::from("account ancestor is not a directory"));
        }
        return Ok(());
    }
    if !info.is_file() || info.mode() & 0o777 != 0o600 || info.len() > 64 {
        return Err(String::from("unsafe account marker"));
    }
    Ok(())
}

fn lookup_user(name: &str) -> Result<(u32, u32, String, String), String> {
    let cname = CString::new(name).map_err(|_| format!("user: unknown user {name}"))?;
    let mut pwd: libc::passwd = unsafe { std::mem::zeroed() };
    let mut buf = [0u8; 16384];
    let mut result: *mut libc::passwd = std::ptr::null_mut();
    let rc = unsafe {
        libc::getpwnam_r(
            cname.as_ptr(),
            &mut pwd,
            buf.as_mut_ptr() as *mut libc::c_char,
            buf.len(),
            &mut result,
        )
    };
    if rc != 0 || result.is_null() {
        return Err(format!("user: unknown user {name}"));
    }
    let gecos = unsafe { CStr::from_ptr(pwd.pw_gecos) }
        .to_string_lossy()
        .into_owned();
    let dir = unsafe { CStr::from_ptr(pwd.pw_dir) }
        .to_string_lossy()
        .into_owned();
    Ok((pwd.pw_uid, pwd.pw_gid, gecos, dir))
}

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
