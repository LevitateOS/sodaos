use super::*;
use crate::project::Executor;
use std::sync::Mutex;
use std::time::{Duration, Instant};

pub(super) fn spec() -> Spec {
    Spec {
        id: format!("a{}", "1".repeat(24)),
        revision: 1,
        name: "TaskBoard".into(),
        artifact_sha256: "2".repeat(64),
        bundle_sha256: "3".repeat(64),
        tools_sha256: "4".repeat(64),
        runtime_image: format!("sha256:{}", "5".repeat(64)),
        image: format!("sha256:{}", "6".repeat(64)),
        data: format!("a{}", "7".repeat(24)),
        owner_team: "12345678-1234-1234-1234-123456789abc".into(),
        origin: "https://board.soda.test".into(),
        port: 22000,
    }
}

struct Native {
    calls: Mutex<Vec<(String, Vec<String>)>>,
    running: Mutex<bool>,
    http: Mutex<Vec<u8>>,
    valid_ingress: Mutex<bool>,
}

impl Native {
    fn new() -> Self {
        Self {
            calls: Mutex::new(vec![]),
            running: Mutex::new(true),
            http: Mutex::new(b"200".to_vec()),
            valid_ingress: Mutex::new(false),
        }
    }
}

impl Executor for Native {
    fn run(&self, _: &[u8], cmd: &str, args: &[&str], _: Instant) -> Result<Vec<u8>, String> {
        self.calls
            .lock()
            .unwrap()
            .push((cmd.into(), args.iter().map(|s| s.to_string()).collect()));
        let spec = spec();
        match cmd {
            "podman" if args[0] == "container" => Ok(serde_json::json!({
                "container": "8".repeat(64), "image": spec.image.strip_prefix("sha256:").unwrap(), "running": *self.running.lock().unwrap(),
                "labels": { "io.soda.canlang.id": spec.id, "io.soda.canlang.data": spec.data,
                    "io.soda.canlang.owner": spec.owner_team, "io.soda.canlang.origin": spec.origin,
                    "io.soda.canlang.port": spec.port.to_string(), "io.soda.canlang.revision": "1",
                    "io.canlang.artifact": spec.artifact_sha256, "io.canlang.bundle": spec.bundle_sha256 }
            }).to_string().into_bytes()),
            "systemctl" if args[0] == "disable" || args[0] == "enable" => Ok(vec![]),
            "systemctl" if args[0] == "stop" => { *self.running.lock().unwrap() = false; Ok(vec![]) }
            "curl" => Ok(self.http.lock().unwrap().clone()),
            "podman" if args[0] == "exec" && *self.valid_ingress.lock().unwrap() => Ok(vec![]),
            "systemctl" if args == ["restart", "soda-proxy.service"] && *self.valid_ingress.lock().unwrap() => Ok(vec![]),
            _ => Err("unexpected native mutation".into()),
        }
    }
}

#[test]
fn rejects_unbounded_references_and_noncanonical_origins() {
    let mut input = spec();
    input.validate().unwrap();
    for origin in [
        "http://board.soda.test",
        "https://board.soda.test/",
        "https://board.soda.test/path",
        "https://user@board.soda.test",
        "https://127.0.0.1",
        "https://localhost",
        "https://board.soda.test#x",
    ] {
        input.origin = origin.into();
        assert!(input.validate().is_err(), "{origin}");
    }
    input = spec();
    input.id = "../../foreign".into();
    assert!(input.validate().is_err());
    input = spec();
    input.image = "latest".into();
    assert!(input.validate().is_err());
}

#[test]
fn install_and_replace_refuse_unselected_runtime_or_tools_before_native_effects() {
    for action in ["install", "replace"] {
        for binding in [
            None,
            Some(ReleaseBinding {
                image: format!("sha256:{}", "9".repeat(64)),
                tools_sha256: spec().tools_sha256,
            }),
            Some(ReleaseBinding {
                image: spec().runtime_image,
                tools_sha256: "9".repeat(64),
            }),
        ] {
            let runtime = Runtime {
                exec: Native::new(),
                release: binding,
            };
            let request = Request {
                application: spec(),
                action: action.into(),
                expected_container: if action == "install" {
                    String::new()
                } else {
                    "8".repeat(64)
                },
            };
            assert!(runtime
                .operate(&request, Instant::now() + Duration::from_secs(1))
                .is_err());
            assert!(runtime.exec.calls.lock().unwrap().is_empty());
        }
    }
}

#[test]
fn changed_appliance_release_does_not_prevent_stopping_existing_application() {
    let runtime = Runtime {
        exec: Native::new(),
        release: Some(ReleaseBinding {
            image: format!("sha256:{}", "9".repeat(64)),
            tools_sha256: "9".repeat(64),
        }),
    };
    let request = Request {
        application: spec(),
        action: "stop".into(),
        expected_container: "8".repeat(64),
    };
    assert!(
        !runtime
            .operate(&request, Instant::now() + Duration::from_secs(1))
            .unwrap()
            .running
    );
}

#[test]
fn stale_container_cannot_stop_a_replacement() {
    let runtime = Runtime {
        exec: Native::new(),
        release: Some(ReleaseBinding {
            image: spec().runtime_image,
            tools_sha256: spec().tools_sha256,
        }),
    };
    let request = Request {
        application: spec(),
        action: "stop".into(),
        expected_container: "9".repeat(64),
    };
    assert!(runtime
        .operate(&request, Instant::now() + Duration::from_secs(5))
        .unwrap_err()
        .contains("generation"));
    assert_eq!(runtime.exec.calls.lock().unwrap().len(), 1);
    assert!(*runtime.exec.running.lock().unwrap());
}

