// soda-candidate-setup prepares this machine so `sudo soda-candidate`
// runs without a coding agent: wrapper on sudo's PATH, admitted controller,
// worker directories, restricted worker config, and a fixture-only media
// authority. Development and fixture scope only: it never creates
// qualification or signing configs, and the generated keys must never
// stand in for release keys.
//
// Rust port of scripts/setup-soda-candidate.sh (plus the
// scripts/candidate-storage.sh defaults it sourced). Messages, exit codes,
// installed paths, file modes, and generated file bytes match the shell.
// Like the script, this binary takes no arguments and ignores any it is
// given. Run from the repository root.

use std::env;
use std::ffi::CString;
use std::fs;
use std::io::{self, Read, Write};
use std::os::unix::ffi::OsStrExt;
use std::os::unix::fs::OpenOptionsExt;
use std::os::unix::process::ExitStatusExt;
use std::path::{Path, PathBuf};
use std::process::{Command, ExitStatus, Stdio};

const FAIL_PREFIX: &str = "setup-soda-candidate";
const PREFIX_DEFAULT: &str = "ghcr.io/levitateos/sodaos";
const STORAGE_ROOT_DEFAULT: &str = "/home/soda-candidate";
const ROOTFS_DIR: &str = "/home/soda-rootfs";
const ADMITTED: &str = "/usr/local/lib/soda/soda-build";
const PINNED_GO: &str = "/usr/local/lib/soda/pinned-go";
const WRAPPER: &str = "/usr/sbin/soda-candidate";
const TOOLS: &str = "/var/lib/soda-candidate-tools";
const AUTHORITY: &str = "/var/lib/soda-candidate-authority";
const LEGACY_HOME: &str = "/var/lib/soda-candidate-home";
const LEGACY_RUN: &str = "/var/lib/soda-candidate-run";
const WORKER_USER: &str = "soda-build-worker";

extern "C" {
    fn flock(fd: i32, op: i32) -> i32;
    fn faccessat(dirfd: i32, path: *const i8, mode: i32, flags: i32) -> i32;
    fn umask(mask: u32) -> u32;
    fn sigaction(signum: i32, act: *const Sigaction, oldact: *mut Sigaction) -> i32;
}

/// Matches glibc's `struct sigaction` on Linux (handler, signal mask, flags,
/// restorer). The glibc wrapper fills in the restorer itself.
#[repr(C)]
#[derive(Clone, Copy)]
struct Sigaction {
    handler: usize,
    mask: [u64; 16],
    flags: i32,
    restorer: usize,
}

const SIGPIPE: i32 = 13;
const SIG_DFL: usize = 0;
const LOCK_EX: i32 = 2;
const LOCK_NB: i32 = 4;
const AT_FDCWD: i32 = -100;
const X_OK: i32 = 1;

/// How the process ends: a `fail()` message with exit 1, or a propagated
/// child status with no extra output (the script's `set -e` behavior).
#[derive(Debug)]
enum Exit {
    Fail(String),
    Propagate(i32),
}

fn fail<T>(msg: impl Into<String>) -> Result<T, Exit> {
    Err(Exit::Fail(msg.into()))
}

fn status_code(status: ExitStatus) -> i32 {
    if let Some(code) = status.code() {
        code
    } else if let Some(sig) = status.signal() {
        128 + sig
    } else {
        1
    }
}

/// `flush_stdout` keeps parent/child output ordered on pipes: the shell's
/// echoes are unbuffered, so flush before every child spawn.
fn flush_stdout() {
    let _ = io::stdout().flush();
}

/// `env_or` mirrors `${VAR:-default}`: unset or empty falls back.
fn env_or(key: &str, default: &str) -> String {
    match env::var(key) {
        Ok(v) if !v.is_empty() => v,
        _ => default.to_string(),
    }
}

/// `current_pwd` mirrors `$PWD`: the inherited logical path when present,
// otherwise the physical working directory.
fn current_pwd() -> String {
    match env::var("PWD") {
        Ok(v) => v,
        Err(_) => env::current_dir()
            .map(|p| p.to_string_lossy().into_owned())
            .unwrap_or_default(),
    }
}

fn is_dir(path: &str) -> bool {
    fs::metadata(path).map(|m| m.is_dir()).unwrap_or(false)
}

fn is_file(path: &str) -> bool {
    fs::metadata(path).map(|m| m.is_file()).unwrap_or(false)
}

/// `stripped` mirrors command substitution: all trailing newlines removed.
fn stripped(out: &[u8]) -> &[u8] {
    let mut end = out.len();
    while end > 0 && out[end - 1] == b'\n' {
        end -= 1;
    }
    &out[..end]
}

fn stripped_string(out: &[u8]) -> String {
    String::from_utf8_lossy(stripped(out)).into_owned()
}

fn current_umask() -> u32 {
    unsafe {
        let mask = umask(0);
        umask(mask);
        mask
    }
}

/// `write_staged` mirrors shell redirection into staging: bytes exact, mode
/// 0666 filtered by the operator umask like `>` and `open("w")`.
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

fn spawn_diag(err: &io::Error) -> (String, i32) {
    match err.kind() {
        io::ErrorKind::NotFound => ("command not found".to_string(), 127),
        io::ErrorKind::PermissionDenied => ("Permission denied".to_string(), 126),
        _ => (err.to_string(), 1),
    }
}

enum Captured {
    SpawnFailed(i32),
    Done(i32, Vec<u8>),
}

/// `capture` runs a command with piped stdout and inherited stderr, like
/// `$(...)`. A spawn failure prints the diagnostic immediately and the
/// caller decides: test position uses the empty substitution, bare position
/// propagates the code (the script's `set -e` behavior).
fn capture(prog: &str, args: &[&str], stderr_null: bool) -> Captured {
    flush_stdout();
    let mut cmd = Command::new(prog);
    cmd.args(args).stdout(Stdio::piped());
    if stderr_null {
        cmd.stderr(Stdio::null());
    }
    let mut child = match cmd.spawn() {
        Ok(child) => child,
        Err(err) => {
            let (msg, code) = spawn_diag(&err);
            // A suppressed child stderr (`2>/dev/null`) also swallows the
            // shell's own "command not found"; stay silent the same way.
            if !stderr_null {
                eprintln!("{FAIL_PREFIX}: {prog}: {msg}");
            }
            return Captured::SpawnFailed(code);
        }
    };
    let mut out = Vec::new();
    if let Some(mut reader) = child.stdout.take() {
        let _ = reader.read_to_end(&mut out);
    }
    let code = child.wait().map(status_code).unwrap_or(1);
    Captured::Done(code, out)
}

/// `run` mirrors a bare command: inherited stdio, propagated status.
fn run(prog: &str, args: &[&str]) -> Result<(), Exit> {
    run_with_io(prog, args, false, false, None, None)
}

