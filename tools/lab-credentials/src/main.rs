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

mod fixture_authority;
mod inventory;
mod process;
mod runbooks;
#[cfg(test)]
mod tests;

use std::env;
use std::fs;
use std::path::PathBuf;

use crate::fixture_authority::rotate_fixture_authority;
use crate::inventory::inventory;
use crate::runbooks::{
    HELP_HEADER, RUNBOOK_CLOUDFLARED, RUNBOOK_FIXTURE, RUNBOOK_LAB_VM, RUNBOOK_RUNNER,
};

pub(crate) const FAIL_PREFIX: &str = "rotate-lab-creds";
pub(crate) const PREFIX_DEFAULT: &str = "ghcr.io/levitateos/sodaos";
pub(crate) const AUTHORITY: &str = "/var/lib/soda-candidate-authority";
pub(crate) const WORKER_USER: &str = "soda-build-worker";

extern "C" {
    pub(crate) fn faccessat(dirfd: i32, path: *const i8, mode: i32, flags: i32) -> i32;
    pub(crate) fn umask(mask: u32) -> u32;
    fn sigaction(signum: i32, act: *const Sigaction, oldact: *mut Sigaction) -> i32;
}
pub(crate) const AT_FDCWD: i32 = -100;
pub(crate) const X_OK: i32 = 1;
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
pub(crate) enum Exit {
    Fail(String),
    Propagate(i32),
}

pub(crate) fn fail<T>(msg: impl Into<String>) -> Result<T, Exit> {
    Err(Exit::Fail(msg.into()))
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
