use std::os::unix::net::UnixStream;
use std::sync::atomic::Ordering;
use std::sync::Arc;
use std::time::{Duration, Instant};

use crate::terminal;

use super::*;

struct ShortIo {
    inner: UnixStream,
    block_first_write: bool,
}

impl std::os::fd::AsRawFd for ShortIo {
    fn as_raw_fd(&self) -> std::os::fd::RawFd {
        self.inner.as_raw_fd()
    }
}
impl std::io::Read for ShortIo {
    fn read(&mut self, buf: &mut [u8]) -> std::io::Result<usize> {
        std::io::Read::read(&mut self.inner, buf)
    }
}
impl std::io::Write for ShortIo {
    fn write(&mut self, buf: &[u8]) -> std::io::Result<usize> {
        if self.block_first_write {
            self.block_first_write = false;
            return Err(std::io::ErrorKind::WouldBlock.into());
        }
        std::io::Write::write(&mut self.inner, &buf[..buf.len().min(3)])
    }
    fn flush(&mut self) -> std::io::Result<()> {
        std::io::Write::flush(&mut self.inner)
    }
}

fn synthetic_attach(
    script: &str,
) -> (
    terminal::NativeAttach,
    u32,
    Arc<std::sync::atomic::AtomicBool>,
) {
    let child = std::process::Command::new("/bin/sh")
        .arg("-c")
        .arg(script)
        .stdin(std::process::Stdio::piped())
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::null())
        .spawn()
        .expect("start synthetic stdio child");
    let pid = child.id();
    let shutdown = Arc::new(std::sync::atomic::AtomicBool::new(false));
    let attach = terminal::NativeAttach::from_child_for_test(child, Arc::clone(&shutdown)).unwrap();
    (attach, pid, shutdown)
}

#[test]
fn production_attached_pump_handles_ping_short_writes_and_flushes_closed_once() {
    use tungstenite::protocol::{Role, WebSocket};
    use tungstenite::Message;
    let (server, client) = UnixStream::pair().unwrap();
    client
        .set_read_timeout(Some(Duration::from_secs(3)))
        .unwrap();
    let server_ws = WebSocket::from_raw_socket(
        ShortIo {
            inner: server,
            block_first_write: true,
        },
        Role::Server,
        None,
    );
    let mut client_ws = WebSocket::from_raw_socket(client, Role::Client, None);
    let (attach, _, shutdown) =
        synthetic_attach("printf '%s\\n' '{\"type\":\"closed\",\"reason\":\"exited\"}'; read line");
    let pump_shutdown = Arc::clone(&shutdown);
    let pump = std::thread::spawn(move || {
        pump_attached(
            server_ws,
            attach,
            Instant::now() + Duration::from_secs(3),
            pump_shutdown,
        )
    });

    client_ws
        .send(Message::Ping(bytes::Bytes::from_static(b"ping")))
        .unwrap();
    let mut got_pong = false;
    let mut text_count = 0;
    for _ in 0..6 {
        match client_ws.read() {
            Err(tungstenite::Error::ConnectionClosed | tungstenite::Error::AlreadyClosed) => break,
            Err(tungstenite::Error::Io(ref e)) if e.kind() == std::io::ErrorKind::TimedOut => break,
            Err(error) => panic!("server websocket read failed: {error}"),
            Ok(message) => match message {
                Message::Pong(payload) => {
                    assert_eq!(payload, bytes::Bytes::from_static(b"ping"));
                    got_pong = true;
                }
                Message::Text(text) => {
                    assert_eq!(
                        terminal::TerminalFrame::decode(text.as_bytes())
                            .unwrap()
                            .frame_type,
                        "closed"
                    );
                    text_count += 1;
                }
                Message::Close(_) => break,
                other => panic!("unexpected websocket message: {other:?}"),
            },
        }
    }
    assert!(got_pong, "same protocol owner answers Ping");
    assert_eq!(
        text_count, 1,
        "WouldBlock retries must not duplicate a terminal frame"
    );
    pump.join().unwrap().unwrap();
}