/// `run_stdout_null` mirrors `... >/dev/null`.
fn run_stdout_null(prog: &str, args: &[&str]) -> Result<(), Exit> {
    run_with_io(prog, args, true, false, None, None)
}

/// `run_stderr_null` mirrors `... 2>/dev/null`.
fn run_stderr_null(prog: &str, args: &[&str]) -> Result<(), Exit> {
    run_with_io(prog, args, false, true, None, None)
}

fn run_with_io(
    prog: &str,
    args: &[&str],
    stdout_null: bool,
    stderr_null: bool,
    stdin_bytes: Option<&[u8]>,
    cwd: Option<&str>,
) -> Result<(), Exit> {
    flush_stdout();
    let mut cmd = Command::new(prog);
    cmd.args(args);
    if stdout_null {
        cmd.stdout(Stdio::null());
    }
    if stderr_null {
        cmd.stderr(Stdio::null());
    }
    if stdin_bytes.is_some() {
        cmd.stdin(Stdio::piped());
    }
    if let Some(dir) = cwd {
        cmd.current_dir(dir);
    }
    let mut child = match cmd.spawn() {
        Ok(child) => child,
        Err(err) => {
            let (msg, code) = spawn_diag(&err);
            // A suppressed child stderr (`2>/dev/null`) also swallows the
            // shell's own "command not found"; stay silent the same way.
            if !stderr_null {
                eprintln!("{FAIL_PREFIX}: {prog}: {msg}");
            }
            return Err(Exit::Propagate(code));
        }
    };
    let mut write_failed = false;
    if let (Some(input), Some(mut stdin)) = (stdin_bytes, child.stdin.take()) {
        if stdin.write_all(input).is_err() {
            write_failed = true;
        }
    }
    let mut code = child.wait().map(status_code).unwrap_or(1);
    if write_failed && code == 0 {
        code = 1;
    }
    if code == 0 {
        Ok(())
    } else {
        Err(Exit::Propagate(code))
    }
}

/// `run_piped_stdin` mirrors `printf ... | sudo tee ... >/dev/null`: exact
/// input bytes on stdin, stdout discarded, stderr inherited.
fn run_piped_stdin(prog: &str, args: &[&str], input: &[u8]) -> Result<(), Exit> {
    run_with_io(prog, args, true, false, Some(input), None)
}

/// `pipe2` mirrors `first | second` under `pipefail`: the exit status is
/// the last non-zero status by pipeline position.
fn pipe2(first: (&str, &[&str]), second: (&str, &[&str])) -> (i32, Vec<u8>) {
    flush_stdout();
    let mut first_cmd = Command::new(first.0);
    first_cmd.args(first.1).stdout(Stdio::piped());
    let mut first_child = match first_cmd.spawn() {
        Ok(child) => child,
        Err(err) => {
            let (msg, code) = spawn_diag(&err);
            eprintln!("{FAIL_PREFIX}: {}: {msg}", first.0);
            // The second stage still runs on empty input; its output is
            // discarded and the pipeline reports the first failure.
            let _ = capture(second.0, second.1, false);
            return (code, Vec::new());
        }
    };
    let mut second_cmd = Command::new(second.0);
    second_cmd.args(second.1).stdout(Stdio::piped());
    if let Some(stdout) = first_child.stdout.take() {
        second_cmd.stdin(stdout);
    }
    let mut second_child = match second_cmd.spawn() {
        Ok(child) => child,
        Err(err) => {
            let (msg, code) = spawn_diag(&err);
            eprintln!("{FAIL_PREFIX}: {}: {msg}", second.0);
            let _ = first_child.wait();
            return (code, Vec::new());
        }
    };
    let mut out = Vec::new();
    if let Some(mut reader) = second_child.stdout.take() {
        let _ = reader.read_to_end(&mut out);
    }
    let first_code = first_child.wait().map(status_code).unwrap_or(1);
    let second_code = second_child.wait().map(status_code).unwrap_or(1);
    let code = if second_code != 0 {
        second_code
    } else {
        first_code
    };
    (code, out)
}

/// `ls_nonempty` mirrors `[ -n "$(ls -A dir 2>/dev/null)" ]`, including the
/// empty treatment when `ls` itself cannot run.
fn ls_nonempty(path: &str) -> bool {
    match capture("ls", &["-A", path], true) {
        Captured::SpawnFailed(_) => false,
        Captured::Done(_, out) => !stripped(&out).is_empty(),
    }
}

/// `id_un` mirrors `$(id -un)` inside `local`: any failure yields the empty
/// string and execution continues.
fn id_un() -> Vec<u8> {
    match capture("id", &["-un"], false) {
        Captured::SpawnFailed(_) => Vec::new(),
        Captured::Done(_, out) => stripped(&out).to_vec(),
    }
}

/// `json_escape` mirrors Python `json.dumps` with `ensure_ascii`: the
/// script generated every JSON file through it, so the bytes stay identical.
fn json_escape(value: &str) -> String {
    let mut escaped = String::with_capacity(value.len() + 2);
    for ch in value.chars() {
        match ch {
            '"' => escaped.push_str("\\\""),
            '\\' => escaped.push_str("\\\\"),
            '\n' => escaped.push_str("\\n"),
            '\r' => escaped.push_str("\\r"),
            '\t' => escaped.push_str("\\t"),
            '\u{08}' => escaped.push_str("\\b"),
            '\u{0c}' => escaped.push_str("\\f"),
            _ if (ch < '\u{20}' || ch == '\u{7f}') => {
                escaped.push_str(&format!("\\u{:04x}", ch as u32));
            }
            _ if ch > '\u{7e}' => {
                let mut encoded = [0u16; 2];
                for unit in ch.encode_utf16(&mut encoded) {
                    escaped.push_str(&format!("\\u{unit:04x}"));
                }
            }
            _ => escaped.push(ch),
        }
    }
    escaped
}

fn json_field(key: &str, value: &str, indent: usize) -> String {
    format!(
        "{:indent$}\"{}\": \"{}\"",
        "",
        key,
        json_escape(value),
        indent = indent
    )
}

fn json_number_field(key: &str, value: u64, indent: usize) -> String {
    format!("{:indent$}\"{}\": {}", "", key, value, indent = indent)
}

/// `worker_json` mirrors the `json.dump(..., indent=2)` for worker.json,
/// including the absent trailing newline.
#[allow(clippy::too_many_arguments)]
fn worker_json(
    executable: &str,
    source: &str,
    forgejo_source: &str,
    output_parent: &str,
    storage_root: &str,
    build_home: &str,
    runtime: &str,
    tools: &str,
    authority: &str,
) -> String {
    [
        "{".to_string(),
        json_field("Executable", executable, 2) + ",",
        json_field("Source", source, 2) + ",",
        json_field("ForgejoSource", forgejo_source, 2) + ",",
        json_field("OutputParent", output_parent, 2) + ",",
        json_field("StorageRoot", storage_root, 2) + ",",
        json_field("BuildHome", build_home, 2) + ",",
        json_field("Runtime", runtime, 2) + ",",
        json_field("Tools", tools, 2) + ",",
        json_field("MediaAuthorityDirectory", authority, 2),
        "}".to_string(),
    ]
    .join("\n")
}

