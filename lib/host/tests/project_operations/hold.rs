use crate::common::{
    deadline, format_inspect, helper_argv, ops, Mock, GO_HOLD_FALSE, GO_HOLD_TRUE, PID,
};
use crate::HoldPreparationReq;

#[test]
fn hold_preparation_confirms_marker() {
    let mock = Mock::new(vec![
        Ok(format_inspect(true)),
        Ok(b"{\"hold\":{\"active\":true,\"revision\":2}}".to_vec()),
    ]);
    let req = HoldPreparationReq::decode(
        format!("{{\"project\":{PID:?},\"hold\":true,\"revision\":2}}").as_bytes(),
    )
    .unwrap();
    let out = ops(&mock).hold_preparation(&req, deadline()).unwrap();
    assert_eq!(out, GO_HOLD_TRUE);
    let calls = mock.calls.borrow();
    assert_eq!(calls.len(), 2);
    assert_eq!(calls[1].2, helper_argv());
    assert_eq!(
        String::from_utf8_lossy(&calls[1].0),
        "{\"op\":\"hold\",\"revision\":2}"
    );
    // Release clears the marker.
    let mock = Mock::new(vec![
        Ok(format_inspect(true)),
        Ok(b"{\"hold\":{\"active\":false,\"revision\":0}}".to_vec()),
    ]);
    let req = HoldPreparationReq::decode(
        format!("{{\"project\":{PID:?},\"hold\":false,\"revision\":0}}").as_bytes(),
    )
    .unwrap();
    let out = ops(&mock).hold_preparation(&req, deadline()).unwrap();
    assert_eq!(out, GO_HOLD_FALSE);
    // A mismatched outcome is not confirmed.
    let mock = Mock::new(vec![
        Ok(format_inspect(true)),
        Ok(b"{\"hold\":{\"active\":false,\"revision\":2}}".to_vec()),
    ]);
    let req = HoldPreparationReq::decode(
        format!("{{\"project\":{PID:?},\"hold\":true,\"revision\":2}}").as_bytes(),
    )
    .unwrap();
    assert_eq!(
        ops(&mock).hold_preparation(&req, deadline()).unwrap_err(),
        "maintenance hold outcome not confirmed"
    );
    // A stopped project cannot take the hold path.
    let mock = Mock::new(vec![Ok(format_inspect(false))]);
    assert_eq!(
        ops(&mock).hold_preparation(&req, deadline()).unwrap_err(),
        "preparation target not ready or isolated"
    );
}
