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
mod tests_rest {
    use super::tests::{
        companion_with, deadline, file_run, image_id, project_id, project_inspect_json,
        test_binding, test_request, test_view, MockExec, MockTailnet,
    };
    use crate::tailnet_domain::{project_has_node, project_status};
    use std::cell::RefCell;

    #[test]
    fn disable_flow_marks_runtime_unconfirmed() {
        let id = project_id();
        let id_clone = id.clone();
        let exec = MockExec::with_handler(move |cmd, args| {
            if cmd == "/usr/bin/systemctl" {
                return Ok(Vec::new());
            }
            assert_eq!(cmd, "/usr/bin/podman");
            assert!(args.last().unwrap() == &format!("soda-{id_clone}"));
            Ok(project_inspect_json(&id_clone, &"b".repeat(64), false))
        });
        let tailnet = MockTailnet {
            project_fn: Box::new(|_, _| Ok(test_view())),
            binding_fn: Box::new(|_| panic!("unexpected run_binding call")),
            enroll_targets: RefCell::new(Vec::new()),
            enroll_err: None,
        };
        let c = companion_with(exec, tailnet, &image_id());
        let view = c
            .observe_project_tailnet(&test_request("disable"), &"c".repeat(64), deadline())
            .unwrap();
        // systemctl accepted the stop, then the missing runtime record
        // overwrote the outcome.
        assert_eq!(view.outcome, "runtime-unconfirmed");
        // The parent container is confirmed stopped.
        assert_eq!(view.state, "stopped");
        let calls = c.exec.calls();
        assert_eq!(
            calls[0],
            (
                "/usr/bin/systemctl".to_string(),
                vec![
                    "stop".to_string(),
                    "--no-block".to_string(),
                    format!("soda-tailnet@{id}.service"),
                ]
            )
        );
        // Queueing the disable directly behaves the same.
        let mut direct = test_view();
        c.queue_project_tailnet_disable(&id, &mut direct, deadline());
        assert_eq!(direct.outcome, "runtime-unconfirmed");
    }

    #[test]
    fn queue_start_gates_on_action_and_systemd() {
        let id = project_id();
        let exec = MockExec::with_handler(move |cmd, args| {
            assert_eq!(cmd, "/usr/bin/systemctl");
            assert_eq!(
                args,
                &[
                    "start".to_string(),
                    "--no-block".to_string(),
                    format!("soda-tailnet@{id}.service"),
                ]
            );
            Ok(Vec::new())
        });
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        let mut view = test_view();
        assert!(c.queue_project_tailnet_start(&test_request("inspect"), &mut view, deadline()));
        assert_eq!(view.outcome, "");
        view.enabled = true;
        assert!(c.queue_project_tailnet_start(&test_request("enable"), &mut view, deadline()));
        assert_eq!(view.outcome, "queued");

        let exec = MockExec::with_handler(|_, _| Err("systemd down".to_string()));
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        let mut view = test_view();
        view.enabled = true;
        assert!(!c.queue_project_tailnet_start(&test_request("enable"), &mut view, deadline()));
        assert_eq!(view.outcome, "");
    }

    #[test]
    fn mark_stopped_distinguishes_confirmed_stop() {
        let id = project_id();
        for (running, want) in [(false, "stopped"), (true, "")] {
            let id_clone = id.clone();
            let exec = MockExec::with_handler(move |_, _| {
                Ok(project_inspect_json(&id_clone, &"b".repeat(64), running))
            });
            let c = companion_with(exec, MockTailnet::inert(), &image_id());
            let mut view = test_view();
            c.mark_stopped_project(&id, &mut view, deadline());
            assert_eq!(view.state, want);
        }
        // Failed observation leaves the state untouched.
        let exec = MockExec::with_handler(|_, _| Err("no container".to_string()));
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        let mut view = test_view();
        c.mark_stopped_project(&id, &mut view, deadline());
        assert_eq!(view.state, "");
    }

