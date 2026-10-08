use crate::common::{deadline, error_reply, json_reply, sized_lease, FakeBroker};
use crate::{terminal, BrokerClient, Duration, Instant};

#[test]
fn error_codes_map_like_go() {
    // (wire code, expected error): the `stale`/`missing` errors carry the
    // adapter's substrings plus the exact Go typed-error text.
    let cases = [
        ("denied", "identity authority denied"),
        ("busy", "subscription is in use"),
        ("stale", "identity generation changed (stale)"),
        ("reauth", "subscription requires reconnection"),
        ("missing", "identity execution missing (not found)"),
        ("nope", "identity operation failed"),
        ("", "identity operation failed"),
        ("  stale\n", "identity generation changed (stale)"),
        ("denied and more", "identity operation failed"),
    ];
    for (code, expected) in cases {
        let owned = code.to_string();
        let broker = FakeBroker::start("err", move |_| error_reply(403, "Forbidden", &owned));
        let client = BrokerClient::new(&broker.path);
        let err = client.available(1, "p", deadline()).unwrap_err();
        assert_eq!(err, expected, "code {code:?}");
    }
    // Go keys on `!= 200`, not on 4xx/5xx: a 201 with a JSON body still
    // fails through the error decoder.
    let broker = FakeBroker::start("err201", |_| json_reply(201, "Created", "{}"));
    let client = BrokerClient::new(&broker.path);
    assert_eq!(
        client.available(1, "p", deadline()).unwrap_err(),
        "identity operation failed"
    );
    // ... and a 200 with a non-list body fails in the data decoder,
    // proving the 200 path was taken.
    let broker = FakeBroker::start("err200", |_| json_reply(200, "OK", "{}"));
    let client = BrokerClient::new(&broker.path);
    let err = client.available(1, "p", deadline()).unwrap_err();
    assert!(!err.is_empty());
}

#[test]
fn substring_contract_holds() {
    for (code, needle) in [
        ("denied", "denied"),
        ("missing", "not found"),
        ("stale", "stale"),
    ] {
        let owned = code.to_string();
        let broker = FakeBroker::start("substr", move |_| error_reply(403, "Forbidden", &owned));
        let client = BrokerClient::new(&broker.path);
        let err = client.reconcile_lease("l", deadline()).unwrap_err();
        assert!(err.contains(needle), "{code:?} -> {err:?}");
    }
}

#[test]
fn response_limit_is_512kib() {
    const LIMIT: usize = 512 * 1024;
    // Exactly at the limit decodes; one byte over fails like Go's
    // `LimitReader(body, 512<<10+1)` length check.
    for (size, ok) in [(LIMIT, true), (LIMIT + 1, false)] {
        let body = sized_lease(size);
        assert_eq!(body.len(), size);
        let broker = FakeBroker::start("limit", move |_| json_reply(200, "OK", &body));
        let client = BrokerClient::new(&broker.path);
        let req = terminal::AcquireRequest {
            deadline_secs: 1907050150,
            ..terminal::AcquireRequest::default()
        };
        let out = client.acquire(&req, deadline());
        assert_eq!(out.is_ok(), ok, "size {size}");
        if !ok {
            assert_eq!(out.unwrap_err(), "identity response exceeds limit");
        }
    }
    // A hostile Content-Length never allocates: the client reads only
    // limit+1 bytes, then reports the violation without hanging.
    let broker = FakeBroker::start("huge", |_| {
        format!(
            "HTTP/1.1 200 OK\r\nContent-Length: 100000000\r\nConnection: close\r\n\r\n{}",
            "x".repeat(LIMIT + 1)
        )
        .into_bytes()
    });
    let client = BrokerClient::new(&broker.path);
    let req = terminal::AcquireRequest {
        deadline_secs: 1907050150,
        ..terminal::AcquireRequest::default()
    };
    assert_eq!(
        client.acquire(&req, deadline()).unwrap_err(),
        "identity response exceeds limit"
    );
}

