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
    use tungstenite::{Error as WsError, Message};

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
    let first = loop {
        if shutdown.load(Ordering::Acquire) {
            return Ok(());
        }
        let remaining = first_deadline.saturating_duration_since(Instant::now());
        if remaining.is_zero() {
            return Ok(());
        }
        match ws.read() {
            Ok(Message::Text(text)) if text.len() <= TERMINAL_REQUEST_LIMIT => {
                break text.as_bytes().to_vec()
            }
            Ok(Message::Ping(_)) => {
                let _ = ws.flush();
            }
            Ok(Message::Pong(_)) => {}
            Err(WsError::Io(ref e)) if e.kind() == std::io::ErrorKind::WouldBlock => {}
            Ok(_) | Err(_) => return Ok(()),
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
            return Ok(());
        }
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
    let container = match service.project_container(&request.project, true, inspect_deadline) {
        Ok(cid) => cid,
        Err(_) => {
            let _ = ws.send(Message::Text(closed_frame("launch_failed").encode().into()));
            return Ok(());
        }
    };
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
        return Ok(());
    }
    if shutdown.load(Ordering::Acquire) {
        return Ok(());
    }
    // NativeAttach::attach is a synchronous spawn outside the shared
    // one-shot deadline; shutdown custody for that child remains separate.
    let attach = match terminal::NativeAttach::attach(&container, &request, Arc::clone(&shutdown)) {
        Ok(attach) => attach,
        Err(_) => {
            let _ = ws.send(Message::Text(closed_frame("launch_failed").encode().into()));
            return Ok(());
        }
    };
    pump_attached(ws, attach, session_deadline, shutdown)
}

