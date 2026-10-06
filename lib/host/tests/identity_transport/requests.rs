use crate::common::{
    deadline, full_binding, full_request, json_reply, FakeBroker, DELIVERY_JSON, LEASE_JSON,
};
use crate::{execution_is_terminal, muse, terminal, BrokerClient, Execution};

#[test]
fn acquire_body_matches_go_oracle() {
    let broker = FakeBroker::start("acquire", |_| json_reply(200, "OK", LEASE_JSON));
    let client = BrokerClient::new(&broker.path);
    let req = terminal::AcquireRequest {
        repository_id: 7,
        provider_id: "codex".to_string(),
        execution_id: "exec".to_string(),
        actor_id: 2,
        connection_id: "conn".to_string(),
        project_id: "project".to_string(),
        kind: "factory".to_string(),
        deadline_secs: 1893553445,
        deadline_nanos: 123456000,
        role: "role-a".to_string(),
    };
    let lease = client.acquire(&req, deadline()).unwrap();
    // Body captured from the live Go client; only the header framing is
    // this client's documented minimal shape.
    let body = "{\"provider_id\":\"\",\"owner_id\":\"0\",\"id\":\"\",\"label\":\"\",\"project_id\":\"\",\"acquire\":{\"repository_id\":\"7\",\"provider_id\":\"codex\",\"execution_id\":\"exec\",\"actor_id\":\"2\",\"connection_id\":\"conn\",\"project_id\":\"project\",\"kind\":\"factory\",\"deadline\":\"2030-01-02T03:04:05.123456Z\",\"role\":\"role-a\"}}\n";
    assert_eq!(body.len(), 280);
    assert_eq!(broker.request(), full_request("/acquire", body));
    assert_eq!(lease.id, "lease-1");
    assert_eq!(lease.actor_id, 2);
    assert_eq!(lease.grant_id, "grant-1");
    assert_eq!(lease.grant_revision, 3);
    assert_eq!(lease.deadline_raw, "2030-01-01T00:00:00Z");
    assert!(lease.deadline.is_some());
    let binding = lease.binding.expect("binding decoded");
    assert_eq!(binding.id, "container");
    assert_eq!(binding.generation, 1);
}

#[test]
fn acquire_omits_zero_repository_and_role() {
    let broker = FakeBroker::start("acquire2", |_| json_reply(200, "OK", LEASE_JSON));
    let client = BrokerClient::new(&broker.path);
    let req = terminal::AcquireRequest {
        provider_id: "muse".to_string(),
        execution_id: "exec-2".to_string(),
        actor_id: 3,
        connection_id: "conn-2".to_string(),
        project_id: "proj-2".to_string(),
        kind: "terminal".to_string(),
        deadline_secs: 1907050150,
        ..terminal::AcquireRequest::default()
    };
    client.acquire(&req, deadline()).unwrap();
    let body = "{\"provider_id\":\"\",\"owner_id\":\"0\",\"id\":\"\",\"label\":\"\",\"project_id\":\"\",\"acquire\":{\"provider_id\":\"muse\",\"execution_id\":\"exec-2\",\"actor_id\":\"3\",\"connection_id\":\"conn-2\",\"project_id\":\"proj-2\",\"kind\":\"terminal\",\"deadline\":\"2030-06-07T08:09:10Z\"}}\n";
    assert_eq!(body.len(), 240);
    assert_eq!(broker.request(), full_request("/acquire", body));
}

#[test]
fn register_full_binding_matches_go_oracle() {
    let broker = FakeBroker::start("register", |_| json_reply(200, "OK", DELIVERY_JSON));
    let client = BrokerClient::new(&broker.path);
    let delivery = client
        .register("lease-9", &full_binding(), deadline())
        .unwrap();
    let body = "{\"provider_id\":\"\",\"owner_id\":\"0\",\"id\":\"lease-9\",\"label\":\"\",\"project_id\":\"\",\"binding\":{\"child_id\":\"child\",\"uid\":1000,\"gid\":1000,\"scope\":\"muse-project\",\"credential_root\":\"/run/cred\",\"invocation_id\":\"inv\",\"kind\":\"terminal\",\"id\":\"term\",\"project\":\"proj\",\"login\":\"login\",\"generation\":4}}\n";
    assert_eq!(body.len(), 282);
    assert_eq!(broker.request(), full_request("/register", body));
    assert_eq!(delivery.credential, Some(b"{}".to_vec()));
    assert_eq!(delivery.lease.id, "lease-9");
    assert_eq!(delivery.lease.kind, "terminal");
}

