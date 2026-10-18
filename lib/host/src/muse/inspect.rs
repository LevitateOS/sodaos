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

/// Feed one free-text inspection field. Returns whether `key` belonged to
/// this family.
fn feed_inspection_text<'de, A>(
    out: &mut MuseInspection,
    key: &str,
    map: &mut A,
) -> Result<bool, A::Error>
where
    A: MapAccess<'de>,
{
    let slot = if key.eq_ignore_ascii_case("id") {
        &mut out.id
    } else if key.eq_ignore_ascii_case("project") {
        &mut out.project
    } else if key.eq_ignore_ascii_case("userns") {
        &mut out.userns
    } else {
        return Ok(false);
    };
    if let Some(v) = map.next_value::<Option<String>>()? {
        *slot = v;
    }
    Ok(true)
}

/// Feed one inspection flag. Returns whether `key` belonged to this family.
fn feed_inspection_flag<'de, A>(
    out: &mut MuseInspection,
    key: &str,
    map: &mut A,
) -> Result<bool, A::Error>
where
    A: MapAccess<'de>,
{
    let slot = if key.eq_ignore_ascii_case("running") {
        &mut out.running
    } else if key.eq_ignore_ascii_case("privileged") {
        &mut out.privileged
    } else {
        return Ok(false);
    };
    if let Some(v) = map.next_value::<Option<bool>>()? {
        *slot = v;
    }
    Ok(true)
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
                    if key.eq_ignore_ascii_case("pid") {
                        if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                            out.pid = v.0;
                        }
                        continue;
                    }
                    if feed_inspection_text(&mut out, &key, &mut map)? {
                        continue;
                    }
                    if feed_inspection_flag(&mut out, &key, &mut map)? {
                        continue;
                    }
                    return Err(de::Error::unknown_field(
                        &key,
                        &["id", "pid", "project", "running", "privileged", "userns"],
                    ));
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