fn pump_attached<S>(
    mut ws: tungstenite::protocol::WebSocket<S>,
    mut attach: terminal::NativeAttach,
    deadline: Instant,
    shutdown: Arc<std::sync::atomic::AtomicBool>,
) -> Result<(), String>
where
    S: std::io::Read + std::io::Write + std::os::fd::AsRawFd + Send + 'static,
{
    use crate::daemon::admission::TERMINAL_FRAME_LIMIT;
    use std::io::Read;
    use std::io::Write;
    use std::os::fd::AsRawFd;
    use std::sync::atomic::{AtomicBool, Ordering};
    use std::sync::mpsc::{self, TryRecvError, TrySendError};
    use tungstenite::{Error as WsError, Message};
    let socket_fd = ws.get_ref().as_raw_fd();
    let flags = unsafe { libc::fcntl(socket_fd, libc::F_GETFL) };
    if flags < 0 || unsafe { libc::fcntl(socket_fd, libc::F_SETFL, flags | libc::O_NONBLOCK) } < 0 {
        let close_error = attach.close().err();
        if let Some(error) = close_error {
            attach.retain_child_until_exit();
            return Err(format!("terminal transport ended; {error}"));
        }
        return Err("terminal transport ended".to_string());
    }
    let mut reader = match attach.take_reader() {
        Some(reader) => reader,
        None => {
            let close_error = attach.close().err();
            if let Some(error) = close_error {
                attach.retain_child_until_exit();
                return Err(format!("terminal output ended; {error}"));
            }
            return Err("terminal output ended".to_string());
        }
    };

    // A bounded child-output queue keeps a quiet or slow socket from pinning
    // the stdout reader. Cancellation is polled even if a descendant retains
    // the pipe after the direct child exits.
    let (out_tx, out_rx) = mpsc::sync_channel::<terminal::TerminalFrame>(8);
    let (wake_reader, mut wake_writer) =
        UnixStream::pair().map_err(|_| "terminal transport ended")?;
    wake_reader.set_nonblocking(true).ok();
    wake_writer.set_nonblocking(true).ok();
    let cancel_reader = Arc::new(AtomicBool::new(false));
    let worker_cancel = Arc::clone(&cancel_reader);
    let output = std::thread::spawn(move || {
        let mut line = Vec::with_capacity(1024);
        let mut buf = [0u8; 4096];
        while !worker_cancel.load(Ordering::Acquire) {
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
                            if let Ok(frame) = terminal::parse_output_line(&line) {
                                line.clear();
                                let mut frame = frame;
                                loop {
                                    if worker_cancel.load(Ordering::Acquire) {
                                        return;
                                    }
                                    match out_tx.try_send(frame) {
                                        Ok(()) => {
                                            let _ = wake_writer.write(&[1]);
                                            break;
                                        }
                                        Err(TrySendError::Full(returned)) => {
                                            frame = returned;
                                            std::thread::sleep(Duration::from_millis(5));
                                        }
                                        Err(TrySendError::Disconnected(_)) => return,
                                    }
                                }
                            } else {
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
    });

    let deadline = deadline;
    let mut pending_send = false;
    let mut pending_message: Option<Message> = None;
    let mut write_deadline = None;
    let mut closed = false;
    let mut close_received = false;
    // `read()` may have pulled later complete frames into tungstenite's own
    // buffer with the first text request. Drain that before sleeping on fd.
    let mut drain_buffered = true;
    while Instant::now() < deadline && !shutdown.load(Ordering::Acquire) {
        if write_deadline.is_some_and(|until| Instant::now() >= until) {
            break;
        }
        let mut pollfds = [
            libc::pollfd {
                fd: socket_fd,
                events: libc::POLLIN
                    | if pending_send || pending_message.is_some() {
                        libc::POLLOUT
                    } else {
                        0
                    },
                revents: 0,
            },
            libc::pollfd {
                fd: wake_reader.as_raw_fd(),
                events: libc::POLLIN,
                revents: 0,
            },
        ];
        let ready = if drain_buffered && !pending_send && pending_message.is_none() {
            1
        } else {
            unsafe { libc::poll(pollfds.as_mut_ptr(), pollfds.len() as _, 100) }
        };
        if ready < 0 && std::io::Error::last_os_error().kind() != std::io::ErrorKind::Interrupted {
            break;
        }
        if pollfds[0].revents & (libc::POLLERR | libc::POLLHUP | libc::POLLNVAL) != 0 {
            break;
        }
        if pollfds[1].revents & libc::POLLIN != 0 {
            let mut wake_bytes = [0u8; 64];
            let _ = wake_reader_read(&wake_reader, &mut wake_bytes);
        }
        if pending_message.is_none() && !pending_send {
            match out_rx.try_recv() {
                Ok(frame) => {
                    closed = frame.frame_type == "closed" || frame.frame_type == "metadata";
                    pending_message = Some(Message::Text(frame.encode().into()));
                    write_deadline = Some(Instant::now() + Duration::from_secs(5));
                }
                Err(TryRecvError::Disconnected) => break,
                Err(TryRecvError::Empty) => {}
            }
        }
        if pending_send {
            if pollfds[0].revents & libc::POLLOUT != 0 {
                match ws.flush() {
                    Ok(()) => {
                        pending_send = false;
                        if pending_message.is_none() {
                            write_deadline = None;
                        }
                    }
                    Err(WsError::Io(ref e)) if e.kind() == std::io::ErrorKind::WouldBlock => {}
                    Err(_) => break,
                }
            }
        }
        if !pending_send {
            if let Some(message) = pending_message.take() {
                write_deadline.get_or_insert_with(|| Instant::now() + Duration::from_secs(5));
                match ws.write(message) {
                    Ok(()) => pending_send = true,
                    Err(WsError::WriteBufferFull(message)) => {
                        pending_message = Some(*message);
                        pending_send = true;
                    }
                    // tungstenite retains the frame in its write buffer after
                    // an I/O WouldBlock; flush it later without resending it.
                    Err(WsError::Io(ref e))
                        if e.kind() == std::io::ErrorKind::WouldBlock
                            || e.kind() == std::io::ErrorKind::TimedOut =>
                    {
                        pending_send = true
                    }
                    Err(_) => break,
                }
            }
        }
        if (closed || close_received) && !pending_send && pending_message.is_none() {
            break;
        }
        if !drain_buffered && pollfds[0].revents & libc::POLLIN == 0 {
            continue;
        }
        match ws.read() {
            Ok(Message::Text(text)) if text.len() <= TERMINAL_FRAME_LIMIT => {
                let Ok(frame) = terminal::TerminalFrame::decode(text.as_bytes()) else {
                    break;
                };
                if !frame.input_valid() {
                    break;
                }
                let is_close = frame.frame_type == "close";
                if attach.input_frame(&frame).is_err() || is_close {
                    break;
                }
                drain_buffered = true;
            }
            Ok(Message::Ping(_)) => {
                pending_send = true;
                write_deadline.get_or_insert_with(|| Instant::now() + Duration::from_secs(5));
                drain_buffered = true;
            }
            Ok(Message::Pong(_)) => {
                drain_buffered = true;
            }
            Ok(Message::Close(_)) => {
                let _ = ws.close(None);
                pending_send = true;
                write_deadline.get_or_insert_with(|| Instant::now() + Duration::from_secs(5));
                close_received = true;
            }
            Ok(_) => break,
            Err(WsError::Io(ref e)) if e.kind() == std::io::ErrorKind::WouldBlock => {
                drain_buffered = false;
            }
            Err(_) => break,
        }
    }
    cancel_reader.store(true, Ordering::Release);
    drop(out_rx);
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

fn wake_reader_read(stream: &UnixStream, buf: &mut [u8]) -> usize {
    use std::io::Read;
    let mut stream = stream;
    stream.read(buf).unwrap_or(0)
}
