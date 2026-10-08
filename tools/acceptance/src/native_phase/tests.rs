use super::*;
use std::cell::RefCell;
use std::os::unix::fs::PermissionsExt;

#[test]
fn phase_request_checks_every_duplicate_occurrence_before_last_wins() {
    let platform = Platform {
        os: "linux".to_string(),
        arch: "x86_64".to_string(),
        hostname: "synthetic-builder".to_string(),
    };
    let revision = "a".repeat(40);
    let work = std::env::temp_dir().join("soda-acceptance-uncreated");
    let work = work.to_string_lossy();
    let raw = format!(
        r#"{{"Revision":false,"Revision":"{revision}","Architecture":"x86_64","Target":"synthetic-builder","Work":"{work}","Phase":"prepare"}}"#
    );
    assert!(phase_request(&raw, &platform).is_err());
    let valid = format!(
        r#"{{"Revision":"wrong","Revision":"{revision}","Architecture":"x86_64","Target":"synthetic-builder","Work":"{work}","Phase":"prepare"}}"#
    );
    assert!(phase_request(&valid, &platform).is_ok());
}

#[cfg(all(target_os = "linux", target_arch = "x86_64"))]
#[test]
fn actual_native_platform_fields_admit_a_matching_phase_request() {
    let platform = native_platform().unwrap();
    assert_eq!(platform.os, "linux");
    assert_eq!(platform.arch, "x86_64");

    let harness = Harness::new();
    std::fs::set_permissions(&harness.dir, std::fs::Permissions::from_mode(0o700)).unwrap();
    let request = harness.request(
        "prepare",
        &[
            ("Architecture", &platform.arch),
            ("Target", &platform.hostname),
        ],
    );
    assert!(phase_request(&request, &platform).is_ok());

    let non_linux = Platform {
        os: "freebsd".to_string(),
        arch: platform.arch.clone(),
        hostname: platform.hostname.clone(),
    };
    assert!(phase_request(&request, &non_linux).is_err());
}

const REVISION: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";

struct FakeRunner {
    calls: RefCell<Vec<Vec<String>>>,
    revision: String,
    dirty: RefCell<Vec<u8>>,
}

impl Runner for FakeRunner {
    fn status(&self, argv: &[String], _cwd: Option<&str>) -> Result<(), PayloadFailure> {
        self.calls.borrow_mut().push(argv.to_vec());
        if argv.iter().any(|a| a == "clone") {
            std::fs::create_dir_all(argv.last().unwrap()).unwrap();
        }
        Ok(())
    }

    fn output(&self, argv: &[String], _cwd: Option<&str>) -> Result<Vec<u8>, PayloadFailure> {
        if argv.get(1).is_some_and(|a| a == "rev-parse") {
            Ok(format!("{}\n", self.revision).into_bytes())
        } else {
            Ok(self.dirty.borrow().clone())
        }
    }
}

struct Harness {
    dir: std::path::PathBuf,
    work: std::path::PathBuf,
    platform: Platform,
    runner: FakeRunner,
}

impl Harness {
    fn new() -> Harness {
        let mut dir = std::env::temp_dir();
        dir.push(format!("soda-native-{}-{}", std::process::id(), fresh_id()));
        std::fs::create_dir_all(&dir).unwrap();
        // Realistic restrictive–but-passing parent for workdir admit checks.
        let work = dir.join("run");
        Harness {
            dir,
            work,
            platform: Platform {
                os: "linux".to_string(),
                arch: "x86_64".to_string(),
                hostname: "synthetic-builder".to_string(),
            },
            runner: FakeRunner {
                calls: RefCell::new(Vec::new()),
                revision: REVISION.to_string(),
                dirty: RefCell::new(Vec::new()),
            },
        }
    }

    fn request(&self, phase: &str, changes: &[(&str, &str)]) -> String {
        let mut fields = HashMap::from([
            ("Revision".to_string(), REVISION.to_string()),
            ("Architecture".to_string(), "x86_64".to_string()),
            ("Target".to_string(), "synthetic-builder".to_string()),
            ("Work".to_string(), self.work.to_string_lossy().into_owned()),
            ("Phase".to_string(), phase.to_string()),
        ]);
        for (key, value) in changes {
            fields.insert(key.to_string(), value.to_string());
        }
        serde_json::to_string(&fields).unwrap()
    }

