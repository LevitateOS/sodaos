// Broker custody/fencing/revoke integration scenarios, split from
// tests/broker.rs (A-R02c). Fixture support lives in tests/common.
mod common;

use base64::Engine;
use common::{
    binding, connection, controller, fixture_key, stub_provider, subscription, Ephemeral,
    StubRuntime,
};
use soda_identity::control::{self, Controller};
use soda_identity::wire::*;
use std::collections::HashMap;
use std::sync::Mutex;

#[test]
fn store_round_trip() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let store = fixture.store(&fixture_key());
    let conn = connection("codex", 1, "conn-1");
    store.save_connection(&conn, &subscription()).unwrap();
    assert_eq!(
        store.connection("conn-1").unwrap().email,
        "soda-tester@example.invalid"
    );
    assert_eq!(store.credential(&conn).unwrap(), subscription());
    assert_eq!(store.connections(1).unwrap().len(), 1);
    assert_eq!(store.available(1, "project").unwrap().len(), 1);
    // A foreign actor sees the connection only through a grant, without email.
    assert!(store.available(2, "project").unwrap().is_empty());
    let grant = Grant {
        id: "grant-1".to_string(),
        connection_id: "conn-1".to_string(),
        user_id: 2,
        project_id: "project".to_string(),
        revision: 1,
        revoked: false,
    };
    store.save_grant(&grant).unwrap();
    let shared = store.available(2, "project").unwrap();
    assert_eq!(shared.len(), 1);
    assert!(shared[0].email.is_empty());
    store.revoke_grant(&grant).unwrap();
    assert!(store.available(2, "project").unwrap().is_empty());
}

#[test]
fn muse_lease_returns_by_forget() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    // The factory seeds the muse connection from file; enrollment plays
    // no part in the lease cycle under test.
    let store = fixture.store(&fixture_key());
    let conn = connection("muse", 1, "conn-muse");
    store.save_connection(&conn, &subscription()).unwrap();
    let mut providers: HashMap<String, Box<dyn control::Provider>> = HashMap::new();
    providers.insert("muse".to_string(), Box::new(stub_provider()));
    let broker = Controller::new(
        store,
        providers,
        Box::new(StubRuntime {
            credential: subscription(),
            calls: Mutex::new(Vec::new()),
        }),
    )
    .unwrap();
    let deadline = UnixTime {
        sec: UnixTime::now().sec + 3600,
        nanos: 0,
    };
    let lease = broker
        .acquire(&AcquireRequest {
            repository_id: 0,
            provider_id: "muse".to_string(),
            execution_id: "execution-muse".to_string(),
            actor_id: 1,
            connection_id: conn.id.clone(),
            project_id: "project".to_string(),
            kind: "factory".to_string(),
            deadline,
            role: String::new(),
        })
        .unwrap();
    let (registered, credential) = broker.register(&lease.id, &binding("factory", 1)).unwrap();
    assert_eq!(credential, subscription());
    assert!(registered.binding.is_some());
    broker
        .return_lease(&lease.id, &binding("factory", 1), &subscription())
        .unwrap();
    // Borrow, not rotation: the connection generation is untouched, the
    // lease is forgotten, the execution is terminal.
    assert_eq!(broker.connections(1).unwrap()[0].generation, 1);
    assert!(broker.leases(1, &conn.id).unwrap().is_empty());
    let execution = broker.get_execution("factory", "execution-muse").unwrap();
    assert_eq!(execution.state, "terminal");
}

#[test]
fn close_execution_fences_late_registration() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let broker = controller(fixture.store(&fixture_key()));
    broker.start_enrollment(1, "codex", "synthetic").unwrap();
    let read = broker.enrollment(1, "enrollment-1").unwrap();
    let conn = read.connection.unwrap();
    let deadline = UnixTime {
        sec: UnixTime::now().sec + 3600,
        nanos: 0,
    };
    let acquire = AcquireRequest {
        repository_id: 0,
        provider_id: "codex".to_string(),
        execution_id: "execution-9".to_string(),
        actor_id: 1,
        connection_id: conn.id.clone(),
        project_id: "project".to_string(),
        kind: "factory".to_string(),
        deadline,
        role: String::new(),
    };
    let lease = broker.acquire(&acquire).unwrap();
    broker.close_execution("factory", "execution-9").unwrap();
    assert!(broker.register(&lease.id, &binding("factory", 1)).is_err());
    assert!(broker.acquire(&acquire).is_err());
}

