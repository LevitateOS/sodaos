use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use super::run::{FACTORY_CLEANUP_SECS, FACTORY_LIVE_SECS};

// ---------- RFC3339Nano deadlines ----------

pub(in crate::factory) const NANOS_PER_SEC: i128 = 1_000_000_000;

/// Parse a Go `time.Time` JSON value to epoch nanos. Accepts the canonical
/// shape Go emits: `YYYY-MM-DDTHH:MM:SS` with an optional 1-9 digit
/// fraction and a `Z` or numeric offset.
pub(in crate::factory) fn parse_deadline(s: &str) -> Option<i128> {
    soda_wire_time::parse_nanos(s)
}

/// Current wall-clock time as Unix epoch nanos.
pub(in crate::factory) fn system_nanos_now() -> i128 {
    match SystemTime::now().duration_since(UNIX_EPOCH) {
        Ok(d) => d.as_nanos() as i128,
        Err(e) => -(e.duration().as_nanos() as i128),
    }
}

/// Go zero time (`0001-01-01T00:00:00Z`) has Unix epoch nanos of the
/// proleptic civil date; anything else, malformed included, is nonzero or
/// unusable. Callers treat `None` (malformed) like a missing deadline: the
/// route decoder owns malformed-deadline reports, the factory never sees
/// them over the wire.
pub(in crate::factory) fn deadline_is_zero(text: &str) -> bool {
    soda_wire_time::parse_nanos(text) == Some(-62_135_596_800_000_000_000)
}

/// Truncated seconds from now until the deadline, like
/// `int64(time.Until(deadline).Seconds())`.
pub(in crate::factory) fn seconds_until(deadline_nanos: i128) -> i64 {
    ((deadline_nanos - system_nanos_now()) / NANOS_PER_SEC)
        .clamp(i64::MIN as i128, i64::MAX as i128) as i64
}

/// Operation deadline capped by the run deadline, like
/// `context.WithDeadline(ctx, run.Deadline)`.
pub(in crate::factory) fn drive_deadline(op: Instant, deadline_nanos: i128) -> Instant {
    let delta = deadline_nanos - system_nanos_now();
    if delta <= 0 {
        return Instant::now();
    }
    let wait = Duration::from_nanos(delta.min(u64::MAX as i128) as u64);
    Instant::now().checked_add(wait).unwrap_or(op).min(op)
}

fn min_instant(a: Instant, b: Instant) -> Instant {
    if a < b {
        a
    } else {
        b
    }
}

pub(in crate::factory) fn cleanup_deadline() -> Instant {
    Instant::now() + Duration::from_secs(FACTORY_CLEANUP_SECS)
}

pub(in crate::factory) fn live_deadline(op: Instant) -> Instant {
    min_instant(op, Instant::now() + Duration::from_secs(FACTORY_LIVE_SECS))
}
