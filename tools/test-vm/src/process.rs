use core::ffi::c_char;
use std::env;
use std::ffi::{CStr, CString};
use std::fs;
use std::io::{self, Read};
use std::os::unix::process::ExitStatusExt;
use std::path::Path;
use std::process::{Command, Stdio};

use super::{
    execvp, faccessat, flush_stdout, status_code, strerror, strsignal, Exit, AT_FDCWD, FAIL_PREFIX,
};

/// `os_error` mirrors the shell's bare strerror text (no `(os error N)`).
pub(super) fn os_error(err: &io::Error) -> String {
    match err.raw_os_error() {
        Some(no) => unsafe { CStr::from_ptr(strerror(no)).to_string_lossy().into_owned() },
        None => err.to_string(),
    }
}

/// `access` mirrors `test -r/-w/-x`: the same access(2) call bash makes,
// including root's bypass semantics.
pub(super) fn access(path: &str, mode: i32) -> bool {
    match CString::new(path) {
        Ok(c) => unsafe { faccessat(AT_FDCWD, c.as_ptr(), mode, 0) == 0 },
        Err(_) => false,
    }
}

pub(super) fn is_file(path: &str) -> bool {
    fs::metadata(path).map(|m| m.is_file()).unwrap_or(false)
}

fn is_dir(path: &Path) -> bool {
    fs::metadata(path).map(|m| m.is_dir()).unwrap_or(false)
}

/// `path_hit` mirrors bash's PATH lookup for a slashless command: the
/// first hit that exists and is not a directory. Directories are skipped
/// (bash reports the command missing), anything else stops the search and
/// its exec error is reported.
fn path_hit(prog: &str) -> bool {
    let raw = env::var_os("PATH").unwrap_or_else(|| "/bin:/usr/bin".into());
    for dir in env::split_paths(&raw) {
        let candidate = dir.join(prog);
        match fs::metadata(&candidate) {
            Ok(meta) if !meta.is_dir() => return true,
            Ok(_) => continue,
            Err(_) => continue,
        }
    }
    false
}

/// `spawn_diag` mirrors a bare command's failure. A slashless miss is
/// "command not found" (127); a direct path reports the raw strerror (127
/// for missing, 126 otherwise); directories say "Is a directory" (126).
pub(super) fn spawn_diag(prog: &str, err: &io::Error) -> (String, i32) {
    if !prog.contains('/') {
        if path_hit(prog) {
            return (os_error(err), 126);
        }
        return ("command not found".to_string(), 127);
    }
    match err.kind() {
        io::ErrorKind::NotFound => (os_error(err), 127),
        io::ErrorKind::PermissionDenied => {
            if is_dir(Path::new(prog)) {
                ("Is a directory".to_string(), 126)
            } else {
                ("Permission denied".to_string(), 126)
            }
        }
        _ => (os_error(err), 126),
    }
}

/// `exec_diag` mirrors `exec` failing. A slashless miss is the one-line
/// "not found" (127); a direct-path miss is the bare strerror (127); every
/// other failure is the bare error plus "cannot execute" (126).
pub(super) fn exec_diag(prog: &str, err: &io::Error) -> (Vec<String>, i32) {
    if !prog.contains('/') {
        if !path_hit(prog) {
            return (vec![format!("{FAIL_PREFIX}: exec: {prog}: not found")], 127);
        }
        let detail = os_error(err);
        return (
            vec![
                format!("{FAIL_PREFIX}: {prog}: {detail}"),
                format!("{FAIL_PREFIX}: exec: {prog}: cannot execute: {detail}"),
            ],
            126,
        );
    }
    if err.kind() == io::ErrorKind::NotFound {
        return (
            vec![format!("{FAIL_PREFIX}: {prog}: {}", os_error(err))],
            127,
        );
    }
    let detail = if is_dir(Path::new(prog)) {
        "Is a directory".to_string()
    } else {
        os_error(err)
    };
    (
        vec![
            format!("{FAIL_PREFIX}: {prog}: {detail}"),
            format!("{FAIL_PREFIX}: exec: {prog}: cannot execute: {detail}"),
        ],
        126,
    )
}

pub(super) enum Captured {
    SpawnFailed,
    Done(Vec<u8>),
}