#[test]
fn stop_confirms_actual_native_state_and_preserves_data() {
    let runtime = Runtime {
        exec: Native::new(),
        release: Some(ReleaseBinding {
            image: spec().runtime_image,
            tools_sha256: spec().tools_sha256,
        }),
    };
    let request = Request {
        application: spec(),
        action: "stop".into(),
        expected_container: "8".repeat(64),
    };
    let observed = runtime
        .operate(&request, Instant::now() + Duration::from_secs(5))
        .unwrap();
    assert!(!observed.running && !observed.ready);
    assert_eq!(observed.container, request.expected_container);
    let calls = runtime.exec.calls.lock().unwrap();
    assert_eq!(calls.len(), 4);
    assert_eq!(
        calls[1],
        (
            "systemctl".into(),
            vec![
                "disable".into(),
                format!("soda-canlang@{}.service", spec().id)
            ]
        )
    );
    assert_eq!(calls[2].1[0], "stop");
}

#[test]
fn artifact_upgrade_refuses_before_stopping_current_writer() {
    let runtime = Runtime {
        exec: Native::new(),
        release: Some(ReleaseBinding {
            image: spec().runtime_image,
            tools_sha256: spec().tools_sha256,
        }),
    };
    let mut next = spec();
    next.artifact_sha256 = "9".repeat(64);
    next.revision = 2;
    let request = Request {
        application: next,
        action: "replace".into(),
        expected_container: "8".repeat(64),
    };
    assert!(runtime
        .operate(&request, Instant::now() + Duration::from_secs(5))
        .unwrap_err()
        .contains("migration"));
    assert_eq!(runtime.exec.calls.lock().unwrap().len(), 1);
    assert!(*runtime.exec.running.lock().unwrap());
}

#[test]
fn readiness_uses_actual_bounded_backend_and_generation() {
    let runtime = Runtime {
        exec: Native::new(),
        release: Some(ReleaseBinding {
            image: spec().runtime_image,
            tools_sha256: spec().tools_sha256,
        }),
    };
    let observed = runtime
        .wait_ready(
            &spec(),
            &"8".repeat(64),
            Instant::now() + Duration::from_secs(1),
        )
        .unwrap();
    assert!(observed.running && observed.ready);
    assert_eq!(
        observed.image,
        spec().image,
        "native raw Podman image IDs become canonical wire references"
    );
    let calls = runtime.exec.calls.lock().unwrap();
    assert_eq!(calls.len(), 2);
    assert_eq!(calls[1].0, "curl");
    assert_eq!(
        calls[1].1.last().unwrap(),
        "http://127.0.0.1:22000/auth/login"
    );
    assert!(calls.iter().all(|(cmd, _)| cmd != "systemctl"));
}

#[test]
fn stopped_backend_cannot_report_started() {
    let runtime = Runtime {
        exec: Native::new(),
        release: Some(ReleaseBinding {
            image: spec().runtime_image,
            tools_sha256: spec().tools_sha256,
        }),
    };
    *runtime.exec.running.lock().unwrap() = false;
    assert!(runtime
        .wait_ready(
            &spec(),
            &"8".repeat(64),
            Instant::now() + Duration::from_millis(10)
        )
        .is_err());
}

#[test]
fn ingress_validation_failure_never_restarts_shared_proxy() {
    let runtime = Runtime {
        exec: Native::new(),
        release: Some(ReleaseBinding {
            image: spec().runtime_image,
            tools_sha256: spec().tools_sha256,
        }),
    };
    assert!(runtime
        .activate_ingress(Instant::now() + Duration::from_secs(1))
        .is_err());
    let calls = runtime.exec.calls.lock().unwrap();
    assert_eq!(calls.len(), 1);
    assert_eq!(
        calls[0].1,
        [
            "exec",
            "soda-proxy",
            "caddy",
            "validate",
            "--config",
            "/etc/caddy/Caddyfile",
            "--adapter",
            "caddyfile"
        ]
    );
}

#[test]
fn running_unready_backend_has_a_finite_failure() {
    let runtime = Runtime {
        exec: Native::new(),
        release: Some(ReleaseBinding {
            image: spec().runtime_image,
            tools_sha256: spec().tools_sha256,
        }),
    };
    *runtime.exec.http.lock().unwrap() = b"503".to_vec();
    let began = Instant::now();
    assert!(runtime
        .wait_ready(&spec(), &"8".repeat(64), began + Duration::from_millis(10))
        .unwrap_err()
        .contains("deadline"));
    assert!(began.elapsed() < Duration::from_secs(1));
}

#[test]
fn validated_install_restarts_only_existing_shared_proxy() {
    let runtime = Runtime {
        exec: Native::new(),
        release: Some(ReleaseBinding {
            image: spec().runtime_image,
            tools_sha256: spec().tools_sha256,
        }),
    };
    *runtime.exec.valid_ingress.lock().unwrap() = true;
    runtime
        .activate_ingress(Instant::now() + Duration::from_secs(1))
        .unwrap();
    let calls = runtime.exec.calls.lock().unwrap();
    assert_eq!(calls.len(), 2);
    assert_eq!(calls[0].0, "podman");
    assert_eq!(
        calls[1],
        (
            "systemctl".into(),
            vec!["restart".into(), "soda-proxy.service".into()]
        )
    );
}
