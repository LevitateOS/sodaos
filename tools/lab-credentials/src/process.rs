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

/// `json_escape` mirrors Python `json.dumps` with `ensure_ascii`: the
/// script generated every JSON file through it, so the bytes stay identical.
pub(crate) fn json_escape(value: &str) -> String {
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
pub(crate) fn trust_json(prefix: &str, now: u64, pubs: [&str; 4]) -> String {
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
pub(crate) fn config_json() -> String {
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