#[test]
fn register_minimal_binding_matches_go_oracle() {
    let broker = FakeBroker::start("register2", |_| json_reply(200, "OK", DELIVERY_JSON));
    let client = BrokerClient::new(&broker.path);
    let binding = terminal::Binding {
        kind: "factory".to_string(),
        id: "container".to_string(),
        generation: 1,
        ..terminal::Binding::default()
    };
    client.register("lease-9", &binding, deadline()).unwrap();
    let body = "{\"provider_id\":\"\",\"owner_id\":\"0\",\"id\":\"lease-9\",\"label\":\"\",\"project_id\":\"\",\"binding\":{\"kind\":\"factory\",\"id\":\"container\",\"project\":\"\",\"login\":\"\",\"generation\":1}}\n";
    assert_eq!(body.len(), 161);
    assert_eq!(broker.request(), full_request("/register", body));
}

#[test]
fn reconcile_end_available_bodies_match_go_oracle() {
    let broker = FakeBroker::start("simple", |raw| {
        let head = String::from_utf8_lossy(raw).into_owned();
        if head.starts_with("POST /available ") {
            // Full connection records, including a lossless >2^53 `,string`
            // owner; the client projects id/provider/state only.
            json_reply(
                200,
                "OK",
                r#"[{"provider_id":"muse","id":"conn-1","owner_id":"9007199254740993","label":"sub","email":"e","plan":"plus","generation":7,"state":"ready"},{"provider_id":"codex","id":"conn-2","owner_id":"2","label":"","email":"","plan":"","generation":1,"state":"reauth"}]"#,
            )
        } else {
            json_reply(200, "OK", "{}")
        }
    });
    let client = BrokerClient::new(&broker.path);
    client.reconcile_lease("lease-9", deadline()).unwrap();
    let reconcile = "{\"provider_id\":\"\",\"owner_id\":\"0\",\"id\":\"lease-9\",\"label\":\"\",\"project_id\":\"\"}\n";
    assert_eq!(reconcile.len(), 76);
    assert_eq!(
        broker.request(),
        full_request("/reconcile-lease", reconcile)
    );
    client.end_lease(2, "lease-9", deadline()).unwrap();
    let end = "{\"provider_id\":\"\",\"owner_id\":\"2\",\"id\":\"lease-9\",\"label\":\"\",\"project_id\":\"\"}\n";
    assert_eq!(end.len(), 76);
    assert_eq!(broker.request(), full_request("/lease/end", end));
    let connections = client.available(2, "project", deadline()).unwrap();
    let available =
        "{\"provider_id\":\"\",\"owner_id\":\"2\",\"id\":\"\",\"label\":\"\",\"project_id\":\"project\"}\n";
    assert_eq!(available.len(), 76);
    assert_eq!(broker.request(), full_request("/available", available));
    assert_eq!(
        connections,
        vec![
            muse::MuseConnection {
                id: "conn-1".to_string(),
                provider_id: "muse".to_string(),
                state: "ready".to_string(),
            },
            muse::MuseConnection {
                id: "conn-2".to_string(),
                provider_id: "codex".to_string(),
                state: "reauth".to_string(),
            },
        ]
    );
}

