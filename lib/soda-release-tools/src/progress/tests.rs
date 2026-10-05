use super::*;

use std::sync::atomic::{AtomicU64, Ordering};

struct EnvGuard {
    key: &'static str,
    prior: Option<String>,
}

impl EnvGuard {
    fn set(key: &'static str, value: &str) -> EnvGuard {
        let prior = std::env::var(key).ok();
        unsafe { std::env::set_var(key, value) };
        EnvGuard { key, prior }
    }
}

impl Drop for EnvGuard {
    fn drop(&mut self) {
        unsafe {
            match &self.prior {
                Some(v) => std::env::set_var(self.key, v),
                None => std::env::remove_var(self.key),
            }
        }
    }
}

fn fixed_clock() -> Box<dyn Fn() -> Duration + Send> {
    let tick = Arc::new(AtomicU64::new(0));
    Box::new(move || {
        let n = tick.fetch_add(1, Ordering::SeqCst);
        Duration::from_nanos(1_000_000_000 + n)
    })
}

static ENV_LOCK: std::sync::Mutex<()> = std::sync::Mutex::new(());

fn fresh_progress(title: &str) -> (BuildProgress, Arc<Mutex<Vec<u8>>>) {
    let prior = std::env::var("SODA_BUILD_START_NS").ok();
    unsafe { std::env::remove_var("SODA_BUILD_START_NS") };
    let mut progress = BuildProgress::new_with_clock(title, fixed_clock()).unwrap();
    let buf = progress.capture();
    unsafe {
        match prior {
            Some(v) => std::env::set_var("SODA_BUILD_START_NS", v),
            None => std::env::remove_var("SODA_BUILD_START_NS"),
        }
    }
    (progress, buf)
}

#[test]
fn phase_lifecycle_emits_wire_format() {
    let _lock = ENV_LOCK.lock().unwrap();
    let (mut p, buf) = fresh_progress("title");
    p.phase("P1 / Build runtime").unwrap();
    p.end_phase(None).unwrap();
    p.finish(None).unwrap();
    let text = String::from_utf8(buf.lock().unwrap().clone()).unwrap();
    assert!(text.contains("START    P1 / Build runtime\n"), "{text}");
    assert!(
        text.contains("DONE     P1 / Build runtime | phase "),
        "{text}"
    );
    assert!(text.contains("SUCCESS  title | total "), "{text}");
}

#[test]
fn failure_carries_reason() {
    let _lock = ENV_LOCK.lock().unwrap();
    let (mut p, buf) = fresh_progress("title");
    p.next("Compile it").unwrap();
    p.note_reason("x.go: permission denied");
    let err = ToolError::msg("boom");
    p.end(Some(&err)).unwrap();
    let text = String::from_utf8(buf.lock().unwrap().clone()).unwrap();
    assert!(text.contains("FAILED   Compile it | section "), "{text}");
    assert!(text.contains("reason x.go: permission denied"), "{text}");
}

#[test]
fn bad_inherited_origin_refused() {
    let _lock = ENV_LOCK.lock().unwrap();
    let _g = EnvGuard::set("SODA_BUILD_START_NS", "bogus");
    assert!(BuildProgress::new("t").is_err());
}