#[test]
fn close_preserves_lease_on_non_not_found_read_error() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let broker = controller(fixture.store(&fixture_key()));
    broker.start_enrollment(1, "codex", "synthetic").unwrap();
    let conn = broker
        .enrollment(1, "enrollment-1")
        .unwrap()
        .connection
        .unwrap();
    let acquire = AcquireRequest {
        repository_id: 0,
        provider_id: "codex".to_string(),
        execution_id: "execution-close-read-error".to_string(),
        actor_id: 1,
        connection_id: conn.id.clone(),
        project_id: "project".to_string(),
        kind: "factory".to_string(),
        deadline: UnixTime {
            sec: UnixTime::now().sec + 3600,
            nanos: 0,
        },
        role: String::new(),
    };
    let lease = broker.acquire(&acquire).unwrap();
    let serialized = serde_json::to_vec(&lease).unwrap();
    let encoded = base64::engine::general_purpose::STANDARD.encode(serialized);
    assert!(fixture.exec(&format!(
        "UPDATE identity_leases SET data='null'::jsonb WHERE id='{}'",
        lease.id
    )));

    let error = broker
        .close_execution("factory", "execution-close-read-error")
        .unwrap_err();
    assert_eq!(error.kind(), ErrorKind::Uncertain);
    let execution = broker
        .get_execution("factory", "execution-close-read-error")
        .unwrap();
    assert_eq!(execution.state, "terminal");
    assert_eq!(execution.lease_id, lease.id);

    assert!(fixture.exec(&format!(
        "UPDATE identity_leases SET data=convert_from(decode('{encoded}','base64'),'UTF8')::jsonb WHERE id='{}'",
        lease.id
    )));
    broker
        .close_execution("factory", "execution-close-read-error")
        .unwrap();
    let closed = broker
        .get_execution("factory", "execution-close-read-error")
        .unwrap();
    assert!(closed.lease_id.is_empty());
    assert!(broker.leases(1, &conn.id).unwrap().is_empty());
}

#[test]
fn acquire_reservation_and_execution_link_roll_back_together() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let broker = controller(fixture.store(&fixture_key()));
    broker.start_enrollment(1, "codex", "synthetic").unwrap();
    let conn = broker
        .enrollment(1, "enrollment-1")
        .unwrap()
        .connection
        .unwrap();
    let acquire = AcquireRequest {
        repository_id: 0,
        provider_id: "codex".to_string(),
        execution_id: "execution-atomic-reserve".to_string(),
        actor_id: 1,
        connection_id: conn.id.clone(),
        project_id: "project".to_string(),
        kind: "factory".to_string(),
        deadline: UnixTime {
            sec: UnixTime::now().sec + 3600,
            nanos: 0,
        },
        role: String::new(),
    };
    assert!(fixture.exec(
        "CREATE FUNCTION fail_live_execution_link() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.state = 'live' THEN RAISE EXCEPTION 'injected link failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER fail_live_execution_link BEFORE UPDATE ON identity_executions FOR EACH ROW EXECUTE FUNCTION fail_live_execution_link()"
    ));
    assert!(broker.acquire(&acquire).is_err());
    let pending = broker
        .get_execution("factory", "execution-atomic-reserve")
        .unwrap();
    assert_eq!(pending.state, "pending");
    assert!(pending.lease_id.is_empty());
    assert!(broker.leases(1, &conn.id).unwrap().is_empty());
    assert!(fixture.exec(
        "DO $$ BEGIN IF EXISTS (SELECT 1 FROM identity_events WHERE data->>'execution_id'='execution-atomic-reserve' AND data->>'action'='reserved') THEN RAISE EXCEPTION 'reservation event escaped rollback'; END IF; END $$"
    ));
    assert!(fixture.exec(
        "DROP TRIGGER fail_live_execution_link ON identity_executions; DROP FUNCTION fail_live_execution_link()"
    ));

    let lease = broker.acquire(&acquire).unwrap();
    let execution = broker
        .get_execution("factory", "execution-atomic-reserve")
        .unwrap();
    assert_eq!(execution.state, "live");
    assert_eq!(execution.lease_id, lease.id);
    assert_eq!(broker.leases(1, &conn.id).unwrap().len(), 1);
    assert!(fixture.exec(
        "DO $$ BEGIN IF (SELECT count(*) FROM identity_events WHERE data->>'execution_id'='execution-atomic-reserve' AND data->>'action'='reserved') <> 1 THEN RAISE EXCEPTION 'expected exactly one reservation event'; END IF; END $$"
    ));
}

