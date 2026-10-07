use std::io::{self, Read, Write};
use std::os::unix::process::ExitStatusExt;
use std::process::{Command, ExitStatus, Stdio};

use crate::{Exit, FAIL_PREFIX};

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

/// `stripped` mirrors command substitution: all trailing newlines removed.
fn stripped(out: &[u8]) -> &[u8] {
    let mut end = out.len();
    while end > 0 && out[end - 1] == b'\n' {
        end -= 1;
    }
    &out[..end]
}

pub(crate) fn stripped_string(out: &[u8]) -> String {
    String::from_utf8_lossy(stripped(out)).into_owned()
}

fn spawn_diag(err: &io::Error) -> (String, i32) {
    match err.kind() {
        io::ErrorKind::NotFound => ("command not found".to_string(), 127),
        io::ErrorKind::PermissionDenied => ("Permission denied".to_string(), 126),
        _ => (err.to_string(), 1),
    }
}

pub(crate) enum Captured {
    SpawnFailed,
    Done(i32, Vec<u8>),
}

/// `capture` runs a command with piped stdout and inherited stderr, like
/// `$(...)`. A spawn failure prints the diagnostic immediately and the
/// caller decides: test position uses the empty substitution, bare position
/// propagates the code (the script's `set -e` behavior).
pub(crate) fn capture(prog: &str, args: &[&str], stderr_null: bool) -> Captured {
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
pub(crate) fn run(prog: &str, args: &[&str]) -> Result<(), Exit> {
    run_with_io(prog, args, false, false)
}

/// `run_stdout_null` mirrors `... >/dev/null`.
pub(crate) fn run_stdout_null(prog: &str, args: &[&str]) -> Result<(), Exit> {
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

fn pretty_json(value: &impl serde::Serialize) -> String {
    serde_json::to_string_pretty(value).expect("serializing a JSON record cannot fail")
}

#[derive(serde::Serialize)]
#[allow(non_snake_case)]
struct TrustRecord<'a> {
    Format: u8,
    Prefix: &'a str,
    Epoch: u8,
    Keys: TrustKeys<'a>,
    NotBefore: u64,
    MaxAgeSeconds: u16,
    ClockSkewSeconds: u8,
    MinimumSequence: MinimumSequence,
}

#[derive(serde::Serialize)]
struct TrustKeys<'a> {
    artifact: [&'a str; 1],
    candidate: [&'a str; 1],
    preview: [&'a str; 1],
    stable: [&'a str; 1],
}

#[derive(serde::Serialize)]
struct MinimumSequence {
    candidate: u8,
    preview: u8,
    stable: u8,
}

#[derive(serde::Serialize)]
#[allow(non_snake_case)]
struct ConfigRecord {
    Trust: &'static str,
    Keys: ConfigKeys,
}

#[derive(serde::Serialize)]
#[allow(non_snake_case)]
struct ConfigKeys {
    Key: &'static str,
    Passphrase: &'static str,
}

/// `trust_json` emits the trust record with its required final LF.
pub(crate) fn trust_json(prefix: &str, now: u64, pubs: [&str; 4]) -> String {
    let record = TrustRecord {
        Format: 1,
        Prefix: prefix,
        Epoch: 1,
        Keys: TrustKeys {
            artifact: [pubs[0]],
            candidate: [pubs[1]],
            preview: [pubs[2]],
            stable: [pubs[3]],
        },
        NotBefore: now - 600,
        MaxAgeSeconds: 3600,
        ClockSkewSeconds: 10,
        MinimumSequence: MinimumSequence {
            candidate: 1,
            preview: 1,
            stable: 1,
        },
    };
    pretty_json(&record) + "\n"
}

/// `config_json` emits the authority configuration record with its required final LF.
pub(crate) fn config_json() -> String {
    pretty_json(&ConfigRecord {
        Trust: "/run/soda-media-authority/trust.json",
        Keys: ConfigKeys {
            Key: "/run/soda-media-authority/artifact.private",
            Passphrase: "/run/soda-media-authority/passphrase",
        },
    }) + "\n"
}