#[test]
fn return_execution_bodies_match_go_oracle() {
    let broker = FakeBroker::start("retexec", |raw| {
        let head = String::from_utf8_lossy(raw).into_owned();
        if head.starts_with("POST /execution/get ") {
            json_reply(
                200,
                "OK",
                r#"{"binding":{"kind":"factory","id":"exec-9","project":"","login":"","generation":1},"kind":"factory","execution_id":"exec-9","digest":"d","state":"terminal","lease_id":"lease-1"}"#,
            )
        } else {
            json_reply(200, "OK", "{}")
        }
    });
    let client = BrokerClient::new(&broker.path);
    let binding = terminal::Binding {
        kind: "factory".to_string(),
        id: "container".to_string(),
        generation: 1,
        ..terminal::Binding::default()
    };
    client
        .return_lease("lease-9", &binding, b"{}", deadline())
        .unwrap();
    let with_cred = "{\"provider_id\":\"\",\"owner_id\":\"0\",\"id\":\"lease-9\",\"label\":\"\",\"project_id\":\"\",\"binding\":{\"kind\":\"factory\",\"id\":\"container\",\"project\":\"\",\"login\":\"\",\"generation\":1},\"credential\":\"e30=\"}\n";
    assert_eq!(with_cred.len(), 181);
    assert_eq!(broker.request(), full_request("/return", with_cred));
    client
        .return_lease("lease-9", &binding, b"", deadline())
        .unwrap();
    let without_cred = "{\"provider_id\":\"\",\"owner_id\":\"0\",\"id\":\"lease-9\",\"label\":\"\",\"project_id\":\"\",\"binding\":{\"kind\":\"factory\",\"id\":\"container\",\"project\":\"\",\"login\":\"\",\"generation\":1}}\n";
    assert_eq!(without_cred.len(), 161);
    assert_eq!(broker.request(), full_request("/return", without_cred));
    let execution = client
        .get_execution("factory", "exec-9", deadline())
        .unwrap();
    let exec_body = "{\"provider_id\":\"\",\"owner_id\":\"0\",\"id\":\"\",\"label\":\"\",\"project_id\":\"\",\"kind\":\"factory\",\"execution_id\":\"exec-9\"}\n";
    assert_eq!(exec_body.len(), 110);
    assert_eq!(broker.request(), full_request("/execution/get", exec_body));
    assert_eq!(execution.kind, "factory");
    assert_eq!(execution.execution_id, "exec-9");
    assert_eq!(execution.digest, "d");
    assert_eq!(execution.state, "terminal");
    assert_eq!(execution.lease_id, "lease-1");
    assert_eq!(execution.binding.as_ref().unwrap().id, "exec-9");
    assert!(execution_is_terminal(&execution));
    client
        .close_execution("factory", "exec-9", deadline())
        .unwrap();
    assert_eq!(
        broker.request(),
        full_request("/execution/close", exec_body)
    );
}

#[test]
fn execution_is_terminal_matches_factory() {
    // `internal/host/project/factory.go` settles custody exactly when the
    // execution state is `terminal`; every other state (or error) retries.
    for (state, terminal) in [
        ("terminal", true),
        ("live", false),
        ("pending", false),
        ("", false),
        ("Terminal", false),
    ] {
        let e = Execution {
            state: state.to_string(),
            ..Execution::default()
        };
        assert_eq!(execution_is_terminal(&e), terminal, "state {state:?}");
    }
}

#[test]
fn deadline_format_matches_go_time_json() {
    // (secs, nanos, Go `time.Time` JSON): epoch, leap day, year starts,
    // pre-epoch, fraction trimming and nanosecond rollover.
    let cases = [
        (0, 0, "1970-01-01T00:00:00Z"),
        (1582934400, 0, "2020-02-29T00:00:00Z"),
        (1893456000, 0, "2030-01-01T00:00:00Z"),
        (2147483647, 0, "2038-01-19T03:14:07Z"),
        (-1, 0, "1969-12-31T23:59:59Z"),
        (1893456000, 100000000, "2030-01-01T00:00:00.1Z"),
        (1893456000, 123000000, "2030-01-01T00:00:00.123Z"),
        (1893456000, 123456789, "2030-01-01T00:00:00.123456789Z"),
        (1893455999, 1000000000, "2030-01-01T00:00:00Z"),
    ];
    let broker = FakeBroker::start("dates", |_| json_reply(200, "OK", LEASE_JSON));
    let client = BrokerClient::new(&broker.path);
    for (secs, nanos, expected) in cases {
        let req = terminal::AcquireRequest {
            deadline_secs: secs,
            deadline_nanos: nanos,
            ..terminal::AcquireRequest::default()
        };
        client.acquire(&req, deadline()).unwrap();
        let raw = broker.request();
        let text = String::from_utf8(raw).unwrap();
        assert!(
            text.contains(&format!("\"deadline\":\"{expected}\"")),
            "{secs}.{nanos}: {text}"
        );
    }
}

#[test]
fn html_escaping_matches_go_encoder() {
    let broker = FakeBroker::start("escape", |_| json_reply(200, "OK", "[]"));
    let client = BrokerClient::new(&broker.path);
    client.available(2, "a&b<c>d\"q\\", deadline()).unwrap();
    let body = "{\"provider_id\":\"\",\"owner_id\":\"2\",\"id\":\"\",\"label\":\"\",\"project_id\":\"a\\u0026b\\u003cc\\u003ed\\\"q\\\\\"}\n";
    assert_eq!(broker.request(), full_request("/available", body));
}
