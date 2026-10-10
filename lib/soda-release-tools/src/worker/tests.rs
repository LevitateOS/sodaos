use super::*;

use std::os::unix::fs::PermissionsExt;
use std::path::PathBuf;
use std::sync::atomic::{AtomicU64, Ordering};

use crate::build_spec::{ImageResult, Request};
use crate::digest::hash_file;
use crate::progress::BuildProgress;

static COUNTER: AtomicU64 = AtomicU64::new(0);

fn temp_dir(tag: &str) -> PathBuf {
    let id = COUNTER.fetch_add(1, Ordering::SeqCst);
    let path = std::env::temp_dir().join(format!(
        "soda-reltools-worker-{tag}-{}-{id}",
        std::process::id()
    ));
    let _ = std::fs::remove_dir_all(&path);
    std::fs::create_dir_all(&path).unwrap();
    path
}

#[test]
fn successful_attempt_requires_runtime_cleanup() {
    assert_eq!(
        super::runtime::combine_attempt_cleanup(Ok(()), Err("permission denied".to_owned())),
        Err(WorkerError::Failed(
            "worker attempt completed but runtime cleanup failed: permission denied".to_owned()
        ))
    );
}

#[test]
fn failed_attempt_keeps_primary_and_cleanup_errors() {
    assert_eq!(
        super::runtime::combine_attempt_cleanup::<()>(
            Err(WorkerError::Failed("build failed".to_owned())),
            Err("cleanup denied".to_owned())
        ),
        Err(WorkerError::Failed(
            "build failed\nworker runtime cleanup also failed: cleanup denied".to_owned()
        ))
    );
}

#[test]
fn attempt_runtime_release_requires_terminal_unit_custody() {
    let parent_path = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../.artifacts")
        .join(format!(
            "soda-worker-custody-{}-{}",
            std::process::id(),
            COUNTER.fetch_add(1, Ordering::SeqCst)
        ));
    std::fs::create_dir(&parent_path).unwrap();
    let parent = parent_path.to_string_lossy().into_owned();
    let uid = unsafe { libc::getuid() };
    let gid = unsafe { libc::getgid() };

    let retained = claim_attempt_runtime(&parent, "/source/out/retained", uid, gid).unwrap();
    let error = runtime::finish_attempt(
        Err::<(), _>(WorkerError::Cancelled("stop timed out".to_owned())),
        WorkerUnitCustody::ExactUnitUnconfirmed,
        &parent,
        &retained,
    )
    .unwrap_err();
    assert_eq!(error, WorkerError::Cancelled(format!(
        "stop timed out\nworker runtime retained at {retained}: exact unit termination is unconfirmed"
    )));
    assert!(std::fs::metadata(&retained).unwrap().is_dir());

    let completed = claim_attempt_runtime(&parent, "/source/out/completed", uid, gid).unwrap();
    runtime::finish_attempt(
        Err::<(), _>(WorkerError::Failed("result parse failed".to_owned())),
        WorkerUnitCustody::ExactUnitTerminal,
        &parent,
        &completed,
    )
    .unwrap_err();
    assert!(std::fs::metadata(&completed).is_err());

    let undispatched =
        claim_attempt_runtime(&parent, "/source/out/undispatched", uid, gid).unwrap();
    runtime::finish_attempt(
        Err::<(), _>(WorkerError::Failed("admission failed".to_owned())),
        WorkerUnitCustody::NeverDispatched,
        &parent,
        &undispatched,
    )
    .unwrap_err();
    assert!(std::fs::metadata(&undispatched).is_err());

    std::fs::remove_dir_all(parent_path).unwrap();
}

#[test]
fn systemd_unit_state_requires_exact_terminal_observation() {
    assert_eq!(
        execution::unit_state_is_terminal("not-found", "", false),
        Ok(true)
    );
    assert_eq!(
        execution::unit_state_is_terminal("loaded", "inactive", false),
        Ok(true)
    );
    assert_eq!(
        execution::unit_state_is_terminal("loaded", "failed", false),
        Ok(true)
    );
    assert_eq!(
        execution::unit_state_is_terminal("loaded", "active", false),
        Ok(false)
    );
    assert!(execution::unit_state_is_terminal("not-found", "active", false).is_err());
    assert!(execution::unit_state_is_terminal("loaded", "", false).is_err());
    assert!(execution::unit_state_is_terminal("loaded", "unknown", false).is_err());
    assert!(execution::unit_state_is_terminal("activating", "activating", false).is_err());
    assert!(execution::unit_state_is_terminal("loaded", "inactive", true).is_err());
}

