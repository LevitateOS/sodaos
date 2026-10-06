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
use crate::proto::{self, ControlFrame};
use crate::pty_process::{child_exited, end_child, set_size, spawn_login_pty};
use crate::pyemit;
use crate::sys;

#[link(name = "util")]
extern "C" {}

/// Stop flag set by SIGTERM/SIGHUP/SIGINT during the relay.
static STOPPED: AtomicBool = AtomicBool::new(false);

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

fn read_fd(fd: i32, buf: &mut [u8]) -> io::Result<usize> {
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

fn write_fd(fd: i32, buf: &[u8]) -> io::Result<usize> {
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

/// Apply one decoded control frame; `Ok(Some(lease))` renews the heartbeat.
pub fn apply_control_frame(
    frame: &ControlFrame,
    master: i32,
    to_pty: &mut Vec<u8>,
    stopped: &mut bool,
) -> Result<Option<f64>, String> {
    match frame {
        ControlFrame::Close => {
            *stopped = true;
            Ok(None)
        }
        ControlFrame::Heartbeat => Ok(Some(sys::monotonic() + proto::HEARTBEAT_SECONDS as f64)),
        ControlFrame::Resize { cols, rows } => {
            set_size(master, *cols, *rows).map_err(|e| e.to_string())?;
            Ok(None)
        }
        ControlFrame::Input(data) => {
            if to_pty.len() + data.len() <= proto::QUEUE_LIMIT {
                to_pty.extend_from_slice(data);
                Ok(None)
            } else {
                Err("input queue full".to_string())
            }
        }
    }
}

/// Ingest complete control lines; returns the (possibly renewed) lease.
pub fn ingest_control_bytes(
    incoming: &mut Vec<u8>,
    master: i32,
    to_pty: &mut Vec<u8>,
    lease: f64,
    stopped: &mut bool,
) -> Result<f64, String> {
    let mut lease = lease;
    while let Some(pos) = incoming.iter().position(|b| *b == b'\n') {
        let line: Vec<u8> = incoming.drain(..=pos).collect();
        let raw = &line[..line.len() - 1];
        if raw.len() > proto::FRAME_LIMIT {
            return Err("frame size".to_string());
        }
        let frame = proto::decode_frame(raw).ok_or_else(|| "unsupported control".to_string())?;
        if let Some(renewed) = apply_control_frame(&frame, master, to_pty, stopped)? {
            lease = renewed;
        }
        if *stopped {
            break;
        }
    }
    if incoming.len() > proto::FRAME_LIMIT {
        return Err("frame size".to_string());
    }
    Ok(lease)
}

/// Read one PTY chunk; the first output also emits `ready`. Returns
/// `(Some("exited"), _)` on EOF/EIO.
pub fn read_pty_output(
    master: i32,
    outgoing: &mut Vec<u8>,
    attaching: bool,
) -> Result<(Option<String>, bool), String> {
    let mut buf = [0u8; 4096];
    let got = match read_fd(master, &mut buf) {
        Ok(got) => got,
        Err(err) if err.raw_os_error() == Some(libc::EIO) => 0,
        Err(err) => return Err(err.to_string()),
    };
    if got == 0 {
        return Ok((Some("exited".to_string()), attaching));
    }
    let mut attaching = attaching;
    if attaching {
        // exec success is not tmux input readiness: wait for the first
        // output (retained) before forwarding any queued input.
        outgoing.extend_from_slice(&ready_line());
        attaching = false;
    }
    outgoing.extend_from_slice(&output_line(&buf[..got]));
    Ok((None, attaching))
}

/// Pure stop rule: `(halt, reason, deadline)`.
pub fn session_should_stop(
    now: f64,
    deadline: f64,
    lease: f64,
    attaching: bool,
    attach_deadline: f64,
    exited: bool,
    reason: &str,
) -> (bool, String, f64) {
    if now >= deadline.min(lease) {
        let reason = if reason != "exited" {
            "expired".to_string()
        } else {
            reason.to_string()
        };
        return (true, reason, deadline);
    }
    if attaching && now >= attach_deadline {
        return (true, "launch_failed".to_string(), deadline);
    }
    if reason != "exited" && exited {
        return (false, "exited".to_string(), deadline.min(now + 0.5));
    }
    (false, reason.to_string(), deadline)
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

/// Read control stdin; `Some("eof"|"stop")` ends the relay.
pub fn take_control_input(
    incoming: &mut Vec<u8>,
    master: i32,
    to_pty: &mut Vec<u8>,
    lease: f64,
    stopped: &mut bool,
) -> Result<(f64, Option<String>), String> {
    let mut buf = [0u8; 4096];
    let got = read_fd(0, &mut buf).map_err(|e| e.to_string())?;
    if got == 0 {
        return Ok((lease, Some("eof".to_string())));
    }
    incoming.extend_from_slice(&buf[..got]);
    let lease = ingest_control_bytes(incoming, master, to_pty, lease, stopped)?;
    Ok((
        lease,
        if *stopped {
            Some("stop".to_string())
        } else {
            None
        },
    ))
}

/// Flush ready queues (partial writes retained).
pub fn flush_pty_queues(
    master: i32,
    to_pty: &mut Vec<u8>,
    outgoing: &mut Vec<u8>,
    writable: &[i32],
) -> Result<(), String> {
    if writable.contains(&master) && !to_pty.is_empty() {
        let wrote = write_fd(master, to_pty).map_err(|e| e.to_string())?;
        to_pty.drain(..wrote);
    }
    if writable.contains(&1) && !outgoing.is_empty() {
        let wrote = write_fd(1, outgoing).map_err(|e| e.to_string())?;
        outgoing.drain(..wrote);
    }
    Ok(())
}

/// Main relay; returns the close reason.
pub fn relay_pty_session(
    pid: i32,
    master: i32,
    seconds: i64,
    stopped: &mut bool,
    outgoing: &mut Vec<u8>,
) -> Result<String, String> {
    let mut incoming = Vec::new();
    let mut to_pty = Vec::new();
    let mut attaching = true;
    let attach_deadline = sys::monotonic() + 5.0;
    let mut deadline = sys::monotonic() + seconds as f64;
    let mut lease = sys::monotonic() + proto::HEARTBEAT_SECONDS as f64;
    let mut reason = "disconnected".to_string();
    while !*stopped && !STOPPED.load(Ordering::SeqCst) {
        let now = sys::monotonic();
        let exited = child_exited(pid).map_err(|e| e.to_string())?;
        let (halt, updated, moved) = session_should_stop(
            now,
            deadline,
            lease,
            attaching,
            attach_deadline,
            exited,
            &reason,
        );
        reason = updated;
        deadline = moved;
        if halt {
            return Ok(reason);
        }
        let timeout = 0.1f64.min(deadline - now).min(lease - now);
        let (readable, writable) =
            match pty_select(master, to_pty.len(), outgoing.len(), attaching, timeout) {
                Ok(sets) => sets,
                Err(err) if is_eintr(&err) => continue,
                Err(err) => return Err(err.to_string()),
            };
        if readable.contains(&0) {
            let (renewed, event) =
                take_control_input(&mut incoming, master, &mut to_pty, lease, stopped)?;
            lease = renewed;
            if event.is_some() {
                break;
            }
        }
        if readable.contains(&master) {
            let (ended, now_attaching) = read_pty_output(master, outgoing, attaching)?;
            attaching = now_attaching;
            if let Some(done) = ended {
                return Ok(done);
            }
        }
        flush_pty_queues(master, &mut to_pty, outgoing, &writable)?;
    }
    Ok(reason)
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
