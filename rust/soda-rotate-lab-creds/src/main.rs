// soda-rotate-lab-creds — lab credential inventory and owner-gated
// rotation (B7).
//
// Context: the D3 rootfs HTTP exposure served live guest disks and images
// world-readable over HTTP. Any secret that ever lived in those bytes must
// be treated as exposed until the owner rotates it. Rotation itself is an
// explicit owner decision: this tool defaults to read-only inventory and
// prints runbooks; only --execute with SODA_ROTATE_ACK=<class> mutates, and
// only for fully scriptable classes. Secrets travel via 0600 files, never
// argv, environment values in logs, or stdout.
//
// Rust port of scripts/ops/rotate-lab-creds.sh. Messages, exit codes,
// inspected paths, file modes, and generated file bytes match the shell,
// including its argument handling (only $1/$2/$3 are read; anything else is
// ignored) and its set -u crashes on unset $USER/$HOME.

use std::env;
use std::ffi::CString;
use std::fs;
use std::io::{self, Read, Write};
use std::os::unix::ffi::OsStrExt;
use std::os::unix::fs::OpenOptionsExt;
use std::os::unix::process::ExitStatusExt;
use std::path::{Path, PathBuf};
use std::process::{Command, ExitStatus, Stdio};

const FAIL_PREFIX: &str = "rotate-lab-creds";
const PREFIX_DEFAULT: &str = "ghcr.io/levitateos/sodaos";
const AUTHORITY: &str = "/var/lib/soda-candidate-authority";
const WORKER_USER: &str = "soda-build-worker";

extern "C" {
    fn faccessat(dirfd: i32, path: *const i8, mode: i32, flags: i32) -> i32;
    fn umask(mask: u32) -> u32;
    fn sigaction(signum: i32, act: *const Sigaction, oldact: *mut Sigaction) -> i32;
}
const AT_FDCWD: i32 = -100;
const X_OK: i32 = 1;
const SIGPIPE: i32 = 13;
const SIG_DFL: usize = 0;

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