fn test_config() -> WorkerConfig {
    WorkerConfig {
        executable: "/bin/true".to_owned(),
        source: "/source".to_owned(),
        forgejo_source: "/forgejo".to_owned(),
        output_parent: "/source/.artifacts/releases/isolated".to_owned(),
        tools: "/tools".to_owned(),
        media_authority_directory: "/authority".to_owned(),
        ..WorkerConfig::default()
    }
}

#[test]
fn development_worker_inputs_and_completion() {
    let c = test_config();
    for target in ["candidate", "media", ""] {
        let mut r = Request {
            source: c.source.clone(),
            forgejo_source: c.forgejo_source.clone(),
            forgejo_revision: "a".repeat(40),
            out: format!("{}/test", c.output_parent),
            development: !target.is_empty(),
            target: target.to_owned(),
            ..Request::default()
        };
        if r.wants_media() {
            r.rootfs_base_url = "https://example.invalid".to_owned();
        }
        let w = build_worker(&c, &r).unwrap();
        assert_eq!(w.user, "soda-build-worker");
        assert!(w
            .read_only
            .contains(&format!("{}:{WORKER_FORGEJO}", c.forgejo_source)));
        assert!(w.arguments.contains(&WORKER_FORGEJO.to_owned()));
        assert!(w.arguments.contains(&r.forgejo_revision));
        if target == "candidate" {
            assert!(!w.read_only.join(" ").contains("authority"));
            assert!(!w.arguments.contains(&"--rootfs-base-url".to_owned()));
            assert!(!w.arguments.contains(&"--media-authority".to_owned()));
        } else {
            assert!(w
                .read_only
                .contains(&"/authority:/run/soda-media-authority".to_owned()));
            assert!(w.arguments.contains(&"--rootfs-base-url".to_owned()));
        }
        if target.is_empty() {
            assert!(!w.arguments.contains(&"--development".to_owned()));
        } else {
            assert!(w.arguments.contains(&"--development".to_owned()));
            assert!(w.arguments.contains(&target.to_owned()));
        }
    }
}

#[test]
fn worker_forwards_controller_live_inputs() {
    let c = test_config();
    let mut r = Request {
        source: c.source.clone(),
        out: format!("{}/test", c.output_parent),
        development: true,
        target: "candidate".to_owned(),
        live_inputs:
            "/run/soda-build-source/.artifacts/releases/isolated/soda-live-inputs-test.json"
                .to_owned(),
        ..Request::default()
    };
    let w = build_worker(&c, &r).unwrap();
    assert!(w.arguments.contains(&"--live-inputs".to_owned()));
    assert!(w.arguments.contains(&r.live_inputs));
    r.live_inputs.clear();
    let w = build_worker(&c, &r).unwrap();
    assert!(!w.arguments.contains(&"--live-inputs".to_owned()));
}

#[test]
fn worker_env_has_no_bun_cache_and_pinned_go_first() {
    let c = test_config();
    let r = Request {
        source: c.source.clone(),
        out: format!("{}/test", c.output_parent),
        development: true,
        target: "media".to_owned(),
        rootfs_base_url: "https://example.invalid".to_owned(),
        ..Request::default()
    };
    let w = build_worker(&c, &r).unwrap();
    for env in &w.environment {
        assert!(!env.contains("BUN_INSTALL_CACHE_DIR"), "{env}");
    }
    assert!(w.environment.contains(&format!("HOME={WORKER_HOME}")));
    assert!(w
        .environment
        .contains(&format!("CARGO_HOME={WORKER_HOME}/cargo")));
    assert!(w
        .environment
        .contains(&format!("CARGO_TARGET_DIR={WORKER_HOME}/cargo-target")));
    assert!(w.environment.contains(&"CARGO_NET_OFFLINE=true".to_owned()));
    let path = w
        .environment
        .iter()
        .find(|e| e.starts_with("PATH="))
        .unwrap();
    assert!(
        path.starts_with(&format!("PATH={PINNED_GO_ROOT}/bin:")),
        "{path}"
    );
    assert!(path.contains("/rust/bin:"), "{path}");
    assert!(!path.contains("soda-build-tools/go"), "{path}");
}

