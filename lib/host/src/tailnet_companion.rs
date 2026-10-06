//! Tailnet companion orchestration: container lifecycle and observation.
//! Lane D owns this file.
//!
//! Rust port of `internal/host/tailnet/companion.go`. Deadlines are absolute
//! [`Instant`]s passed last; sub-deadlines derive from `Instant::now()` plus a
//! fixed duration, mirroring Go's `context.WithTimeout` nesting.

use std::time::Instant;

use crate::tailnet_domain::{ProjectRequest, ProjectView, RunBinding, RunTarget};
use crate::tailnet_runtime::ProjectRun;

pub const RUNTIME_ROOT: &str = "/run/soda-tailnet";
pub const COMPANION_INSPECT: &str = "{\"id\":{{json .ID}},\"image\":{{json .Image}},\"command\":{{json .Config.CreateCommand}},\"running\":{{json .State.Running}},\"pid\":{{json .State.Pid}},\"started\":{{json .State.StartedAt}},\"execs\":{{json .ExecIDs}}}";

#[path = "tailnet/companion/identity.rs"]
mod identity;

pub use identity::{
    companion_command_matches, companion_execs_valid, companion_identity_matches,
    match_companion_namespaces, validate_companion_record, CompanionRecord,
};

use identity::{companion_resolver, decode_companion_record, stat_metadata};

/// Policy/enrollment surface the companion needs from the Tailnet control
/// plane. Mirrors `domain.Control.{Project,RunBinding,EnrollRun}`.
pub trait TailnetControl {
    fn project(
        &self,
        req: &ProjectRequest,
        cid: &str,
        deadline: Instant,
    ) -> Result<ProjectView, String>;
    fn run_binding(&self, target: &RunTarget, deadline: Instant) -> Result<RunBinding, String>;
    fn enroll_run(
        &self,
        target: &RunTarget,
        recheck: &dyn Fn(Instant) -> Result<(), String>,
        consume: &dyn Fn(Instant, &str) -> Result<(), String>,
        deadline: Instant,
    ) -> Result<(), String>;
}

#[allow(clippy::type_complexity)]
pub struct Companion<E, T> {
    pub exec: E,
    pub tailnet: T,
    pub image: String,
    pub enabled_check:
        Option<Box<dyn Fn(&str, &str, Instant) -> Result<bool, String> + Send + Sync>>,
}

/// Tags the actual failure stage with a fixed label. Native/provider output
/// never enters the chain, so the typed cause survives as a substring.
pub fn preparation_error(stage: &str, err: Option<String>) -> Option<String> {
    err.map(|cause| format!("{stage}: {cause}"))
}

/// Infallible [`preparation_error`] for call sites holding a definite cause.
fn stage_error(stage: &str, err: String) -> String {
    match preparation_error(stage, Some(err)) {
        Some(tagged) => tagged,
        None => unreachable!("preparation_error with a cause always tags"),
    }
}

fn arg_refs(args: &[String]) -> Vec<&str> {
    args.iter().map(String::as_str).collect()
}

fn companion_still_running(
    before: &CompanionRecord,
    after: Option<&CompanionRecord>,
    cli_ok: bool,
) -> bool {
    cli_ok
        && matches!(after, Some(a) if a.id == before.id && a.pid == before.pid && a.started == before.started && a.running)
}

fn is_companion_run_fresh(
    current_err: Option<&String>,
    previous: &ProjectRun,
    run: &ProjectRun,
) -> bool {
    match current_err {
        Some(cause) if cause == "runtime record not found" => true,
        None => previous.target.run != run.target.run,
        _ => false,
    }
}

fn apply_companion_idle_state(view: &mut ProjectView, running: bool) -> bool {
    if running {
        return view.enabled;
    }
    if !view.enabled {
        view.state = "off".to_string();
    }
    false // Off intent is not confirmed disconnection.
}

#[path = "tailnet/companion/execute.rs"]
mod execute;

#[path = "tailnet/companion/start.rs"]
mod start;

#[path = "tailnet/companion/enroll.rs"]
mod enroll;

#[path = "tailnet/companion/stop.rs"]
mod stop;

#[path = "tailnet/companion/view.rs"]
mod view;

#[cfg(test)]
#[path = "tailnet/companion/tests.rs"]
mod tests;

#[cfg(test)]
#[path = "tailnet/companion/identity_tests.rs"]
mod identity_tests;

#[cfg(test)]
#[path = "tailnet/companion/lifecycle_tests.rs"]
mod lifecycle_tests;

#[cfg(test)]
#[path = "tailnet/companion/view_tests.rs"]
mod view_tests;
