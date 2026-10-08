use super::common::{
    deadline, drive_to_start, receipt_bytes, sample_launch, sample_run, wired_factory,
    write_running,
};
use super::mocks::script_success;
use crate::factory::*;

#[test]
fn launch_failed_start_retires_cleanly() {
    let (dir, factory, _exec, term, broker) = wired_factory("launch-startfail");
    let req = sample_launch();
    drive_to_start(&term, &broker, &req.run);
    term.start
        .borrow_mut()
        .push_back(Err(FactoryError::msg("start blew up")));
    term.stop.borrow_mut().push_back(Ok(()));
    term.capture
        .borrow_mut()
        .push_back(Ok(b"{\"auth\":2}".to_vec()));
    broker.returns.borrow_mut().push_back(Ok(()));
    broker.close.borrow_mut().push_back(Ok(()));
    let state = factory.launch(&req, deadline()).unwrap();
    assert_eq!(state.phase, "failed");
    assert_eq!(state.reason, "start-unconfirmed");
    assert_eq!(state.retirement, "confirmed");
    assert!(state.credential_returned);
    let text = String::from_utf8(receipt_bytes(&dir, &req.run.project, &req.run.id)).unwrap();
    assert!(text.contains("\"phase\":\"failed\""), "{text}");
}

#[test]
fn launch_failed_start_uncertainty_matrix() {
    // Each custody failure flips the run uncertain.
    for case in ["return", "capture", "close"] {
        let (_dir, factory, _exec, term, broker) = wired_factory("launch-startunc");
        let req = sample_launch();
        drive_to_start(&term, &broker, &req.run);
        term.start
            .borrow_mut()
            .push_back(Err(FactoryError::msg("nope")));
        term.stop.borrow_mut().push_back(Ok(()));
        match case {
            "return" => {
                term.capture.borrow_mut().push_back(Ok(b"{}".to_vec()));
                broker
                    .returns
                    .borrow_mut()
                    .push_back(Err(FactoryError::msg("return blew up")));
            }
            "capture" => {
                term.capture
                    .borrow_mut()
                    .push_back(Err(FactoryError::msg("capture blew up")));
            }
            _ => {
                term.capture.borrow_mut().push_back(Ok(b"{}".to_vec()));
                broker.returns.borrow_mut().push_back(Ok(()));
            }
        }
        broker.close.borrow_mut().push_back(if case == "close" {
            Err(FactoryError::msg("close blew up"))
        } else {
            Ok(())
        });
        let state = factory.launch(&req, deadline()).unwrap();
        assert_eq!(state.phase, "uncertain", "case {case}");
        assert_eq!(state.reason, "start-unconfirmed", "case {case}");
        assert_eq!(state.retirement, "uncertain", "case {case}");
    }
}

#[test]
fn launch_timed_out_wait_records_deadline() {
    let (_dir, factory, _exec, term, broker) = wired_factory("launch-timeout");
    let req = sample_launch();
    drive_to_start(&term, &broker, &req.run);
    term.start.borrow_mut().push_back(Ok(()));
    term.wait
        .borrow_mut()
        .push_back(Err(FactoryError::DeadlineExceeded));
    term.stop.borrow_mut().push_back(Ok(()));
    term.capture.borrow_mut().push_back(Ok(b"{}".to_vec()));
    broker.returns.borrow_mut().push_back(Ok(()));
    broker.close.borrow_mut().push_back(Ok(()));
    let state = factory.launch(&req, deadline()).unwrap();
    assert_eq!(state.phase, "failed");
    assert_eq!(state.reason, "deadline-exceeded");
    assert_eq!(state.retirement, "confirmed");
    assert!(state.credential_returned);
}

#[test]
fn launch_timed_out_stop_failure_is_uncertain() {
    let (_dir, factory, _exec, term, broker) = wired_factory("launch-timeout-stop");
    let req = sample_launch();
    drive_to_start(&term, &broker, &req.run);
    term.start.borrow_mut().push_back(Ok(()));
    term.wait
        .borrow_mut()
        .push_back(Err(FactoryError::DeadlineExceeded));
    term.stop
        .borrow_mut()
        .push_back(Err(FactoryError::msg("stop blew up")));
    term.capture.borrow_mut().push_back(Ok(b"{}".to_vec()));
    broker.returns.borrow_mut().push_back(Ok(()));
    broker.close.borrow_mut().push_back(Ok(()));
    let state = factory.launch(&req, deadline()).unwrap();
    assert_eq!(state.phase, "uncertain");
    assert_eq!(state.reason, "deadline-exceeded");
    assert_eq!(state.retirement, "uncertain");
}

#[test]
fn launch_finish_maps_exit_codes() {
    // Nonzero exit fails the run but keeps the receipt confirmed.
    let (_dir, factory, _exec, term, broker) = wired_factory("launch-exit3");
    let req = sample_launch();
    script_success(&term, &broker, &req.run, 3, "boom");
    let state = factory.launch(&req, deadline()).unwrap();
    assert_eq!(state.phase, "failed");
    assert_eq!(state.reason, "execution-failed");
    assert_eq!(state.retirement, "confirmed");
    assert_eq!(state.exit_code, Some(3));
    assert_eq!(state.output, "boom");
    assert!(state.credential_returned);

    // Negative exits record no code (Go's `exit >= 0` guard).
    let (_dir, factory, _exec, term, broker) = wired_factory("launch-exitneg");
    let req = sample_launch();
    script_success(&term, &broker, &req.run, -1, "huh");
    let state = factory.launch(&req, deadline()).unwrap();
    assert_eq!(state.phase, "failed");
    assert_eq!(state.exit_code, None);
}

