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
mod state;

#[cfg(test)]
use self::process::{exec_diag, signal_name, spawn_diag};
use self::process::{exec_replace, os_error};
use self::start::action_start;
#[cfg(test)]
use self::state::pid_alive;
use self::state::{pid_display, ssh_args, vm_dir, vm_running};

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

fn action_status(vm: &str) -> Result<(), Exit> {
    let pid_path = format!("{vm}/qemu.pid");
    if vm_running(&pid_path)? {
        // A second read for the display value; if the pidfile vanished in
        // between, the substitution fails and aborts like the first one.
        let display = match fs::read(&pid_path) {
            Ok(bytes) => pid_display(&bytes),
            Err(err) => {
                eprintln!("{FAIL_PREFIX}: {pid_path}: {}", os_error(&err));
                return Err(Exit::Propagate(1));
            }
        };
        println!("soda-test is running (PID {display}); SSH at 127.0.0.1:22220");
        Ok(())
    } else {
        // The script echoes to stdout here, so this is not a refusal.
        println!("soda-test is not running");
        Err(Exit::Propagate(1))
    }
}

fn action_ssh(vm: &str, extra: &[String]) -> ! {
    let mut argv = vec!["ssh".to_string()];
    argv.extend(ssh_args(vm));
    argv.push("root@127.0.0.1".to_string());
    argv.extend(extra.iter().cloned());
    exec_replace("ssh", &argv);
}

fn action_tunnel(vm: &str, web: bool) -> ! {
    let mut argv = vec!["ssh".to_string()];
    argv.extend(ssh_args(vm));
    argv.push("-o".to_string());
    argv.push("ExitOnForwardFailure=yes".to_string());
    argv.push("-NT".to_string());
    if web {
        println!("Keep this running: Forgejo + Sodaspaces https://localhost:24444");
        argv.push("-L".to_string());
        argv.push("127.0.0.1:24444:127.0.0.1:24444".to_string());
    } else {
        println!(
            "Keep this running: Forgejo http://localhost:23000; Cockpit https://localhost:29090"
        );
        argv.push("-L".to_string());
        argv.push("127.0.0.1:23000:127.0.0.1:3000".to_string());
        argv.push("-L".to_string());
        argv.push("127.0.0.1:29090:127.0.0.1:9090".to_string());
    }
    argv.push("root@127.0.0.1".to_string());
    exec_replace("ssh", &argv);
}

fn action_console(vm: &str) -> ! {
    exec_replace(
        "tail",
        &[
            "tail".to_string(),
            "-n".to_string(),
            "80".to_string(),
            "-f".to_string(),
            format!("{vm}/console.log"),
        ],
    );
}

