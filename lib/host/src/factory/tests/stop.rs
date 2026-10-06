use super::common::{
    deadline, receipt_bytes, run_id, sample_launch, sample_lease, sample_run, wired_factory,
    write_running,
};
use super::mocks::script_success;
use crate::factory::receipt::FactoryReceipt;
use crate::factory::*;

fn stop_req(run: &FactoryRun) -> FactoryStop {
    FactoryStop {
        project: run.project.clone(),
        id: run.id.clone(),
    }
}

#[test]
fn stop_before_start_writes_exact_tombstone() {
    let (dir, factory, _exec, _term, broker) = wired_factory("stop-tomb");
    let run = sample_run();
    broker.close.borrow_mut().push_back(Ok(()));
    let state = factory.stop(&stop_req(&run), deadline()).unwrap();
    assert_eq!(state.phase, "stopped");
    assert_eq!(state.retirement, "confirmed");
    assert_eq!(state.reason, "stop-before-start");
    assert_eq!(state.id, run.id);
    assert_eq!(state.unit, factory_unit_name(&run.id));
    let data = receipt_bytes(&dir, &run.project, &run.id);
    let text = String::from_utf8(data).unwrap();
    assert!(
        text.contains("\"deadline\":\"0001-01-01T00:00:00Z\""),
        "{text}"
    );
    assert!(
        text.ends_with("\"retirement\":\"confirmed\",\"reason\":\"stop-before-start\"}"),
        "{text}"
    );
    // A relaunch refuses against the tombstone.
    assert_eq!(broker.close_calls.borrow().len(), 1);
}

#[test]
fn stop_before_start_close_failure_retries() {
    let (_dir, factory, _exec, _term, broker) = wired_factory("stop-retry");
    let run = sample_run();
    broker
        .close
        .borrow_mut()
        .push_back(Err(FactoryError::msg("close blew up")));
    let state = factory.stop(&stop_req(&run), deadline()).unwrap();
    assert_eq!(state.phase, "stopped");
    assert_eq!(state.reason, "broker-close-uncertain");
    // A second stop retries the fence and converges.
    broker.close.borrow_mut().push_back(Ok(()));
    let state = factory.stop(&stop_req(&run), deadline()).unwrap();
    assert_eq!(state.reason, "stop-before-start");
    assert_eq!(broker.close_calls.borrow().len(), 2);
}

#[test]
fn stop_approved_run_uses_unbound_stop() {
    let (_dir, factory, _exec, term, broker) = wired_factory("stop-approved");
    let req = sample_launch();
    broker
        .acquire
        .borrow_mut()
        .push_back(Err(FactoryError::Busy));
    let launched = factory.launch(&req, deadline()).unwrap();
    assert_eq!(launched.phase, "approved");
    // No lease, no binding: unbound stop, reconcile, close.
    term.stop_unbound.borrow_mut().push_back(Ok(()));
    broker.close.borrow_mut().push_back(Ok(()));
    let state = factory.stop(&stop_req(&req.run), deadline()).unwrap();
    assert_eq!(state.phase, "stopped");
    assert_eq!(state.reason, "stopped");
    assert_eq!(state.retirement, "confirmed");
    assert_eq!(broker.reconcile_calls.borrow().len(), 0);
}

#[test]
fn launch_path_respects_stop_owned_uncertain() {
    // A concurrent stop that fenced uncertain owns the outcome: the
    // launch path neither advances past it nor overwrites it.
    let (dir, factory, _exec, _term, broker) = wired_factory("stop-owned");
    let run = sample_run();
    // Abandon fences first, then defers to the stop's outcome.
    broker.close.borrow_mut().push_back(Ok(()));
    let receipt = FactoryReceipt {
        run: run.clone(),
        lease: Some(sample_lease(&run)),
        generation: 1,
        phase: "uncertain".to_string(),
        retirement: "uncertain".to_string(),
        reason: "stop-uncertain".to_string(),
        ..FactoryReceipt::default()
    };
    factory.store_receipt(&receipt).unwrap();
    // Abandon after a failed register keeps the stop's outcome.
    let state = factory
        .abandon_run(&receipt, "register-refused", deadline())
        .unwrap();
    assert_eq!(state.phase, "uncertain");
    assert_eq!(state.reason, "stop-uncertain");
    assert_eq!(state.retirement, "uncertain");
    // A failed acquire keeps it too.
    let state = factory
        .fail_run(&receipt, FactoryError::msg("nope"), deadline())
        .unwrap();
    assert_eq!(state.phase, "uncertain");
    // The start gate refuses to advance past it.
    let mut probe = receipt.clone();
    let held = factory.consume_start(&mut probe, deadline()).unwrap();
    assert_eq!(held.unwrap().phase, "uncertain");
    let text = String::from_utf8(receipt_bytes(&dir, &run.project, &run.id)).unwrap();
    assert!(text.contains("\"phase\":\"uncertain\""), "{text}");
    assert!(text.contains("\"reason\":\"stop-uncertain\""), "{text}");
}

