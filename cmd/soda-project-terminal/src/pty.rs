//! PTY attach relay (port of `spawn_login_pty` … `run_terminal`).
//!
//! Byte-exact stdout lines (`ready`/`output`/`closed`) via [`pyemit`]. No
//! transcript or diagnostic ever carries terminal bytes: every error surface
//! is a fixed reason string.
//!
//! EINTR is retried wherever CPython's PEP 475 would retry; EAGAIN on the
//! relay descriptors surfaces as `stream_failed`, matching the `.py` which
//! raises on any unexpected `BlockingIOError` there.

use std::io;
use std::sync::atomic::{AtomicBool, Ordering};

use crate::account::Account;
use crate::b64;
use crate::proto;
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

fn set_nonblocking(fd: i32, nonblocking: bool) -> io::Result<()> {
    let flags = unsafe { libc::fcntl(fd, libc::F_GETFL) };
    if flags < 0 {
        return Err(io::Error::last_os_error());
    }
    let updated = if nonblocking {
        flags | libc::O_NONBLOCK
    } else {
        flags & !libc::O_NONBLOCK
    };
    if unsafe { libc::fcntl(fd, libc::F_SETFL, updated) } != 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(())
}

pub(crate) fn is_eintr(err: &io::Error) -> bool {
    err.raw_os_error() == Some(libc::EINTR)
}

pub(crate) fn read_fd(fd: i32, buf: &mut [u8]) -> io::Result<usize> {
    loop {
        let got = unsafe { libc::read(fd, buf.as_mut_ptr() as *mut libc::c_void, buf.len()) };
        if got < 0 {
            let err = io::Error::last_os_error();
            if is_eintr(&err) {
                continue;
            }
            return Err(err);
        }
        return Ok(got as usize);
    }
}

pub(crate) fn write_fd(fd: i32, buf: &[u8]) -> io::Result<usize> {
    loop {
        let got = unsafe { libc::write(fd, buf.as_ptr() as *const libc::c_void, buf.len()) };
        if got < 0 {
            let err = io::Error::last_os_error();
            if is_eintr(&err) {
                continue;
            }
            return Err(err);
        }
        return Ok(got as usize);
    }
}

fn write_all_blocking(fd: i32, mut buf: &[u8]) -> io::Result<()> {
    while !buf.is_empty() {
        let wrote = write_fd(fd, buf)?;
        if wrote == 0 {
            return Err(io::Error::other("short write"));
        }
        buf = &buf[wrote..];
    }
    Ok(())
}

/// `{"type":"ready"}` line.
pub fn ready_line() -> Vec<u8> {
    pyemit::line(&soda_json::JsonValue::Object(vec![(
        "type".to_string(),
        soda_json::JsonValue::Str("ready".to_string()),
    )]))
}

/// `{"type":"output","data":<base64>}` line.
pub fn output_line(data: &[u8]) -> Vec<u8> {
    pyemit::line(&soda_json::JsonValue::Object(vec![
        (
            "type".to_string(),
            soda_json::JsonValue::Str("output".to_string()),
        ),
        (
            "data".to_string(),
            soda_json::JsonValue::Str(b64::encode(data)),
        ),
    ]))
}

/// `{"type":"closed","reason":<reason>}` line.
pub fn closed_line(reason: &str) -> Vec<u8> {
    pyemit::line(&soda_json::JsonValue::Object(vec![
        (
            "type".to_string(),
            soda_json::JsonValue::Str("closed".to_string()),
        ),
        (
            "reason".to_string(),
            soda_json::JsonValue::Str(reason.to_string()),
        ),
    ]))
}

/// `select` with the `.py` backpressure sets; returns `(readable, writable)`.
pub fn pty_select(
    master: i32,
    to_pty_len: usize,
    outgoing_len: usize,
    attaching: bool,
    timeout: f64,
) -> io::Result<(Vec<i32>, Vec<i32>)> {
    let mut read_set: libc::fd_set = unsafe { std::mem::zeroed() };
    let mut write_set: libc::fd_set = unsafe { std::mem::zeroed() };
    unsafe {
        libc::FD_ZERO(&mut read_set);
        libc::FD_ZERO(&mut write_set);
    }
    let mut watch_read = Vec::new();
    let mut watch_write = Vec::new();
    if to_pty_len <= proto::QUEUE_LIMIT - 16384 {
        unsafe {
            libc::FD_SET(0, &mut read_set);
        }
        watch_read.push(0);
    }
    if outgoing_len <= proto::QUEUE_LIMIT - 8192 {
        unsafe {
            libc::FD_SET(master, &mut read_set);
        }
        watch_read.push(master);
    }
    if outgoing_len > 0 {
        unsafe {
            libc::FD_SET(1, &mut write_set);
        }
        watch_write.push(1);
    }
    if to_pty_len > 0 && !attaching {
        unsafe {
            libc::FD_SET(master, &mut write_set);
        }
        watch_write.push(master);
    }
    let clamped = timeout.max(0.0);
    let mut wait = libc::timeval {
        tv_sec: clamped.floor() as libc::time_t,
        tv_usec: ((clamped - clamped.floor()) * 1_000_000.0) as libc::suseconds_t,
    };
    let top = watch_read
        .iter()
        .chain(watch_write.iter())
        .copied()
        .max()
        .unwrap_or(0)
        + 1;
    let rc = unsafe {
        libc::select(
            top,
            &mut read_set,
            &mut write_set,
            std::ptr::null_mut(),
            &mut wait,
        )
    };
    if rc < 0 {
        return Err(io::Error::last_os_error());
    }
    let readable = watch_read
        .into_iter()
        .filter(|fd| unsafe { libc::FD_ISSET(*fd, &read_set) })
        .collect();
    let writable = watch_write
        .into_iter()
        .filter(|fd| unsafe { libc::FD_ISSET(*fd, &write_set) })
        .collect();
    Ok((readable, writable))
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

fn wait_readable(fd: i32, timeout: f64) -> io::Result<bool> {
    let until = sys::monotonic() + timeout;
    loop {
        let remaining = until - sys::monotonic();
        if remaining <= 0.0 {
            return Ok(false);
        }
        let mut read_set: libc::fd_set = unsafe { std::mem::zeroed() };
        unsafe {
            libc::FD_ZERO(&mut read_set);
            libc::FD_SET(fd, &mut read_set);
        }
        let mut wait = libc::timeval {
            tv_sec: remaining.floor() as libc::time_t,
            tv_usec: ((remaining - remaining.floor()) * 1_000_000.0) as libc::suseconds_t,
        };
        let rc = unsafe {
            libc::select(
                fd + 1,
                &mut read_set,
                std::ptr::null_mut(),
                std::ptr::null_mut(),
                &mut wait,
            )
        };
        if rc < 0 {
            let err = io::Error::last_os_error();
            if is_eintr(&err) {
                continue;
            }
            return Err(err);
        }
        return Ok(rc > 0);
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
