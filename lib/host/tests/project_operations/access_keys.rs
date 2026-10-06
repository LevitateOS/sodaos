use crate::common::{
    binding_argv, deadline, format_inspect, ops, Mock, ED, GO_ACCESS_KEY_STATE,
    GO_ACCESS_KEY_STATE_EMPTY, GO_KEYS_BODY_APPLY, PID, REV,
};
use crate::{account, AccessKeysReq};

#[test]
fn access_keys_observe_matches_golden() {
    let observed = format!("{{\"revision\":{REV:?},\"keys\":[{ED:?}]}}").into_bytes();
    let mock = Mock::new(vec![Ok(format_inspect(true)), Ok(observed)]);
    let req = AccessKeysReq::decode(
        format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7}}").as_bytes(),
    )
    .unwrap();
    let out = ops(&mock).access_keys(&req, deadline()).unwrap();
    assert_eq!(out, GO_ACCESS_KEY_STATE);
    let calls = mock.calls.borrow();
    assert_eq!(calls.len(), 2);
    assert_eq!(calls[0].1, "/usr/bin/podman");
    assert_eq!(calls[0].2, binding_argv());
    assert_eq!(
        calls[1].2,
        vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            "--interactive".to_string(),
            REV.to_string(),
            account::AGENT_PROGRAM.to_string(),
            "keys".to_string(),
        ]
    );
    // Sorted map keys, exactly like the Go request body.
    assert_eq!(
        String::from_utf8_lossy(&calls[1].0),
        "{\"apply\":false,\"identity\":7,\"keys\":[],\"login\":\"alice\",\"revision\":\"\"}"
    );
}

#[test]
fn access_keys_apply_rechecks_revision() {
    let observed = format!("{{\"revision\":{REV:?},\"keys\":[{ED:?}]}}").into_bytes();
    // Apply binds the container before the preview re-observes it.
    let mock = Mock::new(vec![
        Ok(format_inspect(true)),
        Ok(format_inspect(true)),
        Ok(observed.clone()),
        Ok(observed),
    ]);
    let raw = format!(
        "{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"revision\":{REV:?},\"keys\":[{ED:?}],\"apply\":true}}"
    );
    let req = AccessKeysReq::decode(raw.as_bytes()).unwrap();
    let out = ops(&mock).access_keys(&req, deadline()).unwrap();
    assert_eq!(out, GO_ACCESS_KEY_STATE);
    let calls = mock.calls.borrow();
    assert_eq!(calls.len(), 4);
    assert_eq!(String::from_utf8_lossy(&calls[3].0), GO_KEYS_BODY_APPLY);
}

#[test]
fn access_keys_rejects_revision_drift() {
    let drifted = format!("{{\"revision\":{:?},\"keys\":[{ED:?}]}}", "0".repeat(64)).into_bytes();
    let mock = Mock::new(vec![Ok(format_inspect(true)), Ok(drifted)]);
    let raw = format!(
        "{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"revision\":{REV:?},\"keys\":[{ED:?}],\"apply\":true}}"
    );
    let req = AccessKeysReq::decode(raw.as_bytes()).unwrap();
    assert_eq!(
        ops(&mock).access_keys(&req, deadline()).unwrap_err(),
        "native keys changed or are not managed canonical keys"
    );
    // The apply exec never runs after a drifted preview.
    assert_eq!(mock.calls.borrow().len(), 2);
}

#[test]
fn access_keys_validates_before_exec() {
    // Trailing comment: noncanonical (apply shape passes request validation first).
    let commented = format!("{ED} alice@host");
    let raw = format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"revision\":{REV:?},\"keys\":[{commented:?}],\"apply\":true}}");
    let mock = Mock::new(vec![]);
    let req = AccessKeysReq::decode(raw.as_bytes()).unwrap();
    assert_eq!(
        ops(&mock).access_keys(&req, deadline()).unwrap_err(),
        "noncanonical or duplicate development key"
    );
    // 33 keys exceed the bound (apply shape passes request validation first).
    let many = vec![format!("{ED:?}"); 33].join(",");
    let raw = format!(
        "{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7,\"revision\":{REV:?},\"keys\":[{many}],\"apply\":true}}"
    );
    let req = AccessKeysReq::decode(raw.as_bytes()).unwrap();
    assert_eq!(
        ops(&mock).access_keys(&req, deadline()).unwrap_err(),
        "too many development keys"
    );
    assert!(mock.calls.borrow().is_empty());
}

#[test]
fn oracle_access_key_state_empty_encoding() {
    let mock = Mock::new(vec![
        Ok(format_inspect(true)),
        Ok(format!("{{\"revision\":{REV:?},\"keys\":[]}}").into_bytes()),
    ]);
    let req = AccessKeysReq::decode(
        format!("{{\"project\":{PID:?},\"login\":\"alice\",\"identity\":7}}").as_bytes(),
    )
    .unwrap();
    assert_eq!(
        ops(&mock).access_keys(&req, deadline()).unwrap(),
        GO_ACCESS_KEY_STATE_EMPTY
    );
}
