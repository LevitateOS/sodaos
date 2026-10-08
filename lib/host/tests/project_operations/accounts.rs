use crate::common::{
    container_inspect, deadline, format_inspect, ops, Mock, ED, GO_ACCOUNT_BODY, PID, REV,
};
use crate::AccountReq;
use soda_host::domain::ProjectAccessRequest;

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

#[test]
fn project_access_uses_bound_container_and_confirms_false() {
    let mock = Mock::new(vec![
        Ok(format_inspect(true)),
        Ok(br#"{"login":"alice","identity":7,"administrator":false}"#.to_vec()),
    ]);
    let request = ProjectAccessRequest {
        project: PID.to_string(),
        login: "alice".to_string(),
        identity: 7,
    };
    let status = ops(&mock).project_access(&request, deadline()).unwrap();
    let status: serde_json::Value = serde_json::from_str(&status).unwrap();
    assert_eq!(status["project"], PID);
    assert_eq!(status["login"], "alice");
    assert_eq!(status["identity"], 7);
    assert_eq!(status["administrator"], false);
    let calls = mock.calls.borrow();
    assert_eq!(calls.len(), 2);
    assert_eq!(calls[0].1, "/usr/bin/podman");
    assert_eq!(calls[0].2.last().unwrap(), &format!("soda-{PID}"));
    assert_eq!(calls[1].0, br#"{"login":"alice","identity":7}"#);
    assert_eq!(calls[1].1, "/usr/bin/podman");
    assert_eq!(
        calls[1].2,
        vec![
            "exec".to_string(),
            "--interactive".to_string(),
            REV.to_string(),
            "/usr/libexec/soda/project-account".to_string(),
            "--status".to_string(),
        ]
    );
}

#[test]
fn project_access_refuses_unconfirmed_or_invalid_observations() {
    let request = ProjectAccessRequest {
        project: PID.to_string(),
        login: "alice".to_string(),
        identity: 7,
    };
    for body in [
        br#"{"login":"bob","identity":7,"administrator":true}"#.as_slice(),
        br#"{"login":"alice","identity":8,"administrator":true}"#.as_slice(),
        br#"{"login":"alice","identity":7}"#.as_slice(),
        br#"{"login":"alice","identity":7,"administrator":null}"#.as_slice(),
        br#"{"login":"alice","identity":7,"administrator":true,"extra":1}"#.as_slice(),
    ] {
        let mock = Mock::new(vec![Ok(format_inspect(true)), Ok(body.to_vec())]);
        assert!(ops(&mock).project_access(&request, deadline()).is_err());
    }
    let mock = Mock::new(vec![Ok(format_inspect(false))]);
    assert!(ops(&mock).project_access(&request, deadline()).is_err());
    assert_eq!(mock.calls.borrow().len(), 1);
    let invalid = ProjectAccessRequest {
        login: "root".to_string(),
        ..request
    };
    let mock = Mock::new(vec![]);
    assert!(ops(&mock).project_access(&invalid, deadline()).is_err());
    assert!(mock.calls.borrow().is_empty());
}