/// `capture` runs a command with piped stdout and inherited stderr, like
/// `$(...)`. A spawn failure prints the diagnostic immediately (unless the
/// script redirected it away) and the caller decides how to continue.
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
            let (msg, _code) = spawn_diag(prog, &err);
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
    // Reap the child; like `[[ $(...) == ... ]]` the status is ignored and
    // only the output bytes decide. A substitution child killed by signal
    // stays silent here (bash reports only foreground deaths this way).
    let _ = child.wait();
    Captured::Done(out)
}

/// `TYPED_QEMU_CMD` is the QEMU invocation exactly as typed in the retired
/// script (lines 30-35): bash reprints the parsed command unexpanded,
/// space-joined, quotes and redirects kept, in signal-death notices.
const TYPED_QEMU_CMD: &str = "\"$qemu\" -name soda-test -machine q35,accel=kvm -cpu host -smp 4 -m 8192 -drive \"if=virtio,format=qcow2,file=$vm/disk.qcow2\" -fw_cfg \"name=opt/com.coreos/config,file=$vm/soda.ign\" -nic user,model=virtio-net-pci,hostfwd=tcp:127.0.0.1:22220-:22 -display none -serial \"file:$vm/console.log\" -monitor none -daemonize -pidfile \"$vm/qemu.pid\" 9>&-";

/// `signal_name` mirrors the signal strings bash prints (glibc strsignal).
pub(super) fn signal_name(sig: i32) -> String {
    unsafe {
        CStr::from_ptr(strsignal(sig))
            .to_string_lossy()
            .into_owned()
    }
}

/// `signal_notice` mirrors what a script-running bash prints when its
/// foreground child dies by signal: silence for INT and PIPE, bare
/// "Terminated" for TERM, and otherwise the pid, the signal name padded to
/// 24, "(core dumped)" for core-dumping signals (printed unconditionally,
/// even when no core is written), and the typed command. The script path
/// and line number collapse to the fixed tool prefix, like every other
/// crash prefix in this port.
fn signal_notice(sig: i32, pid: u32) {
    if sig == 2 || sig == 13 {
        return;
    }
    if sig == 15 {
        eprintln!("Terminated");
        return;
    }
    let name = signal_name(sig);
    // Core-dumping signals per signal(7): QUIT ILL TRAP ABRT BUS FPE SEGV
    // XCPU XFSZ SYS.
    let core = matches!(sig, 3 | 4 | 5 | 6 | 7 | 8 | 11 | 24 | 25 | 31);
    if core {
        eprintln!("{FAIL_PREFIX}: {pid} {name:<24}(core dumped) {TYPED_QEMU_CMD}");
    } else {
        eprintln!("{FAIL_PREFIX}: {pid} {name:<24}{TYPED_QEMU_CMD}");
    }
}

/// `run` mirrors a bare command: inherited stdio, signal notice on violent
/// death, propagated status.
pub(super) fn run(prog: &str, args: &[&str]) -> Result<(), Exit> {
    flush_stdout();
    let mut cmd = Command::new(prog);
    cmd.args(args);
    let mut child = match cmd.spawn() {
        Ok(child) => child,
        Err(err) => {
            let (msg, code) = spawn_diag(prog, &err);
            eprintln!("{FAIL_PREFIX}: {prog}: {msg}");
            return Err(Exit::Propagate(code));
        }
    };
    let pid = child.id();
    let status = match child.wait() {
        Ok(status) => status,
        Err(_) => return Err(Exit::Propagate(1)),
    };
    if let Some(sig) = status.signal() {
        signal_notice(sig, pid);
    }
    let code = status_code(status);
    if code == 0 {
        Ok(())
    } else {
        Err(Exit::Propagate(code))
    }
}

/// `exec_replace` mirrors `exec`: the current process becomes `prog`, so it
/// never returns on success. Failure prints the `exec_diag` shapes and exits.
pub(super) fn exec_replace(prog: &str, args: &[String]) -> ! {
    flush_stdout();
    let prog_c = CString::new(prog).expect("exec prog must not contain NUL");
    let arg_cs: Vec<CString> = args
        .iter()
        .map(|arg| CString::new(arg.as_str()).expect("exec argv must not contain NUL"))
        .collect();
    let mut argv: Vec<*const c_char> = arg_cs.iter().map(|arg| arg.as_ptr()).collect();
    argv.push(std::ptr::null());
    unsafe {
        execvp(prog_c.as_ptr(), argv.as_ptr());
    }
    let err = io::Error::last_os_error();
    let (lines, code) = exec_diag(prog, &err);
    for line in lines {
        eprintln!("{line}");
    }
    std::process::exit(code);
}
