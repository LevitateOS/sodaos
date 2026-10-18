use std::os::unix::net::UnixStream;
use std::sync::Arc;
use std::time::{Duration, Instant};

use crate::{project, terminal};

use super::{native_deadline, DaemonBackend};

#[cfg(test)]
mod tests;

fn closed_frame(reason: &str) -> terminal::TerminalFrame {
    terminal::TerminalFrame {
        frame_type: "closed".into(),
        data: String::new(),
        cols: 0,
        rows: 0,
        reason: reason.into(),
        terminals: None,
    }
}

fn read_first_text_request(
    ws: &mut tungstenite::protocol::WebSocket<UnixStream>,
    shutdown: &Arc<std::sync::atomic::AtomicBool>,
    socket_fd: std::os::fd::RawFd,
    first_deadline: Instant,
) -> Option<Vec<u8>> {
    use crate::daemon::admission::TERMINAL_REQUEST_LIMIT;
    use std::sync::atomic::Ordering;
    use tungstenite::{Error as WsError, Message};

    loop {
        if shutdown.load(Ordering::Acquire) {
            return None;
        }
        let remaining = first_deadline.saturating_duration_since(Instant::now());
        if remaining.is_zero() {
            return None;
        }
        match ws.read() {
            Ok(Message::Text(text)) if text.len() <= TERMINAL_REQUEST_LIMIT => {
                break Some(text.as_bytes().to_vec())
            }
            Ok(Message::Ping(_)) => {
                let _ = ws.flush();
            }
            Ok(Message::Pong(_)) => {}
            Err(WsError::Io(ref e)) if e.kind() == std::io::ErrorKind::WouldBlock => {}
            Ok(_) | Err(_) => return None,
        }
        let mut pollfd = libc::pollfd {
            fd: socket_fd,
            events: libc::POLLIN,
            revents: 0,
        };
        let wait = first_deadline
            .saturating_duration_since(Instant::now())
            .as_millis()
            .min(100) as i32;
        if unsafe { libc::poll(&mut pollfd, 1, wait.max(1)) } < 0
            && std::io::Error::last_os_error().kind() != std::io::ErrorKind::Interrupted
        {
            return None;
        }
    }
}

struct AdmittedAttach {
    attach: terminal::NativeAttach,
    admission: project::admission::AdmissionGuard,
}

fn admit_terminal_session(
    service: &terminal::Service<project::Native>,
    backend: &DaemonBackend,
    ws: &mut tungstenite::protocol::WebSocket<UnixStream>,
    request: &terminal::TerminalRequest,
    operation_deadline: Instant,
    inspect_deadline: Instant,
    shutdown: &Arc<std::sync::atomic::AtomicBool>,
) -> Option<AdmittedAttach> {
    use tungstenite::Message;

    let container = match service.project_container(&request.project, true, inspect_deadline) {
        Ok(cid) => cid,
        Err(_) => {
            let _ = ws.send(Message::Text(closed_frame("launch_failed").encode().into()));
            return None;
        }
    };
    let admission = match project::admission::admit(Some(&request.project)) {
        Ok(guard) => guard,
        Err(reason) => {
            let code = if reason == project::admission::MAINTENANCE {
                "project_maintenance"
            } else {
                terminal::UNAVAILABLE
            };
            let _ = ws.send(Message::Text(closed_frame(code).encode().into()));
            return None;
        }
    };
    let support = service.support_in_container(&container, inspect_deadline);
    if let Some(reason) = support.failure_code() {
        let _ = ws.send(Message::Text(closed_frame(reason).encode().into()));
        return None;
    }
    let end_hook = |actor: i64, lease_id: &str| {
        backend
            .broker
            .end_lease(actor, lease_id, operation_deadline)
    };
    if service
        .managed_end(&container, &request, Some(&end_hook), operation_deadline)
        .is_err()
    {
        let _ = ws.send(Message::Text(
            closed_frame("cleanup_unconfirmed").encode().into(),
        ));
        return None;
    }
    // NativeAttach::attach is a synchronous spawn outside the shared
    // one-shot deadline; shutdown custody for that child remains separate.
    let attach = match terminal::NativeAttach::attach(&container, request, Arc::clone(shutdown)) {
        Ok(attach) => attach,
        Err(_) => {
            let _ = ws.send(Message::Text(closed_frame("launch_failed").encode().into()));
            return None;
        }
    };
    Some(AdmittedAttach { attach, admission })
}

