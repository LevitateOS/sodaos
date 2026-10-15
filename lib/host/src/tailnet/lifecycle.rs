//! Tailnet lifecycle owner (RCX-2 T01): trusted-pin validation, bounded
//! dependency acquisition, and factual observation rendering for the strict
//! `POST /tailnet/prepare` and `POST /tailnet/lifecycle` host endpoints.
//!
//! Authority stays with Fountain's manager and is verified by the Go
//! service before this hop; this owner only matches the verified tuple
//! against the trusted release pin, runs the selected dependency
//! acquisition, and reports facts. Rejected, replayed, wrong-variant or
//! unbound authority never reaches an effect: the mux validates before
//! dispatch, and every fallible step maps to a bounded wire error.
//!
//! T02 owns the operational grant, boot-closed admission, start
//! serialization and drain, host/project runtime integration, and the
//! generation-bound run records. Until then activation reports facts
//! without claiming them: a present-but-unverified host is unknown, never
//! running, and admission is never ready.

use crate::tcontrol_wire::{self, LifecycleCall};
use std::time::Instant;

/// Privileged release-owned importer. Prepare invokes its fixed ensure
/// command; there is no Tailnet downloader or second acquisition path.
pub const IMAGE_IMPORT_BINARY: &str = "/usr/libexec/soda/soda-image-import";

/// Upper bound for one dependency acquisition inside a lifecycle call.
/// The caller clips this to the call's remaining whole-operation budget.
pub const ENSURE_TIMEOUT_SECS: u64 = 300;

/// Trusted tailnet package pin from the selected release payload: the exact
/// catalog version and archive digest the call must match. Empty pins mean
/// no trusted release was loaded; every lifecycle effect is then
/// unavailable, never default-open.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ReleasePin {
    pub package_version: String,
    pub package_sha256: String,
}

/// Match the verified call against the trusted release pin. A missing pin
/// or any version/digest drift is an identity change, never a silent adopt.
pub fn match_release_pin(call: &LifecycleCall, pin: &ReleasePin) -> Result<(), String> {
    if pin.package_version.is_empty() || pin.package_sha256.is_empty() {
        return Err("unavailable: trusted Tailnet release pin is unavailable".to_string());
    }
    if call.package_version != pin.package_version || call.package_sha256 != pin.package_sha256 {
        return Err("conflict: lifecycle package identity changed".to_string());
    }
    Ok(())
}

/// Run the fixed importer ensure for the tailnet set against the trusted
/// current payload: `soda-image-import ensure --payload ABS --set tailnet`.
/// Quiet exit 0 means the selected manifest/config identity is verified
/// locally. This validates and pulls the selected dependency only; it never
/// opens admission or starts a daemon, so a revoked controller after
/// download leaves a harmless verified cache. The child runs in its own
/// process group and the whole group is killed when the deadline passes.
pub fn ensure_tailnet_dependency(
    binary: &str,
    payload_path: &str,
    deadline: Instant,
) -> Result<(), String> {
    if !binary.starts_with('/') || !payload_path.starts_with('/') {
        return Err("unavailable: Tailnet dependency acquisition is unavailable".to_string());
    }
    if Instant::now() >= deadline {
        return Err("unavailable: Tailnet dependency acquisition is unavailable".to_string());
    }
    use std::os::unix::process::CommandExt;
    use std::process::{Command, Stdio};
    let child = Command::new(binary)
        .args(["ensure", "--payload", payload_path, "--set", "tailnet"])
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .process_group(0)
        .spawn()
        .map_err(|_| unavailable())?;
    match wait_bounded(child, deadline) {
        Ok(status) => {
            if status.success() {
                Ok(())
            } else {
                Err(unavailable())
            }
        }
        Err(_) => Err(unavailable()),
    }
}

fn unavailable() -> String {
    "unavailable: Tailnet dependency acquisition is unavailable".to_string()
}

/// Wait for one spawned child until the deadline, then kill its whole
/// process group and reap it. The caller owns the argv; this owns the
/// clock and the group kill.
fn wait_bounded(
    mut child: std::process::Child,
    deadline: Instant,
) -> Result<std::process::ExitStatus, String> {
    loop {
        match child.try_wait() {
            Ok(Some(status)) => return Ok(status),
            Ok(None) => {}
            Err(_) => return Err("child wait failed".to_string()),
        }
        if Instant::now() >= deadline {
            let pid = child.id() as i32;
            // Best effort: the whole group dies with the deadline; reap
            // whatever remains before reporting.
            unsafe {
                libc::killpg(pid, libc::SIGKILL);
            }
            let _ = child.wait();
            return Err("child deadline exceeded".to_string());
        }
        std::thread::sleep(std::time::Duration::from_millis(50));
    }
}

fn observation_for(
    call: &LifecycleCall,
    state: &str,
    detail: &str,
) -> tcontrol_wire::LifecycleObservation {
    tcontrol_wire::LifecycleObservation {
        controller_id: call.controller_id.clone(),
        controller_instance_id: call.controller_instance_id.clone(),
        package_id: call.package_id.clone(),
        package_version: call.package_version.clone(),
        package_sha256: call.package_sha256.clone(),
        installation_id: call.installation_id.clone(),
        generation: call.generation.clone(),
        operation: call.operation.clone(),
        deadline_unix: call.deadline_unix,
        state: state.to_string(),
        effect_generation: None,
        admission_ready: false,
        detail: detail.to_string(),
        error: None,
    }
}