/// `current_pwd` mirrors the script's bare `$PWD` in the metadata glob: the
/// inherited value verbatim (even when stale or empty), or the `set -u`
/// crash when unset. Only the crash prefix differs (fixed here instead of
/// `$0`-and-line), since invocation paths differ inherently.
fn current_pwd() -> Result<String, Exit> {
    match env::var("PWD") {
        Ok(v) => Ok(v),
        Err(_) => {
            eprintln!("{FAIL_PREFIX}: PWD: unbound variable");
            Err(Exit::Propagate(1))
        }
    }
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

fn spawn_diag(err: &io::Error) -> (String, i32) {
    match err.kind() {
        io::ErrorKind::NotFound => ("command not found".to_string(), 127),
        io::ErrorKind::PermissionDenied => ("Permission denied".to_string(), 126),
        _ => (err.to_string(), 1),
    }
}

enum Captured {
    SpawnFailed,
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
            let (msg, _code) = spawn_diag(&err);
            // A suppressed child stderr (`2>/dev/null`) also swallows the
            // shell's own "command not found"; stay silent the same way.
            if !stderr_null {
                eprintln!("{FAIL_PREFIX}: {prog}: {msg}");
            }
            return Captured::SpawnFailed;
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
    run_with_io(prog, args, false, false)
}

/// `run_stdout_null` mirrors `... >/dev/null`.
fn run_stdout_null(prog: &str, args: &[&str]) -> Result<(), Exit> {
    run_with_io(prog, args, true, false)
}

fn run_with_io(
    prog: &str,
    args: &[&str],
    stdout_null: bool,
    stderr_null: bool,
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
    let code = child.wait().map(status_code).unwrap_or(1);
    if code == 0 {
        Ok(())
    } else {
        Err(Exit::Propagate(code))
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

/// `HELP_HEADER` is the retired script's lines 2-10 verbatim: `help` printed
/// them via `sed -n '2,10p' "$0"`, so the port embeds the same bytes.
const HELP_HEADER: &str = "# rotate-lab-creds.sh — lab credential inventory and owner-gated rotation (B7).\n#\n# Context: the D3 rootfs HTTP exposure served live guest disks and images\n# world-readable over HTTP. Any secret that ever lived in those bytes must\n# be treated as exposed until the owner rotates it. Rotation itself is an\n# explicit owner decision: this script defaults to read-only inventory and\n# prints runbooks; only --execute with SODA_ROTATE_ACK=<class> mutates, and\n# only for fully scriptable classes. Secrets travel via 0600 files, never\n# argv, environment values in logs, or stdout.\n";

const RUNBOOK_FIXTURE: &str = "-- runbook: fixture-authority (fully scriptable with --execute)\nRegenerates the fixture-only media authority (never release keys):\n  SODA_ROTATE_ACK=fixture-authority cargo run -p soda-rotate-lab-creds --rotate fixture-authority --execute\nEffect: old fixture signatures stop verifying; in-flight development\nattempts using the old authority fail closed and must rerun setup.\n";

const RUNBOOK_CLOUDFLARED: &str = "-- runbook: cloudflared-token (owner-manual; dashboard-issued)\n1. Rotate the tunnel token in the Cloudflare dashboard.\n2. As root, install it with secret-file handling only:\n     install -m 0640 -o root -g cloudflared /path/to/new.token \\\n       /etc/cloudflared/dimensionlab-forgejo-https.token\n   Never pass the token on a command line; shred the staging copy after.\n3. Restart the tunnel service and re-run this script (inventory).\n";

const RUNBOOK_RUNNER: &str = "-- runbook: forgejo-runner (owner-manual; needs a Forgejo admin token)\n1. Revoke the runner registration in Forgejo and create a new token.\n2. Stop the runner, replace the secret file (0600) without argv exposure,\n   re-register, and restart the runner.\n3. Re-run this script (inventory) to confirm modes.\n";

const RUNBOOK_LAB_VM: &str = "-- runbook: lab-vm-operator (owner-manual; protected VMs)\nGuest operator credentials possibly baked into the served QCOW2s cannot be\naudited from the host (live disks are never mounted here). After the D3\nreplacement is installed, rotate operator/SSH material from the guest\nconsoles, then record the rotation date with the owner.\n";

/// `note` mirrors `printf '%-7s %s\n'`: the tag padded to 7, then the text.
fn note(tag: &str, msg: &str) -> String {
    format!("{tag:<7} {msg}")
}

/// `stat_line` mirrors `stat_one`: mode/owner/size/mtime for a path, never
/// content. Unset `$USER` on a missing path crashes like the script's
/// `set -u` expansion; only the crash prefix differs (fixed here instead of
/// `$0`-and-line), since invocation paths differ inherently.
fn stat_line(path: &str, want: &str) -> Result<String, Exit> {
    if fs::metadata(path).is_err() {
        let user = match env::var("USER") {
            Ok(user) => user,
            Err(_) => {
                eprintln!("{FAIL_PREFIX}: USER: unbound variable");
                return Err(Exit::Propagate(1));
            }
        };
        return Ok(note(
            "SKIP",
            &format!("{path} absent or not visible to {user}"),
        ));
    }
    // The script suppresses this stat's stderr (`2>/dev/null`); the two
    // mode checks below inherit it, like the script's unredirected stats.
    let full = match capture("stat", &["-c", "%a %U:%G %s %y", path], true) {
        Captured::SpawnFailed => {
            return Ok(note(
                "SKIP",
                &format!("{path} unreadable (run with read access)"),
            ));
        }
        Captured::Done(code, out) => {
            if code != 0 {
                return Ok(note(
                    "SKIP",
                    &format!("{path} unreadable (run with read access)"),
                ));
            }
            stripped_string(&out)
        }
    };
    let mode = match capture("stat", &["-c", "%a", path], false) {
        Captured::SpawnFailed => String::new(),
        Captured::Done(_, out) => stripped_string(&out),
    };
    if mode == want {
        Ok(note("PASS", &format!("{path} [{full}]")))
    } else {
        let shown = match capture("stat", &["-c", "%a", path], false) {
            Captured::SpawnFailed => String::new(),
            Captured::Done(_, out) => stripped_string(&out),
        };
        Ok(note(
            "WARN",
            &format!("{path} mode is {shown}, want {want} [{full}]"),
        ))
    }
}

/// `first_live_inputs` mirrors the build-metadata glob: the first
/// `soda-live-inputs-*.json` under the isolated releases directory in byte
/// (LC_ALL=C collation) order, or nothing when the glob matches nothing.
/// Like the script's `[ -e ]` guard, entries that do not resolve (dangling
/// symlinks, unreadable paths) are skipped, not reported.
fn first_live_inputs(pwd: &str) -> Option<String> {
    let dir = format!("{pwd}/.artifacts/releases/isolated");
    let entries = fs::read_dir(&dir).ok()?;
    let mut names: Vec<String> = entries
        .filter_map(|entry| entry.ok())
        .map(|entry| entry.file_name().to_string_lossy().into_owned())
        .filter(|name| {
            name.starts_with("soda-live-inputs-")
                && name.ends_with(".json")
                && fs::metadata(format!("{dir}/{name}")).is_ok()
        })
        .collect();
    names.sort();
    names.into_iter().next().map(|name| format!("{dir}/{name}"))
}

fn inventory() -> Result<(), Exit> {
    println!("-- fixture-only media authority (dev scope; regenerable)");
    for name in [
        "artifact.private",
        "passphrase",
        "config.json",
        "trust.json",
    ] {
        println!("{}", stat_line(&format!("{AUTHORITY}/{name}"), "600")?);
    }
    println!("{}", stat_line(&format!("{AUTHORITY}/worker.json"), "600")?);
    println!("-- cloudflared tunnel credentials (dashboard-issued)");
    println!(
        "{}",
        stat_line("/etc/cloudflared/dimensionlab-forgejo-https.token", "640")?
    );
    println!(
        "{}",
        stat_line("/etc/cloudflared/dimensionlab-forgejo-https.id", "640")?
    );
    println!("-- forgejo runner registration (host service config)");
    let home = match env::var("HOME") {
        Ok(home) => home,
        Err(_) => {
            eprintln!("{FAIL_PREFIX}: HOME: unbound variable");
            return Err(Exit::Propagate(1));
        }
    };
    println!(
        "{}",
        stat_line(
            &format!("{home}/containers/forgejo-runner/data/config.yaml"),
            "600"
        )?
    );
    println!(
        "{}",
        stat_line(
            &format!("{home}/containers/forgejo-runner/data/.runner"),
            "600"
        )?
    );
    println!("-- build metadata (public pins only; informational)");
    if let Some(first) = first_live_inputs(&current_pwd()?) {
        println!(
            "{}",
            note(
                "INFO",
                &format!("{first} carries public URLs/hashes only; 0644 is expected")
            )
        );
    }
    println!("-- D3 exposure reminder");
    println!(
        "{}",
        note(
            "INFO",
            "served QCOW2/ISO/rootfs bytes are 0644-or-readable over HTTP;"
        )
    );
    println!(
        "{}",
        note(
            "INFO",
            "guest runtime secrets inside them are UNKNOWN until rotated."
        )
    );
    println!("inventory complete; no values printed, nothing mutated.");
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

fn rotate_fixture_authority(cleanup: &mut Vec<PathBuf>) -> Result<(), Exit> {
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

fn run_rotate(argv0: &str, args: &[String], cleanup: &mut Vec<PathBuf>) -> Result<(), Exit> {
    let class = args
        .first()
        .map(|arg| arg.as_str())
        .filter(|arg| !arg.is_empty())
        .unwrap_or("inventory");
    let class = match class {
        "-h" | "--help" | "help" => "help",
        other => other,
    };
    // Only $2/$3 can carry --execute; anything further is ignored.
    let execute = args.get(1).is_some_and(|arg| arg == "--execute")
        || args.get(2).is_some_and(|arg| arg == "--execute");
    match class {
        "inventory" => inventory(),
        "help" => {
            print!("{HELP_HEADER}");
            println!("usage: {argv0} [inventory|--rotate CLASS [--execute]]");
            println!("classes: fixture-authority cloudflared-token forgejo-runner lab-vm-operator");
            Ok(())
        }
        "--rotate" => {
            let target = args.get(1).map(|arg| arg.as_str()).unwrap_or("");
            match target {
                "fixture-authority" => {
                    if execute {
                        rotate_fixture_authority(cleanup)
                    } else {
                        print!("{RUNBOOK_FIXTURE}");
                        Ok(())
                    }
                }
                "cloudflared-token" => {
                    print!("{RUNBOOK_CLOUDFLARED}");
                    Ok(())
                }
                "forgejo-runner" => {
                    print!("{RUNBOOK_RUNNER}");
                    Ok(())
                }
                "lab-vm-operator" => {
                    print!("{RUNBOOK_LAB_VM}");
                    Ok(())
                }
                _ => fail(format!("unknown class '{target}'; see --help")),
            }
        }
        _ => fail(format!("unknown command '{class}'; see --help")),
    }
}

fn main() {
    // Die by SIGPIPE like the shell instead of panicking on a closed pipe,
    // and keep the script's `umask 077` for staging and children.
    unsafe {
        let restore_pipe = Sigaction {
            handler: SIG_DFL,
            mask: [0; 16],
            flags: 0,
            restorer: 0,
        };
        sigaction(SIGPIPE, &restore_pipe, std::ptr::null_mut());
        umask(0o077);
    }
    let mut argv: Vec<String> = env::args().collect();
    let argv0 = argv.first().cloned().unwrap_or_default();
    let args = if argv.is_empty() {
        Vec::new()
    } else {
        argv.split_off(1)
    };
    let mut cleanup: Vec<PathBuf> = Vec::new();
    let result = run_rotate(&argv0, &args, &mut cleanup);
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
    fn note_pads_tag_to_seven() {
        assert_eq!(note("PASS", "/x [s]"), "PASS    /x [s]");
        assert_eq!(note("SKIP", "m"), "SKIP    m");
        assert_eq!(note("INFO", "m"), "INFO    m");
    }

    #[test]
    fn stat_line_reports_modes_without_content() {
        let dir = env::temp_dir().join(format!("soda-rotate-stat-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(&dir).unwrap();
        let secret = dir.join("secret.token");
        fs::write(&secret, b"SYNTHETIC_SECRET_MARKER").unwrap();
        fs::set_permissions(&secret, fs::Permissions::from_mode(0o600)).unwrap();
        let path = secret.to_string_lossy().into_owned();
        let pass = stat_line(&path, "600").unwrap();
        assert!(pass.starts_with("PASS   "), "{pass}");
        assert!(pass.contains(&path), "{pass}");
        assert!(!pass.contains("SYNTHETIC_SECRET_MARKER"), "{pass}");
        let warn = stat_line(&path, "640").unwrap();
        assert!(warn.starts_with("WARN   "), "{warn}");
        assert!(warn.contains("mode is 600, want 640"), "{warn}");
        assert!(!warn.contains("SYNTHETIC_SECRET_MARKER"), "{warn}");
        fs::set_permissions(&secret, fs::Permissions::from_mode(0o640)).unwrap();
        let pass640 = stat_line(&path, "640").unwrap();
        assert!(pass640.starts_with("PASS   "), "{pass640}");
        let missing = dir.join("absent.token").to_string_lossy().into_owned();
        let skip = stat_line(&missing, "600").unwrap();
        assert!(skip.starts_with("SKIP   "), "{skip}");
        assert!(skip.contains("absent or not visible to"), "{skip}");
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn first_live_inputs_takes_sorted_first_match() {
        let root = env::temp_dir().join(format!("soda-rotate-glob-{}", std::process::id()));
        let _ = fs::remove_dir_all(&root);
        let dir = root.join(".artifacts/releases/isolated");
        fs::create_dir_all(&dir).unwrap();
        let base = root.to_string_lossy().into_owned();
        assert_eq!(first_live_inputs(&base), None);
        fs::write(dir.join("soda-live-inputs-b.json"), b"b").unwrap();
        fs::write(dir.join("soda-live-inputs-a.json"), b"a").unwrap();
        fs::write(dir.join("notes.txt"), b"n").unwrap();
        assert_eq!(
            first_live_inputs(&base),
            Some(format!(
                "{}/.artifacts/releases/isolated/soda-live-inputs-a.json",
                base
            ))
        );
        let _ = fs::remove_dir_all(&root);
    }

    #[test]
    fn first_live_inputs_skips_dangling_symlinks() {
        use std::os::unix::fs::symlink;
        let root = env::temp_dir().join(format!("soda-rotate-dangle-{}", std::process::id()));
        let _ = fs::remove_dir_all(&root);
        let dir = root.join(".artifacts/releases/isolated");
        fs::create_dir_all(&dir).unwrap();
        let base = root.to_string_lossy().into_owned();
        symlink("nowhere-target.json", dir.join("soda-live-inputs-zz.json")).unwrap();
        assert_eq!(first_live_inputs(&base), None);
        fs::write(dir.join("soda-live-inputs-aa.json"), b"a").unwrap();
        assert_eq!(
            first_live_inputs(&base),
            Some(format!(
                "{}/.artifacts/releases/isolated/soda-live-inputs-aa.json",
                base
            ))
        );
        let _ = fs::remove_dir_all(&root);
    }

    #[test]
    fn passphrase_is_64_lowercase_hex_without_newline() {
        let passphrase = random_hex_passphrase().unwrap();
        assert_eq!(passphrase.len(), 64);
        assert!(passphrase
            .bytes()
            .all(|b| b.is_ascii_hexdigit() && !b.is_ascii_uppercase()));
    }
}
