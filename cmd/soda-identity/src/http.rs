// Unix HTTP listener, extracted from http.rs (A05.M).
use crate::control::Controller;
use crate::http_routes::dispatch;
use crate::http_wire::{error_response, read_request, Admission};
use std::io::Write;
use std::os::unix::net::{UnixListener, UnixStream};
use std::sync::atomic::{AtomicBool, AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::Duration;
pub struct Server {
    controller: Arc<Controller>,
    runtime_allowed: bool,
    shutdown: Arc<AtomicBool>,
    inflight: Arc<AtomicUsize>,
}

impl Server {
    pub fn new(
        controller: Arc<Controller>,
        runtime_allowed: bool,
        shutdown: Arc<AtomicBool>,
        inflight: Arc<AtomicUsize>,
    ) -> Server {
        Server {
            controller,
            runtime_allowed,
            shutdown,
            inflight,
        }
    }

    pub fn serve(&self, listener: &UnixListener) {
        listener.set_nonblocking(true).ok();
        let mut fatal_errors = 0u32;
        loop {
            if self.shutdown.load(Ordering::SeqCst) {
                return;
            }
            match listener.accept() {
                Ok((stream, _)) => {
                    fatal_errors = 0;
                    self.inflight.fetch_add(1, Ordering::SeqCst);
                    let controller = Arc::clone(&self.controller);
                    let inflight = Arc::clone(&self.inflight);
                    let runtime_allowed = self.runtime_allowed;
                    std::thread::spawn(move || {
                        handle_connection(stream, &controller, runtime_allowed);
                        inflight.fetch_sub(1, Ordering::SeqCst);
                    });
                }
                Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {
                    fatal_errors = 0;
                    std::thread::sleep(Duration::from_millis(50));
                }
                Err(e)
                    if e.kind() == std::io::ErrorKind::Interrupted
                        || e.kind() == std::io::ErrorKind::ConnectionAborted =>
                {
                    // Transient or per-connection failure: keep serving.
                    fatal_errors = 0;
                    std::thread::sleep(Duration::from_millis(50));
                }
                Err(_) => {
                    if self.shutdown.load(Ordering::SeqCst) {
                        return;
                    }
                    fatal_errors += 1;
                    // Fail fast for supervisor restart, like Go's Serve on
                    // a fatal listener error, instead of spinning deaf.
                    if fatal_errors >= 20 {
                        return;
                    }
                    std::thread::sleep(Duration::from_millis(50));
                }
            }
        }
    }
}

fn handle_connection(mut stream: UnixStream, controller: &Controller, runtime_allowed: bool) {
    let response = match read_request(&mut stream) {
        Ok((request, body)) => dispatch(controller, request, body, runtime_allowed),
        Err(Admission::Invalid) => error_response(400, "invalid request"),
        Err(Admission::HeadersTooLarge) => error_response(431, "header too large"),
    };
    let _ = stream.write_all(&response);
}
