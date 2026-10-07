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

fn listed_connections(bytes: Vec<u8>) -> Vec<Connection> {
    serde_json::from_slice(&bytes).unwrap()
}

fn listed_leases(bytes: Vec<u8>) -> Vec<Lease> {
    serde_json::from_slice(&bytes).unwrap()
}

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
    assert_eq!(listed_connections(store.connections(1).unwrap()).len(), 1);
    assert_eq!(listed_connections(store.available(1, "project").unwrap()).len(), 1);
    // A foreign actor sees the connection only through a grant, without email.
    assert!(listed_connections(store.available(2, "project").unwrap()).is_empty());
    let grant = Grant {
        id: "grant-1".to_string(),
        connection_id: "conn-1".to_string(),
        user_id: 2,
        project_id: "project".to_string(),
        revision: 1,
        revoked: false,
    };
    store.save_grant(&grant).unwrap();
    let grants: Vec<Grant> = serde_json::from_slice(&store.grants_for("conn-1").unwrap()).unwrap();
    assert_eq!(grants.len(), 1);
    let shared = listed_connections(store.available(2, "project").unwrap());
    assert_eq!(shared.len(), 1);
    assert!(shared[0].email.is_empty());
    store.revoke_grant(&grant).unwrap();
    assert!(listed_connections(store.available(2, "project").unwrap()).is_empty());
}

#[test]
fn connection_listing_refuses_oversized_rows_and_aggregate_output() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let store = fixture.store(&fixture_key());
    let mut first = connection("codex", 1, "large-1");
    first.label = "x".repeat(270_000);
    let mut second = connection("codex", 1, "large-2");
    second.label = "x".repeat(270_000);
    store.save_connection(&first, &subscription()).unwrap();
    store.save_connection(&second, &subscription()).unwrap();
    assert!(store.connections(1).is_err(), "the list must refuse instead of returning a partial prefix");

    let mut oversized = connection("codex", 2, "oversized-row");
    oversized.label = "x".repeat(530_000);
    store.save_connection(&oversized, &subscription()).unwrap();
    assert!(store.connections(2).is_err(), "a single oversized row must fail before JSON decode");

    let grant_conn = connection("codex", 3, "large-grant-list");
    store.save_connection(&grant_conn, &subscription()).unwrap();
    // Keep each indexed project id safely below PostgreSQL's index-entry
    // limit while making the complete compact JSON list exceed 512 KiB.
    for n in 0..600 {
        store.save_grant(&Grant {
            id: format!("large-grant-{n}"),
            connection_id: grant_conn.id.clone(),
            user_id: 4,
            project_id: format!("{}-{n:04}", "p".repeat(900)),
            revision: 1,
            revoked: false,
        }).unwrap();
    }
    assert!(store.grants_for(&grant_conn.id).is_err(), "grant lists also refuse a partial prefix over the response budget");
}

#[test]
fn reconcile_keyset_walk_finishes_while_each_page_is_deleted() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let store = fixture.store(&fixture_key());
    let conn = connection("muse", 1, "many-muse-leases");
    store.save_connection(&conn, &subscription()).unwrap();
    let mut providers: HashMap<String, Box<dyn control::Provider>> = HashMap::new();
    providers.insert("muse".to_string(), Box::new(stub_provider()));
    let broker = Controller::new(
        store,
        providers,
        Box::new(StubRuntime { credential: subscription(), calls: Mutex::new(Vec::new()) }),
    ).unwrap();
    let deadline = UnixTime { sec: UnixTime::now().sec + 3600, nanos: 0 };
    for n in 0..65 {
        broker.acquire(&AcquireRequest {
            repository_id: 0,
            provider_id: "muse".to_string(),
            execution_id: format!("execution-page-{n}"),
            actor_id: 1,
            connection_id: conn.id.clone(),
            project_id: "project".to_string(),
            kind: "factory".to_string(),
            deadline,
            role: String::new(),
        }).unwrap();
    }
    assert_eq!(listed_leases(broker.leases(1, &conn.id).unwrap()).len(), 65);
    broker.reconcile().unwrap();
    assert!(listed_leases(broker.leases(1, &conn.id).unwrap()).is_empty());
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
    assert_eq!(listed_connections(broker.connections(1).unwrap())[0].generation, 1);
    assert!(listed_leases(broker.leases(1, &conn.id).unwrap()).is_empty());
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
    assert!(listed_leases(broker.leases(1, &conn.id).unwrap()).is_empty());
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
    assert!(listed_leases(broker.leases(1, &conn.id).unwrap()).is_empty());
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
    assert_eq!(listed_leases(broker.leases(1, &conn.id).unwrap()).len(), 1);
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
    assert_eq!(listed_connections(broker.connections(1).unwrap())[0].generation, 2);
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
    assert_eq!(listed_connections(broker.connections(1).unwrap())[0].state, "revoked");
    assert!(listed_leases(broker.leases(1, &conn.id).unwrap()).is_empty());
}

