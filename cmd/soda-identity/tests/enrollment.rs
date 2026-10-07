// Enrollment-to-lease integration scenario, split from tests/broker.rs
// (A-R02c). Fixture support lives in tests/common.
mod common;

use common::{binding, controller, fixture_key, subscription, Ephemeral};
use soda_identity::wire::*;

#[test]
fn enrollment_to_lease_lifecycle() {
    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let broker = controller(fixture.store(&fixture_key()));
    let started = broker.start_enrollment(1, "codex", "synthetic").unwrap();
    assert_eq!(started.id, "enrollment-1");
    // A second enrollment while one is unretained is busy.
    assert!(broker.start_enrollment(1, "codex", "other").is_err());
    let read = broker.enrollment(1, "enrollment-1").unwrap();
    let conn = read
        .connection
        .expect("completed enrollment retains a connection");
    assert_eq!(conn.generation, 1);
    assert_eq!(conn.state, "ready");
    // Acquire, register, return.
    let deadline = UnixTime {
        sec: UnixTime::now().sec + 3600,
        nanos: 0,
    };
    let lease = broker
        .acquire(&AcquireRequest {
            repository_id: 0,
            provider_id: "codex".to_string(),
            execution_id: "execution-1".to_string(),
            actor_id: 1,
            connection_id: conn.id.clone(),
            project_id: "project".to_string(),
            kind: "factory".to_string(),
            deadline,
            role: String::new(),
        })
        .unwrap();
    assert_eq!(lease.connection_id, conn.id);
    // Same digest replays the recorded lease instead of reserving another.
    let replay = broker
        .acquire(&AcquireRequest {
            repository_id: 0,
            provider_id: "codex".to_string(),
            execution_id: "execution-1".to_string(),
            actor_id: 1,
            connection_id: conn.id.clone(),
            project_id: "project".to_string(),
            kind: "factory".to_string(),
            deadline,
            role: String::new(),
        })
        .unwrap();
    assert_eq!(replay.id, lease.id);
    let (registered, credential) = broker.register(&lease.id, &binding("factory", 1)).unwrap();
    assert_eq!(credential, subscription());
    assert!(registered.binding.is_some());
    // Owner lease listing strips bindings.
    let listed: Vec<Lease> = serde_json::from_slice(&broker.leases(1, &conn.id).unwrap()).unwrap();
    assert_eq!(listed.len(), 1);
    assert!(listed[0].binding.is_none());
    broker
        .return_lease(&lease.id, &binding("factory", 1), &subscription())
        .unwrap();
    // Return rotated the credential generation.
    let connections: Vec<Connection> = serde_json::from_slice(&broker.connections(1).unwrap()).unwrap();
    assert_eq!(connections[0].generation, 2);
    let execution = broker.get_execution("factory", "execution-1").unwrap();
    assert_eq!(execution.state, "terminal");
}