#[test]
fn strict_response_decode() {
    // Unknown fields, trailing data and mistyped `,string`/integer fields
    // all fail, like Go's `DisallowUnknownFields` binding.
    let cases = [
        (
            r#"{"id":"l","bogus":1,"connection_id":"c","generation":1,"actor_id":"2","project_id":"p","execution_id":"e","kind":"factory","provider_id":"x","deadline":"2030-01-01T00:00:00Z"}"#,
            "unknown field",
        ),
        (
            r#"{"id":"l","connection_id":"c","generation":1,"actor_id":"2","project_id":"p","execution_id":"e","kind":"factory","provider_id":"x","deadline":"2030-01-01T00:00:00Z"} trailing"#,
            "trailing",
        ),
        (
            r#"{"id":"l","connection_id":"c","generation":1,"actor_id":"x","project_id":"p","execution_id":"e","kind":"factory","provider_id":"x","deadline":"2030-01-01T00:00:00Z"}"#,
            "actor_id",
        ),
        (
            r#"{"id":"l","connection_id":"c","generation":1.5,"actor_id":"2","project_id":"p","execution_id":"e","kind":"factory","provider_id":"x","deadline":"2030-01-01T00:00:00Z"}"#,
            "",
        ),
        (
            r#"{"id":"l","connection_id":"c","generation":1,"actor_id":"2","project_id":"p","execution_id":"e","kind":"factory","provider_id":"x","deadline":"not-a-time"}"#,
            "deadline",
        ),
        (r#"[]"#, ""),
    ];
    for (body, needle) in cases {
        let owned = body.to_string();
        let broker = FakeBroker::start("strict", move |_| json_reply(200, "OK", &owned));
        let client = BrokerClient::new(&broker.path);
        let req = terminal::AcquireRequest {
            deadline_secs: 1907050150,
            ..terminal::AcquireRequest::default()
        };
        let err = client.acquire(&req, deadline()).unwrap_err();
        if needle.is_empty() {
            // Malformed numeric and object shapes are rejected; Serde's
            // diagnostic wording is not a caller contract.
            assert_ne!(err, "identity broker unavailable", "{body:?} -> {err:?}");
        } else {
            assert!(err.contains(needle), "{body:?} -> {err:?}");
        }
    }
    // Available: element-level strictness too.
    for (body, needle) in [
        (
            r#"[{"provider_id":"muse","id":"c","owner_id":"1","label":"","email":"","plan":"","generation":1,"state":"ready","bogus":1}]"#,
            "unknown field",
        ),
        (
            r#"[{"provider_id":"muse","id":"c","owner_id":"nope","label":"","email":"","plan":"","generation":1,"state":"ready"}]"#,
            "owner_id",
        ),
        (
            r#"[{"provider_id":"muse","id":"c","owner_id":"1","label":"","email":"","plan":"","generation":"1","state":"ready"}]"#,
            "generation",
        ),
    ] {
        let owned = body.to_string();
        let broker = FakeBroker::start("strict2", move |_| json_reply(200, "OK", &owned));
        let client = BrokerClient::new(&broker.path);
        let err = client.available(1, "p", deadline()).unwrap_err();
        if needle == "generation" {
            assert_ne!(err, "identity broker unavailable");
        } else {
            assert!(err.contains(needle), "{body:?} -> {err:?}");
        }
    }
}

#[test]
fn execution_binding_reuses_terminal_wire_decode() {
    let broker = FakeBroker::start("binding-wire", |_| {
        json_reply(
            200,
            "OK",
            r#"{"binding":{"CHILD_ID":"first","child_id":"second","Child_Id":null,"UID":7,"uid":null,"kind":"factory","id":"exec-9","generation":1},"kind":"factory"}"#,
        )
    });
    let execution = BrokerClient::new(&broker.path)
        .get_execution("factory", "exec-9", deadline())
        .unwrap();
    let binding = execution.binding.unwrap();
    assert_eq!(binding.child_id, "second");
    assert_eq!(binding.uid, 7);

    let broker = FakeBroker::start("binding-unknown", |_| {
        json_reply(
            200,
            "OK",
            r#"{"binding":{"kind":"factory","unknown":1},"kind":"factory"}"#,
        )
    });
    assert!(BrokerClient::new(&broker.path)
        .get_execution("factory", "exec-9", deadline())
        .is_err());

    // Whole-document lease decoding stays strict even though broker response
    // decoding accepts case-folded duplicates with the last non-null value.
    assert!(terminal::Lease::decode(br#"{"binding":{"id":"first","id":"second"}}"#).is_err());
}

#[test]
fn deadline_and_transport_failures() {
    // An expired deadline fails before connecting.
    let broker = FakeBroker::start("expired", |_| json_reply(200, "OK", "{}"));
    let client = BrokerClient::new(&broker.path);
    let err = client
        .reconcile_lease("l", Instant::now() - Duration::from_secs(1))
        .unwrap_err();
    assert_eq!(err, "identity broker unavailable");
    assert_eq!(broker.request_count(), 0);
    // A missing socket fails the same way Go's dial error does.
    let ghost = BrokerClient::new("target/iclient-oracle/no-such-socket.sock");
    assert_eq!(
        ghost.reconcile_lease("l", deadline()).unwrap_err(),
        "identity broker unavailable"
    );
    // A silent broker hits the socket timeout. The stub sleeps past the
    // client deadline but briefly, so the drop-time join stays fast.
    let broker = FakeBroker::start("silent", |_| {
        std::thread::sleep(Duration::from_secs(2));
        json_reply(200, "OK", "{}")
    });
    let client = BrokerClient::new(&broker.path);
    let err = client
        .reconcile_lease("l", Instant::now() + Duration::from_millis(300))
        .unwrap_err();
    assert_eq!(err, "identity broker unavailable");
    // Malformed framing fails as unavailable, like Go's Transport errors.
    for (tag, reply) in [
        ("garbage", b"this is not http\r\n\r\n".to_vec()),
        (
            "badlen",
            b"HTTP/1.1 200 OK\r\nContent-Length: nope\r\n\r\n{}".to_vec(),
        ),
        (
            "duplen",
            b"HTTP/1.1 200 OK\r\nContent-Length: 2\r\nContent-Length: 3\r\n\r\n{}".to_vec(),
        ),
        (
            "short",
            b"HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\n{}".to_vec(),
        ),
        ("empty", b"".to_vec()),
    ] {
        let broker = FakeBroker::start(tag, move |_| reply.clone());
        let client = BrokerClient::new(&broker.path);
        assert_eq!(
            client.reconcile_lease("l", deadline()).unwrap_err(),
            "identity broker unavailable",
            "case {tag}",
        );
    }
}

#[test]
fn framing_edge_cases_match_go_transport() {
    // Close-delimited body without Content-Length.
    let broker = FakeBroker::start("close", |_| {
        b"HTTP/1.1 200 OK\r\nConnection: close\r\n\r\n{}".to_vec()
    });
    BrokerClient::new(&broker.path)
        .reconcile_lease("l", deadline())
        .unwrap();
    // Chunked body.
    let broker = FakeBroker::start("chunked", |_| {
        b"HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n1\r\n{\r\n1\r\n}\r\n0\r\n\r\n"
            .to_vec()
    });
    BrokerClient::new(&broker.path)
        .reconcile_lease("l", deadline())
        .unwrap();
    // Hyper's selected HTTP/1 parser accepts the bare-LF response accepted
    // by Go's textproto parser.
    let broker = FakeBroker::start("barelf", |_| {
        b"HTTP/1.1 200 OK\nContent-Length: 2\n\n{}".to_vec()
    });
    BrokerClient::new(&broker.path)
        .reconcile_lease("l", deadline())
        .unwrap();
    // Chunk extensions and mixed-case framing names.
    let broker = FakeBroker::start("chunkext", |_| {
        b"HTTP/1.1 200 OK\r\ntransfer-encoding: Chunked\r\n\r\n2;ext=1\r\n{}\r\n0\r\n\r\n".to_vec()
    });
    BrokerClient::new(&broker.path)
        .reconcile_lease("l", deadline())
        .unwrap();
}
