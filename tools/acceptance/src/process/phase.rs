use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};

use crate::error::Error;

struct PhaseInner {
    deadline: Option<Instant>,
    cancelled: AtomicBool,
}

/// Cancellable phase context with an optional deadline, like Go's
/// `context.Context`: cancellation and expiry flow downward only, so
/// cancelling a child never disturbs its parent or siblings.
#[derive(Clone)]
pub struct Phase {
    inner: Arc<PhaseInner>,
    parent: Option<Box<Phase>>,
}

impl Phase {
    fn new(deadline: Option<Instant>, parent: Option<Phase>) -> Phase {
        Phase {
            inner: Arc::new(PhaseInner {
                deadline,
                cancelled: AtomicBool::new(false),
            }),
            parent: parent.map(Box::new),
        }
    }

    /// Unbounded, uncancelled-until-asked phase.
    pub fn background() -> Phase {
        Phase::new(None, None)
    }

    /// Phase expiring after `duration`.
    pub fn timeout(duration: Duration) -> Phase {
        Phase::new(Some(Instant::now() + duration), None)
    }

    /// Child phase with a tighter deadline. It observes its parent's
    /// cancellation and expiry, but cancelling it stays local.
    pub fn child(&self, duration: Duration) -> Phase {
        let child_deadline = Instant::now() + duration;
        let deadline = self
            .inner
            .deadline
            .map_or(child_deadline, |parent| child_deadline.min(parent));
        Phase::new(Some(deadline), Some(self.clone()))
    }

    /// Cancel this phase and its children.
    pub fn cancel(&self) {
        self.inner.cancelled.store(true, Ordering::SeqCst);
    }

    /// True after [`Phase::cancel`], here or on any parent.
    pub fn is_cancelled(&self) -> bool {
        self.inner.cancelled.load(Ordering::SeqCst)
            || self.parent.as_ref().is_some_and(|p| p.is_cancelled())
    }

    /// True once the deadline has passed, here or on any parent.
    pub fn expired(&self) -> bool {
        self.inner.deadline.is_some_and(|d| Instant::now() >= d)
            || self.parent.as_ref().is_some_and(|p| p.expired())
    }

    /// Fail when cancelled or expired, like `ctx.Err()`.
    pub fn check(&self) -> Result<(), Error> {
        if self.is_cancelled() {
            return Err(Error::Cancelled);
        }
        if self.expired() {
            return Err(Error::msg("context deadline exceeded"));
        }
        Ok(())
    }

    /// Current deadline, if any.
    pub fn deadline(&self) -> Option<Instant> {
        self.inner.deadline
    }
}