/// Terminal attach loop. Tungstenite owns RFC6455 framing and the upgraded
/// connection; one thread owns the protocol for the entire session.
pub(super) fn pump_terminal(
    service: &terminal::Service<project::Native>,
    backend: &DaemonBackend,
    mut ws: tungstenite::protocol::WebSocket<UnixStream>,
    shutdown: Arc<std::sync::atomic::AtomicBool>,
) -> Result<(), String> {
    use crate::daemon::admission::{TERMINAL_FRAME_LIMIT, TERMINAL_REQUEST_LIMIT};
    use std::os::fd::AsRawFd;
    use std::sync::atomic::Ordering;
    use tungstenite::Message;

    let socket_fd = ws.get_ref().as_raw_fd();
    let flags = unsafe { libc::fcntl(socket_fd, libc::F_GETFL) };
    if flags < 0 || unsafe { libc::fcntl(socket_fd, libc::F_SETFL, flags | libc::O_NONBLOCK) } < 0 {
        return Err("terminal transport ended".to_string());
    }
    ws.set_config(|config| {
        config.max_message_size = Some(TERMINAL_REQUEST_LIMIT);
        config.max_frame_size = Some(TERMINAL_REQUEST_LIMIT);
    });
    let first_deadline = Instant::now() + Duration::from_secs(5);
    let Some(first) = read_first_text_request(&mut ws, &shutdown, socket_fd, first_deadline) else {
        return Ok(());
    };
    let request = match terminal::TerminalRequest::decode(&first).ok() {
        Some(request) if request.valid(terminal::now_unix()) => request,
        _ => return Ok(()),
    };
    ws.set_config(|config| {
        config.max_message_size = Some(TERMINAL_FRAME_LIMIT);
        config.max_frame_size = Some(TERMINAL_FRAME_LIMIT);
    });
    let expiry = Instant::now()
        + Duration::from_secs(request.expires.saturating_sub(terminal::now_unix()).max(0) as u64);
    let session_deadline = expiry;
    let operation_deadline = native_deadline().min(session_deadline);
    let inspect_deadline = (Instant::now() + Duration::from_secs(10)).min(operation_deadline);
    let Some(admitted) = admit_terminal_session(
        service,
        backend,
        &mut ws,
        &request,
        operation_deadline,
        inspect_deadline,
        &shutdown,
    ) else {
        return Ok(());
    };
    if shutdown.load(Ordering::Acquire) {
        return Ok(());
    }
    pump_attached_admitted(
        ws,
        admitted.attach,
        session_deadline,
        shutdown,
        Some(admitted.admission),
    )
}

#[cfg(test)]
fn pump_attached<S>(
    ws: tungstenite::protocol::WebSocket<S>,
    attach: terminal::NativeAttach,
    deadline: Instant,
    shutdown: Arc<std::sync::atomic::AtomicBool>,
) -> Result<(), String>
where
    S: std::io::Read + std::io::Write + std::os::fd::AsRawFd + Send + 'static,
{
    pump_attached_admitted(ws, attach, deadline, shutdown, None)
}

fn close_attached_error(attach: &mut terminal::NativeAttach, ended: &str) -> String {
    if let Some(error) = attach.close().err() {
        attach.retain_child_until_exit();
        return format!("{ended}; {error}");
    }
    ended.to_string()
}

fn admit_attached_transport<S>(
    ws: &tungstenite::protocol::WebSocket<S>,
    attach: &mut terminal::NativeAttach,
) -> Result<(std::os::fd::RawFd, std::io::BufReader<std::fs::File>), String>
where
    S: std::io::Read + std::io::Write + std::os::fd::AsRawFd,
{
    use std::os::fd::AsRawFd;

    let socket_fd = ws.get_ref().as_raw_fd();
    let flags = unsafe { libc::fcntl(socket_fd, libc::F_GETFL) };
    if flags < 0 || unsafe { libc::fcntl(socket_fd, libc::F_SETFL, flags | libc::O_NONBLOCK) } < 0 {
        return Err(close_attached_error(attach, "terminal transport ended"));
    }
    match attach.take_reader() {
        Some(reader) => Ok((socket_fd, reader)),
        None => Err(close_attached_error(attach, "terminal output ended")),
    }
}

