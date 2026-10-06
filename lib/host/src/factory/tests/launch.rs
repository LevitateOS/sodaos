use super::common::{
    container_id, deadline, deadline_text, receipt_bytes, sample_binding, sample_launch,
    sample_lease, wired_factory,
};
use super::mocks::script_success;
use crate::factory::*;

#[test]
fn launch_success_drives_to_completed() {
    let (dir, factory, _exec, term, broker) = wired_factory("launch-ok");
    let req = sample_launch();
    script_success(&term, &broker, &req.run, 0, "all done");
    let state = factory.launch(&req, deadline()).unwrap();
    assert_eq!(state.id, req.run.id);
    assert_eq!(state.project, req.run.project);
    assert_eq!(state.role, "soda-coder");
    assert_eq!(state.phase, "completed");
    assert_eq!(state.exit_code, Some(0));
    assert_eq!(state.output, "all done");
    assert_eq!(state.retirement, "confirmed");
    assert_eq!(state.reason, "");
    assert!(state.credential_returned);
    assert!(state.delivered);
    assert!(!state.live);
    assert_eq!(state.generation, 3);
    assert_eq!(state.lease_id, "lease-1");
    assert_eq!(state.container, container_id());
    assert_eq!(state.unit, factory_unit_name(&req.run.id));
    assert_eq!(state.invocation, "09".repeat(16));
    assert_eq!(state.login, "soda-coder");
    assert_eq!(state.uid, 1001);
    assert_eq!(state.gid, 1001);
    // The broker saw the exact acquisition identity.
    let calls = broker.acquire_calls.borrow();
    assert_eq!(calls.len(), 1);
    assert_eq!(
        calls[0],
        AcquireRequest {
            provider_id: "codex".to_string(),
            execution_id: req.run.id.clone(),
            actor_id: 7,
            connection_id: "conn-1".to_string(),
            project_id: req.run.project.clone(),
            kind: "factory".to_string(),
            deadline: req.run.deadline.clone(),
            role: "soda-coder".to_string(),
        }
    );
    // Reserve got the caller pin and a sane second bound.
    let reserves = term.reserve_calls.borrow();
    assert_eq!(reserves.len(), 1);
    assert_eq!(reserves[0].0, "f".repeat(64));
    assert!(reserves[0].1 > 3500 && reserves[0].1 <= 3600);
    // The execution fence closed exactly once.
    assert_eq!(
        broker.close_calls.borrow().as_slice(),
        &[(String::from("factory"), req.run.id.clone())]
    );
    // The durable receipt carries the outcome.
    let data = receipt_bytes(&dir, &req.run.project, &req.run.id);
    let text = String::from_utf8(data).unwrap();
    assert!(text.contains("\"phase\":\"completed\""), "{text}");
    assert!(text.contains("\"exit_code\":0"), "{text}");
    assert!(text.contains("\"credential_returned\":true"), "{text}");
    // Exact state encoding.
    assert!(state.encode().starts_with(
        "{\"exit_code\":0,\"generation\":3,\"uid\":1001,\"gid\":1001,\"credential_returned\":true,\"live\":false,\"delivered\":true"
    ));
}

#[test]
fn launch_muse_harness_acquires_muse_provider() {
    let (_dir, factory, _exec, term, broker) = wired_factory("launch-muse-provider");
    let mut req = sample_launch();
    req.run.harness = FACTORY_HARNESS_MUSE.to_string();
    script_success(&term, &broker, &req.run, 0, "done");
    let state = factory.launch(&req, deadline()).unwrap();
    assert_eq!(state.phase, "completed");
    // The broker saw the muse acquisition identity: a codex
    // provider against a muse connection denies the run.
    let calls = broker.acquire_calls.borrow();
    assert_eq!(calls.len(), 1);
    assert_eq!(calls[0].provider_id, "muse");
    assert_eq!(calls[0].execution_id, req.run.id);
}

#[test]
fn launch_duplicate_returns_recorded_state() {
    let (_dir, factory, _exec, term, broker) = wired_factory("launch-dup");
    let req = sample_launch();
    script_success(&term, &broker, &req.run, 0, "done");
    let first = factory.launch(&req, deadline()).unwrap();
    assert_eq!(first.phase, "completed");
    // No further scripts: a duplicate must not touch any seam.
    let second = factory.launch(&req, deadline()).unwrap();
    assert_eq!(second, first);
    assert_eq!(broker.acquire_calls.borrow().len(), 1);
}