#[test]
fn admit_storage_root_matrix() {
    let good = WorkerConfig {
        storage_root: "/home/soda-candidate".to_owned(),
        build_home: "/home/soda-candidate/home".to_owned(),
        runtime: "/home/soda-candidate/run".to_owned(),
        ..WorkerConfig::default()
    };
    assert!(admit_storage_root(&good).is_ok());
    for (name, bad) in [
        (
            "missing root",
            WorkerConfig {
                storage_root: String::new(),
                ..good.clone()
            },
        ),
        (
            "relative root",
            WorkerConfig {
                storage_root: "home/soda-candidate".to_owned(),
                ..good.clone()
            },
        ),
        (
            "home on root",
            WorkerConfig {
                build_home: "/var/lib/soda-candidate-home".to_owned(),
                ..good.clone()
            },
        ),
        (
            "runtime on root",
            WorkerConfig {
                runtime: "/var/lib/soda-candidate-run".to_owned(),
                ..good.clone()
            },
        ),
        (
            "sibling prefix",
            WorkerConfig {
                build_home: "/home/soda-candidate-evil".to_owned(),
                ..good.clone()
            },
        ),
    ] {
        assert!(admit_storage_root(&bad).is_err(), "{name}");
    }
}

#[test]
fn fast_media_worker_selection() {
    let c = test_config();
    let r = Request {
        source: c.source.clone(),
        out: format!("{}/test", c.output_parent),
        development: true,
        target: "media".to_owned(),
        media_compression: "fast".to_owned(),
        rootfs_base_url: "https://example.invalid".to_owned(),
        ..Request::default()
    };
    let w = build_worker(&c, &r).unwrap();
    assert!(w.arguments.join(" ").contains("--media-compression fast"));
    let r = Request {
        development: false,
        target: String::new(),
        ..r
    };
    assert!(build_worker(&c, &r).is_err());
}

#[test]
fn worker_result_binds_target_and_candidate() {
    let source = temp_dir("result");
    let out = source.join(".artifacts/releases/isolated/test");
    std::fs::create_dir_all(out.join("artifacts")).unwrap();
    std::fs::write(out.join("artifacts/candidate.json"), b"fixture").unwrap();
    let out = out.to_string_lossy().into_owned();
    let hash = hash_file(&format!("{out}/artifacts/candidate.json")).unwrap();
    let r = Request {
        source: source.to_string_lossy().into_owned(),
        out: out.clone(),
        revision: "a".repeat(40),
        arch: "x86_64".to_owned(),
        development: true,
        target: "candidate".to_owned(),
        ..Request::default()
    };
    let original = ImageResult {
        revision: r.revision.clone(),
        architecture: r.arch.clone(),
        candidate: format!(
            "{WORKER_SOURCE}/.artifacts/releases/isolated/test/artifacts/candidate.json"
        ),
        candidate_sha256: hash,
        purpose: "development".to_owned(),
        requested_target: "candidate".to_owned(),
        completed_target: "candidate".to_owned(),
        ..ImageResult::default()
    };
    for mode in [
        "valid",
        "media",
        "purpose",
        "requested",
        "completed",
        "hash",
        "compression",
    ] {
        let mut result = original.clone();
        match mode {
            "media" => result.media = "unexpected-media.json".to_owned(),
            "purpose" => result.purpose = "production".to_owned(),
            "requested" => result.requested_target = "release".to_owned(),
            "completed" => result.completed_target = "media".to_owned(),
            "compression" => result.media_compression = "fast".to_owned(),
            "hash" => result.candidate_sha256 = "b".repeat(64),
            _ => {}
        }
        let err = validate_worker_result(&r, &mut result);
        if mode == "valid" {
            assert!(err.is_ok(), "{err:?}");
            assert_eq!(result.candidate, format!("{out}/artifacts/candidate.json"));
            assert!(result.media.is_empty());
        } else {
            assert!(err.is_err(), "{mode}");
        }
    }
    let _ = std::fs::remove_dir_all(&source);
}

