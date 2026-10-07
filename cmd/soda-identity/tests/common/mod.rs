// Broker integration fixtures over an ephemeral PostgreSQL database. They
// skip when the A10 fixture environment (SODA_PG_HOST, SODA_PG_PORT,
// SODA_PG_SUPER_PASSWORD_FILE) is absent, exactly like the Go suite.
// Each integration target compiles this module but uses a different
// subset of fixtures; allow the resulting per-target dead code.
#![allow(dead_code)]

use soda_identity::control::{self, Controller};
use soda_identity::store::Store;
use soda_identity::wire::*;
use std::collections::HashMap;
use std::sync::Mutex;
use std::time::Duration;
use tokio::runtime::Builder;
use tokio::time::{timeout_at, Instant};
use tokio_postgres::{config::SslMode, Config, NoTls};

const ADMIN_OPERATION_BUDGET: Duration = Duration::from_secs(30);
const ADMIN_CLEANUP_BUDGET: Duration = Duration::from_millis(250);

pub fn super_dsn() -> Option<String> {
    let host = std::env::var("SODA_PG_HOST").ok()?;
    let port = std::env::var("SODA_PG_PORT").ok()?;
    let password_file = std::env::var("SODA_PG_SUPER_PASSWORD_FILE").ok()?;
    if host.is_empty() || port.is_empty() || password_file.is_empty() {
        return None;
    }
    let raw = std::fs::read_to_string(password_file).ok()?;
    let password = raw.trim();
    if password.is_empty() || password.contains(['\r', '\n', '\0']) {
        return None;
    }
    Some(format!(
        "postgres://postgres:{password}@{host}:{port}/postgres?sslmode=disable"
    ))
}

pub struct Ephemeral {
    dsn: String,
    super_dsn: String,
    name: String,
}

fn admin_command(dsn: &str, sql: &str) -> bool {
    let Ok(mut config) = dsn.parse::<Config>() else {
        return false;
    };
    config.ssl_mode(SslMode::Disable);
    let Ok(runtime) = Builder::new_current_thread().enable_all().build() else {
        return false;
    };
    let deadline = Instant::now() + ADMIN_OPERATION_BUDGET;
    let work_deadline = deadline - ADMIN_CLEANUP_BUDGET;
    runtime.block_on(async {
        let Ok(Ok((client, connection))) =
            timeout_at(work_deadline, config.connect(NoTls)).await
        else {
            return false;
        };
        let mut driver = tokio::spawn(connection);
        let command_ok = {
            let future = client.batch_execute(sql);
            tokio::pin!(future);
            match timeout_at(work_deadline, &mut future).await {
                Ok(Ok(())) => true,
                Ok(Err(_)) => false,
                Err(_) => {
                    let drain_deadline = deadline - Duration::from_millis(50);
                    let canceled = timeout_at(
                        drain_deadline,
                        client.cancel_token().cancel_query(NoTls),
                    )
                    .await;
                    if canceled.is_ok() {
                        let _ = timeout_at(drain_deadline, &mut future).await;
                    }
                    false
                }
            }
        };
        drop(client);
        match timeout_at(deadline - Duration::from_millis(50), &mut driver).await {
            Ok(Ok(Ok(()))) => command_ok,
            _ => {
                driver.abort();
                matches!(timeout_at(deadline, &mut driver).await, Ok(Err(error)) if error.is_cancelled())
                    && command_ok
            }
        }
    })
}

impl Ephemeral {
    pub fn create() -> Option<Ephemeral> {
        let super_dsn = super_dsn()?;
        let mut random = [0u8; 8];
        use std::io::Read;
        std::fs::File::open("/dev/urandom")
            .ok()?
            .read_exact(&mut random)
            .ok()?;
        let name = format!("soda_ephem_{}", hex(&random));
        if !admin_command(&super_dsn, &format!("CREATE DATABASE \"{name}\"")) {
            return None;
        }
        let dsn = super_dsn.replace("/postgres?sslmode", &format!("/{name}?sslmode"));
        Some(Ephemeral {
            dsn,
            super_dsn,
            name,
        })
    }

    pub fn store(&self, key: &[u8]) -> Store {
        Store::open_encrypted(&self.dsn, key).expect("open ephemeral store")
    }

    pub fn exec(&self, sql: &str) -> bool {
        admin_command(&self.dsn, sql)
    }
}

