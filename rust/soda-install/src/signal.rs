//! Cancellation scopes mirroring Go's `signal.NotifyContext` trees: a
//! process-wide terminated flag (SIGTERM/SIGHUP), a SIGINT counter consumed
//! against per-scope baselines, and optional deadlines.

use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::Arc;
use std::time::Instant;

use crate::errors::Error;

static TERMINATED: AtomicBool = AtomicBool::new(false);
static SIGINT_COUNT: AtomicU64 = AtomicU64::new(0);

extern "C" fn handle_signal(signo: libc::c_int) {
    match signo {
        libc::SIGINT => {
            SIGINT_COUNT.fetch_add(1, Ordering::SeqCst);
        }
        _ => {
            TERMINATED.store(true, Ordering::SeqCst);
        }
    }
}

/// Install process signal handling exactly like `main.go`: SIGTERM/SIGHUP
/// always terminate the root scope; whether SIGINT cancels the root scope
/// is decided per action by the interruptibility of the root [`Ctx`].
pub fn install_handlers() {
    unsafe {
        let mut action: libc::sigaction = std::mem::zeroed();
        action.sa_sigaction = handle_signal as usize;
        action.sa_flags = 0; // No SA_RESTART: event polls surface EINTR.
        libc::sigemptyset(&mut action.sa_mask);
        for signo in [libc::SIGINT, libc::SIGTERM, libc::SIGHUP] {
            libc::sigaction(signo, &action, std::ptr::null_mut());
        }
    }
}

#[derive(Debug, Clone)]
enum CancelSource {
    Global,
    Owned(Arc<AtomicBool>),
}

/// Cancellation scope. Scopes share their parent's termination source and
/// deadline; interruptible scopes additionally observe SIGINT arrivals past
/// their baseline, mirroring nested `signal.NotifyContext(ctx, SIGINT)`.
#[derive(Debug, Clone)]
pub struct Ctx {
    source: CancelSource,
    deadline: Option<Instant>,
    interruptible: bool,
    sigint_base: u64,
}

impl Ctx {
    /// Root scope: terminated by SIGTERM/SIGHUP, and by SIGINT when the
    /// action is not `disk` (mirroring `main.go`'s signal set).
    pub fn root(interruptible: bool) -> Ctx {
        Ctx { source: CancelSource::Global, deadline: None, interruptible, sigint_base: SIGINT_COUNT.load(Ordering::SeqCst) }
    }

    /// Test scope with a private termination flag. Test scopes are not
    /// SIGINT-interruptible: the process-wide counter is shared with other
    /// parallel tests, so SIGINT behavior is covered by the dedicated scope
    /// test below instead.
    pub fn test() -> (Ctx, Arc<AtomicBool>) {
        let flag = Arc::new(AtomicBool::new(false));
        let ctx = Ctx {
            source: CancelSource::Owned(flag.clone()),
            deadline: None,
            interruptible: false,
            sigint_base: u64::MAX,
        };
        (ctx, flag)
    }

    /// Child scope observing SIGINT from this point on, mirroring
    /// `signal.NotifyContext(ctx, syscall.SIGINT)`.
    pub fn interrupt_scope(&self) -> Ctx {
        Ctx {
            source: self.source.clone(),
            deadline: self.deadline,
            interruptible: true,
            sigint_base: SIGINT_COUNT.load(Ordering::SeqCst),
        }
    }

    /// Detached scope for cleanup, mirroring `context.WithTimeout` over
    /// `context.Background()`: a fresh flag, no parent cancellation or
    /// SIGINT, only the optional deadline.
    pub fn detached(deadline: Option<Instant>) -> Ctx {
        Ctx {
            source: CancelSource::Owned(Arc::new(AtomicBool::new(false))),
            deadline,
            interruptible: false,
            sigint_base: u64::MAX,
        }
    }

    /// Child scope with a deadline, mirroring `context.WithTimeout`.
    pub fn with_timeout(&self, deadline: Instant) -> Ctx {
        let deadline = match self.deadline {
            Some(outer) if outer < deadline => Some(outer),
            _ => Some(deadline),
        };
        Ctx {
            source: self.source.clone(),
            deadline,
            interruptible: self.interruptible,
            sigint_base: self.sigint_base,
        }
    }

    fn terminated(&self) -> bool {
        match &self.source {
            CancelSource::Global => TERMINATED.load(Ordering::SeqCst),
            CancelSource::Owned(flag) => flag.load(Ordering::SeqCst),
        }
    }

    /// Go `ctx.Deadline()`: the scope's absolute timeout, if any.
    pub fn deadline(&self) -> Option<Instant> {
        self.deadline
    }

    /// Go `ctx.Err()`: termination first, then deadline, then SIGINT.
    pub fn err(&self) -> Option<Error> {
        if self.terminated() {
            return Some(Error::Canceled);
        }
        if let Some(deadline) = self.deadline {
            if Instant::now() >= deadline {
                return Some(Error::DeadlineExceeded);
            }
        }
        if self.interruptible && SIGINT_COUNT.load(Ordering::SeqCst) > self.sigint_base {
            return Some(Error::Canceled);
        }
        None
    }
}

#[cfg(test)]
pub fn test_deliver_sigint() {
    SIGINT_COUNT.fetch_add(1, Ordering::SeqCst);
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Duration;

    static SERIAL: std::sync::Mutex<()> = std::sync::Mutex::new(());

    #[test]
    fn scopes_observe_cancel_deadline_and_sigint() {
        let _guard = SERIAL.lock().unwrap();
        let (ctx, flag) = Ctx::test();
        assert!(ctx.err().is_none());
        flag.store(true, Ordering::SeqCst);
        assert_eq!(ctx.err(), Some(Error::Canceled));
        flag.store(false, Ordering::SeqCst);

        let expired = ctx.with_timeout(Instant::now() - Duration::from_secs(1));
        assert_eq!(expired.err(), Some(Error::DeadlineExceeded));
        let live = ctx.with_timeout(Instant::now() + Duration::from_secs(60));
        assert!(live.err().is_none());

        // SIGINT cancels scopes created before the arrival only.
        let before = ctx.interrupt_scope();
        test_deliver_sigint();
        let after = ctx.interrupt_scope();
        assert_eq!(before.err(), Some(Error::Canceled));
        assert!(after.err().is_none());

        // Non-interruptible scopes ignore SIGINT.
        let root = Ctx {
            source: CancelSource::Owned(Arc::new(AtomicBool::new(false))),
            deadline: None,
            interruptible: false,
            sigint_base: 0,
        };
        assert!(root.err().is_none());
    }
}