fn emit_output_line(
    line: &mut Vec<u8>,
    out_tx: &std::sync::mpsc::SyncSender<terminal::TerminalFrame>,
    wake_writer: &mut UnixStream,
    cancel: &Arc<std::sync::atomic::AtomicBool>,
) -> bool {
    use std::io::Write;
    use std::sync::atomic::Ordering;
    use std::sync::mpsc::TrySendError;

    let Ok(frame) = terminal::parse_output_line(line) else {
        return false;
    };
    line.clear();
    let mut frame = frame;
    loop {
        if cancel.load(Ordering::Acquire) {
            return false;
        }
        match out_tx.try_send(frame) {
            Ok(()) => {
                let _ = wake_writer.write(&[1]);
                return true;
            }
            Err(TrySendError::Full(returned)) => {
                frame = returned;
                std::thread::sleep(Duration::from_millis(5));
            }
            Err(TrySendError::Disconnected(_)) => return false,
        }
    }
}

fn spawn_output_reader(
    mut reader: std::io::BufReader<std::fs::File>,
    cancel: Arc<std::sync::atomic::AtomicBool>,
    out_tx: std::sync::mpsc::SyncSender<terminal::TerminalFrame>,
    mut wake_writer: UnixStream,
) -> std::thread::JoinHandle<()> {
    use std::io::Read;
    use std::os::fd::AsRawFd;
    use std::sync::atomic::Ordering;

    std::thread::spawn(move || {
        let mut line = Vec::with_capacity(1024);
        let mut buf = [0u8; 4096];
        while !cancel.load(Ordering::Acquire) {
            let mut pollfd = libc::pollfd {
                fd: reader.get_ref().as_raw_fd(),
                events: libc::POLLIN,
                revents: 0,
            };
            let ready = unsafe { libc::poll(&mut pollfd, 1, 100) };
            if ready < 0 {
                if std::io::Error::last_os_error().kind() == std::io::ErrorKind::Interrupted {
                    continue;
                } else {
                    break;
                }
            }
            if ready == 0 {
                continue;
            }
            match reader.get_mut().read(&mut buf) {
                Ok(0) => break,
                Ok(n) => {
                    for &byte in &buf[..n] {
                        if byte == b'\n' {
                            if line.last() == Some(&b'\r') {
                                line.pop();
                            }
                            if !emit_output_line(&mut line, &out_tx, &mut wake_writer, &cancel) {
                                return;
                            }
                        } else {
                            line.push(byte);
                            if line.len() > 131072 {
                                return;
                            }
                        }
                    }
                }
                Err(e)
                    if e.kind() == std::io::ErrorKind::WouldBlock
                        || e.kind() == std::io::ErrorKind::Interrupted =>
                {
                    continue
                }
                Err(_) => break,
            }
        }
    })
}

struct AttachedPump {
    pending_send: bool,
    pending_message: Option<tungstenite::Message>,
    write_deadline: Option<Instant>,
    closed: bool,
    close_received: bool,
    drain_buffered: bool,
}

fn forward_queued_frame(
    out_rx: &std::sync::mpsc::Receiver<terminal::TerminalFrame>,
    admission: &mut Option<project::admission::AdmissionGuard>,
    pump: &mut AttachedPump,
) -> bool {
    use std::sync::mpsc::TryRecvError;

    if pump.pending_message.is_none() && !pump.pending_send {
        match out_rx.try_recv() {
            Ok(frame) => {
                if frame.frame_type == "ready" || frame.frame_type == "metadata" {
                    drop(admission.take());
                }
                pump.closed = frame.frame_type == "closed" || frame.frame_type == "metadata";
                pump.pending_message = Some(tungstenite::Message::Text(frame.encode().into()));
                pump.write_deadline = Some(Instant::now() + Duration::from_secs(5));
            }
            Err(TryRecvError::Disconnected) => return true,
            Err(TryRecvError::Empty) => {}
        }
    }
    false
}

fn flush_pending_send<S>(
    ws: &mut tungstenite::protocol::WebSocket<S>,
    pump: &mut AttachedPump,
    writable: bool,
) -> bool
where
    S: std::io::Read + std::io::Write,
{
    use tungstenite::Error as WsError;

    if pump.pending_send && writable {
        match ws.flush() {
            Ok(()) => {
                pump.pending_send = false;
                if pump.pending_message.is_none() {
                    pump.write_deadline = None;
                }
            }
            Err(WsError::Io(ref e)) if e.kind() == std::io::ErrorKind::WouldBlock => {}
            Err(_) => return true,
        }
    }
    false
}