#[test]
fn grant_revoke_preserves_ungranted_and_other_grant_leases() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let store = fixture.store(&fixture_key());
    let target_conn = connection("codex", 1, "grant-target-connection");
    let other_conn = connection("codex", 1, "grant-other-connection");
    let ungranted_conn = connection("codex", 1, "grant-ungranted-connection");
    for conn in [&target_conn, &other_conn, &ungranted_conn] {
        store.save_connection(conn, &subscription()).unwrap();
    }
    let target_grant = Grant {
        id: "grant-target".to_string(),
        connection_id: target_conn.id.clone(),
        user_id: 2,
        project_id: "target-project".to_string(),
        revision: 1,
        revoked: false,
    };
    let other_grant = Grant {
        id: "grant-other".to_string(),
        connection_id: other_conn.id.clone(),
        user_id: 2,
        project_id: "other-project".to_string(),
        revision: 1,
        revoked: false,
    };
    store.save_grant(&target_grant).unwrap();
    store.save_grant(&other_grant).unwrap();
    let broker = controller(store);
    let deadline = UnixTime { sec: UnixTime::now().sec + 3600, nanos: 0 };
    for (execution, actor, conn, project) in [
        ("target-execution", 2, &target_conn, "target-project"),
        ("other-execution", 2, &other_conn, "other-project"),
        ("ungranted-execution", 1, &ungranted_conn, "owner-project"),
    ] {
        broker.acquire(&AcquireRequest {
            repository_id: 0,
            provider_id: "codex".to_string(),
            execution_id: execution.to_string(),
            actor_id: actor,
            connection_id: conn.id.clone(),
            project_id: project.to_string(),
            kind: "factory".to_string(),
            deadline,
            role: String::new(),
        }).unwrap();
    }

    broker.revoke_grant(1, &target_grant.id).unwrap();
    assert!(listed_leases(broker.leases(1, &target_conn.id).unwrap()).is_empty());
    assert_eq!(listed_leases(broker.leases(1, &other_conn.id).unwrap()).len(), 1);
    assert_eq!(listed_leases(broker.leases(1, &ungranted_conn.id).unwrap()).len(), 1);
}

#[test]
fn malformed_scoped_lease_selector_refuses_before_retirement() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let store = fixture.store(&fixture_key());
    let conn = connection("codex", 1, "malformed-scope-connection");
    let connection = connection("codex", 1, "malformed-connection-scope");
    store.save_connection(&conn, &subscription()).unwrap();
    store.save_connection(&connection, &subscription()).unwrap();
    let grant = Grant {
        id: "malformed-scope-grant".to_string(),
        connection_id: conn.id.clone(),
        user_id: 2,
        project_id: "project".to_string(),
        revision: 1,
        revoked: false,
    };
    store.save_grant(&grant).unwrap();
    let broker = controller(store);
    let deadline = UnixTime { sec: UnixTime::now().sec + 3600, nanos: 0 };
    let lease = broker.acquire(&AcquireRequest {
        repository_id: 0,
        provider_id: "codex".to_string(),
        execution_id: "malformed-selector-execution".to_string(),
        actor_id: 2,
        connection_id: conn.id.clone(),
        project_id: "project".to_string(),
        kind: "factory".to_string(),
        deadline,
        role: String::new(),
    }).unwrap();
    let conn_lease = broker.acquire(&AcquireRequest {
        repository_id: 0,
        provider_id: "codex".to_string(),
        execution_id: "malformed-connection-execution".to_string(),
        actor_id: 1,
        connection_id: connection.id.clone(),
        project_id: "project".to_string(),
        kind: "factory".to_string(),
        deadline,
        role: String::new(),
    }).unwrap();
    assert!(fixture.exec("UPDATE identity_leases SET data=jsonb_set(data,'{grant_id}','7'::jsonb) WHERE data->>'execution_id'='malformed-selector-execution'"));

    assert!(broker.revoke_grant(1, &grant.id).is_err());
    let execution = broker.get_execution("factory", "malformed-selector-execution").unwrap();
    assert_eq!(execution.state, "live", "invalid grant selectors must fail before any lease retirement");
    assert_eq!(execution.lease_id, lease.id);

    // The independent owner lease still exists; its JSON selector is now
    // inconsistent with the relational connection scope.
    assert!(fixture.exec("UPDATE identity_leases SET data=jsonb_set(data,'{connection_id}','7'::jsonb) WHERE data->>'execution_id'='malformed-connection-execution'"));
    assert!(broker.revoke(1, &connection.id).is_err());
    let execution = broker.get_execution("factory", "malformed-connection-execution").unwrap();
    assert_eq!(execution.state, "live", "invalid connection selectors must fail before any lease retirement");
    assert_eq!(execution.lease_id, conn_lease.id);
}

#[test]
fn connection_lists_refuse_relational_json_owner_and_state_mismatch() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let store = fixture.store(&fixture_key());
    let conn = connection("codex", 1, "malformed-connection-owner");
    store.save_connection(&conn, &subscription()).unwrap();
    store.save_grant(&Grant {
        id: "malformed-connection-grant".to_string(),
        connection_id: conn.id.clone(),
        user_id: 2,
        project_id: "shared".to_string(),
        revision: 1,
        revoked: false,
    }).unwrap();

    assert!(fixture.exec("UPDATE identity_connections SET data=jsonb_set(data,'{owner_id}','\"2\"'::jsonb) WHERE id='malformed-connection-owner'"));
    assert!(store.connections(1).is_err(), "owner listing must reject inconsistent owner metadata");
    assert!(store.available(2, "shared").is_err(), "a granted actor must not receive email from mismatched JSON owner metadata");

    assert!(fixture.exec("UPDATE identity_connections SET data=jsonb_set(jsonb_set(data,'{owner_id}','\"1\"'::jsonb),'{state}','\"revoked\"'::jsonb) WHERE id='malformed-connection-owner'"));
    assert!(store.available(2, "shared").is_err(), "available listing must reject inconsistent state metadata");
}
