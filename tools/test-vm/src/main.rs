// soda-test-vm — manage the isolated x86_64 test VM prepared under
// .artifacts/test-vm. Never installs on the builder, deletes a disk, or
// changes host networking.
//
// Rust port of scripts/test-vm.sh. Messages, exit codes, the QEMU command
// line, ssh/tail argv, lock handling, and pidfile checks match the shell,
// including its argument handling (only $1 names the action; only `ssh`
// passes the rest through) and its empty-$1 default to status.
//
// Deliberate deltas: the tool operates relative to the current directory
// (invoke it from the repository root; `$PWD` is honored only when it
// names the cwd) instead of cd-ing to the script's own location, which
// is meaningless for an installed binary; the usage
// and started lines print the working `cargo run` invocation instead of
// the deleted script path; and `exec` uses execvp PATH search, which skips
// a broken shadow entry where bash would stop and fail.

use core::ffi::{c_char, c_int};
use std::env;
use std::fs;
use std::io::{self, Write};
use std::os::unix::process::ExitStatusExt;
use std::process::ExitStatus;

mod process;
mod start;
mod transport;

use self::transport::run_vm;
mod state;

const FAIL_PREFIX: &str = "test-vm";
const QEMU_DEFAULT: &str = "/usr/libexec/qemu-kvm";
const SSH_PORT: &str = "22220";

extern "C" {
    fn faccessat(dirfd: i32, path: *const i8, mode: i32, flags: i32) -> i32;
    fn umask(mask: u32) -> u32;
    fn sigaction(signum: i32, act: *const Sigaction, oldact: *mut Sigaction) -> i32;
    fn flock(fd: i32, op: i32) -> i32;
    fn kill(pid: c_int, sig: c_int) -> c_int;
    fn execvp(file: *const c_char, argv: *const *const c_char) -> c_int;
    fn strerror(errnum: c_int) -> *const c_char;
    fn strsignal(sig: c_int) -> *const c_char;
}

const AT_FDCWD: i32 = -100;
const R_OK: i32 = 4;
const W_OK: i32 = 2;
const X_OK: i32 = 1;
const LOCK_EX: i32 = 2;
const LOCK_NB: i32 = 4;
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

/// How the process ends: a propagated status with no extra output (the
/// script's `set -e` behavior). The script's own refusal echoes are bare
/// (`echo ... >&2; exit 1` with no prefix); only bash's own diagnostics
/// carry a prefix, mirrored with `test-vm: ` at their call sites.
#[derive(Debug)]
enum Exit {
    Propagate(i32),
}

/// `refuse` mirrors `echo ... >&2; exit 1`: the bare script message.
fn refuse<T>(msg: impl Into<String>) -> Result<T, Exit> {
    eprintln!("{}", msg.into());
    Err(Exit::Propagate(1))
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

/// `current_pwd` mirrors the script's post-`cd` `$PWD`: the inherited
/// logical path when it names the current directory (symlinked invocations
/// keep their logical spelling like the script), otherwise the physical
/// working directory. Callers that fix the cwd without fixing `$PWD` still
/// resolve correctly.
fn current_pwd() -> String {
    let physical = env::current_dir().unwrap_or_default();
    match env::var("PWD") {
        Ok(logical) if !logical.is_empty() => match fs::canonicalize(&logical) {
            Ok(resolved) if resolved == physical => logical,
            _ => physical.to_string_lossy().into_owned(),
        },
        _ => physical.to_string_lossy().into_owned(),
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
    let argv: Vec<String> = env::args().collect();
    let args = if argv.is_empty() {
        Vec::new()
    } else {
        argv[1..].to_vec()
    };
    match run_vm(&args) {
        Ok(()) => {}
        Err(Exit::Propagate(code)) => std::process::exit(code),
    }
}

#[cfg(test)]
mod tests;
