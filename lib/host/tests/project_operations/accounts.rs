use crate::common::{container_inspect, deadline, ops, Mock, ED, GO_ACCOUNT_BODY, PID};
use crate::AccountReq;

#[test]
fn account_provisions_login() {
    let mock = Mock::new(vec![
        Ok(container_inspect("7", true, "10.0.0.5")),
        Ok(b"{\"login\":\"alice\",\"identity\":7}".to_vec()),
    ]);
    let raw =
        format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"keys\":[{ED:?}]}}");
    let req = AccountReq::decode(raw.as_bytes()).unwrap();
    ops(&mock).account(&req, deadline()).unwrap();
    let calls = mock.calls.borrow();
    assert_eq!(calls.len(), 2);
    assert_eq!(
        calls[1].2,
        vec![
            "exec".to_string(),
            "--interactive".to_string(),
            format!("soda-{PID}"),
            "/usr/libexec/soda/project-account".to_string(),
        ]
    );
    // Identity equals owner, so admin:true; keys carry MarshalAuthorizedKey newlines.
    assert_eq!(String::from_utf8_lossy(&calls[1].0), GO_ACCOUNT_BODY);
}

#[test]
fn account_refuses_stopped_and_root() {
    let mock = Mock::new(vec![Ok(container_inspect("7", false, ""))]);
    let raw = format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"keys\":[]}}");
    let req = AccountReq::decode(raw.as_bytes()).unwrap();
    assert_eq!(
        ops(&mock).account(&req, deadline()).unwrap_err(),
        "project is stopped"
    );
    assert_eq!(mock.calls.borrow().len(), 1);
    let mock = Mock::new(vec![]);
    let raw = format!("{{\"project\":{PID:?},\"login\":\"root\",\"identity\":7,\"keys\":[]}}");
    let req = AccountReq::decode(raw.as_bytes()).unwrap();
    assert_eq!(
        ops(&mock).account(&req, deadline()).unwrap_err(),
        "invalid project account"
    );
    assert!(mock.calls.borrow().is_empty());
}