    #[test]
    fn observe_passthrough_and_fast_paths() {
        // Policy errors propagate with the caller's view dropped, as in Go.
        let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
        let tailnet = MockTailnet {
            project_fn: Box::new(|_, _| Err("policy down".to_string())),
            binding_fn: Box::new(|_| panic!("unexpected run_binding call")),
            enroll_targets: RefCell::new(Vec::new()),
            enroll_err: None,
        };
        let c = companion_with(exec, tailnet, &image_id());
        assert_eq!(
            c.observe_project_tailnet(&test_request("inspect"), &"c".repeat(64), deadline()),
            Err("policy down".to_string())
        );
        // An empty image passes the policy view through untouched.
        let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
        let tailnet = MockTailnet {
            project_fn: Box::new(|_, _| {
                let mut view = test_view();
                view.enabled = true;
                view.state = "policy-state".to_string();
                Ok(view)
            }),
            binding_fn: Box::new(|_| panic!("unexpected run_binding call")),
            enroll_targets: RefCell::new(Vec::new()),
            enroll_err: None,
        };
        let c = companion_with(exec, tailnet, "");
        let view = c
            .observe_project_tailnet(&test_request("inspect"), &"c".repeat(64), deadline())
            .unwrap();
        assert_eq!(view.state, "policy-state");
        assert!(c.exec.calls().is_empty());
        // An unresolvable parent marks a confirmed stop without queueing.
        let id = project_id();
        let id_clone = id.clone();
        let exec = MockExec::with_handler(move |cmd, args| {
            assert_ne!(cmd, "/usr/bin/systemctl");
            assert!(args.last().unwrap() == &format!("soda-{id_clone}"));
            Ok(project_inspect_json(&id_clone, &"b".repeat(64), false))
        });
        let tailnet = MockTailnet {
            project_fn: Box::new(|_, _| {
                let mut view = test_view();
                view.enabled = true;
                Ok(view)
            }),
            binding_fn: Box::new(|_| panic!("unexpected run_binding call")),
            enroll_targets: RefCell::new(Vec::new()),
            enroll_err: None,
        };
        let c = companion_with(exec, tailnet, &image_id());
        let view = c
            .observe_project_tailnet(&test_request("inspect"), &"c".repeat(64), deadline())
            .unwrap();
        assert_eq!(view.state, "stopped");
        assert_eq!(view.outcome, "");
    }

    #[test]
    fn observe_companion_status_fails_closed() {
        let run = file_run();
        // A binding failure leaves the view untouched without any call.
        let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
        let tailnet = MockTailnet {
            project_fn: Box::new(|_, _| panic!("unexpected project call")),
            binding_fn: Box::new(|_| Err("binding down".to_string())),
            enroll_targets: RefCell::new(Vec::new()),
            enroll_err: None,
        };
        let c = companion_with(exec, tailnet, &image_id());
        let mut view = test_view();
        c.observe_companion_status(&run, &mut view, deadline());
        assert_eq!(view.state, "");
        assert!(c.exec.calls().is_empty());
        // A status failure (no runtime state here) also leaves it untouched.
        let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
        let tailnet = MockTailnet {
            project_fn: Box::new(|_, _| panic!("unexpected project call")),
            binding_fn: Box::new(|_| Ok(test_binding())),
            enroll_targets: RefCell::new(Vec::new()),
            enroll_err: None,
        };
        let c = companion_with(exec, tailnet, &image_id());
        let mut view = test_view();
        c.observe_companion_status(&run, &mut view, deadline());
        assert_eq!(view.state, "");
        assert!(view.addresses.is_empty());
        assert_eq!(view.dns_name, "");
    }

    #[test]
    fn project_status_maps_observations() {
        // Pre-login states short-circuit before prefs and binding checks.
        let (state, addresses, dns) = project_status(
            br#"{"BackendState":"NeedsLogin","HaveNodeKey":false}"#,
            b"{}",
            &test_binding(),
        )
        .unwrap();
        assert_eq!(state, "needs-login");
        assert!(addresses.is_empty());
        assert_eq!(dns, "");
        // A matched running node reports connected with addresses and DNS.
        let (state, addresses, dns) = project_status(
            br#"{"BackendState":"Running","HaveNodeKey":true,"CurrentTailnet":{"Name":"tail-abc"},"Self":{"ID":"node1","DNSName":"soda-abc.tail-abc.ts.net","TailscaleIPs":["100.64.0.5"],"Tags":["tag:soda"],"Online":true,"Expired":false}}"#,
            br#"{"WantRunning":true,"CorpDNS":true,"RouteAll":false,"RunSSH":false,"ExitNodeID":"","ExitNodeIP":"","AdvertiseRoutes":[]}"#,
            &test_binding(),
        )
        .unwrap();
        assert_eq!(state, "connected");
        assert_eq!(addresses, vec!["100.64.0.5".to_string()]);
        assert_eq!(dns, "soda-abc.tail-abc.ts.net");
    }

    #[test]
    fn project_has_node_reports_presence() {
        assert_eq!(
            project_has_node(br#"{"BackendState":"Running","HaveNodeKey":true}"#),
            Ok(true)
        );
        assert_eq!(
            project_has_node(br#"{"BackendState":"NeedsLogin","HaveNodeKey":false}"#),
            Ok(false)
        );
    }

    #[test]
    fn confirm_stopped_resolver_ignores_gone_parent() {
        // An unresolvable parent needs no resolver confirmation.
        let exec = MockExec::with_handler(|_, _| Err("no container".to_string()));
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        let run = file_run();
        assert!(c.confirm_stopped_resolver(&run, deadline()).is_ok());
    }
}