impl Drop for Ephemeral {
    fn drop(&mut self) {
        let _ = admin_command(
            &self.super_dsn,
            &format!("DROP DATABASE IF EXISTS \"{}\"", self.name),
        );
    }
}

pub fn hex(bytes: &[u8]) -> String {
    bytes.iter().map(|b| format!("{b:02x}")).collect()
}

pub fn fixture_key() -> Vec<u8> {
    vec![9u8; 32]
}

pub fn subscription() -> Vec<u8> {
    br#"{"schema_version":1,"providers":{"meta":{"access_token":"synthetic","api_key":"synthetic","api_base_url":"https://api.meta.ai/v1","mechanism":"oauth","obtained_via":"device_code"}}}"#.to_vec()
}

pub fn connection(provider: &str, owner: i64, id: &str) -> Connection {
    Connection {
        provider_id: provider.to_string(),
        id: id.to_string(),
        owner_id: owner,
        label: "synthetic".to_string(),
        email: "soda-tester@example.invalid".to_string(),
        plan: "plus".to_string(),
        generation: 1,
        state: "ready".to_string(),
    }
}

pub struct StubSession {
    snapshot: Mutex<Enrollment>,
    connection: Connection,
    credential: Vec<u8>,
}

impl control::EnrollmentSession for StubSession {
    fn snapshot(&self) -> Enrollment {
        self.snapshot.lock().unwrap().clone()
    }
    fn finish(&self) -> Result<(Connection, Vec<u8>), Error> {
        Ok((self.connection.clone(), self.credential.clone()))
    }
    fn close(&self) -> Result<(), Error> {
        Ok(())
    }
}

pub struct StubProvider {
    enrollment: Enrollment,
    connection: Connection,
    credential: Vec<u8>,
}

impl control::Provider for StubProvider {
    fn start(&self, _owner: i64) -> Result<Box<dyn control::EnrollmentSession>, Error> {
        Ok(Box::new(StubSession {
            snapshot: Mutex::new(self.enrollment.clone()),
            connection: self.connection.clone(),
            credential: self.credential.clone(),
        }))
    }
}

pub struct StubRuntime {
    pub credential: Vec<u8>,
    pub calls: Mutex<Vec<String>>,
}

impl control::Runtime for StubRuntime {
    fn validate(&self, _lease: &Lease) -> Result<(), Error> {
        self.calls.lock().unwrap().push("validate".to_string());
        Ok(())
    }
    fn stop(&self, _lease: &Lease) -> Result<(), Error> {
        self.calls.lock().unwrap().push("stop".to_string());
        Ok(())
    }
    fn finish(&self, _lease: &Lease) -> Result<Vec<u8>, Error> {
        self.calls.lock().unwrap().push("finish".to_string());
        Ok(self.credential.clone())
    }
}

pub fn stub_provider() -> StubProvider {
    StubProvider {
        enrollment: Enrollment {
            provider_id: "codex".to_string(),
            id: "enrollment-1".to_string(),
            verification_url: "https://auth.openai.com/codex/device".to_string(),
            user_code: "ABCD-1234".to_string(),
            state: "completed".to_string(),
            error: String::new(),
            connection: None,
        },
        connection: Connection {
            provider_id: String::new(),
            id: String::new(),
            owner_id: 0,
            label: String::new(),
            email: "soda-tester@example.invalid".to_string(),
            plan: "plus".to_string(),
            generation: 0,
            state: String::new(),
        },
        credential: subscription(),
    }
}

pub fn controller(store: Store) -> Controller {
    let mut providers: HashMap<String, Box<dyn control::Provider>> = HashMap::new();
    providers.insert("codex".to_string(), Box::new(stub_provider()));
    Controller::new(
        store,
        providers,
        Box::new(StubRuntime {
            credential: subscription(),
            calls: Mutex::new(Vec::new()),
        }),
    )
    .unwrap()
}

pub fn binding(kind: &str, generation: i64) -> Binding {
    Binding {
        child_id: String::new(),
        uid: 0,
        gid: 0,
        scope: String::new(),
        credential_root: String::new(),
        invocation_id: String::new(),
        kind: kind.to_string(),
        id: "execution-1".to_string(),
        project: String::new(),
        login: String::new(),
        generation,
    }
}