fn run_vm(args: &[String]) -> Result<(), Exit> {
    // Like `${1:-status}`, an empty or missing $1 defaults to status.
    let action = args
        .first()
        .map(|arg| arg.as_str())
        .filter(|arg| !arg.is_empty())
        .unwrap_or("status");
    let vm = vm_dir();
    match action {
        "start" => action_start(&vm),
        "status" => action_status(&vm),
        "ssh" => action_ssh(&vm, &args[1..]),
        "tunnel" => action_tunnel(&vm, false),
        "web-tunnel" => action_tunnel(&vm, true),
        "console" => action_console(&vm),
        _ => {
            eprintln!(
                "usage: cargo run -p soda-test-vm -- [start|status|ssh [COMMAND...]|tunnel|web-tunnel|console]"
            );
            Err(Exit::Propagate(2))
        }
    }
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
mod tests {
    use super::*;

    #[test]
    fn pid_display_keeps_spaces_drops_newlines() {
        assert_eq!(pid_display(b"  3512  \n\n"), "  3512  ");
        assert_eq!(pid_display(b"42"), "42");
        assert_eq!(pid_display(b""), "");
    }

    #[test]
    fn pid_alive_matches_kill_zero_semantics() {
        let mine = std::process::id().to_string();
        assert!(pid_alive(mine.as_bytes()));
        assert!(pid_alive(format!("  {mine}  \n").as_bytes()));
        assert!(pid_alive(format!("0{mine}\n").as_bytes()));
        assert!(pid_alive(b"0"));
        assert!(!pid_alive(b""));
        assert!(!pid_alive(b"\n\n"));
        assert!(!pid_alive(b"abc\n"));
        assert!(!pid_alive(b"12 3\n"));
        assert!(!pid_alive(b"-5\n"));
        assert!(!pid_alive(b"2147483647\n"));
        assert!(!pid_alive(b"9999999999\n"));
    }

    #[test]
    fn ssh_args_pin_key_paths_and_options() {
        assert_eq!(
            ssh_args("/repo/.artifacts/test-vm"),
            vec![
                "-p",
                "22220",
                "-i",
                "/repo/.artifacts/test-vm/operator",
                "-o",
                "IdentitiesOnly=yes",
                "-o",
                "StrictHostKeyChecking=yes",
                "-o",
                "UserKnownHostsFile=/repo/.artifacts/test-vm/known_hosts",
            ]
            .into_iter()
            .map(|part| part.to_string())
            .collect::<Vec<String>>(),
        );
    }

    #[test]
    fn stripped_drops_all_trailing_newlines() {
        assert_eq!(stripped(b"Linux\n"), b"Linux");
        assert_eq!(stripped(b"x\n\n\n"), b"x");
        assert_eq!(stripped(b""), b"");
    }

    #[test]
    fn signal_names_come_from_strsignal() {
        assert_eq!(signal_name(15), "Terminated");
        assert_eq!(signal_name(11), "Segmentation fault");
        assert_eq!(signal_name(1), "Hangup");
        assert_eq!(signal_name(6), "Aborted");
    }

    #[test]
    fn spawn_diag_matrix_matches_bash() {
        // Slashless misses say "command not found" (PATH lookups here run
        // against the ambient PATH, where these names do not exist).
        let (msg, code) = spawn_diag(
            "definitely-missing-soda-vm-probe",
            &io::Error::from_raw_os_error(2),
        );
        assert_eq!((msg.as_str(), code), ("command not found", 127));
        // Direct paths report the raw strerror.
        let (msg, code) = spawn_diag(
            "/nonexistent-soda-vm-probe/foo",
            &io::Error::from_raw_os_error(2),
        );
        assert_eq!((msg.as_str(), code), ("No such file or directory", 127));
        // Directories are reported as such.
        let (msg, code) = spawn_diag("/tmp", &io::Error::from_raw_os_error(13));
        assert_eq!((msg.as_str(), code), ("Is a directory", 126));
    }

    #[test]
    fn exec_diag_matrix_matches_bash() {
        // Slashless misses use the one-line "not found" shape.
        let (lines, code) = exec_diag(
            "definitely-missing-soda-vm-probe",
            &io::Error::from_raw_os_error(2),
        );
        assert_eq!(code, 127);
        assert_eq!(
            lines,
            vec!["test-vm: exec: definitely-missing-soda-vm-probe: not found".to_string()]
        );
        // Direct-path misses use the bare strerror without the exec part.
        let (lines, code) = exec_diag(
            "/nonexistent-soda-vm-probe/foo",
            &io::Error::from_raw_os_error(2),
        );
        assert_eq!(code, 127);
        assert_eq!(
            lines,
            vec!["test-vm: /nonexistent-soda-vm-probe/foo: No such file or directory".to_string()]
        );
        // Directories get the two-line shape reporting "Is a directory".
        let (lines, code) = exec_diag("/tmp", &io::Error::from_raw_os_error(13));
        assert_eq!(code, 126);
        assert_eq!(
            lines,
            vec![
                "test-vm: /tmp: Is a directory".to_string(),
                "test-vm: exec: /tmp: cannot execute: Is a directory".to_string(),
            ]
        );
    }
}
