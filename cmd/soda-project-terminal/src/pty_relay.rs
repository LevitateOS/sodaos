use std::sync::atomic::Ordering;

use crate::proto::{self, ControlFrame};
use crate::pty::{output_line, ready_line, STOPPED};
use crate::pty_io::{is_eintr, pty_select, read_fd, write_fd};
use crate::pty_process::{child_exited, set_size};
use crate::sys;

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