/// `trust_json` mirrors the `json.dumps(..., indent=2) + "\n"` trust file.
fn trust_json(prefix: &str, now: u64, pubs: [&str; 4]) -> String {
    let roles = ["artifact", "candidate", "preview", "stable"];
    let mut lines = vec![
        "{".to_string(),
        json_number_field("Format", 1, 2) + ",",
        json_field("Prefix", prefix, 2) + ",",
        json_number_field("Epoch", 1, 2) + ",",
        "  \"Keys\": {".to_string(),
    ];
    for (index, (role, key)) in roles.iter().zip(pubs.iter()).enumerate() {
        let comma = if index + 1 < roles.len() { "," } else { "" };
        lines.push(format!("    \"{role}\": ["));
        lines.push(format!("      \"{}\"", json_escape(key)));
        lines.push(format!("    ]{comma}"));
    }
    lines.push("  },".to_string());
    lines.push(json_number_field("NotBefore", now - 600, 2) + ",");
    lines.push(json_number_field("MaxAgeSeconds", 3600, 2) + ",");
    lines.push(json_number_field("ClockSkewSeconds", 10, 2) + ",");
    lines.push("  \"MinimumSequence\": {".to_string());
    lines.push(json_number_field("candidate", 1, 4) + ",");
    lines.push(json_number_field("preview", 1, 4) + ",");
    lines.push(json_number_field("stable", 1, 4));
    lines.push("  }".to_string());
    lines.push("}".to_string());
    lines.join("\n") + "\n"
}

/// `config_json` mirrors the `json.dumps(..., indent=2) + "\n"` config file.
fn config_json() -> String {
    [
        "{".to_string(),
        json_field("Trust", "/run/soda-media-authority/trust.json", 2) + ",",
        "  \"Keys\": {".to_string(),
        json_field("Key", "/run/soda-media-authority/artifact.private", 4) + ",",
        json_field("Passphrase", "/run/soda-media-authority/passphrase", 4),
        "  }".to_string(),
        "}".to_string(),
    ]
    .join("\n")
        + "\n"
}

/// `units_have_active` mirrors the `awk` over `systemctl list-units`: any
/// unit whose third field is active, activating, or deactivating blocks.
fn units_have_active(output: &[u8]) -> bool {
    let text = String::from_utf8_lossy(output);
    text.lines().any(|line| {
        matches!(
            line.split_whitespace().nth(2),
            Some("active" | "activating" | "deactivating")
        )
    })
}

/// `bridge_ip` mirrors the `ip ... | awk ... | cut ... | head -1` pickup:
/// the first line's fourth field before any `/`.
fn bridge_ip(ip_output: &[u8]) -> Option<String> {
    let text = String::from_utf8_lossy(ip_output);
    let line = text.lines().next()?;
    let field = line.split_whitespace().nth(3)?;
    let addr = field.split('/').next()?;
    if addr.is_empty() {
        None
    } else {
        Some(addr.to_string())
    }
}

fn refuse_active_build() -> Result<(), Exit> {
    let args = [
        "list-units",
        "--all",
        "--type=service",
        "--no-legend",
        "--plain",
        "soda-build-*",
    ];
    match capture("systemctl", &args, true) {
        Captured::SpawnFailed(_) => {
            fail("cannot list worker units; refusing to touch shared build state")
        }
        Captured::Done(code, out) => {
            if code != 0 {
                return fail("cannot list worker units; refusing to touch shared build state");
            }
            if units_have_active(&out) {
                return fail("a candidate build is still active; finish it before rerunning setup");
            }
            Ok(())
        }
    }
}

struct Storage {
    root: String,
    home: String,
    run: String,
    scratch: String,
}

fn migrate_candidate_home(storage: &Storage) -> Result<(), Exit> {
    if !is_dir(LEGACY_HOME) {
        return Ok(());
    }
    if ls_nonempty(&storage.home) {
        println!("-- new worker home already populated; legacy {LEGACY_HOME} preserved untouched");
        return Ok(());
    }
    refuse_active_build()?;
    if command_v("pgrep").is_some() {
        // Both streams silenced like `>/dev/null 2>&1`; captured stdout is
        // discarded and only the exit status decides.
        let runs = match capture("pgrep", &["-u", WORKER_USER], true) {
            Captured::SpawnFailed(_) => false,
            Captured::Done(code, _) => code == 0,
        };
        if runs {
            return fail(
                "soda-build-worker still owns processes; finish them before migrating heavy state",
            );
        }
    }
    println!(
        "-- migrating legacy worker home to {} (legacy preserved)",
        storage.home
    );
    let from = format!("{LEGACY_HOME}/.");
    run("sudo", &["cp", "-a", &from, &storage.home])?;
    let owned = format!("{WORKER_USER}:{WORKER_USER}");
    run("sudo", &["chown", "-R", &owned, &storage.home])?;
    println!("-- legacy {LEGACY_HOME} preserved; retire it explicitly (D2) after the new home proves itself");
    Ok(())
}

/// `fcontext_add_or_modify` mirrors `semanage fcontext -a ... || semanage
/// fcontext -m ...`: add the entry, or modify it when it already exists.
fn fcontext_add_or_modify(file_type: &str, pattern: &str) -> Result<(), Exit> {
    if run_stderr_null(
        "sudo",
        &["semanage", "fcontext", "-a", "-t", file_type, pattern],
    )
    .is_err()
    {
        run(
            "sudo",
            &["semanage", "fcontext", "-m", "-t", file_type, pattern],
        )?;
    }
    Ok(())
}

/// `selinux_has_type` mirrors `sudo stat -c %C path | grep -q ":type:"`.
fn selinux_has_type(path: &str, marker: &str) -> bool {
    match capture("sudo", &["stat", "-c", "%C", path], false) {
        Captured::SpawnFailed(_) => false,
        Captured::Done(code, out) => code == 0 && String::from_utf8_lossy(&out).contains(marker),
    }
}

/// `run_in_dir` mirrors `(cd dir && cmd ...)`: a missing directory fails
/// like `cd`, anything else runs the command there.
fn run_in_dir(prog: &str, args: &[&str], dir: &str, stdout_null: bool) -> Result<(), Exit> {
    if !is_dir(dir) {
        eprintln!("{FAIL_PREFIX}: cd: {dir}: No such file or directory");
        return Err(Exit::Propagate(1));
    }
    run_with_io(prog, args, stdout_null, false, None, Some(dir))
}

