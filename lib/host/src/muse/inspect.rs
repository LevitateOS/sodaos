use std::time::{Duration, Instant};

use super::MusePeer;
use crate::json::SignedInteger;
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

/// Strict-decoded Muse container inspection.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(in crate::muse) struct MuseInspection {
    pub(in crate::muse) id: String,
    pub(in crate::muse) pid: i64,
    pub(in crate::muse) project: String,
    pub(in crate::muse) running: bool,
    pub(in crate::muse) privileged: bool,
    pub(in crate::muse) userns: String,
}

impl<'de> Deserialize<'de> for MuseInspection {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct InspectionVisitor;
        impl<'de> Visitor<'de> for InspectionVisitor {
            type Value = MuseInspection;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a Muse container inspection object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = MuseInspection::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.id = v;
                        }
                    } else if key.eq_ignore_ascii_case("pid") {
                        if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                            out.pid = v.0;
                        }
                    } else if key.eq_ignore_ascii_case("project") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.project = v;
                        }
                    } else if key.eq_ignore_ascii_case("running") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            out.running = v;
                        }
                    } else if key.eq_ignore_ascii_case("privileged") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            out.privileged = v;
                        }
                    } else if key.eq_ignore_ascii_case("userns") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.userns = v;
                        }
                    } else {
                        return Err(de::Error::unknown_field(
                            &key,
                            &["id", "pid", "project", "running", "privileged", "userns"],
                        ));
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(InspectionVisitor)
    }
}

pub(in crate::muse) fn muse_peer_alive(peer: &MusePeer) -> bool {
    use std::os::fd::AsRawFd;
    let mut fds = [libc::pollfd {
        fd: peer.pidfd.as_raw_fd(),
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
