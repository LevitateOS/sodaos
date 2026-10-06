use std::path::Path;
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::Instant;

use super::{
    geteuid, native_import, require_native, signal, Payload, IMAGES_PATH, IMPORT_TIMEOUT, PODMAN,
    RELEASE_PATH, SIGINT, SIGTERM,
};

pub(super) fn run() -> i32 {
    // Never panic on non-UTF-8 argv; only the count matters here.
    let argc = std::env::args_os().count();
    if let Err(msg) = admit(unsafe { geteuid() }, argc) {
        eprintln!("{msg}");
        return 1;
    }
    let payload = match Payload::load(Path::new(RELEASE_PATH)) {
        Ok(p) => p,
        Err(_) => {
            eprintln!("appliance release metadata unavailable");
            return 1;
        }
    };
    if let Err(e) = require_native(&payload.architecture) {
        eprintln!("{e}");
        return 1;
    }
    install_cancel_handlers();
    let ctx = ImportCtx {
        deadline: Instant::now() + IMPORT_TIMEOUT,
        cancelled: &CANCELLED,
    };
    match native_import(&payload, IMAGES_PATH, PODMAN, &ctx) {
        Ok(()) => 0,
        Err(e) => {
            eprintln!("{e}");
            1
        }
    }
}

pub(super) fn admit(euid: u32, argc: usize) -> Result<(), &'static str> {
    if euid != 0 || argc != 1 {
        return Err("root and no arguments required");
    }
    Ok(())
}

static CANCELLED: AtomicBool = AtomicBool::new(false);

extern "C" fn on_cancel(_sig: i32) {
    CANCELLED.store(true, Ordering::SeqCst);
}

fn install_cancel_handlers() {
    unsafe {
        signal(SIGINT, on_cancel);
        signal(SIGTERM, on_cancel);
    }
}

/// Import deadline + cancellation, mirroring the Go signal/notify + timeout
/// context. Pre-checks report context text; a cancel/timeout during a podman
/// run kills the child and reports that call's generic failure, exactly as
/// the killed `exec.CommandContext` exit maps in Go.
pub(super) struct ImportCtx<'a> {
    pub(super) deadline: Instant,
    pub(super) cancelled: &'a AtomicBool,
}

impl ImportCtx<'_> {
    pub(super) fn check(&self) -> Result<(), String> {
        if self.cancelled.load(Ordering::SeqCst) {
            return Err("context canceled".to_string());
        }
        if Instant::now() >= self.deadline {
            return Err("context deadline exceeded".to_string());
        }
        Ok(())
    }
}