fn run_setup(cleanup: &mut Vec<PathBuf>) -> Result<(), Exit> {
    // Like the script, which parsed none, all arguments are ignored.
    let prefix = env_or("SODA_REPOSITORY_PREFIX", PREFIX_DEFAULT);
    let refresh = env_or("SODA_REFRESH_AUTHORITY", "0");
    let forgejo_source = env_or("SODA_FORGEJO_SOURCE", "");
    let storage_root = env_or("SODA_CANDIDATE_ROOT", STORAGE_ROOT_DEFAULT);
    let storage = Storage {
        home: env_or("SODA_CANDIDATE_HOME", &format!("{storage_root}/home")),
        run: env_or("SODA_CANDIDATE_RUN", &format!("{storage_root}/run")),
        scratch: env_or("SODA_CANDIDATE_SCRATCH", &format!("{storage_root}/scratch")),
        root: storage_root,
    };

    let mut rootfs_url = String::new();
    if let Captured::Done(code, out) = capture("ip", &["-4", "-o", "addr", "show", "virbr0"], true)
    {
        if code == 0 {
            if let Some(addr) = bridge_ip(&out) {
                rootfs_url = format!("http://{addr}:8080");
            }
        }
    }

    let pwd = current_pwd();
    let output_parent = format!("{pwd}/.artifacts/releases/isolated");
    let worker_json_path = format!("{AUTHORITY}/worker.json");

    if !is_file("go.mod") {
        return fail("run from the repository root");
    }
    if forgejo_source.is_empty() {
        return fail("set SODA_FORGEJO_SOURCE to the clean canonical Forgejo fork checkout");
    }
    let forgejo_git = format!("{forgejo_source}/.git");
    let canonical_ok = is_dir(&forgejo_git)
        && match capture("realpath", &[forgejo_source.as_str()], false) {
            Captured::SpawnFailed(_) => false,
            Captured::Done(code, out) => code == 0 && stripped(&out) == forgejo_source.as_bytes(),
        };
    if !canonical_ok {
        return fail("canonical Forgejo checkout required");
    }
    let safe_dir = format!("safe.directory={forgejo_source}");
    let forgejo_clean = match capture(
        "git",
        &[
            "-c",
            &safe_dir,
            "-C",
            &forgejo_source,
            "status",
            "--porcelain",
            "--untracked-files=normal",
        ],
        false,
    ) {
        Captured::SpawnFailed(_) => true,
        Captured::Done(_, out) => stripped(&out).is_empty(),
    };
    if !forgejo_clean {
        return fail("Forgejo source must be clean and committed");
    }
    let arch_ok = match capture("uname", &["-m"], false) {
        Captured::SpawnFailed(_) => false,
        Captured::Done(_, out) => stripped(&out) == b"x86_64",
    };
    if !arch_ok {
        return fail("matching native x86_64 required on this host");
    }
    for tool in ["go", "bun", "podman", "skopeo", "python3", "flock"] {
        if command_v(tool).is_none() {
            return fail(format!(
                "pinned go, bun, podman, skopeo, python3 and flock required (missing {tool})"
            ));
        }
    }
    let user_ok = match capture("id", &[WORKER_USER], true) {
        Captured::SpawnFailed(_) => false,
        Captured::Done(code, _) => code == 0,
    };
    if !user_ok {
        return fail("soda-build-worker user missing");
    }
    let dirty = match capture(
        "git",
        &["status", "--porcelain", "--untracked-files=no"],
        false,
    ) {
        Captured::SpawnFailed(_) => false,
        Captured::Done(_, out) => !stripped(&out).is_empty(),
    };
    if dirty {
        return fail("commit or stash tracked changes first; the controller refuses dirty source");
    }
    let (pin_code, pin_out) = pipe2(("grep", &["^go ", "go.mod"]), ("awk", &["{print $2}"]));
    if pin_code != 0 {
        return Err(Exit::Propagate(pin_code));
    }
    let pinned = stripped_string(&pin_out);
    if pinned.is_empty() {
        return fail("go.mod pins no Go version");
    }
    env::set_var("GOTOOLCHAIN", format!("go{pinned}"));
    if run_stdout_null("go", &["version"]).is_err() {
        return fail(format!("cannot fetch Go {pinned}"));
    }
    let pinned_goroot = match capture("go", &["env", "GOROOT"], false) {
        Captured::SpawnFailed(code) => return Err(Exit::Propagate(code)),
        Captured::Done(code, out) => {
            if code != 0 {
                return Err(Exit::Propagate(code));
            }
            stripped_string(&out)
        }
    };
    let want = format!("go version go{pinned} linux/amd64");

    let user = String::from_utf8_lossy(&id_un()).into_owned();
    let lock_path = format!("/tmp/soda-setup-{user}.lock");
    let lease = match fs::OpenOptions::new()
        .write(true)
        .create(true)
        .truncate(true)
        .open(&lock_path)
    {
        Ok(file) => file,
        Err(err) => {
            eprintln!("{FAIL_PREFIX}: {lock_path}: {err}");
            return fail(format!("cannot open setup lease {lock_path}"));
        }
    };
    {
        use std::os::unix::io::AsRawFd;
        if unsafe { flock(lease.as_raw_fd(), LOCK_EX | LOCK_NB) } != 0 {
            return fail("another setup is already running for this operator");
        }
    }
    refuse_active_build()?;

    println!("-- candidate storage root ({})", storage.root);
    run("sudo", &["mkdir", "-p", &storage.scratch])?;
    let scratch_owner = match env::var("SUDO_USER") {
        Ok(owner) if !owner.is_empty() => owner,
        _ => String::from_utf8_lossy(&id_un()).into_owned(),
    };
    run("sudo", &["chown", &scratch_owner, &storage.scratch])?;

    println!("-- build tools from committed source");
    let bindir_template = format!("{}/setup-bindir.XXXXXXXX", storage.scratch);
    let bindir = match capture("mktemp", &["-d", &bindir_template], false) {
        Captured::SpawnFailed(code) => return Err(Exit::Propagate(code)),
        Captured::Done(code, out) => {
            if code != 0 {
                return Err(Exit::Propagate(code));
            }
            stripped_string(&out)
        }
    };
    cleanup.push(PathBuf::from(&bindir));
    let soda_build = format!("{bindir}/soda-build");
    let soda_candidate = format!("{bindir}/soda-candidate");
    run("go", &["build", "-o", &soda_build, "./tools/soda-build"])?;
    run(
        "go",
        &["build", "-o", &soda_candidate, "./tools/soda-candidate"],
    )?;
    let stamp_ok = match capture("go", &["version", &soda_build], false) {
        Captured::SpawnFailed(_) => false,
        Captured::Done(_, out) => stripped(&out) == format!("{soda_build}: go{pinned}").as_bytes(),
    };
    if !stamp_ok {
        return fail(format!(
            "controller stamp is not Go {pinned}; refusing to admit it"
        ));
    }

    println!("-- install wrapper and admitted controller");
    run("sudo", &["install", "-m", "0755", &soda_candidate, WRAPPER])?;
    run(
        "sudo",
        &["install", "-D", "-m", "0755", &soda_build, ADMITTED],
    )?;
    let which_out = match capture("sudo", &["which", "soda-candidate"], false) {
        Captured::SpawnFailed(_) => Vec::new(),
        Captured::Done(_, out) => stripped(&out).to_vec(),
    };
    let which_str = String::from_utf8_lossy(&which_out);
    let got_wrapper = match capture("sudo", &["readlink", "-f", which_str.as_ref()], false) {
        Captured::SpawnFailed(code) => return Err(Exit::Propagate(code)),
        Captured::Done(code, out) => {
            if code != 0 {
                return Err(Exit::Propagate(code));
            }
            stripped(&out).to_vec()
        }
    };
    let want_wrapper = match capture("readlink", &["-f", WRAPPER], false) {
        Captured::SpawnFailed(code) => return Err(Exit::Propagate(code)),
        Captured::Done(code, out) => {
            if code != 0 {
                return Err(Exit::Propagate(code));
            }
            stripped(&out).to_vec()
        }
    };
    if got_wrapper != want_wrapper {
        return fail(format!(
            "wrapper not visible on sudo secure_path (got {})",
            String::from_utf8_lossy(&got_wrapper)
        ));
    }

    println!("-- worker directories");
    let tools_bin = format!("{TOOLS}/bin");
    run(
        "sudo",
        &[
            "mkdir",
            "-p",
            &output_parent,
            &storage.home,
            &storage.run,
            &tools_bin,
            AUTHORITY,
        ],
    )?;
    migrate_candidate_home(&storage)?;
    if ls_nonempty(LEGACY_RUN) {
        println!(
            "-- legacy {LEGACY_RUN} holds leftovers; preserved untouched (new runs use {})",
            storage.run
        );
    }
    let pinned_new = format!("{PINNED_GO}.new");
    let bun_new = format!("{tools_bin}/bun.new");
    run("sudo", &["rm", "-rf", &pinned_new, &bun_new])?;
    run("sudo", &["cp", "-a", &pinned_goroot, &pinned_new])?;
    let staged_go = format!("{pinned_new}/bin/go");
    let staged_ok = match capture(&staged_go, &["version"], false) {
        Captured::SpawnFailed(_) => false,
        Captured::Done(_, out) => stripped(&out) == want.as_bytes(),
    };
    if !staged_ok {
        return fail(format!("staged Go is not {pinned}; refusing to publish it"));
    }
    let bun_path = command_v("bun").map(|p| p.to_string_lossy().into_owned());
    let bun_src = bun_path.as_deref().unwrap_or("");
    run("sudo", &["cp", bun_src, &bun_new])?;
    if run_stdout_null("sudo", &["-u", WORKER_USER, &bun_new, "--version"]).is_err() {
        return fail("staged bun is not worker-runnable; refusing to publish it");
    }
    refuse_active_build()?;
    let tools_go = format!("{TOOLS}/go");
    run("sudo", &["rm", "-rf", &tools_go, PINNED_GO])?;
    run("sudo", &["mv", &pinned_new, PINNED_GO])?;
    let bun_final = format!("{tools_bin}/bun");
    run("sudo", &["mv", &bun_new, &bun_final])?;
    run("sudo", &["chown", "-R", "root:root", PINNED_GO, TOOLS])?;
    if command_v("semanage").is_some() {
        let pattern = format!("{TOOLS}(/.*)?");
        fcontext_add_or_modify("bin_t", &pattern)?;
    }
    if command_v("restorecon").is_some() {
        run("sudo", &["restorecon", "-R", PINNED_GO, TOOLS])?;
    }
    run(
        "sudo",
        &[
            "find", PINNED_GO, "-type", "d", "-exec", "chmod", "0755", "{}", "+",
        ],
    )?;
    run(
        "sudo",
        &[
            "find", PINNED_GO, "-type", "f", "-exec", "chmod", "a+r", "{}", "+",
        ],
    )?;
    if !selinux_has_type(&format!("{PINNED_GO}/bin/go"), ":lib_t:") {
        return fail("pinned GOROOT is not lib_t; the sandboxed worker could not execute it");
    }
    let owned = format!("{WORKER_USER}:{WORKER_USER}");
    run(
        "sudo",
        &["chown", &owned, &output_parent, &storage.home, &storage.run],
    )?;
    if command_v("semanage").is_some() {
        let pattern = format!("{}(/.*)?", storage.root);
        fcontext_add_or_modify("var_lib_t", &pattern)?;
    }
    if command_v("restorecon").is_some() {
        run("sudo", &["restorecon", &storage.root])?;
    }
    if command_v("semanage").is_some() {
        let pattern = format!("{output_parent}(/.*)?");
        fcontext_add_or_modify("var_lib_t", &pattern)?;
    }
    if command_v("restorecon").is_some() {
        run("sudo", &["restorecon", "-R", &output_parent])?;
    }
    run("sudo", &["chmod", "0755", TOOLS, &tools_bin])?;
    run("sudo", &["chmod", "0700", AUTHORITY])?;

    println!("-- verify tools as the worker user");
    let verify_ok = match capture(
        "sudo",
        &[
            "-u",
            WORKER_USER,
            "env",
            "GOTOOLCHAIN=local",
            &format!("HOME={}", storage.home),
            &format!("{PINNED_GO}/bin/go"),
            "version",
        ],
        false,
    ) {
        Captured::SpawnFailed(_) => false,
        Captured::Done(_, out) => stripped(&out) == want.as_bytes(),
    };
    if !verify_ok {
        return fail(format!(
            "provisioned Go is not {pinned} or not worker-runnable"
        ));
    }
    if run(
        "sudo",
        &[
            "-u",
            WORKER_USER,
            "test",
            "-r",
            &format!("{PINNED_GO}/src/net/textproto/header.go"),
        ],
    )
    .is_err()
    {
        return fail("provisioned GOROOT sources are not worker-readable");
    }
    if run_stdout_null("sudo", &["-u", WORKER_USER, &bun_final, "--version"]).is_err() {
        return fail("provisioned bun is not worker-runnable");
    }

    println!("-- warm worker caches (the isolated worker has no network)");
    let go_mod = format!("{}/go-mod", storage.home);
    let go_build_cache = format!("{}/go-build", storage.home);
    run("sudo", &["mkdir", "-p", &go_mod, &go_build_cache])?;
    let toolchain = format!("go{pinned}");
    let pinned_go_bin = format!("{PINNED_GO}/bin/go");
    let home_env = format!("HOME={}", storage.home);
    let modcache_env = format!("GOMODCACHE={go_mod}");
    let gocache_env = format!("GOCACHE={go_build_cache}");
    let toolchain_env = format!("GOTOOLCHAIN={toolchain}");
    let warm_env = [
        home_env.as_str(),
        "GOPROXY=https://proxy.golang.org,direct",
        "GOSUMDB=sum.golang.org",
        modcache_env.as_str(),
        gocache_env.as_str(),
        toolchain_env.as_str(),
        "GOFLAGS=-mod=readonly",
        "CGO_ENABLED=0",
    ];
    // `sudo env ... go build ./...`: argv assembled exactly like the script.
    let mut build_args: Vec<&str> = vec!["env"];
    build_args.extend(warm_env.iter().copied());
    build_args.push(&pinned_go_bin);
    build_args.push("build");
    build_args.push("./...");
    if run("sudo", &build_args).is_err() {
        return fail("cannot warm Go module cache");
    }
    let mut download_args: Vec<&str> = vec!["env"];
    download_args.extend(warm_env.iter().copied());
    download_args.push(&pinned_go_bin);
    download_args.push("mod");
    download_args.push("download");
    download_args.push("all");
    if run("sudo", &download_args).is_err() {
        return fail("cannot warm full Go module set");
    }
    if run(
        "sudo",
        &[
            "-u",
            WORKER_USER,
            "env",
            &format!("HOME={}", storage.home),
            &pinned_go_bin,
            "env",
            "-w",
            "GOPROXY=off",
        ],
    )
    .is_err()
    {
        return fail("cannot lock worker Go offline");
    }
    let bun_template = format!("{}/setup-bun.XXXXXXXX", storage.scratch);
    let tmpw = match capture("mktemp", &["-d", &bun_template], false) {
        Captured::SpawnFailed(code) => return Err(Exit::Propagate(code)),
        Captured::Done(code, out) => {
            if code != 0 {
                return Err(Exit::Propagate(code));
            }
            stripped_string(&out)
        }
    };
    // Like the script's `trap` at this point, the bun scratch tree is only
    // removed by the explicit `sudo rm -rf` after a successful warm: a
    // failure here leaks it, worker-owned, under the scratch root.
    let tmpw_tools = format!("{tmpw}/tools");
    if fs::create_dir_all(&tmpw_tools).is_err() {
        return Err(Exit::Propagate(1));
    }
    run("cp", &["package.json", "bun.lock", "bunfig.toml", &tmpw])?;
    run("cp", &["-a", "tools/lit-check", &tmpw_tools])?;
    run("sudo", &["chown", "-R", &owned, &tmpw])?;
    run(
        "sudo",
        &[
            "find", &tmpw, "-type", "d", "-exec", "chmod", "0755", "{}", "+",
        ],
    )?;
    if run_in_dir(
        "sudo",
        &[
            "-u",
            WORKER_USER,
            "env",
            &format!("HOME={}", storage.home),
            &bun_final,
            "install",
            "--frozen-lockfile",
        ],
        &tmpw,
        true,
    )
    .is_err()
    {
        return fail("cannot warm Bun cache");
    }
    let browsers = format!("{}/browsers", storage.home);
    run(
        "sudo",
        &[
            "install",
            "-d",
            "-o",
            WORKER_USER,
            "-g",
            WORKER_USER,
            &browsers,
        ],
    )?;
    if run_in_dir(
        "sudo",
        &[
            "-u",
            WORKER_USER,
            "env",
            &format!("HOME={}", storage.home),
            &format!("PLAYWRIGHT_BROWSERS_PATH={browsers}"),
            &bun_final,
            "x",
            "playwright",
            "install",
            "chromium",
        ],
        &tmpw,
        false,
    )
    .is_err()
    {
        return fail("cannot stage Playwright chromium");
    }
    run("sudo", &["rm", "-rf", &tmpw])?;
    run("sudo", &["chown", "-R", &owned, &storage.home])?;

    println!("-- worker SELinux policy (process groups plus Go cache mapping)");
    for tool in ["checkmodule", "semodule_package", "semodule"] {
        if command_v(tool).is_none() {
            return fail(format!(
                "policycoreutils tooling required for the worker SELinux module (missing {tool})"
            ));
        }
    }
    let worker_mod = format!("{bindir}/soda-build-worker.mod");
    let worker_pp = format!("{bindir}/soda-build-worker.pp");
    run(
        "checkmodule",
        &[
            "-M",
            "-m",
            "-o",
            &worker_mod,
            "scripts/selinux/soda-build-worker.te",
        ],
    )?;
    run("semodule_package", &["-o", &worker_pp, "-m", &worker_mod])?;
    refuse_active_build()?;
    let _ = run_stderr_null("sudo", &["semodule", "-r", "soda-build-setpgid"]);
    run("sudo", &["semodule", "-i", &worker_pp])?;
    let go_config = format!("{}/.config", storage.home);
    run(
        "sudo",
        &["mkdir", "-p", &go_build_cache, &go_mod, &go_config],
    )?;
    let containers_dir = format!("{go_config}/containers");
    run(
        "sudo",
        &[
            "install",
            "-d",
            "-o",
            WORKER_USER,
            "-g",
            WORKER_USER,
            &containers_dir,
        ],
    )?;
    let containers_conf = format!("{containers_dir}/containers.conf");
    run_piped_stdin(
        "sudo",
        &["tee", &containers_conf],
        b"[engine]\ncgroup_manager = \"cgroupfs\"\n",
    )?;
    run("sudo", &["chown", &owned, &containers_conf])?;
    run("sudo", &["chmod", "0644", &containers_conf])?;
    if command_v("semanage").is_some() {
        for cache in [&go_build_cache, &go_mod, &go_config] {
            let pattern = format!("{cache}(/.*)?");
            fcontext_add_or_modify("soda_build_cache_t", &pattern)?;
        }
        let pattern = format!("{}(/.*)?", storage.run);
        fcontext_add_or_modify("soda_build_runtime_t", &pattern)?;
    }
    if command_v("restorecon").is_some() {
        run(
            "sudo",
            &[
                "restorecon",
                "-R",
                &go_build_cache,
                &go_mod,
                &go_config,
                &storage.run,
            ],
        )?;
    }
    if !selinux_has_type(&go_build_cache, ":soda_build_cache_t:") {
        return fail(
            "worker Go cache is not soda_build_cache_t; the sandboxed worker could not map it",
        );
    }
    if !selinux_has_type(&storage.run, ":soda_build_runtime_t:") {
        return fail(
            "worker runtime is not soda_build_runtime_t; pasta could not use its netns dir",
        );
    }

    println!("-- worker git ownership exception");
    let gitconfig = "/etc/gitconfig";
    let needs_gitconfig = if !is_file(gitconfig) {
        true
    } else {
        match capture(
            "grep",
            &["-qF", "directory = /run/soda-build-source", gitconfig],
            false,
        ) {
            Captured::SpawnFailed(_) => false,
            Captured::Done(code, _) => code == 1,
        }
    };
    if needs_gitconfig {
        run_piped_stdin(
            "sudo",
            &["tee", "-a", gitconfig],
            b"# Soda build worker: the isolated worker sees the canonical checkout\n# only at /run/soda-build-source, owned by the operator. Mark it expected.\n[safe]\n\tdirectory = /run/soda-build-source\n",
        )?;
        run("sudo", &["chmod", "0644", gitconfig])?;
    }

    println!("-- restricted worker config");
    let worker_config = worker_json(
        ADMITTED,
        &pwd,
        &forgejo_source,
        &output_parent,
        &storage.root,
        &storage.home,
        &storage.run,
        TOOLS,
        AUTHORITY,
    );
    run_piped_stdin(
        "sudo",
        &["tee", &worker_json_path],
        worker_config.as_bytes(),
    )?;
    run("sudo", &["chmod", "0600", &worker_json_path])?;

    let trust_path = format!("{AUTHORITY}/trust.json");
    let artifact_path = format!("{AUTHORITY}/artifact.private");
    let passphrase_path = format!("{AUTHORITY}/passphrase");
    let config_path = format!("{AUTHORITY}/config.json");
    if is_file(&artifact_path) && refresh != "1" {
        println!("-- fixture authority already exists; keeping it (SODA_REFRESH_AUTHORITY=1 to regenerate)");
    } else {
        println!("-- fixture-only media authority (never release keys)");
        refuse_active_build()?;
        let authority_template = format!("{}/setup-authority.XXXXXXXX", storage.scratch);
        let tmpd = match capture("mktemp", &["-d", &authority_template], false) {
            Captured::SpawnFailed(code) => return Err(Exit::Propagate(code)),
            Captured::Done(code, out) => {
                if code != 0 {
                    return Err(Exit::Propagate(code));
                }
                stripped_string(&out)
            }
        };
        cleanup.push(PathBuf::from(&tmpd));
        let staged_passphrase = format!("{tmpd}/passphrase");
        stage_file(&staged_passphrase, random_hex_passphrase()?.as_bytes())?;
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
        for name in [
            "trust.json",
            "artifact.private",
            "passphrase",
            "config.json",
        ] {
            run(
                "sudo",
                &[
                    "install",
                    "-m",
                    "0600",
                    &format!("{tmpd}/{name}"),
                    &format!("{AUTHORITY}/{name}"),
                ],
            )?;
        }
        run("sudo", &["chmod", "0700", AUTHORITY])?;
    }
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

    run("sudo", &["mkdir", "-p", ROOTFS_DIR])?;
    let rootfs_owner = String::from_utf8_lossy(&id_un()).into_owned();
    run("sudo", &["chown", &rootfs_owner, ROOTFS_DIR])?;
    if command_v("restorecon").is_some() {
        run("sudo", &["restorecon", ROOTFS_DIR])?;
    }

    println!("-- ready. Development candidate command from {pwd}:");
    println!("  sudo {ADMITTED} --worker-config {worker_json_path} --arch x86_64 \\");
    println!("    --out {output_parent}/manual-01 --development --target candidate \\");
    println!("    --forgejo-source {forgejo_source}");
    println!("-- installer rootfs pickup: {ROOTFS_DIR} (served by soda-rootfs-server.service)");
    println!("  after a media build, copy its hash-named rootfs here:");
    println!("  sudo cp <out>/artifacts/media/*-rootfs.img {ROOTFS_DIR}/ && sudo chown root:root {ROOTFS_DIR}/*-rootfs.img && sudo chmod 0644 {ROOTFS_DIR}/*-rootfs.img");
    if !rootfs_url.is_empty() {
        println!("-- guests fetch the filed rootfs from: {rootfs_url}");
    }
    Ok(())
}