/// Hand-write an approved receipt holding a lease, as if a launch
/// were inside reserve (pre-start, pre-delivery).
fn write_approved_with_lease(dir: &std::path::Path, run: &FactoryRun) {
    let receipt = FactoryReceipt {
        run: run.clone(),
        lease: Some(sample_lease(run)),
        generation: 1,
        phase: "approved".to_string(),
        ..FactoryReceipt::default()
    };
    receipt.validate().unwrap();
    std::fs::write(
        dir.join(format!("{}-{}.json", run.project, run.id)),
        receipt.encode().as_bytes(),
    )
    .unwrap();
}

#[test]
fn stop_before_delivery_confirms_despite_native_failure() {
    // ST15 dependant race: the stop lands while reserve is in flight
    // and the native unit is absent. The fence confirms, the run
    // never started and never took delivery: stopped, not uncertain.
    let (dir, factory, _exec, term, broker) = wired_factory("stop-predelivery");
    let run = sample_run();
    write_approved_with_lease(&dir, &run);
    term.stop_unbound
        .borrow_mut()
        .push_back(Err(FactoryError::msg("no unit yet")));
    broker.close.borrow_mut().push_back(Ok(()));
    let state = factory.stop(&stop_req(&run), deadline()).unwrap();
    assert_eq!(state.phase, "stopped");
    assert_eq!(state.reason, "stopped");
    assert_eq!(state.retirement, "confirmed");
}

#[test]
fn stop_close_retries_before_uncertain() {
    // A transient broker close heals inside the stop; only a fence
    // that never confirms fences the run.
    let (dir, factory, _exec, term, broker) = wired_factory("stop-closeretry");
    let run = sample_run();
    write_approved_with_lease(&dir, &run);
    term.stop_unbound.borrow_mut().push_back(Ok(()));
    broker
        .close
        .borrow_mut()
        .push_back(Err(FactoryError::msg("blip")));
    broker.close.borrow_mut().push_back(Ok(()));
    let state = factory.stop(&stop_req(&run), deadline()).unwrap();
    assert_eq!(state.phase, "stopped");
    assert_eq!(state.retirement, "confirmed");
    assert_eq!(broker.close_calls.borrow().len(), 2);
}

#[test]
fn stop_running_run_reconciles_custody() {
    let (dir, factory, _exec, term, broker) = wired_factory("stop-running");
    let run = sample_run();
    write_running(&dir, &run, true);
    term.stop.borrow_mut().push_back(Ok(()));
    term.capture.borrow_mut().push_back(Ok(b"{}".to_vec()));
    broker.returns.borrow_mut().push_back(Ok(()));
    broker.close.borrow_mut().push_back(Ok(()));
    let state = factory.stop(&stop_req(&run), deadline()).unwrap();
    assert_eq!(state.phase, "stopped");
    assert_eq!(state.reason, "stopped");
    assert!(state.credential_returned);
}

#[test]
fn stop_converges_when_launch_returned_first() {
    let (dir, factory, _exec, term, broker) = wired_factory("stop-converge");
    let run = sample_run();
    write_running(&dir, &run, true);
    term.stop.borrow_mut().push_back(Ok(()));
    term.capture.borrow_mut().push_back(Ok(b"{}".to_vec()));
    broker
        .returns
        .borrow_mut()
        .push_back(Err(FactoryError::msg("already returned")));
    // Terminal execution: custody settled without this stop.
    broker.terminal.borrow_mut().push_back(Ok(true));
    broker.close.borrow_mut().push_back(Ok(()));
    let state = factory.stop(&stop_req(&run), deadline()).unwrap();
    assert_eq!(state.phase, "stopped");
    assert_eq!(state.reason, "stopped");
    // This stop did not return: the flag stays false.
    assert!(!state.credential_returned);
}

#[test]
fn stop_uncertain_when_custody_unsettled() {
    let (dir, factory, _exec, term, broker) = wired_factory("stop-uncertain");
    let run = sample_run();
    write_running(&dir, &run, true);
    term.stop
        .borrow_mut()
        .push_back(Err(FactoryError::msg("stop blew up")));
    term.capture
        .borrow_mut()
        .push_back(Err(FactoryError::msg("capture blew up")));
    broker.close.borrow_mut().push_back(Ok(()));
    let state = factory.stop(&stop_req(&run), deadline()).unwrap();
    assert_eq!(state.phase, "uncertain");
    assert_eq!(state.reason, "stop-uncertain");
    assert_eq!(state.retirement, "uncertain");
    assert_eq!(
        broker.reconcile_calls.borrow().as_slice(),
        &["lease-1".to_string()]
    );
}

#[test]
fn stop_terminal_receipt_is_final() {
    let (_dir, factory, _exec, term, broker) = wired_factory("stop-final");
    let req = sample_launch();
    script_success(&term, &broker, &req.run, 0, "done");
    factory.launch(&req, deadline()).unwrap();
    // No scripts left: terminal receipts never touch a seam.
    let state = factory.stop(&stop_req(&req.run), deadline()).unwrap();
    assert_eq!(state.phase, "completed");
    // Invalid addresses reject.
    assert_eq!(
        factory
            .stop(
                &FactoryStop {
                    project: "bogus".to_string(),
                    id: run_id()
                },
                deadline()
            )
            .unwrap_err()
            .message(),
        "invalid factory run address"
    );
}
