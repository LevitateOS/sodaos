use super::paths::{errno_str, go_base, go_quote_rune, go_strerror, is_clean_absolute_path};
use super::store_resolve::{enforce_receipt, resolve_muse_native};
use super::tool::Tool;
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
    let tool = Tool::current();
    let env = match tool {
        Tool::Muse => match execution_state(root) {
            Ok(state) => muse_environment(root, &state),
            Err(e) => return (1, Some(e)),
        },
        Tool::Codex => match link_codex_home(root) {
            Ok(()) => Vec::new(),
            Err(e) => return (1, Some(e)),
        },
    };
    // The shared store's selected bundle wins over the staged binary; a
    // known-stale or unusable selection refuses instead of running old
    // bytes, and the staged receipt must agree with the staged file.
    let binary_path = match resolve_muse_native() {
        Ok(path) => path,
        Err(e) => return (1, Some(e)),
    };
    if let Err(e) = enforce_receipt(&binary_path) {
        return (1, Some(e));
    }
    let env = if tool == Tool::Codex {
        codex_environment(root, &binary_path)
    } else {
        env
    };
    // execve only returns on failure; success replaces this process.
    let binary = match CString::new(binary_path) {
        Ok(binary) => binary,
        Err(_) => {
            return (
                1,
                Some(String::from("muse execution failed: invalid argument")),
            )
        }
    };
    let mut argv: Vec<CString> = vec![CString::new(tool.name()).unwrap()];
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

// Codex keeps its credential in the private, writable CODEX_HOME the host
// staged; everything else there links back to the account's ~/.codex, so
// config, sessions and history persist across runs.
fn link_codex_home(root: &str) -> Result<(), String> {
    let home = env_lossy_opt("HOME").unwrap_or_default();
    if !is_clean_absolute_path(&home) {
        return Err(String::from("codex needs an absolute $HOME"));
    }
    let persistent = format!("{home}/.codex");
    let private = format!("{root}/codex");
    ensure_dir(&persistent)?;
    ensure_dir(&format!("{persistent}/sessions"))?;
    let history = format!("{persistent}/history.jsonl");
    if fs::symlink_metadata(&history).is_err() {
        use std::os::unix::fs::OpenOptionsExt;
        fs::OpenOptions::new()
            .create(true)
            .append(true)
            .mode(0o600)
            .open(&history)
            .map_err(|e| format!("create {history}: {e}"))?;
    }
    let entries = fs::read_dir(&persistent).map_err(|e| format!("read {persistent}: {e}"))?;
    for entry in entries {
        let entry = entry.map_err(|e| format!("read {persistent}: {e}"))?;
        let name = entry.file_name();
        if name == "auth.json" {
            continue;
        }
        let target = std::path::Path::new(&private).join(&name);
        if fs::symlink_metadata(&target).is_ok() {
            continue;
        }
        std::os::unix::fs::symlink(entry.path(), &target)
            .map_err(|e| format!("link {}: {e}", target.display()))?;
    }
    Ok(())
}

fn ensure_dir(path: &str) -> Result<(), String> {
    match fs::symlink_metadata(path) {
        Ok(meta) if meta.is_dir() => Ok(()),
        Ok(_) => Err(format!("{path} is not a directory")),
        Err(_) => mkdir_mode(path, 0o700),
    }
}

fn codex_environment(root: &str, binary: &str) -> Vec<String> {
    // The bundle ships helper tools (ripgrep, bubblewrap) beside the
    // executable.
    let helpers = std::path::Path::new(binary)
        .parent()
        .and_then(|bin| bin.parent())
        .map(|bundle| {
            format!(
                "{}:{}:",
                bundle.join("codex-path").display(),
                bundle.join("codex-resources").display()
            )
        })
        .unwrap_or_default();
    let mut env = vec![
        format!("PATH={helpers}/usr/local/bin:/usr/bin:/bin"),
        String::from("LANG=C.UTF-8"),
        format!("CODEX_HOME={root}/codex"),
    ];
    for name in ["HOME", "USER", "LOGNAME", "TERM", "COLORTERM"] {
        if let Some(value) = env_lossy_opt(name) {
            if !value.is_empty() {
                env.push(format!("{name}={value}"));
            }
        }
    }
    env
}
