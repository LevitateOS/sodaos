//! Monotonic clock (`clock_linux.go` / `clock_other.go`).

use std::time::Duration;

/// Monotonic time for build progress; panics when the clock fails, as the
/// Go owner does.
#[cfg(target_os = "linux")]
pub fn monotonic() -> Duration {
    let mut ts = libc::timespec {
        tv_sec: 0,
        tv_nsec: 0,
    };
    let rc = unsafe { libc::clock_gettime(libc::CLOCK_MONOTONIC, &mut ts) };
    if rc != 0 {
        panic!("clock_gettime: {}", std::io::Error::last_os_error());
    }
    Duration::new(ts.tv_sec.max(0) as u64, ts.tv_nsec.max(0) as u32)
}

/// Non-Linux fallback: production admission requires native Linux.
#[cfg(not(target_os = "linux"))]
pub fn monotonic() -> Duration {
    use std::sync::OnceLock;
    use std::time::Instant;
    static ORIGIN: OnceLock<Instant> = OnceLock::new();
    ORIGIN.get_or_init(Instant::now).elapsed()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn monotonic_advances() {
        let first = monotonic();
        std::thread::sleep(Duration::from_millis(2));
        assert!(monotonic() >= first);
    }
}
