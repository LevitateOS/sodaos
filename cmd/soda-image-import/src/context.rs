use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::Instant;

use super::payload::load as load_payload;
use super::{
    ensure_images, geteuid, native_import, require_native, run_podman, run_podman_output, signal,
    ImageSet, Payload, IMPORT_TIMEOUT, PODMAN, RELEASE_PATH, SIGINT, SIGTERM,
};

/// Fixed CLI surface. Boot takes no arguments and acquires the six required
/// roles. The two trusted callers (Tailnet prepare, maintenance) invoke the
/// internal command with the immutable payload path and one of exactly two
/// sets. No image URL/digest/registry override exists.
pub(super) enum Invocation {
    Boot,
    Ensure { payload: PathBuf, set: ImageSet },
}

pub(super) const USAGE: &str =
    "soda-image-import ensure --payload ABSOLUTE_PAYLOAD_PATH --set required|tailnet";

pub(super) fn parse(argv: &[String]) -> Result<Invocation, &'static str> {
    if argv.len() == 1 {
        return Ok(Invocation::Boot);
    }
    if argv.len() == 6
        && argv[1] == "ensure"
        && argv[2] == "--payload"
        && argv[4] == "--set"
        && argv[3].starts_with('/')
    {
        return match ImageSet::parse(&argv[5]) {
            Some(set) => Ok(Invocation::Ensure {
                payload: PathBuf::from(&argv[3]),
                set,
            }),
            None => Err(USAGE),
        };
    }
    Err(USAGE)
}

pub(super) fn run() -> i32 {
    if unsafe { geteuid() } != 0 {
        eprintln!("root required");
        return 1;
    }
    let argv: Vec<String> = std::env::args_os()
        .map(|arg| arg.into_string().unwrap_or_default())
        .collect();
    let invocation = match parse(&argv) {
        Ok(invocation) => invocation,
        Err(msg) => {
            eprintln!("{msg}");
            return 1;
        }
    };
    install_cancel_handlers();
    let ctx = ImportCtx {
        deadline: Instant::now() + IMPORT_TIMEOUT,
        cancelled: &CANCELLED,
    };
    let result = match invocation {
        Invocation::Boot => match load_payload(Path::new(RELEASE_PATH)) {
            Ok(payload) => boot(&payload, &ctx),
            Err(_) => {
                eprintln!("appliance release metadata unavailable");
                return 1;
            }
        },
        Invocation::Ensure { payload, set } => match load_payload(&payload) {
            Ok(loaded) => selected(&loaded, set, &ctx),
            Err(_) => {
                eprintln!("appliance release metadata unavailable");
                return 1;
            }
        },
    };
    match result {
        Ok(()) => 0,
        Err(e) => {
            eprintln!("{e}");
            1
        }
    }
}

/// Boot acquisition: the six required roles, no Tailnet intent read.
fn boot(payload: &Payload, ctx: &ImportCtx) -> Result<(), String> {
    require_native(&payload.architecture)?;
    native_import(payload, PODMAN, ctx)
}

fn selected(payload: &Payload, set: ImageSet, ctx: &ImportCtx) -> Result<(), String> {
    require_native(&payload.architecture)?;
    ensure_images(payload, set, PODMAN, ctx, &mut |cmd, args| {
        if args.get(1).is_some_and(|op| op == "image")
            && args.get(2).is_some_and(|op| op == "inspect")
        {
            run_podman_output(cmd, args, ctx)
        } else {
            run_podman(cmd, args, ctx)
        }
    })
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
