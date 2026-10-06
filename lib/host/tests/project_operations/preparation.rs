use crate::common::{
    approve_response, binding_argv, deadline, fixture_digest, format_inspect, helper_argv,
    helper_ops, helper_state, ops, prepare_body, Mock, FID, GO_HELPER_INSPECT, GO_PREPARE_FULL,
    GO_PREPARE_MISSING, GO_PREPARE_READY, GO_PREPARE_STOPPED, PID, REV,
};
use crate::{InspectPreparationReq, PrepareReq, StopPreparationReq};

#[test]
fn prepare_ready_short_circuit_matches_golden() {
    let digest = fixture_digest();
    let ready = helper_state(
        "soda-coder",
        "ready",
        REV,
        "",
        "",
        false,
        true,
        None,
        "",
        "",
    );
    let mock = Mock::new(vec![
        Ok(format_inspect(true)),
        Ok(b"{}".to_vec()),
        Ok(approve_response(FID, "soda-coder")),
        Ok(ready),
    ]);
    let req = PrepareReq::decode(&prepare_body(FID, "soda-coder", &digest)).unwrap();
    let out = ops(&mock).prepare(&req, deadline()).unwrap();
    assert_eq!(out, GO_PREPARE_READY);
    // ensure, approve, inspect: no clone, no start for an already-ready state.
    assert_eq!(helper_ops(&mock), vec!["ensure", "approve", "inspect"]);
    assert_eq!(mock.calls.borrow().len(), 4);
}

#[test]
fn prepare_validates_before_exec() {
    let digest = fixture_digest();
    let mock = Mock::new(vec![]);
    let bad = prepare_body("../escape", "soda-coder", &digest);
    let req = PrepareReq::decode(&bad).unwrap();
    assert_eq!(
        ops(&mock).prepare(&req, deadline()).unwrap_err(),
        "invalid preparation identity"
    );
    let bad = prepare_body(FID, "soda-coder", REV);
    let req = PrepareReq::decode(&bad).unwrap();
    assert_eq!(
        ops(&mock).prepare(&req, deadline()).unwrap_err(),
        "approved inputs do not match their digest"
    );
    assert!(mock.calls.borrow().is_empty());
}

#[test]
fn inspect_preparation_matches_golden() {
    let ready = helper_state(
        "soda-coder",
        "ready",
        REV,
        "",
        "",
        false,
        true,
        None,
        "",
        "",
    );
    let mock = Mock::new(vec![Ok(format_inspect(false)), Ok(ready)]);
    let req =
        InspectPreparationReq::decode(format!("{{\"project\":{PID:?},\"id\":{FID:?}}}").as_bytes())
            .unwrap();
    let out = ops(&mock).inspect_preparation(&req, deadline()).unwrap();
    assert_eq!(out, GO_PREPARE_READY);
    let calls = mock.calls.borrow();
    assert_eq!(calls.len(), 2);
    assert_eq!(calls[0].2, binding_argv());
    assert_eq!(calls[1].2, helper_argv());
    assert_eq!(String::from_utf8_lossy(&calls[1].0), GO_HELPER_INSPECT);
}

#[test]
fn stop_preparation_confirms_retirement() {
    let stopped = helper_state(
        "soda-coder",
        "stopped",
        REV,
        "",
        "",
        true,
        false,
        None,
        "",
        "",
    );
    let stop_ok = format!("{{\"stopped\":{FID:?},\"retirement\":\"confirmed\",\"known\":true}}");
    let mock = Mock::new(vec![
        Ok(format_inspect(false)),
        Ok(stop_ok.into_bytes()),
        Ok(stopped),
    ]);
    let req =
        StopPreparationReq::decode(format!("{{\"project\":{PID:?},\"id\":{FID:?}}}").as_bytes())
            .unwrap();
    let out = ops(&mock).stop_preparation(&req, deadline()).unwrap();
    assert_eq!(out, GO_PREPARE_STOPPED);
    assert_eq!(helper_ops(&mock), vec!["stop", "inspect"]);
    // An unrecognized retirement marker stays an error, not a state.
    let stop_bad = format!("{{\"stopped\":{FID:?},\"retirement\":\"bogus\",\"known\":true}}");
    let mock = Mock::new(vec![Ok(format_inspect(false)), Ok(stop_bad.into_bytes())]);
    assert_eq!(
        ops(&mock).stop_preparation(&req, deadline()).unwrap_err(),
        "preparation stop unconfirmed"
    );
}

// ---------- response-shape oracles (Go-captured goldens) ----------
#[test]
fn oracle_prepare_state_full_encoding() {
    let tools = "{\"name\":\"python3\",\"path\":\"/usr/bin/python3\",\"version\":\"9.9-test\"}";
    let rich = helper_state(
        "soda-coder",
        "running",
        REV,
        tools,
        "",
        false,
        false,
        Some((0, 1)),
        "ok",
        "fail",
    );
    let mock = Mock::new(vec![Ok(format_inspect(true)), Ok(rich)]);
    let req =
        InspectPreparationReq::decode(format!("{{\"project\":{PID:?},\"id\":{FID:?}}}").as_bytes())
            .unwrap();
    assert_eq!(
        ops(&mock).inspect_preparation(&req, deadline()).unwrap(),
        GO_PREPARE_FULL
    );
}

#[test]
fn oracle_prepare_state_missing_encoding() {
    let waiting = helper_state(
        "soda-coder",
        "waiting",
        REV,
        "",
        "node",
        false,
        false,
        None,
        "",
        "",
    );
    let mock = Mock::new(vec![Ok(format_inspect(true)), Ok(waiting)]);
    let req =
        InspectPreparationReq::decode(format!("{{\"project\":{PID:?},\"id\":{FID:?}}}").as_bytes())
            .unwrap();
    assert_eq!(
        ops(&mock).inspect_preparation(&req, deadline()).unwrap(),
        GO_PREPARE_MISSING
    );
}