/// `random_hex_passphrase` mirrors `head -c 32 /dev/urandom | od -An -tx1
/// | tr -d ' \n'`: 64 lowercase hex characters, no trailing newline.
fn random_hex_passphrase() -> Result<String, Exit> {
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
/// as setup errors here instead of tracebacks; the file bytes are identical.
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

fn main() {
    // Rust ignores SIGPIPE at startup (a write to a closed pipe would return
    // EPIPE and println! would panic instead of dying like the shell);
    // restore the default disposition for byte parity.
    unsafe {
        let restore_pipe = Sigaction {
            handler: SIG_DFL,
            mask: [0; 16],
            flags: 0,
            restorer: 0,
        };
        sigaction(SIGPIPE, &restore_pipe, std::ptr::null_mut());
    }
    let mut cleanup: Vec<PathBuf> = Vec::new();
    let result = run_setup(&mut cleanup);
    for path in &cleanup {
        let _ = fs::remove_dir_all(path);
    }
    match result {
        Ok(()) => {}
        Err(Exit::Fail(msg)) => {
            eprintln!("{FAIL_PREFIX}: {msg}");
            std::process::exit(1);
        }
        Err(Exit::Propagate(code)) => std::process::exit(code),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::fs::PermissionsExt;

    #[test]
    fn json_escape_matches_python_dumps() {
        let input = "a\"b\\c\nd\re\tf\x08g\x0ch\x01i\x7fj\u{80}ké😀l\x1fm";
        assert_eq!(
            json_escape(input),
            "a\\\"b\\\\c\\nd\\re\\tf\\bg\\fh\\u0001i\\u007fj\\u0080k\\u00e9\\ud83d\\ude00l\\u001fm"
        );
        assert_eq!(json_escape("plain / path-_name.1"), "plain / path-_name.1");
        assert_eq!(json_escape(""), "");
    }

    #[test]
    fn worker_json_matches_python_dump_without_trailing_newline() {
        let got = worker_json(
            "/usr/local/lib/soda/soda-build",
            "/home/op/sodaos",
            "/home/op/forgejo",
            "/home/op/sodaos/.artifacts/releases/isolated",
            "/home/soda-candidate",
            "/home/soda-candidate/home",
            "/home/soda-candidate/run",
            "/var/lib/soda-candidate-tools",
            "/var/lib/soda-candidate-authority",
        );
        let want = "{\n  \"Executable\": \"/usr/local/lib/soda/soda-build\",\n  \"Source\": \"/home/op/sodaos\",\n  \"ForgejoSource\": \"/home/op/forgejo\",\n  \"OutputParent\": \"/home/op/sodaos/.artifacts/releases/isolated\",\n  \"StorageRoot\": \"/home/soda-candidate\",\n  \"BuildHome\": \"/home/soda-candidate/home\",\n  \"Runtime\": \"/home/soda-candidate/run\",\n  \"Tools\": \"/var/lib/soda-candidate-tools\",\n  \"MediaAuthorityDirectory\": \"/var/lib/soda-candidate-authority\"\n}";
        assert_eq!(got, want);
    }

    #[test]
    fn trust_json_matches_python_dump_with_trailing_newline() {
        let got = trust_json(
            "ghcr.io/levitateos/sodaos",
            424242,
            [
                "artifact-pub\n",
                "candidate-pub\n",
                "preview-pub\n",
                "stable-pub\n",
            ],
        );
        let want = "{\n  \"Format\": 1,\n  \"Prefix\": \"ghcr.io/levitateos/sodaos\",\n  \"Epoch\": 1,\n  \"Keys\": {\n    \"artifact\": [\n      \"artifact-pub\\n\"\n    ],\n    \"candidate\": [\n      \"candidate-pub\\n\"\n    ],\n    \"preview\": [\n      \"preview-pub\\n\"\n    ],\n    \"stable\": [\n      \"stable-pub\\n\"\n    ]\n  },\n  \"NotBefore\": 423642,\n  \"MaxAgeSeconds\": 3600,\n  \"ClockSkewSeconds\": 10,\n  \"MinimumSequence\": {\n    \"candidate\": 1,\n    \"preview\": 1,\n    \"stable\": 1\n  }\n}\n";
        assert_eq!(got, want);
    }

    #[test]
    fn config_json_matches_python_dump_with_trailing_newline() {
        let got = config_json();
        let want = "{\n  \"Trust\": \"/run/soda-media-authority/trust.json\",\n  \"Keys\": {\n    \"Key\": \"/run/soda-media-authority/artifact.private\",\n    \"Passphrase\": \"/run/soda-media-authority/passphrase\"\n  }\n}\n";
        assert_eq!(got, want);
    }

    #[test]
    fn units_have_active_matches_awk_third_field() {
        assert!(units_have_active(
            b"soda-build-x.service loaded active running desc\n"
        ));
        assert!(units_have_active(
            b"soda-build-x.service loaded activating start desc\n"
        ));
        assert!(units_have_active(
            b"soda-build-x.service loaded deactivating stop desc\n"
        ));
        assert!(!units_have_active(
            b"soda-build-x.service loaded failed failed desc\n"
        ));
        assert!(!units_have_active(b""));
        assert!(!units_have_active(b"short line\n"));
        assert!(!units_have_active(
            b"other.service loaded inactive dead desc\nsoda-build-y.service loaded failed failed desc\n"
        ));
        assert!(units_have_active(
            b"other.service loaded inactive dead desc\nsoda-build-y.service loaded active running desc\n"
        ));
    }

    #[test]
    fn bridge_ip_takes_first_line_fourth_field_before_slash() {
        assert_eq!(
            bridge_ip(
                b"2: virbr0    inet 192.168.122.1/24 brd 192.168.122.255 scope global virbr0\n"
            ),
            Some("192.168.122.1".to_string())
        );
        assert_eq!(bridge_ip(b""), None);
        assert_eq!(bridge_ip(b"2: virbr0\n"), None);
        assert_eq!(
            bridge_ip(b"2: virbr0    inet 10.0.0.1 scope global\n"),
            Some("10.0.0.1".to_string())
        );
    }

    #[test]
    fn env_or_falls_back_on_unset_or_empty() {
        let key = "SODA_CANDIDATE_SETUP_TEST_ENV_OR";
        env::remove_var(key);
        assert_eq!(env_or(key, "dflt"), "dflt");
        env::set_var(key, "");
        assert_eq!(env_or(key, "dflt"), "dflt");
        env::set_var(key, "value");
        assert_eq!(env_or(key, "dflt"), "value");
        env::remove_var(key);
    }

    #[test]
    fn stripped_removes_only_trailing_newlines() {
        assert_eq!(stripped(b"a\n"), b"a");
        assert_eq!(stripped(b"a\n\n\n"), b"a");
        assert_eq!(stripped(b"a"), b"a");
        assert_eq!(stripped(b""), b"");
        assert_eq!(stripped(b"a\nb\n"), b"a\nb");
        assert_eq!(stripped(b"\n"), b"");
    }

    #[test]
    fn command_v_searches_path_for_executables() {
        let dir = env::temp_dir().join(format!("soda-setup-cmdv-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(&dir).unwrap();
        let tool = dir.join("soda-test-tool");
        let flat = dir.join("soda-test-flat");
        fs::write(&tool, b"#!/bin/sh\nexit 0\n").unwrap();
        fs::write(&flat, b"data").unwrap();
        fs::set_permissions(&tool, fs::Permissions::from_mode(0o755)).unwrap();
        fs::set_permissions(&flat, fs::Permissions::from_mode(0o644)).unwrap();
        let found = command_v_in("soda-test-tool", dir.as_os_str());
        let missing = command_v_in("soda-test-flat", dir.as_os_str()).is_none()
            && command_v_in("soda-test-absent", dir.as_os_str()).is_none();
        let _ = fs::remove_dir_all(&dir);
        assert_eq!(found, Some(tool));
        assert!(missing);
    }

    #[test]
    fn pipe2_reports_last_nonzero_by_position() {
        assert_eq!(pipe2(("true", &[]), ("true", &[])), (0, Vec::new()));
        assert_eq!(pipe2(("true", &[]), ("false", &[])).0, 1);
        // pipefail: the first failure still fails the pipeline.
        assert_eq!(pipe2(("false", &[]), ("true", &[])).0, 1);
        let (code, out) = pipe2(("echo", &["hi"]), ("tr", &["a-z", "A-Z"]));
        assert_eq!((code, out), (0, b"HI\n".to_vec()));
    }

    #[test]
    fn write_staged_matches_redirection_bytes_and_mode() {
        let path = env::temp_dir().join(format!("soda-setup-stage-{}", std::process::id()));
        let _ = fs::remove_file(&path);
        write_staged(&path, b"bytes\n").unwrap();
        assert_eq!(fs::read(&path).unwrap(), b"bytes\n");
        let mode = fs::metadata(&path).unwrap().permissions().mode() & 0o777;
        assert_eq!(mode, 0o666 & !current_umask() & 0o777);
        let _ = fs::remove_file(&path);
    }

    #[test]
    fn passphrase_is_64_lowercase_hex_without_newline() {
        let passphrase = random_hex_passphrase().unwrap();
        assert_eq!(passphrase.len(), 64);
        assert!(passphrase
            .bytes()
            .all(|b| b.is_ascii_hexdigit() && !b.is_ascii_uppercase()));
        assert_ne!(
            random_hex_passphrase().unwrap(),
            random_hex_passphrase().unwrap()
        );
    }
}