/// Report stopped: no host node answers owned-resource inspection.
/// Admission is never ready here; the T02 grant owns readiness.
pub fn stopped_observation(call: &LifecycleCall) -> Vec<u8> {
    let detail = "No host node answers inspection; project companions join enumeration in T02.";
    tcontrol_wire::encode_lifecycle_observation(&observation_for(
        call,
        tcontrol_wire::LIFECYCLE_STOPPED,
        detail,
    ))
    .into_bytes()
}

/// Report unknown: a host node answers but ownership, image and generation
/// are unverified without T02 records, so the runtime is genuinely
/// incomplete evidence, never adopted as running.
pub fn unknown_observation(call: &LifecycleCall) -> Vec<u8> {
    let detail = "A host node answers but ownership is unverified; retry after T02 enumeration.";
    tcontrol_wire::encode_lifecycle_observation(&observation_for(
        call,
        tcontrol_wire::LIFECYCLE_UNKNOWN,
        detail,
    ))
    .into_bytes()
}

/// Report prepared: the selected dependency is verified locally. Prepare
/// never opens admission and never starts a daemon.
pub fn prepared_observation(call: &LifecycleCall, host_present: bool) -> Vec<u8> {
    if host_present {
        return unknown_observation(call);
    }
    let detail = "Selected dependency verified locally; admission never opens here.";
    tcontrol_wire::encode_lifecycle_observation(&observation_for(
        call,
        tcontrol_wire::LIFECYCLE_STOPPED,
        detail,
    ))
    .into_bytes()
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Duration;

    fn test_call() -> LifecycleCall {
        LifecycleCall {
            controller_id: "soda".to_string(),
            controller_instance_id: "dddddddddddddddddddddddddddddddd".to_string(),
            package_id: "tailnet".to_string(),
            package_version: "1.0.0".to_string(),
            package_sha256: "b".repeat(64),
            installation_id: "550e8400-e29b-41d4-a716-446655440000".to_string(),
            generation: "7".to_string(),
            operation: "activate".to_string(),
            deadline_unix: 1792100300,
        }
    }

    fn test_pin() -> ReleasePin {
        ReleasePin {
            package_version: "1.0.0".to_string(),
            package_sha256: "b".repeat(64),
        }
    }

    #[test]
    fn pin_match_accepts_exact_and_rejects_drift() {
        let call = test_call();
        assert!(match_release_pin(&call, &test_pin()).is_ok());
        let mut drifted = test_pin();
        drifted.package_sha256 = "c".repeat(64);
        let err = match_release_pin(&call, &drifted).unwrap_err();
        assert!(err.contains("conflict"), "{err}");
        let err = match_release_pin(&call, &ReleasePin::default()).unwrap_err();
        assert!(err.contains("unavailable"), "{err}");
    }

    #[test]
    fn ensure_refuses_without_spawn() {
        // Relative binary or payload, and a spent deadline, never spawn.
        let err = ensure_tailnet_dependency(
            "relative",
            "/payload.json",
            Instant::now() + Duration::from_secs(5),
        )
        .unwrap_err();
        assert!(err.contains("unavailable"), "{err}");
        let err = ensure_tailnet_dependency(
            "/bin/true",
            "relative.json",
            Instant::now() + Duration::from_secs(5),
        )
        .unwrap_err();
        assert!(err.contains("unavailable"), "{err}");
        let err =
            ensure_tailnet_dependency("/bin/true", "/payload.json", Instant::now()).unwrap_err();
        assert!(err.contains("unavailable"), "{err}");
    }

    #[test]
    fn ensure_maps_exit_status() {
        let far = Instant::now() + Duration::from_secs(30);
        assert!(ensure_tailnet_dependency("/bin/true", "/payload.json", far).is_ok());
        let err = ensure_tailnet_dependency("/bin/false", "/payload.json", far).unwrap_err();
        assert!(err.contains("unavailable"), "{err}");
        let err =
            ensure_tailnet_dependency("/nonexistent-binary", "/payload.json", far).unwrap_err();
        assert!(err.contains("unavailable"), "{err}");
    }

    #[test]
    fn ensure_kills_group_on_deadline() {
        // A real 30-second sleep must die at the 300ms deadline, proving
        // the waiter owns the child process group rather than merely
        // abandoning it.
        use std::os::unix::process::CommandExt;
        use std::process::{Command, Stdio};
        let child = Command::new("/bin/sleep")
            .args(["30"])
            .stdin(Stdio::null())
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .process_group(0)
            .spawn()
            .expect("sleep must spawn");
        let start = Instant::now();
        let err = wait_bounded(child, start + Duration::from_millis(300)).unwrap_err();
        assert!(err.contains("deadline"), "{err}");
        assert!(
            start.elapsed() < Duration::from_secs(10),
            "wait outlived its deadline"
        );
    }

    #[test]
    fn observations_are_stopped_or_unknown_without_grant() {
        let call = test_call();
        let stopped = stopped_observation(&call);
        let observation: tcontrol_wire::LifecycleObservation =
            crate::json::decode_strict_as(&stopped).expect("stopped must decode");
        assert_eq!(observation.state, "stopped");
        assert!(observation.effect_generation.is_none());
        assert!(!observation.admission_ready);
        assert!(observation.error.is_none());
        tcontrol_wire::validate_lifecycle_observation(&observation, &call).expect("valid");
        let unknown = unknown_observation(&call);
        let observation: tcontrol_wire::LifecycleObservation =
            crate::json::decode_strict_as(&unknown).expect("unknown must decode");
        assert_eq!(observation.state, "unknown");
        assert!(!observation.admission_ready);
        tcontrol_wire::validate_lifecycle_observation(&observation, &call).expect("valid");
    }
}
