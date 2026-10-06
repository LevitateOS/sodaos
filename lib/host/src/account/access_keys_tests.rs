use super::*;
use super::tests::{container_payload, deadline, pid, test_config, Mock, ED};

#[test]
fn access_keys_preview_observes_without_applying() {
    let id = pid();
    let rev = "d".repeat(64);
    let mock = Mock::new(vec![
        Ok(container_payload(&id)),
        Ok(format!("{{\"revision\":{rev:?},\"keys\":[{ED:?}]}}").into_bytes()),
    ]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    let input = AccessKeys {
        project: id.clone(),
        login: "alice".to_string(),
        identity: 1,
        revision: String::new(),
        keys: vec![],
        apply: false,
    };
    let state = rt.access_keys(&input, deadline()).unwrap();
    assert_eq!(state.revision, rev);
    assert_eq!(state.keys, vec![ED.to_string()]);
    let calls = mock.calls.borrow();
    assert_eq!(calls.len(), 2);
    assert_eq!(calls[1].1, "/usr/bin/podman");
    assert_eq!(
        &calls[1].2[..6],
        &[
            "--remote=false".to_string(),
            "exec".to_string(),
            "--interactive".to_string(),
            "f".repeat(64),
            AGENT_PROGRAM.to_string(),
            "keys".to_string(),
        ]
    );
    assert_eq!(
        String::from_utf8(calls[1].0.clone()).unwrap(),
        "{\"apply\":false,\"identity\":1,\"keys\":[],\"login\":\"alice\",\"revision\":\"\"}"
    );
}

#[test]
fn access_keys_apply_round_trips_preview_and_confirms_set() {
    let id = pid();
    let rev = "e".repeat(64);
    let observed = format!("{{\"revision\":{rev:?},\"keys\":[{ED:?}]}}");
    let mock = Mock::new(vec![
        Ok(container_payload(&id)),
        Ok(container_payload(&id)),
        Ok(observed.clone().into_bytes()),
        Ok(observed.into_bytes()),
    ]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    let input = AccessKeys {
        project: id.clone(),
        login: "alice".to_string(),
        identity: 1,
        revision: rev.clone(),
        keys: vec![ED.to_string()],
        apply: true,
    };
    let state = rt.access_keys(&input, deadline()).unwrap();
    assert_eq!(state.revision, rev);
    assert_eq!(mock.calls.borrow().len(), 4);
    // The preview call carries an empty revision and key set.
    assert!(String::from_utf8(mock.calls.borrow()[2].0.clone())
        .unwrap()
        .contains("\"revision\":\"\""));
}

#[test]
fn access_keys_rejects_drift_and_bad_requests() {
    let id = pid();
    // Invalid request shapes never exec.
    for input in [
        AccessKeys {
            project: id.clone(),
            login: "root".to_string(),
            identity: 1,
            revision: String::new(),
            keys: vec![],
            apply: false,
        },
        AccessKeys {
            project: id.clone(),
            login: "alice".to_string(),
            identity: 1,
            revision: "zz".to_string(),
            keys: vec![],
            apply: true,
        },
        AccessKeys {
            project: id.clone(),
            login: "alice".to_string(),
            identity: 1,
            revision: String::new(),
            keys: vec![ED.to_string()],
            apply: false,
        },
    ] {
        let mock = Mock::new(vec![]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.access_keys(&input, deadline()).unwrap_err(),
            "invalid own-account key operation"
        );
        assert!(mock.calls.borrow().is_empty());
    }
    // Preview revision drift.
    let rev = "f".repeat(64);
    let other = "0".repeat(64);
    let mock = Mock::new(vec![
        Ok(container_payload(&id)),
        Ok(container_payload(&id)),
        Ok(format!("{{\"revision\":{other:?},\"keys\":[]}}").into_bytes()),
    ]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    let input = AccessKeys {
        project: id.clone(),
        login: "alice".to_string(),
        identity: 1,
        revision: rev,
        keys: vec![],
        apply: true,
    };
    assert_eq!(
        rt.access_keys(&input, deadline()).unwrap_err(),
        "native keys changed or are not managed canonical keys"
    );
    // Applied result differs from the requested set.
    let rev = "1".repeat(64);
    let observed = format!("{{\"revision\":{rev:?},\"keys\":[]}}");
    let mock = Mock::new(vec![
        Ok(container_payload(&id)),
        Ok(container_payload(&id)),
        Ok(observed.clone().into_bytes()),
        Ok(observed.into_bytes()),
    ]);
    let rt = Runtime {
        exec: &mock,
        config: test_config(),
    };
    let input = AccessKeys {
        project: id.clone(),
        login: "alice".to_string(),
        identity: 1,
        revision: rev,
        keys: vec![ED.to_string()],
        apply: true,
    };
    assert_eq!(
        rt.access_keys(&input, deadline()).unwrap_err(),
        "native key result differs from requested set"
    );
}