#[test]
fn end_lease_fences_same_id_reacquire() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let broker = controller(fixture.store(&fixture_key()));
    broker.start_enrollment(1, "codex", "synthetic").unwrap();
    let read = broker.enrollment(1, "enrollment-1").unwrap();
    let conn = read.connection.unwrap();
    let deadline = UnixTime {
        sec: UnixTime::now().sec + 3600,
        nanos: 0,
    };
    // Completed End of a bound lease keeps the execution terminal (I06-F1).
    let acquire = AcquireRequest {
        repository_id: 0,
        provider_id: "codex".to_string(),
        execution_id: "execution-end".to_string(),
        actor_id: 1,
        connection_id: conn.id.clone(),
        project_id: "project".to_string(),
        kind: "factory".to_string(),
        deadline,
        role: String::new(),
    };
    let lease = broker.acquire(&acquire).unwrap();
    broker.register(&lease.id, &binding("factory", 1)).unwrap();
    broker.end_lease(1, &lease.id).unwrap();
    assert_eq!(broker.connections(1).unwrap()[0].generation, 2);
    let execution = broker.get_execution("factory", "execution-end").unwrap();
    assert_eq!(execution.state, "terminal");
    assert!(broker.acquire(&acquire).is_err());
    // Completed End of an admitted-but-unbound lease is terminal too.
    let unbound = AcquireRequest {
        repository_id: 0,
        provider_id: "codex".to_string(),
        execution_id: "execution-end-unbound".to_string(),
        actor_id: 1,
        connection_id: conn.id.clone(),
        project_id: "project".to_string(),
        kind: "factory".to_string(),
        deadline,
        role: String::new(),
    };
    let lease2 = broker.acquire(&unbound).unwrap();
    broker.end_lease(1, &lease2.id).unwrap();
    let execution2 = broker
        .get_execution("factory", "execution-end-unbound")
        .unwrap();
    assert_eq!(execution2.state, "terminal");
    assert!(broker.acquire(&unbound).is_err());
}

#[test]
fn reconcile_unbound_preserves_recovery() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let broker = controller(fixture.store(&fixture_key()));
    broker.start_enrollment(1, "codex", "synthetic").unwrap();
    let read = broker.enrollment(1, "enrollment-1").unwrap();
    let conn = read.connection.unwrap();
    let deadline = UnixTime {
        sec: UnixTime::now().sec + 3600,
        nanos: 0,
    };
    // Recovery of an admitted-but-unreserved lease releases back to pending:
    // the same identity may reserve again (I06-F1 legitimate recovery).
    let acquire = AcquireRequest {
        repository_id: 0,
        provider_id: "codex".to_string(),
        execution_id: "execution-recover".to_string(),
        actor_id: 1,
        connection_id: conn.id.clone(),
        project_id: "project".to_string(),
        kind: "factory".to_string(),
        deadline,
        role: String::new(),
    };
    let lease = broker.acquire(&acquire).unwrap();
    broker.reconcile_lease(&lease.id).unwrap();
    let retry = broker.acquire(&acquire).unwrap();
    assert_ne!(retry.id, lease.id);
    let execution = broker
        .get_execution("factory", "execution-recover")
        .unwrap();
    assert_eq!(execution.state, "live");
}

#[test]
fn close_after_reconcile_is_idempotent() {
    // The ST15 dependant-stop sequence: acquire, reconcile the unbound
    // lease (forget), then close twice (daemon stop, coordinator
    // settle). Every close confirms; late registration stays fenced.
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let broker = controller(fixture.store(&fixture_key()));
    broker.start_enrollment(1, "codex", "synthetic").unwrap();
    let read = broker.enrollment(1, "enrollment-1").unwrap();
    let conn = read.connection.unwrap();
    let deadline = UnixTime {
        sec: UnixTime::now().sec + 3600,
        nanos: 0,
    };
    let lease = broker
        .acquire(&AcquireRequest {
            repository_id: 0,
            provider_id: "codex".to_string(),
            execution_id: "execution-8".to_string(),
            actor_id: 1,
            connection_id: conn.id.clone(),
            project_id: "project".to_string(),
            kind: "factory".to_string(),
            deadline,
            role: String::new(),
        })
        .unwrap();
    broker.reconcile_lease(&lease.id).unwrap();
    broker.close_execution("factory", "execution-8").unwrap();
    broker.close_execution("factory", "execution-8").unwrap();
    assert!(broker.register(&lease.id, &binding("factory", 1)).is_err());
    let execution = broker.get_execution("factory", "execution-8").unwrap();
    assert_eq!(execution.state, "terminal");
}

#[test]
fn revoke_retires_live_leases() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let broker = controller(fixture.store(&fixture_key()));
    broker.start_enrollment(1, "codex", "synthetic").unwrap();
    let read = broker.enrollment(1, "enrollment-1").unwrap();
    let conn = read.connection.unwrap();
    let deadline = UnixTime {
        sec: UnixTime::now().sec + 3600,
        nanos: 0,
    };
    broker
        .acquire(&AcquireRequest {
            repository_id: 0,
            provider_id: "codex".to_string(),
            execution_id: "execution-2".to_string(),
            actor_id: 1,
            connection_id: conn.id.clone(),
            project_id: "project".to_string(),
            kind: "factory".to_string(),
            deadline,
            role: String::new(),
        })
        .unwrap();
    broker.revoke(1, &conn.id).unwrap();
    assert_eq!(broker.connections(1).unwrap()[0].state, "revoked");
    assert!(broker.leases(1, &conn.id).unwrap().is_empty());
}
