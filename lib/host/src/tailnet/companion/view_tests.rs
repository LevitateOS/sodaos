use super::tests::{
    companion_with, deadline, file_run, image_id, project_id, project_inspect_json, test_binding,
    test_request, test_view, MockExec, MockTailnet,
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