fn write_pending_message<S>(
    ws: &mut tungstenite::protocol::WebSocket<S>,
    pump: &mut AttachedPump,
) -> bool
where
    S: std::io::Read + std::io::Write,
{
    use tungstenite::Error as WsError;

    if !pump.pending_send {
        if let Some(message) = pump.pending_message.take() {
            pump.write_deadline
                .get_or_insert_with(|| Instant::now() + Duration::from_secs(5));
            match ws.write(message) {
                Ok(()) => pump.pending_send = true,
                Err(WsError::WriteBufferFull(message)) => {
                    pump.pending_message = Some(*message);
                    pump.pending_send = true;
                }
                // tungstenite retains the frame in its write buffer after
                // an I/O WouldBlock; flush it later without resending it.
                Err(WsError::Io(ref e))
                    if e.kind() == std::io::ErrorKind::WouldBlock
                        || e.kind() == std::io::ErrorKind::TimedOut =>
                {
                    pump.pending_send = true
                }
                Err(_) => return true,
            }
        }
    }
    false
}

fn read_incoming_frame<S>(
    ws: &mut tungstenite::protocol::WebSocket<S>,
    attach: &mut terminal::NativeAttach,
    pump: &mut AttachedPump,
) -> bool
where
    S: std::io::Read + std::io::Write,
{
    use crate::daemon::admission::TERMINAL_FRAME_LIMIT;
    use tungstenite::{Error as WsError, Message};

    match ws.read() {
        Ok(Message::Text(text)) if text.len() <= TERMINAL_FRAME_LIMIT => {
            let Ok(frame) = terminal::TerminalFrame::decode(text.as_bytes()) else {
                return true;
            };
            if !frame.input_valid() {
                return true;
            }
            let is_close = frame.frame_type == "close";
            if attach.input_frame(&frame).is_err() || is_close {
                return true;
            }
            pump.drain_buffered = true;
        }
        Ok(Message::Ping(_)) => {
            pump.pending_send = true;
            pump.write_deadline
                .get_or_insert_with(|| Instant::now() + Duration::from_secs(5));
            pump.drain_buffered = true;
        }
        Ok(Message::Pong(_)) => {
            pump.drain_buffered = true;
        }
        Ok(Message::Close(_)) => {
            let _ = ws.close(None);
            pump.pending_send = true;
            pump.write_deadline
                .get_or_insert_with(|| Instant::now() + Duration::from_secs(5));
            pump.close_received = true;
        }
        Ok(_) => return true,
        Err(WsError::Io(ref e)) if e.kind() == std::io::ErrorKind::WouldBlock => {
            pump.drain_buffered = false;
        }
        Err(_) => return true,
    }
    false
}

fn poll_attached_sockets(
    socket_fd: std::os::fd::RawFd,
    wake_reader: &UnixStream,
    pump: &AttachedPump,
) -> Option<(i16, i16)> {
    let mut pollfds = [
        libc::pollfd {
            fd: socket_fd,
            events: libc::POLLIN
                | if pump.pending_send || pump.pending_message.is_some() {
                    libc::POLLOUT
                } else {
                    0
                },
            revents: 0,
        },
        libc::pollfd {
            fd: {
                use std::os::fd::AsRawFd;
                wake_reader.as_raw_fd()
            },
            events: libc::POLLIN,
            revents: 0,
        },
    ];
    let ready = if pump.drain_buffered && !pump.pending_send && pump.pending_message.is_none() {
        1
    } else {
        unsafe { libc::poll(pollfds.as_mut_ptr(), pollfds.len() as _, 100) }
    };
    if ready < 0 && std::io::Error::last_os_error().kind() != std::io::ErrorKind::Interrupted {
        return None;
    }
    if pollfds[0].revents & (libc::POLLERR | libc::POLLHUP | libc::POLLNVAL) != 0 {
        return None;
    }
    if pollfds[1].revents & libc::POLLIN != 0 {
        let mut wake_bytes = [0u8; 64];
        let _ = wake_reader_read(wake_reader, &mut wake_bytes);
    }
    Some((pollfds[0].revents, pollfds[1].revents))
}