#[test]
fn launch_prechecks_reject_before_any_seam() {
    let (_dir, factory, _exec, term, broker) = wired_factory("launch-pre");
    // Invalid request.
    let mut bad = sample_launch();
    bad.run.id = "short".to_string();
    assert_eq!(
        factory.launch(&bad, deadline()).unwrap_err().message(),
        "invalid factory run identity"
    );
    // Harness mismatch.
    let mut bad = sample_launch();
    bad.harness_sha256 = "0".repeat(64);
    assert_eq!(
        factory.launch(&bad, deadline()).unwrap_err().message(),
        "factory harness is unavailable until its own proof passes"
    );
    // Past deadline.
    let mut bad = sample_launch();
    bad.run.deadline = deadline_text(-10);
    assert_eq!(
        factory.launch(&bad, deadline()).unwrap_err().message(),
        "run deadline is outside the supervised bound"
    );
    // Beyond the three-hour supervised bound.
    let mut bad = sample_launch();
    bad.run.deadline = deadline_text(3 * 3600 + 60);
    assert_eq!(
        factory.launch(&bad, deadline()).unwrap_err().message(),
        "run deadline is outside the supervised bound"
    );
    // Nothing reached the broker or the terminal.
    assert!(broker.acquire_calls.borrow().is_empty());
    assert!(term.reserve_calls.borrow().is_empty());
}

#[test]
fn launch_acquire_refusal_matrix() {
    // (cause, expected phase, expected reason, stores receipt, propagates)
    let cases: Vec<(FactoryError, &str, &str, bool)> = vec![
        (FactoryError::Busy, "approved", "broker-busy", true),
        (FactoryError::Denied, "failed", "broker-denied", true),
        (
            FactoryError::Uncertain,
            "failed",
            "broker-unavailable",
            true,
        ),
        (
            FactoryError::DeadlineExceeded,
            "failed",
            "deadline-exceeded",
            true,
        ),
    ];
    for (cause, phase, reason, _) in cases {
        let (dir, factory, _exec, _term, broker) = wired_factory("launch-acq");
        let req = sample_launch();
        broker.acquire.borrow_mut().push_back(Err(cause));
        let state = factory.launch(&req, deadline()).unwrap();
        assert_eq!(state.phase, phase);
        assert_eq!(state.reason, reason);
        assert_eq!(state.id, req.run.id);
        let text = String::from_utf8(receipt_bytes(&dir, &req.run.project, &req.run.id)).unwrap();
        assert!(text.contains(&format!("\"phase\":\"{phase}\"")), "{text}");
        assert!(text.contains(&format!("\"reason\":\"{reason}\"")), "{text}");
    }
    // Transport failures stay errors and store nothing new.
    let (dir, factory, _exec, _term, broker) = wired_factory("launch-acq-err");
    let req = sample_launch();
    broker
        .acquire
        .borrow_mut()
        .push_back(Err(FactoryError::msg("identity broker unavailable")));
    assert_eq!(
        factory.launch(&req, deadline()).unwrap_err().message(),
        "identity broker unavailable"
    );
    let text = String::from_utf8(receipt_bytes(&dir, &req.run.project, &req.run.id)).unwrap();
    assert!(text.contains("\"phase\":\"approved\""), "{text}");
    assert!(!text.contains("\"reason\""), "{text}");
}

#[test]
fn launch_abandon_paths_record_refusals() {
    for (refusal, queue) in [("reserve-refused", 0), ("register-refused", 1)] {
        let (dir, factory, _exec, term, broker) = wired_factory("launch-abandon");
        let req = sample_launch();
        broker
            .acquire
            .borrow_mut()
            .push_back(Ok(sample_lease(&req.run)));
        if queue == 0 {
            term.reserve
                .borrow_mut()
                .push_back(Err(FactoryError::msg("reserve blew up")));
        } else {
            term.reserve
                .borrow_mut()
                .push_back(Ok(sample_binding(&req.run)));
            broker
                .register
                .borrow_mut()
                .push_back(Err(FactoryError::msg("register blew up")));
            term.stop.borrow_mut().push_back(Ok(()));
        }
        // Register refusal closes twice (drive, then abandonRun), like Go.
        broker.close.borrow_mut().push_back(Ok(()));
        if queue == 1 {
            broker.close.borrow_mut().push_back(Ok(()));
        }
        let state = factory.launch(&req, deadline()).unwrap();
        assert_eq!(state.phase, "failed");
        assert_eq!(state.reason, refusal);
        // Like Go's abandonRun: phase and reason only, no retirement.
        assert_eq!(state.retirement, "");
        assert_eq!(broker.close_calls.borrow().len(), queue + 1);
        let text = String::from_utf8(receipt_bytes(&dir, &req.run.project, &req.run.id)).unwrap();
        assert!(
            text.contains(&format!("\"reason\":\"{refusal}\"")),
            "{text}"
        );
    }
}

#[test]
fn launch_on_tombstoned_id_returns_stop() {
    let (_dir, factory, _exec, _term, broker) = wired_factory("launch-tomb");
    let req = sample_launch();
    broker.close.borrow_mut().push_back(Ok(()));
    let stopped = factory
        .stop(
            &FactoryStop {
                project: req.run.project.clone(),
                id: req.run.id.clone(),
            },
            deadline(),
        )
        .unwrap();
    assert_eq!(stopped.phase, "stopped");
    // No launch scripts: the tombstone refuses without driving.
    let relaunched = factory.launch(&req, deadline()).unwrap();
    assert_eq!(relaunched.phase, "stopped");
    assert_eq!(relaunched.reason, "stop-before-start");
}
