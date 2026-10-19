//! Tailnet lifecycle: trusted dependency acquisition and exact-owned runtime composition.
//! Fountain owns durable intent; this owner reports facts and bounds native effects.

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

/// Render an incomplete native operation using the SDK lifecycle error domain,
/// which is distinct from Fountain's browser/manager result error vocabulary.
pub(crate) fn failed_observation(call: &LifecycleCall) -> tcontrol_wire::LifecycleObservation {
    let mut observation = observation_for(
        call,
        tcontrol_wire::LIFECYCLE_UNKNOWN,
        "Lifecycle work is incomplete; admission stays closed and explicit retry is required.",
    );
    observation.error = Some(tcontrol_wire::LifecycleError {
        code: "effect_failed".into(),
        message: "Native lifecycle work is unconfirmed.".into(),
    });
    observation
}

#[path = "lifecycle/runtime.rs"]
mod runtime;
pub use runtime::Runtime;