#[test]
fn claim_attempt_runtime_isolates_attempts() {
    let parent = temp_dir("runtime");
    let parent = parent.to_string_lossy().into_owned();
    let uid = unsafe { libc::getuid() };
    let gid = unsafe { libc::getgid() };
    let first = claim_attempt_runtime(
        &parent,
        "/source/.artifacts/releases/isolated/manual-01",
        uid,
        gid,
    )
    .unwrap();
    let second = claim_attempt_runtime(
        &parent,
        "/source/.artifacts/releases/isolated/manual-01",
        uid,
        gid,
    )
    .unwrap();
    assert_ne!(first, second);
    let first_unit = runtime::attempt_worker_name(&first).unwrap();
    let second_unit = runtime::attempt_worker_name(&second).unwrap();
    assert_ne!(first_unit, second_unit);
    assert!(valid_worker_name(&first_unit));
    assert!(valid_worker_name(&second_unit));
    assert!(
        runtime::attempt_worker_name("/run/preexisting-00000000000000000000000000000000").is_err()
    );
    for dir in [&first, &second] {
        assert_eq!(dir.rfind('/').map(|i| &dir[..i]), Some(parent.as_str()));
        let st = std::fs::metadata(dir).unwrap();
        assert!(st.file_type().is_dir());
        assert_eq!(st.permissions().mode() & 0o777, 0o700);
    }
    std::fs::write(format!("{first}/a.lock"), b"a").unwrap();
    std::fs::write(format!("{second}/b.lock"), b"b").unwrap();
    release_attempt_runtime(&parent, &first).unwrap();
    assert!(std::fs::metadata(&first).is_err());
    assert!(std::fs::metadata(format!("{second}/b.lock")).is_ok());
    release_attempt_runtime(&parent, &second).unwrap();
    assert!(std::fs::metadata(&second).is_err());
    assert!(std::fs::read_dir(&parent).unwrap().next().is_none());
    let _ = std::fs::remove_dir_all(&parent);
}

#[test]
fn claim_attempt_runtime_refuses_bad_input() {
    let parent = temp_dir("runtime-bad");
    let parent = parent.to_string_lossy().into_owned();
    let uid = unsafe { libc::getuid() };
    let gid = unsafe { libc::getgid() };
    assert!(
        claim_attempt_runtime(&format!("{parent}/missing"), "/source/out/leaf", uid, gid).is_err()
    );
    let plain = format!("{parent}/plain");
    std::fs::write(&plain, b"x").unwrap();
    assert!(claim_attempt_runtime(&plain, "/source/out/leaf", uid, gid).is_err());
    for out in ["", "/"] {
        assert!(
            claim_attempt_runtime(&parent, out, uid, gid).is_err(),
            "{out:?}"
        );
    }
    assert_eq!(std::fs::read_dir(&parent).unwrap().count(), 1);
    let _ = std::fs::remove_dir_all(&parent);
}

#[test]
fn claim_attempt_runtime_entropy_failure_creates_no_directory() {
    let parent = temp_dir("runtime-entropy-failure");
    let parent = parent.to_string_lossy().into_owned();
    let uid = unsafe { libc::getuid() };
    let gid = unsafe { libc::getgid() };
    let result =
        runtime::claim_attempt_runtime_with_random(&parent, "/source/out/leaf", uid, gid, |out| {
            out[0] = 1;
            Err(std::io::Error::other("injected entropy failure"))
        });
    assert_eq!(result.unwrap_err(), "worker runtime randomness unavailable");
    assert_eq!(std::fs::read_dir(&parent).unwrap().count(), 0);
    std::fs::remove_dir_all(parent).unwrap();
}

#[test]
fn release_attempt_runtime_refuses_foreign_paths() {
    let parent = temp_dir("release");
    let parent = parent.to_string_lossy().into_owned();
    let uid = unsafe { libc::getuid() };
    let gid = unsafe { libc::getgid() };
    let owned = claim_attempt_runtime(&parent, "/source/out/leaf", uid, gid).unwrap();
    let outside = temp_dir("release-out");
    let nested = format!("{owned}/nested");
    std::fs::create_dir(&nested).unwrap();
    let link = format!("{parent}/link");
    std::os::unix::fs::symlink(&owned, &link).unwrap();
    let outside = outside.to_string_lossy().into_owned();
    let missing = format!("{parent}/missing");
    for dir in [&parent, &outside, &nested, &link, &missing] {
        assert!(release_attempt_runtime(&parent, dir).is_err(), "{dir}");
    }
    assert!(std::fs::metadata(&owned).is_ok());
    release_attempt_runtime(&parent, &owned).unwrap();
    let _ = std::fs::remove_dir_all(&parent);
    let _ = std::fs::remove_dir_all(&outside);
}

