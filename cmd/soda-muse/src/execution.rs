use super::paths::{errno_str, go_base, go_quote_rune, go_strerror, is_clean_absolute_path};
use super::MUSE_NATIVE;
use std::ffi::CString;
use std::fs;
use std::time::{Duration, Instant};

// execute is fixed product code started by the native supervisor.
pub(crate) fn execute(root: &str, cwd: &str, args: &[String]) -> (i32, Option<String>) {
    let ccwd = match CString::new(cwd) {
        Ok(c) => c,
        Err(_) => return (1, Some(format!("chdir {cwd}: invalid argument"))),
    };
    if unsafe { libc::chdir(ccwd.as_ptr()) } != 0 {
        return (1, Some(format!("chdir {cwd}: {}", errno_str())));
    }
    if !is_clean_absolute_path(root) || !root.starts_with("/run/soda-muse/") {
        return (1, Some(String::from("invalid Muse execution root")));
    }
    if let Err(e) = await_admission(root) {
        return (1, Some(e));
    }
    let state = match execution_state(root) {
        Ok(s) => s,
        Err(e) => return (1, Some(e)),
    };
    let env = muse_environment(root, &state);
    // execve only returns on failure; success replaces this process.
    let binary = CString::new(MUSE_NATIVE).unwrap();
    let mut argv: Vec<CString> = vec![CString::new("muse").unwrap()];
    for a in args {
        match CString::new(a.as_str()) {
            Ok(c) => argv.push(c),
            Err(_) => {
                return (
                    1,
                    Some(format!(
                        "muse execution failed: {}",
                        go_strerror(Some(libc::EINVAL))
                    )),
                );
            }
        }
    }
    let mut envp: Vec<CString> = Vec::with_capacity(env.len());
    for e in &env {
        match CString::new(e.as_str()) {
            Ok(c) => envp.push(c),
            Err(_) => {
                return (
                    1,
                    Some(format!(
                        "muse execution failed: {}",
                        go_strerror(Some(libc::EINVAL))
                    )),
                );
            }
        }
    }
    let argv_ptr: Vec<*const libc::c_char> = argv
        .iter()
        .map(|c| c.as_ptr())
        .chain(std::iter::once(std::ptr::null()))
        .collect();
    let env_ptr: Vec<*const libc::c_char> = envp
        .iter()
        .map(|c| c.as_ptr())
        .chain(std::iter::once(std::ptr::null()))
        .collect();
    unsafe { libc::execve(binary.as_ptr(), argv_ptr.as_ptr(), env_ptr.as_ptr()) };
    (1, Some(format!("muse execution failed: {}", errno_str())))
}

fn await_admission(root: &str) -> Result<(), String> {
    let ready = format!("{root}/ready");
    let deadline = Instant::now() + Duration::from_secs(10);
    loop {
        if let Ok(info) = fs::metadata(&ready) {
            if info.is_file() {
                return Ok(());
            }
        }
        if Instant::now() >= deadline {
            return Err(String::from("muse admission did not complete"));
        }
        std::thread::sleep(Duration::from_millis(25));
    }
}

fn execution_state(root: &str) -> Result<String, String> {
    let id = go_base(root);
    if id.len() != 32 {
        return Err(String::from("invalid Muse execution ID"));
    }
    if let Some(b) = id.bytes().find(|b| !b.is_ascii_hexdigit()) {
        return Err(format!("encoding/hex: invalid byte: {}", go_quote_rune(b)));
    }
    let state = format!("/tmp/soda-muse-state-{id}");
    mkdir_mode(&state, 0o700)?;
    for name in ["state", "cache", "data", "tmp"] {
        mkdir_mode(&format!("{state}/{name}"), 0o700)?;
    }
    Ok(state)
}

fn mkdir_mode(path: &str, mode: u32) -> Result<(), String> {
    let c = match CString::new(path) {
        Ok(c) => c,
        Err(_) => return Err(format!("mkdir {path}: invalid argument")),
    };
    if unsafe { libc::mkdir(c.as_ptr(), mode) } != 0 {
        return Err(format!("mkdir {path}: {}", errno_str()));
    }
    Ok(())
}

fn muse_environment(root: &str, state: &str) -> Vec<String> {
    muse_environment_with(root, state, &|n| env_lossy_opt(n))
}

fn env_lossy_opt(name: &str) -> Option<String> {
    std::env::var_os(name).map(|v| v.to_string_lossy().into_owned())
}

pub(crate) fn muse_environment_with(
    root: &str,
    state: &str,
    lookup: &dyn Fn(&str) -> Option<String>,
) -> Vec<String> {
    let mut env = vec![
        String::from("PATH=/usr/local/bin:/usr/bin:/bin"),
        String::from("LANG=C.UTF-8"),
        String::from("TBH_CREDENTIAL_BACKEND=file"),
        format!("XDG_CONFIG_HOME={root}/config"),
        format!("XDG_STATE_HOME={state}/state"),
        format!("XDG_CACHE_HOME={state}/cache"),
        format!("XDG_DATA_HOME={state}/data"),
        format!("TMPDIR={state}/tmp"),
    ];
    for name in ["HOME", "USER", "LOGNAME", "TERM", "COLORTERM"] {
        if let Some(value) = lookup(name) {
            if !value.is_empty() {
                env.push(format!("{name}={value}"));
            }
        }
    }
    env
}
