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
    let connections: Vec<Connection> =
        serde_json::from_slice(&broker.connections(1).unwrap()).unwrap();
    assert_eq!(connections[0].generation, 2);
    let execution = broker.get_execution("factory", "execution-1").unwrap();
    assert_eq!(execution.state, "terminal");

    // A retained actor/Project grant survives ordinary Codex credential
    // maintenance between Factory roles. Each new execution leases the current
    // generation; no owner grant or sponsorship rewrite is needed.
    let grant = broker
        .create_grant(
            1,
            &GrantRequest {
                connection_id: conn.id.clone(),
                user_id: 2,
                project_id: "project".into(),
                confirm_subscription: true,
                confirm_credential_exposure: true,
            },
        )
        .unwrap();
    for (role, generation, id) in [
        ("soda-coder", 2, "grant-coder"),
        ("soda-reviewer", 3, "grant-reviewer"),
    ] {
        let lease = broker
            .acquire(&AcquireRequest {
                repository_id: 7,
                provider_id: "codex".into(),
                execution_id: id.into(),
                actor_id: 2,
                connection_id: conn.id.clone(),
                project_id: "project".into(),
                kind: "factory".into(),
                deadline,
                role: role.into(),
            })
            .unwrap();
        assert_eq!(lease.generation, generation);
        assert_eq!(lease.grant_id, grant.id);
        assert_eq!(lease.grant_revision, grant.revision);
        let mut native = binding("factory", generation);
        native.id = id.into();
        broker.register(&lease.id, &native).unwrap();
        broker
            .return_lease(&lease.id, &native, &subscription())
            .unwrap();
        assert_eq!(
            broker.get_execution("factory", id).unwrap().state,
            "terminal"
        );
    }
    let grants: Vec<Grant> = serde_json::from_slice(&broker.grants(1, &conn.id).unwrap()).unwrap();
    assert_eq!(grants.len(), 1);
    assert_eq!(grants[0].id, grant.id);
    assert_eq!(grants[0].revision, grant.revision);
    assert!(!grants[0].revoked);
    let connections: Vec<Connection> =
        serde_json::from_slice(&broker.connections(1).unwrap()).unwrap();
    assert_eq!(connections[0].generation, 4);
    let leases: Vec<Lease> = serde_json::from_slice(&broker.leases(1, &conn.id).unwrap()).unwrap();
    assert!(leases.is_empty());
}

#[test]
fn slow_provider_start_does_not_block_reads_or_admit_a_duplicate() {
    use soda_identity::control::{Controller, EnrollmentSession, Provider, Runtime};
    use std::collections::HashMap;
    use std::sync::{mpsc, Arc, Mutex};
    use std::time::{Duration, Instant};

    struct GatedProvider {
        entered: mpsc::SyncSender<()>,
        release: Mutex<mpsc::Receiver<()>>,
    }
    struct Session(Enrollment);
    impl EnrollmentSession for Session {
        fn snapshot(&self) -> Enrollment {
            self.0.clone()
        }
        fn finish(&self) -> Result<(Connection, Vec<u8>), Error> {
            Err(Error::internal("unused test session"))
        }
        fn close(&self) -> Result<(), Error> {
            Ok(())
        }
    }
    impl Provider for GatedProvider {
        fn bind_new(
            &self,
        ) -> Result<Option<soda_identity::providers::resolution::BoundBinary>, Error> {
            Ok(None)
        }
        fn start(&self, _owner: i64) -> Result<Box<dyn EnrollmentSession>, Error> {
            self.entered.send(()).unwrap();
            self.release.lock().unwrap().recv().unwrap();
            Ok(Box::new(Session(Enrollment {
                provider_id: "codex".into(),
                id: "gated-1".into(),
                verification_url: "https://example.test/device".into(),
                user_code: "ABCD-1234".into(),
                state: "pending".into(),
                error: String::new(),
                connection: None,
            })))
        }
    }
    struct EmptyRuntime;
    impl Runtime for EmptyRuntime {
        fn validate(&self, _: &Lease) -> Result<(), Error> {
            Ok(())
        }
        fn stop(&self, _: &Lease) -> Result<(), Error> {
            Ok(())
        }
        fn finish(&self, _: &Lease) -> Result<Vec<u8>, Error> {
            Ok(Vec::new())
        }
    }

    let Some(fixture) = Ephemeral::create() else {
        eprintln!("SODA_PG_* fixture unavailable");
        return;
    };
    let (entered_tx, entered_rx) = mpsc::sync_channel(1);
    let (release_tx, release_rx) = mpsc::channel();
    let mut providers: HashMap<String, Box<dyn Provider>> = HashMap::new();
    providers.insert(
        "codex".into(),
        Box::new(GatedProvider {
            entered: entered_tx,
            release: Mutex::new(release_rx),
        }),
    );
    let broker = Arc::new(
        Controller::new(
            fixture.store(&fixture_key()),
            providers,
            Box::new(EmptyRuntime),
        )
        .unwrap(),
    );
    let starter = {
        let broker = Arc::clone(&broker);
        std::thread::spawn(move || broker.start_enrollment(1, "codex", "slow"))
    };
    entered_rx
        .recv_timeout(Duration::from_secs(5))
        .expect("provider start admitted");

    // The start is still running: reads answer at once and a duplicate is busy.
    let began = Instant::now();
    broker.connections(1).unwrap();
    assert!(broker.enrollments(1).unwrap().is_empty());
    assert!(broker.start_enrollment(1, "codex", "again").is_err());
    assert!(began.elapsed() < Duration::from_secs(1));

    release_tx.send(()).unwrap();
    let started = starter.join().unwrap().unwrap();
    assert_eq!(started.user_code, "ABCD-1234");
    assert_eq!(broker.enrollments(1).unwrap().len(), 1);
}