    fn invoke(&self, phase: &str, changes: &[(&str, &str)]) -> Result<(), PayloadFailure> {
        run_phase(&self.request(phase, changes), &self.platform, &self.runner)
    }

    fn calls(&self) -> usize {
        self.runner.calls.borrow().len()
    }
}

impl Drop for Harness {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.dir);
    }
}

fn fresh_id() -> u64 {
    use std::sync::atomic::{AtomicU64, Ordering};
    static NEXT: AtomicU64 = AtomicU64::new(0);
    NEXT.fetch_add(1, Ordering::SeqCst)
}

fn seed_candidate(work: &std::path::Path) {
    let candidate = work.join("candidate");
    std::fs::create_dir_all(&candidate).unwrap();
    std::fs::write(candidate.join("payload.json"), b"{}").unwrap();
    std::fs::write(candidate.join("candidate.json"), b"{}").unwrap();
}

#[test]
fn explicit_phase_order_without_automatic_work() {
    let harness = Harness::new();
    harness.invoke("prepare", &[]).unwrap();
    assert_eq!(harness.calls(), 2);
    assert!(!harness.work.join("build.started").exists());
    seed_candidate(&harness.work);
    let before = harness.calls();
    harness.invoke("build", &[]).unwrap();
    assert_eq!(
        harness.calls(),
        before,
        "build admits candidate bytes; it does not run a producer"
    );
    assert!(!harness.work.join("check.started").exists());
    harness.invoke("check", &[]).unwrap();
    let last = harness.runner.calls.borrow().last().cloned().unwrap();
    assert_eq!(
        last[..3],
        [
            "bash".to_string(),
            "scripts/check-native.sh".to_string(),
            "x86_64".to_string()
        ]
    );
    assert!(last[3].ends_with("/candidate"));
    for call in harness.runner.calls.borrow().iter() {
        for part in call {
            assert!(!part.contains("install-native.sh"));
            assert!(!part.contains("build-native.sh"));
        }
    }
}

#[test]
fn unknown_phase_target_arch_and_revision_fail_before_creation() {
    for (phase, changes) in [
        ("install", vec![]),
        ("bundle", vec![]),
        ("prepare", vec![("Target", "other")]),
        ("prepare", vec![("Architecture", "aarch64")]),
        ("prepare", vec![("Revision", "main")]),
    ] {
        let harness = Harness::new();
        let err = harness.invoke(phase, &changes).unwrap_err();
        assert_eq!(err.kind_name(), "ValueError", "{phase} {changes:?}");
        assert!(!harness.work.exists());
        assert_eq!(harness.calls(), 0);
    }
}

#[test]
fn occupied_prepare_and_phase_replay_do_not_run_commands() {
    let harness = Harness::new();
    harness.invoke("prepare", &[]).unwrap();
    let before = harness.calls();
    let err = harness.invoke("prepare", &[]).unwrap_err();
    assert_eq!(err.kind_name(), "FileExistsError");
    assert_eq!(harness.calls(), before);
    seed_candidate(&harness.work);
    harness.invoke("build", &[]).unwrap();
    let before = harness.calls();
    let err = harness.invoke("build", &[]).unwrap_err();
    assert_eq!(err.kind_name(), "FileExistsError");
    assert_eq!(harness.calls(), before);
}

#[test]
fn missing_prerequisite_stale_binding_and_dirty_source() {
    let harness = Harness::new();
    harness.invoke("prepare", &[]).unwrap();
    let before = harness.calls();
    assert!(harness.invoke("check", &[]).is_err());
    assert!(harness
        .invoke("build", &[("Revision", &"b".repeat(40))])
        .is_err());
    *harness.runner.dirty.borrow_mut() = b" M source.go\n".to_vec();
    assert!(harness.invoke("build", &[]).is_err());
    assert_eq!(harness.calls(), before);
    assert!(!harness.work.join("build.started").exists());
}

#[test]
fn build_without_candidate_retains_started_without_completion() {
    let harness = Harness::new();
    harness.invoke("prepare", &[]).unwrap();
    assert!(harness.invoke("build", &[]).is_err());
    assert!(harness.work.join("build.started").is_file());
    assert!(!harness.work.join("build.completed").exists());
}

#[test]
fn remote_dispatch_has_no_runtime_or_publication_phases() {
    let harness = Harness::new();
    let err = harness.invoke("publish", &[]).unwrap_err();
    assert_eq!(err.kind_name(), "ValueError");
    assert_eq!(harness.calls(), 0);
}