fn finish_attached_pump<S>(
    ws: &mut tungstenite::protocol::WebSocket<S>,
    attach: &mut terminal::NativeAttach,
    output: std::thread::JoinHandle<()>,
    socket_fd: std::os::fd::RawFd,
    deadline: Instant,
) -> Result<(), String>
where
    S: std::io::Read + std::io::Write,
{
    use tungstenite::Error as WsError;

    let close_result = attach.close();
    let output_result = output
        .join()
        .map_err(|_| "terminal output reader panicked".to_string());
    let _ = ws.close(None);
    let close_deadline = (Instant::now() + Duration::from_millis(250)).min(deadline);
    while Instant::now() < close_deadline {
        let mut pfd = libc::pollfd {
            fd: socket_fd,
            events: libc::POLLOUT,
            revents: 0,
        };
        let ready = unsafe { libc::poll(&mut pfd, 1, 25) };
        if ready <= 0 {
            continue;
        }
        match ws.flush() {
            Ok(()) => break,
            Err(WsError::Io(ref e)) if e.kind() == std::io::ErrorKind::WouldBlock => continue,
            Err(_) => break,
        }
    }
    match (close_result, output_result) {
        (Err(close_error), Err(output_error)) => {
            // Keep the same pump task as Child owner until wait confirms exit.
            // The host's absolute shutdown deadline bounds this retained task.
            attach.retain_child_until_exit();
            Err(format!("{close_error}; {output_error}"))
        }
        (Err(close_error), Ok(())) => {
            attach.retain_child_until_exit();
            Err(close_error)
        }
        (Ok(()), Err(output_error)) => Err(output_error),
        (Ok(()), Ok(())) => Ok(()),
    }
}

fn pump_attached_admitted<S>(
    mut ws: tungstenite::protocol::WebSocket<S>,
    mut attach: terminal::NativeAttach,
    deadline: Instant,
    shutdown: Arc<std::sync::atomic::AtomicBool>,
    mut admission: Option<project::admission::AdmissionGuard>,
) -> Result<(), String>
where
    S: std::io::Read + std::io::Write + std::os::fd::AsRawFd + Send + 'static,
{
    use std::os::fd::AsRawFd;
    use std::sync::atomic::{AtomicBool, Ordering};
    use std::sync::mpsc;

    let (socket_fd, reader) = admit_attached_transport(&ws, &mut attach)?;

    // A bounded child-output queue keeps a quiet or slow socket from pinning
    // the stdout reader. Cancellation is polled even if a descendant retains
    // the pipe after the direct child exits.
    let (out_tx, out_rx) = mpsc::sync_channel::<terminal::TerminalFrame>(8);
    let (wake_reader, wake_writer) = UnixStream::pair().map_err(|_| "terminal transport ended")?;
    wake_reader.set_nonblocking(true).ok();
    wake_writer.set_nonblocking(true).ok();
    let cancel_reader = Arc::new(AtomicBool::new(false));
    let output = spawn_output_reader(reader, Arc::clone(&cancel_reader), out_tx, wake_writer);

    let mut pump = AttachedPump {
        pending_send: false,
        pending_message: None,
        write_deadline: None,
        closed: false,
        close_received: false,
        // `read()` may have pulled later complete frames into tungstenite's own
        // buffer with the first text request. Drain that before sleeping on fd.
        drain_buffered: true,
    };
    while Instant::now() < deadline && !shutdown.load(Ordering::Acquire) {
        if pump
            .write_deadline
            .is_some_and(|until| Instant::now() >= until)
        {
            break;
        }
        let Some((socket_events, _)) = poll_attached_sockets(socket_fd, &wake_reader, &pump) else {
            break;
        };
        if forward_queued_frame(&out_rx, &mut admission, &mut pump) {
            break;
        }
        if flush_pending_send(&mut ws, &mut pump, socket_events & libc::POLLOUT != 0) {
            break;
        }
        if write_pending_message(&mut ws, &mut pump) {
            break;
        }
        if (pump.closed || pump.close_received)
            && !pump.pending_send
            && pump.pending_message.is_none()
        {
            break;
        }
        if !pump.drain_buffered && socket_events & libc::POLLIN == 0 {
            continue;
        }
        if read_incoming_frame(&mut ws, &mut attach, &mut pump) {
            break;
        }
    }
    cancel_reader.store(true, Ordering::Release);
    drop(out_rx);
    finish_attached_pump(&mut ws, &mut attach, output, socket_fd, deadline)
}

fn wake_reader_read(stream: &UnixStream, buf: &mut [u8]) -> usize {
    use std::io::Read;
    let mut stream = stream;
    stream.read(buf).unwrap_or(0)
}
