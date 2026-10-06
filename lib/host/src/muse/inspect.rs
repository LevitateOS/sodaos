use std::time::{Duration, Instant};

use super::MusePeer;
use crate::json::{Kind, Spec};

/// Strict-decoded Muse container inspection.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(in crate::muse) struct MuseInspection {
    pub(in crate::muse) id: String,
    pub(in crate::muse) project: String,
    pub(in crate::muse) running: bool,
    pub(in crate::muse) privileged: bool,
    pub(in crate::muse) userns: String,
}

pub(in crate::muse) const MUSE_INSPECTION_SPECS: &[Spec] = &[
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "pid",
        kind: Kind::Int,
    },
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "running",
        kind: Kind::Bool,
    },
    Spec {
        name: "privileged",
        kind: Kind::Bool,
    },
    Spec {
        name: "userns",
        kind: Kind::Str,
    },
];

pub(in crate::muse) fn muse_peer_alive(peer: &MusePeer) -> bool {
    let mut fds = [libc::pollfd {
        fd: peer.pidfd,
        events: libc::POLLIN,
        revents: 0,
    }];
    // SAFETY: valid one-element pollfd array.
    let n = unsafe { libc::poll(fds.as_mut_ptr(), 1, 0) };
    n == 0
}

pub(in crate::muse) fn sleep_until(target: Instant, deadline: Instant) -> Result<(), ()> {
    loop {
        let now = Instant::now();
        if now >= deadline {
            return Err(());
        }
        if now >= target {
            return Ok(());
        }
        let slice = (target - now)
            .min(deadline - now)
            .min(Duration::from_millis(10));
        std::thread::sleep(slice);
    }
}
