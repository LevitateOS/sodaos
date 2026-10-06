use std::env;
use std::ffi::CString;
use std::fs;
use std::io::{self, Read, Write};
use std::os::unix::ffi::OsStrExt;
use std::os::unix::fs::OpenOptionsExt;
use std::path::{Path, PathBuf};

use crate::process::{
    capture, config_json, run, run_stdout_null, stripped_string, trust_json, Captured,
};
use crate::{faccessat, fail, umask, Exit, AT_FDCWD, AUTHORITY, PREFIX_DEFAULT, WORKER_USER, X_OK};

/// `env_or` mirrors `${VAR:-default}`: unset or empty falls back.
fn env_or(key: &str, default: &str) -> String {
    match env::var(key) {
        Ok(v) if !v.is_empty() => v,
        _ => default.to_string(),
    }
}

fn current_umask() -> u32 {
    unsafe {
        let mask = umask(0);
        umask(mask);
        mask
    }
}

/// `write_staged` mirrors shell redirection into staging: bytes exact, mode
/// 0666 filtered by the umask like `>` and `open("w")`.
fn write_staged(path: &Path, bytes: &[u8]) -> io::Result<()> {
    let mode = 0o666 & !current_umask();
    let mut opts = fs::OpenOptions::new();
    opts.write(true).create(true).truncate(true).mode(mode);
    opts.open(path)?.write_all(bytes)
}

fn is_executable(path: &Path) -> bool {
    match CString::new(path.as_os_str().as_bytes()) {
        Ok(c) => unsafe { faccessat(AT_FDCWD, c.as_ptr(), X_OK, 0) == 0 },
        Err(_) => false,
    }
}

/// `command_v` mirrors `command -v`: a PATH search with execute permission.
fn command_v(tool: &str) -> Option<PathBuf> {
    let raw = env::var_os("PATH").unwrap_or_else(|| "/bin:/usr/bin".into());
    command_v_in(tool, &raw)
}

fn command_v_in(tool: &str, path_env: &std::ffi::OsStr) -> Option<PathBuf> {
    if tool.contains('/') {
        let path = PathBuf::from(tool);
        return is_executable(&path).then_some(path);
    }
    for dir in env::split_paths(path_env) {
        let candidate = dir.join(tool);
        if is_executable(&candidate) {
            return Some(candidate);
        }
    }
    None
}

/// `random_hex_passphrase` mirrors `head -c 32 /dev/urandom | od -An -tx1
/// | tr -d ' \n'`: 64 lowercase hex characters, no trailing newline.
pub(crate) fn random_hex_passphrase() -> Result<String, Exit> {
    let mut bytes = [0u8; 32];
    let mut urandom = fs::File::open("/dev/urandom").map_err(|_| Exit::Propagate(1))?;
    urandom
        .read_exact(&mut bytes)
        .map_err(|_| Exit::Propagate(1))?;
    let mut hex = String::with_capacity(64);
    for byte in bytes {
        hex.push_str(&format!("{byte:02x}"));
    }
    Ok(hex)
}

fn unix_now() -> Result<u64, Exit> {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|elapsed| elapsed.as_secs())
        .map_err(|_| Exit::Fail("cannot read the current time".to_string()))
}

/// Staging reads and writes that the script did through `python3` surface
/// as rotation errors here instead of tracebacks; the file bytes are
/// identical.
fn read_staged(path: &str) -> Result<String, Exit> {
    match fs::read(path) {
        Ok(bytes) => String::from_utf8(bytes)
            .map_err(|_| Exit::Fail(format!("cannot read staged file {path}"))),
        Err(_) => Err(Exit::Fail(format!("cannot read staged file {path}"))),
    }
}

fn stage_file(path: &str, bytes: &[u8]) -> Result<(), Exit> {
    write_staged(Path::new(path), bytes)
        .map_err(|_| Exit::Fail(format!("cannot write staged file {path}")))
}

pub(crate) fn rotate_fixture_authority(cleanup: &mut Vec<PathBuf>) -> Result<(), Exit> {
    if env_or("SODA_ROTATE_ACK", "") != "fixture-authority" {
        return fail("refusing: set SODA_ROTATE_ACK=fixture-authority to execute");
    }
    if command_v("skopeo").is_none() {
        return fail("skopeo required");
    }
    let tmpd = match capture("mktemp", &["-d"], false) {
        Captured::SpawnFailed => return fail("cannot stage secrets"),
        Captured::Done(code, out) => {
            if code != 0 {
                return fail("cannot stage secrets");
            }
            stripped_string(&out)
        }
    };
    if tmpd.is_empty() || fs::metadata(&tmpd).map(|m| !m.is_dir()).unwrap_or(true) {
        return fail("cannot stage secrets");
    }
    cleanup.push(PathBuf::from(&tmpd));
    run("chmod", &["0700", &tmpd])?;
    let staged_passphrase = format!("{tmpd}/passphrase");
    stage_file(&staged_passphrase, random_hex_passphrase()?.as_bytes())?;
    run("chmod", &["0600", &staged_passphrase])?;
    for role in ["artifact", "candidate", "preview", "stable"] {
        let output_prefix = format!("{tmpd}/{role}");
        run_stdout_null(
            "skopeo",
            &[
                "generate-sigstore-key",
                "--output-prefix",
                &output_prefix,
                "--passphrase-file",
                &staged_passphrase,
            ],
        )?;
    }
    let now = unix_now()?;
    let prefix = env_or("SODA_REPOSITORY_PREFIX", PREFIX_DEFAULT);
    let pubs = [
        read_staged(&format!("{tmpd}/artifact.pub"))?,
        read_staged(&format!("{tmpd}/candidate.pub"))?,
        read_staged(&format!("{tmpd}/preview.pub"))?,
        read_staged(&format!("{tmpd}/stable.pub"))?,
    ];
    stage_file(
        &format!("{tmpd}/trust.json"),
        trust_json(&prefix, now, [&pubs[0], &pubs[1], &pubs[2], &pubs[3]]).as_bytes(),
    )?;
    stage_file(&format!("{tmpd}/config.json"), config_json().as_bytes())?;
    let trust_path = format!("{AUTHORITY}/trust.json");
    let artifact_path = format!("{AUTHORITY}/artifact.private");
    let passphrase_path = format!("{AUTHORITY}/passphrase");
    let config_path = format!("{AUTHORITY}/config.json");
    for (staged, target) in [
        (format!("{tmpd}/trust.json"), &trust_path),
        (format!("{tmpd}/artifact.private"), &artifact_path),
        (format!("{tmpd}/passphrase"), &passphrase_path),
        (format!("{tmpd}/config.json"), &config_path),
    ] {
        run(
            "sudo",
            &["install", "-m", "0600", &staged, &format!("{target}.new")],
        )?;
        run("sudo", &["mv", &format!("{target}.new"), target])?;
    }
    let owned = format!("{WORKER_USER}:{WORKER_USER}");
    run(
        "sudo",
        &[
            "chown",
            &owned,
            AUTHORITY,
            &trust_path,
            &artifact_path,
            &passphrase_path,
            &config_path,
        ],
    )?;
    // Remove staging before the final line, not only at process exit: the
    // script's trap on EXIT also runs when the shell itself dies on SIGPIPE,
    // so a closed stdout at this print must not leave secrets in /tmp.
    let _ = fs::remove_dir_all(&tmpd);
    println!("fixture authority rotated; old fixture signatures no longer verify.");
    Ok(())
}