#[test]
fn launch_finish_uncertainty_matrix() {
    for (case, reason) in [
        ("stop", "retirement-unconfirmed"),
        ("capture", "credential-capture-unconfirmed"),
        ("return", "credential-return-unconfirmed"),
    ] {
        let (_dir, factory, _exec, term, broker) = wired_factory("launch-finunc");
        let req = sample_launch();
        drive_to_start(&term, &broker, &req.run);
        term.start.borrow_mut().push_back(Ok(()));
        term.wait.borrow_mut().push_back(Ok((0, "out".to_string())));
        term.stop.borrow_mut().push_back(if case == "stop" {
            Err(FactoryError::msg("stop blew up"))
        } else {
            Ok(())
        });
        if case != "stop" {
            term.capture.borrow_mut().push_back(if case == "capture" {
                Err(FactoryError::msg("capture blew up"))
            } else {
                Ok(b"{}".to_vec())
            });
        }
        if case == "return" {
            broker
                .returns
                .borrow_mut()
                .push_back(Err(FactoryError::msg("return blew up")));
        }
        broker.close.borrow_mut().push_back(Ok(()));
        let state = factory.launch(&req, deadline()).unwrap();
        assert_eq!(state.phase, "uncertain", "case {case}");
        assert_eq!(state.reason, reason, "case {case}");
        assert_eq!(state.retirement, "uncertain", "case {case}");
    }
}

#[test]
fn stale_finish_update_preserves_stop_outcome_and_finish_facts() {
    for (case, stop_result, phase, retirement, reason) in [
        ("confirmed", Ok(()), FACTORY_STOPPED, "confirmed", "stopped"),
        (
            "uncertain",
            Err(FactoryError::msg("stop blew up")),
            FACTORY_UNCERTAIN,
            "uncertain",
            "stop-uncertain",
        ),
    ] {
        let (dir, factory, _exec, term, broker) = wired_factory("stale-finish-stop");
        let run = sample_run();
        write_running(&dir, &run, true);
        let (mut finishing, exists) = factory.load_receipt(&run.project, &run.id).unwrap();
        assert!(exists);
        finishing.phase = FACTORY_COMPLETED.to_string();
        finishing.exit_code = Some(0);
        finishing.output = "finished output".to_string();
        finishing.credential_returned = true;
        finishing.retirement = "confirmed".to_string();
        finishing.reason.clear();

        term.stop.borrow_mut().push_back(stop_result);
        term.capture
            .borrow_mut()
            .push_back(Ok(b"returned auth".to_vec()));
        broker.returns.borrow_mut().push_back(Ok(()));
        broker.close.borrow_mut().push_back(Ok(()));
        let stopped = factory
            .stop(
                &FactoryStop {
                    project: run.project.clone(),
                    id: run.id.clone(),
                },
                deadline(),
            )
            .unwrap();
        assert_eq!(stopped.phase, phase, "case {case}");

        factory.update_receipt(&mut finishing, deadline()).unwrap();
        let (current, exists) = factory.load_receipt(&run.project, &run.id).unwrap();
        assert!(exists);
        assert_eq!(current.phase, phase, "case {case}");
        assert_eq!(current.retirement, retirement, "case {case}");
        assert_eq!(current.reason, reason, "case {case}");
        assert_eq!(current.exit_code, Some(0), "case {case}");
        assert_eq!(current.output, "finished output", "case {case}");
        assert!(current.credential_returned, "case {case}");
    }
}

#[test]
fn stale_delivery_progress_does_not_clear_known_receipt_facts() {
    let (dir, factory, _exec, term, broker) = wired_factory("stale-delivery-progress");
    let run = sample_run();
    write_running(&dir, &run, true);
    let (mut stale, exists) = factory.load_receipt(&run.project, &run.id).unwrap();
    assert!(exists);

    stale.delivered = false;
    stale.credential_returned = false;
    term.stop.borrow_mut().push_back(Ok(()));
    term.capture
        .borrow_mut()
        .push_back(Ok(b"returned auth".to_vec()));
    broker.returns.borrow_mut().push_back(Ok(()));
    broker.close.borrow_mut().push_back(Ok(()));
    let stopped = factory
        .stop(
            &FactoryStop {
                project: run.project.clone(),
                id: run.id.clone(),
            },
            deadline(),
        )
        .unwrap();
    assert_eq!(stopped.phase, FACTORY_STOPPED);

    let (mut finishing, exists) = factory.load_receipt(&run.project, &run.id).unwrap();
    assert!(exists);
    finishing.phase = FACTORY_COMPLETED.to_string();
    finishing.delivered = true;
    finishing.credential_returned = true;
    finishing.exit_code = Some(0);
    finishing.output = "captured output".to_string();
    finishing.retirement = "confirmed".to_string();
    finishing.reason.clear();
    factory.update_receipt(&mut finishing, deadline()).unwrap();

    factory.update_receipt(&mut stale, deadline()).unwrap();

    let (current, exists) = factory.load_receipt(&run.project, &run.id).unwrap();
    assert!(exists);
    assert_eq!(current.phase, FACTORY_STOPPED);
    assert_eq!(current.retirement, "confirmed");
    assert_eq!(current.reason, "stopped");
    assert!(current.delivered);
    assert!(current.credential_returned);
    assert_eq!(current.exit_code, Some(0));
    assert_eq!(current.output, "captured output");
}
