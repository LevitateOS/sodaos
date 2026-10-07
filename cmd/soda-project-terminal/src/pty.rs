//! PTY attach relay (port of `spawn_login_pty` … `run_terminal`).
//!
//! Byte-exact stdout lines (`ready`/`output`/`closed`) via [`pyemit`]. No
//! transcript or diagnostic ever carries terminal bytes: every error surface
//! is a fixed reason string.
//!
//! EINTR is retried wherever CPython's PEP 475 would retry; EAGAIN on the
//! relay descriptors surfaces as `stream_failed`, matching the `.py` which
//! raises on any unexpected `BlockingIOError` there.

use std::sync::atomic::{AtomicBool, Ordering};

use crate::account::Account;
use crate::b64;
use crate::proto;
use crate::pty_io::{read_fd, set_nonblocking, wait_readable, write_all_blocking, write_fd};
use crate::pty_process::{end_child, set_size, spawn_login_pty};
use crate::pty_relay::relay_pty_session;
use crate::pyemit;
use crate::sys;

#[link(name = "util")]
extern "C" {}

/// Stop flag set by SIGTERM/SIGHUP/SIGINT during the relay.
pub(crate) static STOPPED: AtomicBool = AtomicBool::new(false);

extern "C" fn stop_handler(_signal: libc::c_int) {
    STOPPED.store(true, Ordering::SeqCst);
}

fn install_stop_handlers() -> Vec<(libc::c_int, libc::sigaction)> {
    let mut saved = Vec::new();
    for signal in [libc::SIGTERM, libc::SIGHUP, libc::SIGINT] {
        let mut action: libc::sigaction = unsafe { std::mem::zeroed() };
        action.sa_sigaction = stop_handler as *const () as libc::sighandler_t;
        unsafe {
            libc::sigemptyset(&mut action.sa_mask);
        }
        action.sa_flags = 0; // No SA_RESTART: signals interrupt select.
        let mut old: libc::sigaction = unsafe { std::mem::zeroed() };
        if unsafe { libc::sigaction(signal, &action, &mut old) } == 0 {
            saved.push((signal, old));
        }
    }
    saved
}

fn restore_handlers(saved: &[(libc::c_int, libc::sigaction)]) {
    for (signal, old) in saved {
        unsafe {
            libc::sigaction(*signal, old, std::ptr::null_mut());
        }
    }
}

/// `{"type":"ready"}` line.
pub fn ready_line() -> Vec<u8> {
    pyemit::line(&crate::state_json::StateValue::Object(vec![(
        "type".to_string(),
        crate::state_json::StateValue::Str("ready".to_string()),
    )]))
}

/// `{"type":"output","data":<base64>}` line.
pub fn output_line(data: &[u8]) -> Vec<u8> {
    pyemit::line(&crate::state_json::StateValue::Object(vec![
        (
            "type".to_string(),
            crate::state_json::StateValue::Str("output".to_string()),
        ),
        (
            "data".to_string(),
            crate::state_json::StateValue::Str(b64::encode(data)),
        ),
    ]))
}

/// `{"type":"closed","reason":<reason>}` line.
pub fn closed_line(reason: &str) -> Vec<u8> {
    pyemit::line(&crate::state_json::StateValue::Object(vec![
        (
            "type".to_string(),
            crate::state_json::StateValue::Str("closed".to_string()),
        ),
        (
            "reason".to_string(),
            crate::state_json::StateValue::Str(reason.to_string()),
        ),
    ]))
}

/// Best-effort bounded status delivery after teardown (no transcript).
pub fn flush_closed(outgoing: &mut Vec<u8>) {
    let _ = set_nonblocking(1, true);
    let until = sys::monotonic() + 0.5;
    while !outgoing.is_empty() && sys::monotonic() < until {
        let remaining = (until - sys::monotonic()).max(0.0);
        let mut write_set: libc::fd_set = unsafe { std::mem::zeroed() };
        unsafe {
            libc::FD_ZERO(&mut write_set);
            libc::FD_SET(1, &mut write_set);
        }
        let mut wait = libc::timeval {
            tv_sec: remaining.floor() as libc::time_t,
            tv_usec: ((remaining - remaining.floor()) * 1_000_000.0) as libc::suseconds_t,
        };
        let rc = unsafe {
            libc::select(
                2,
                std::ptr::null_mut(),
                &mut write_set,
                std::ptr::null_mut(),
                &mut wait,
            )
        };
        if rc <= 0 {
            break;
        }
        match write_fd(1, outgoing) {
            Ok(wrote) => {
                outgoing.drain(..wrote);
            }
            Err(_) => break,
        }
    }
}

/// Attach a PTY to the tmux socket; returns the process exit code.
pub fn run_terminal(account: &Account, cols: i64, rows: i64, seconds: i64, sock: &str) -> i32 {
    STOPPED.store(false, Ordering::SeqCst);
    // Bounds failures happen before the spawn in the `.py`, so `main`
    // reports them as `launch_failed` — mirror that here.
    if !proto::dimensions(cols, rows) || !(1..=43200).contains(&seconds) {
        let _ = write_all_blocking(1, &closed_line("launch_failed"));
        return 1;
    }
    let (pid, master, ready_read, gate_write) = match spawn_login_pty(account, sock) {
        Ok(endpoints) => endpoints,
        Err(_) => {
            let _ = write_all_blocking(1, &closed_line("launch_failed"));
            return 1;
        }
    };
    let mut stopped = false;
    let mut outgoing = Vec::new();
    let mut gate_open = true;
    let old_handlers = install_stop_handlers();
    let outcome: Result<String, ()> = (|| {
        set_size(master, cols, rows).map_err(|_| ())?;
        write_all_blocking(gate_write, b"1").map_err(|_| ())?;
        unsafe {
            libc::close(gate_write);
        }
        gate_open = false;
        let ready = wait_readable(ready_read, 5.0).map_err(|_| ())?;
        let mut probe = [0u8; 16];
        // Blocking read: EOF (empty) proves exec via CLOEXEC.
        let got = read_fd(ready_read, &mut probe).map_err(|_| ())?;
        if !ready || got > 0 {
            return Ok("launch_failed".to_string());
        }
        set_nonblocking(0, true).map_err(|_| ())?;
        set_nonblocking(1, true).map_err(|_| ())?;
        set_nonblocking(master, true).map_err(|_| ())?;
        relay_pty_session(pid, master, seconds, &mut stopped, &mut outgoing).map_err(|_| ())
    })();
    let reason = match outcome {
        Ok(reason) => reason,
        Err(()) => "stream_failed".to_string(),
    };
    unsafe {
        libc::close(ready_read);
        if gate_open {
            libc::close(gate_write);
        }
    }
    let status = end_child(pid, master);
    restore_handlers(&old_handlers);
    let delivered = if status.is_some() {
        reason.as_str()
    } else {
        "cleanup_unconfirmed"
    };
    outgoing.extend_from_slice(&closed_line(delivered));
    flush_closed(&mut outgoing);
    if status.is_some() && reason != "launch_failed" && reason != "stream_failed" {
        0
    } else {
        1
    }
}

#[cfg(test)]
#[path = "pty_tests.rs"]
mod pty_tests;
