use std::io::{self, Read, Write};
use std::os::unix::process::ExitStatusExt;
use std::process::{Command, ExitStatus, Stdio};

use super::{fail, is_dir, stripped, Exit, FAIL_PREFIX};

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

fn spawn_diag(err: &io::Error) -> (String, i32) {
    match err.kind() {
        io::ErrorKind::NotFound => ("command not found".to_string(), 127),
        io::ErrorKind::PermissionDenied => ("Permission denied".to_string(), 126),
        _ => (err.to_string(), 1),
    }
}

pub(super) enum Captured {
    SpawnFailed(i32),
    Done(i32, Vec<u8>),
}

/// `capture` runs a command with piped stdout and inherited stderr, like
/// `$(...)`. A spawn failure prints the diagnostic immediately and the
/// caller decides: test position uses the empty substitution, bare position
/// propagates the code (the script's `set -e` behavior).
pub(super) fn capture(prog: &str, args: &[&str], stderr_null: bool) -> Captured {
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

/// Preflight clean-tree check (D01-F3): only a *successful* status with
/// empty output proves clean. Spawn failure or a failing status refuses.
pub(super) fn git_tree_clean(captured: &Captured) -> bool {
    match captured {
        Captured::SpawnFailed(_) => false,
        Captured::Done(code, out) => *code == 0 && stripped(out).is_empty(),
    }
}

/// `run` mirrors a bare command: inherited stdio, propagated status.
pub(super) fn run(prog: &str, args: &[&str]) -> Result<(), Exit> {
    run_with_io(prog, args, false, false, None, None)
}

/// `run_stdout_null` mirrors `... >/dev/null`.
pub(super) fn run_stdout_null(prog: &str, args: &[&str]) -> Result<(), Exit> {
    run_with_io(prog, args, true, false, None, None)
}

/// `run_stderr_null` mirrors `... 2>/dev/null`.
pub(super) fn run_stderr_null(prog: &str, args: &[&str]) -> Result<(), Exit> {
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
pub(super) fn run_piped_stdin(prog: &str, args: &[&str], input: &[u8]) -> Result<(), Exit> {
    run_with_io(prog, args, true, false, Some(input), None)
}

/// `pipe2` mirrors `first | second` under `pipefail`: the exit status is
/// the last non-zero status by pipeline position.
pub(super) fn pipe2(first: (&str, &[&str]), second: (&str, &[&str])) -> (i32, Vec<u8>) {
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
pub(super) fn ls_nonempty(path: &str) -> bool {
    match capture("ls", &["-A", path], true) {
        Captured::SpawnFailed(_) => false,
        Captured::Done(_, out) => !stripped(&out).is_empty(),
    }
}

/// `id_un` mirrors `$(id -un)` inside `local`: any failure yields the empty
/// string and execution continues.
pub(super) fn id_un() -> Vec<u8> {
    match capture("id", &["-un"], false) {
        Captured::SpawnFailed(_) => Vec::new(),
        Captured::Done(_, out) => stripped(&out).to_vec(),
    }
}

/// `run_in_dir` mirrors `(cd dir && cmd ...) || fail(msg)`: a missing
/// directory prints the `cd` diagnostic and then fails like the script's
/// `|| fail`, anything else runs the command there.
pub(super) fn run_in_dir(
    prog: &str,
    args: &[&str],
    dir: &str,
    stdout_null: bool,
    fail_msg: &str,
) -> Result<(), Exit> {
    if !is_dir(dir) {
        eprintln!("{FAIL_PREFIX}: cd: {dir}: No such file or directory");
        return fail(fail_msg.to_string());
    }
    run_with_io(prog, args, stdout_null, false, None, Some(dir))
}