#[test]
fn build_worker_binds_attempt_runtime() {
    let c = WorkerConfig {
        source: "/source".to_owned(),
        output_parent: "/source/.artifacts/releases/isolated".to_owned(),
        runtime: "/run/attempt-xyz".to_owned(),
        tools: "/tools".to_owned(),
        media_authority_directory: "/authority".to_owned(),
        ..WorkerConfig::default()
    };
    let r = Request {
        source: c.source.clone(),
        out: format!("{}/test", c.output_parent),
        development: true,
        target: "candidate".to_owned(),
        ..Request::default()
    };
    let w = build_worker(&c, &r).unwrap();
    assert!(w
        .writable
        .contains(&format!("{}:{WORKER_RUNTIME}", c.runtime)));
}

#[test]
fn worker_name_rule() {
    assert!(valid_worker_name("soda-build-manual-01"));
    assert!(valid_worker_name("soda-qualify-x"));
    assert!(!valid_worker_name("soda-build-"));
    assert!(!valid_worker_name("soda-other-x"));
    assert!(!valid_worker_name("soda-build-UPPER"));
    assert!(!valid_worker_name(&format!(
        "soda-build-{}",
        "a".repeat(49)
    )));
}

#[test]
fn decode_image_result_round_trip() {
    let document = r#"{"Revision":"r","Architecture":"x86_64","Candidate":"c","CandidateSHA256":"s","Scope":"scope","Purpose":"development","RequestedTarget":"candidate","CompletedTarget":"candidate"}"#;
    let result = decode_image_result(document.as_bytes()).unwrap();
    assert_eq!(result.revision, "r");
    assert_eq!(result.scope, "scope");
    assert!(result.media.is_empty());
    assert!(decode_image_result(b"[1,2]").is_err());
}

#[test]
fn image_result_uses_last_exact_raw_slot_and_ignores_unknown_large_numbers() {
    let result = decode_image_result(
        br#"{"Revision":1e400,"Revision":"new","Candidate":"old","Candidate":null,"Unknown":{"number":1e400}}"#,
    )
    .unwrap();
    assert_eq!(result.revision, "new");
    assert!(result.candidate.is_empty());
}

fn argv_executable() -> String {
    for candidate in ["/usr/bin/true", "/bin/true"] {
        if trusted_executable(candidate).is_ok() {
            return candidate.to_owned();
        }
    }
    panic!("no trusted test executable available");
}

fn argv_fixture() -> Worker {
    Worker {
        name: "soda-build-manual-01".to_owned(),
        user: "soda-build-worker".to_owned(),
        executable: argv_executable(),
        directory: "/run/soda-build-source".to_owned(),
        read_only: vec!["/source:/run/soda-build-source".to_owned()],
        writable: vec!["/out:/run/soda-build-source/.artifacts/releases/isolated".to_owned()],
        environment: vec!["HOME=/var/lib/soda-build-worker".to_owned()],
        arguments: vec!["--worker-build".to_owned()],
    }
}

#[test]
fn worker_argv_golden() {
    let w = argv_fixture();
    let argv = worker_argv(&w).unwrap();
    let expected: Vec<String> = [
        "--quiet",
        "--wait",
        "--pipe",
        "--collect",
        "--service-type=exec",
        "--unit=soda-build-manual-01",
        "--property=User=soda-build-worker",
        "--property=Group=soda-build-worker",
        "--property=WorkingDirectory=/run/soda-build-source",
        "--property=ProtectHome=tmpfs",
        "--property=ProtectSystem=strict",
        "--property=PrivateTmp=yes",
        "--property=PrivateMounts=yes",
        "--property=Delegate=yes",
        "--property=CPUQuota=400%",
        "--property=MemoryMax=16G",
        "--property=CPUAffinity=0 1 2 3",
        "--property=KillMode=control-group",
        "--property=TimeoutStopSec=20s",
        "--property=UMask=0077",
        "--property=InaccessiblePaths=-/var/lib/soda-release -/root",
        "--property=BindReadOnlyPaths=/source:/run/soda-build-source",
        "--property=BindPaths=/out:/run/soda-build-source/.artifacts/releases/isolated",
        "--setenv=HOME=/var/lib/soda-build-worker",
        "--",
    ]
    .iter()
    .map(|s| s.to_string())
    .chain([w.executable.clone(), "--worker-build".to_owned()])
    .collect();
    assert_eq!(argv, expected);
}