#[test]
fn production_attached_pump_cancels_full_output_queue_and_reaps_child() {
    use tungstenite::protocol::{Role, WebSocket};
    let (server, _client) = UnixStream::pair().unwrap();
    let server_ws = WebSocket::from_raw_socket(server, Role::Server, None);
    let (attach, pid, shutdown) = synthetic_attach(
        "i=0; while [ $i -lt 20000 ]; do printf '%s\\n' '{\"type\":\"output\",\"data\":\"YQ==\"}'; i=$((i+1)); done; exec /bin/sleep 60",
    );
    let pump_shutdown = Arc::clone(&shutdown);
    let (done_tx, done_rx) = std::sync::mpsc::channel();
    let pump = std::thread::spawn(move || {
        let result = pump_attached(
            server_ws,
            attach,
            Instant::now() + Duration::from_secs(10),
            pump_shutdown,
        );
        let _ = done_tx.send(result);
    });
    std::thread::sleep(Duration::from_millis(200));
    shutdown.store(true, Ordering::Release);
    done_rx
        .recv_timeout(Duration::from_secs(5))
        .expect("cancelled pump joins despite slow peer")
        .unwrap();
    pump.join().unwrap();
    assert_eq!(
        unsafe { libc::kill(pid as i32, 0) },
        -1,
        "NativeAttach must kill and reap its direct child"
    );
}

#[test]
fn production_attached_pump_expires_stalled_websocket_write() {
    use std::os::fd::AsRawFd;
    use tungstenite::protocol::{Role, WebSocket};
    let (server, client) = UnixStream::pair().unwrap();
    let small_buffer: libc::c_int = 4096;
    assert_eq!(
        unsafe {
            libc::setsockopt(
                server.as_raw_fd(),
                libc::SOL_SOCKET,
                libc::SO_SNDBUF,
                &small_buffer as *const _ as *const libc::c_void,
                std::mem::size_of_val(&small_buffer) as libc::socklen_t,
            )
        },
        0
    );
    let (attach, pid, shutdown) = synthetic_attach(
        "i=0; while [ $i -lt 20000 ]; do printf '%s\\n' '{\"type\":\"output\",\"data\":\"YQ==\"}'; i=$((i+1)); done; exec /bin/sleep 60",
    );
    let ws = WebSocket::from_raw_socket(server, Role::Server, None);
    let started = Instant::now();
    pump_attached(
        ws,
        attach,
        Instant::now() + Duration::from_secs(9),
        shutdown,
    )
    .unwrap();
    let elapsed = started.elapsed();
    assert!(
        elapsed >= Duration::from_millis(4500),
        "stalled peer should consume the 5s write budget: {elapsed:?}"
    );
    assert!(
        elapsed < Duration::from_secs(10),
        "5s write budget plus 3s child grace and final flush should bound the pump: {elapsed:?}"
    );
    assert_eq!(
        unsafe { libc::kill(pid as i32, 0) },
        -1,
        "deadline cleanup must reap NativeAttach child"
    );
    drop(client); // Intentionally unread until pump expiry.
}

#[test]
fn native_attach_input_frame_expires_when_child_does_not_read() {
    use crate::terminal::TerminalFrame;
    let (mut attach, pid, _) = synthetic_attach("exec /bin/sleep 60");
    let frame = TerminalFrame {
        frame_type: "input".to_string(),
        data: "QUFB".repeat(5461),
        ..Default::default()
    };
    let started = Instant::now();
    let result = loop {
        match attach.input_frame(&frame) {
            Ok(()) => {}
            Err(error) => break error,
        }
    };
    let elapsed = started.elapsed();
    assert_eq!(result, "terminal input deadline exceeded");
    assert!(
        elapsed >= Duration::from_millis(1900),
        "stdin backpressure should use its 2s budget: {elapsed:?}"
    );
    attach.close().unwrap();
    assert_eq!(
        unsafe { libc::kill(pid as i32, 0) },
        -1,
        "input failure cleanup must reap child"
    );
}