#[test]
fn worker_argv_refuses_bad_identity() {
    let w = argv_fixture();
    let mut bad = w.clone();
    bad.name = "soda-other-x".to_owned();
    assert_eq!(
        worker_argv(&bad).unwrap_err(),
        "exact task worker name required"
    );
    let mut bad = w.clone();
    bad.user = "root".to_owned();
    assert_eq!(
        worker_argv(&bad).unwrap_err(),
        "separate approved worker identity required"
    );
    let mut bad = w.clone();
    bad.directory = "relative".to_owned();
    assert_eq!(
        worker_argv(&bad).unwrap_err(),
        "absolute worker directory required"
    );
    let mut bad = w.clone();
    bad.directory = "/run/soda-build-source:extra".to_owned();
    assert_eq!(
        worker_argv(&bad).unwrap_err(),
        "absolute worker directory required"
    );
}

#[test]
fn worker_argv_refuses_bad_bind_and_env() {
    let w = argv_fixture();
    for bad in [
        "relative:/guest",
        "/host:relative",
        "/host:/guest:extra",
        "/ho st:/guest",
        "/host:/gue\nst",
        "/host:/gue%st",
    ] {
        let mut bound = w.clone();
        bound.writable = vec![bad.to_owned()];
        assert_eq!(
            worker_argv(&bound).unwrap_err(),
            "explicit absolute worker bind pair required",
            "{bad:?}"
        );
    }
    let mut refused = w.clone();
    refused.environment = vec!["SODA_EVIL=1".to_owned()];
    assert_eq!(
        worker_argv(&refused).unwrap_err(),
        "worker environment key refused"
    );
    for bad in ["HOME", "HOME=/a\nb", "HOME=/a\rb", "PATH=/bin\0x"] {
        let mut invalid = w.clone();
        invalid.environment = vec![bad.to_owned()];
        assert_eq!(
            worker_argv(&invalid).unwrap_err(),
            "invalid worker environment",
            "{bad:?}"
        );
    }
}

#[test]
fn live_inputs_paths_are_pure() {
    let (controller, worker) = live_inputs_paths(
        "/source",
        "/source/.artifacts/releases/isolated",
        "/source/.artifacts/releases/isolated/manual-01",
    )
    .unwrap();
    assert_eq!(
        controller,
        "/source/.artifacts/releases/isolated/soda-live-inputs-manual-01.json"
    );
    assert_eq!(
        worker,
        "/run/soda-build-source/.artifacts/releases/isolated/soda-live-inputs-manual-01.json"
    );
    assert_eq!(live_inputs_name("/a/b/c"), "soda-live-inputs-c.json");
    assert!(live_inputs_paths("/source", "/elsewhere", "/source/out").is_err());
}

struct EnvRestore {
    prior: Option<String>,
}

impl Drop for EnvRestore {
    fn drop(&mut self) {
        match &self.prior {
            Some(v) => std::env::set_var("SODA_BUILD_START_NS", v),
            None => std::env::remove_var("SODA_BUILD_START_NS"),
        }
    }
}

#[test]
fn run_build_worker_fails_before_dispatch_without_progress() {
    let _restore = EnvRestore {
        prior: std::env::var("SODA_BUILD_START_NS").ok(),
    };
    std::env::remove_var("SODA_BUILD_START_NS");
    let mut progress = BuildProgress::new("test").unwrap();
    let _captured = progress.capture();
    progress.finish(None).unwrap();
    let config = WorkerConfig::default();
    // Both media branches fail at the phase boundary: no live-input
    // network, no root lookup, no systemd dispatch.
    for request in [
        Request {
            development: true,
            target: "candidate".to_owned(),
            ..Request::default()
        },
        Request::default(),
    ] {
        assert_eq!(
            run_build_worker(&config, &request, &mut progress).unwrap_err(),
            WorkerError::Failed("invalid progress transition".to_owned())
        );
    }
}

#[test]
fn worker_cancel_predicate_observes_interrupt() {
    // D01-F4: the predicate wired into run_worker must reflect the
    // recorded interrupt state without consuming it.
    let _guard = crate::exitcode::interrupt_test_lock();
    let _ = crate::exitcode::take_interrupt();
    assert!(!worker_cancelled());
    crate::exitcode::note_interrupt(130);
    assert!(worker_cancelled());
    assert!(crate::exitcode::has_interrupt());
    let _ = crate::exitcode::take_interrupt();
    assert!(!worker_cancelled());
}
