// Identity store surface over PostgreSQL, mirroring
// internal/store/{identity,identity_events,grants,schema}.go: the same
// statements (with the same `?` placeholder binding), the same row JSON,
// the same AES-GCM credential custody and the same schema bootstrap.
use crate::crypto::{GrantCipher, KEY_BINDING};
use crate::pg::{Client as PgClient, Dsn, Row};
use crate::schema;
use crate::wire::{
    credential_valid, provider_valid, Connection, Error, Event, Execution, Grant, Lease, UnixTime,
};
use std::sync::Mutex;

fn identity_binding(connection_id: &str, generation: i64) -> String {
    format!("soda/identity/{connection_id}/{generation}")
}

// bind rewrites ? placeholders to PostgreSQL $n parameters, skipping
// single-quoted literals, exactly like the Go store.
fn bind(query: &str) -> String {
    let bytes = query.as_bytes();
    let mut out = String::with_capacity(query.len() + 8);
    let mut n = 0u32;
    let mut quoted = false;
    let mut i = 0;
    while i < bytes.len() {
        let c = bytes[i];
        if c == b'\'' {
            if quoted && i + 1 < bytes.len() && bytes[i + 1] == b'\'' {
                out.push_str("''");
                i += 2;
                continue;
            }
            quoted = !quoted;
            out.push('\'');
            i += 1;
            continue;
        }
        if c == b'?' && !quoted {
            n += 1;
            out.push('$');
            out.push_str(&n.to_string());
            i += 1;
            continue;
        }
        out.push(c as char);
        i += 1;
    }
    out
}

pub struct Store {
    client: Mutex<PgClient>,
    grants: Option<GrantCipher>,
}

pub struct Tx<'a> {
    store: &'a Store,
}

impl Store {
    pub fn open_encrypted(dsn: &str, key: &[u8]) -> Result<Store, Error> {
        let cipher = GrantCipher::new(key)?;
        Self::open(dsn, Some(cipher))
    }

    fn open(dsn: &str, grants: Option<GrantCipher>) -> Result<Store, Error> {
        let parsed = Dsn::parse(dsn)?;
        let client = PgClient::connect(&parsed)?;
        let store = Store {
            client: Mutex::new(client),
            grants,
        };
        store.check_grant_key()?;
        store.initialize_schema()?;
        if store.grants.is_some() {
            store.initialize_grant_key()?;
        }
        Ok(store)
    }

    fn query(&self, sql: &str, params: &[Param]) -> Result<(Vec<Row>, u64), Error> {
        let encoded: Vec<Option<Vec<u8>>> = params.iter().map(|p| p.encode()).collect();
        let refs: Vec<Option<&[u8]>> = encoded.iter().map(|o| o.as_deref()).collect();
        self.client.lock().unwrap().query(&bind(sql), &refs)
    }

    fn exec(&self, sql: &str, params: &[Param]) -> Result<u64, Error> {
        Ok(self.query(sql, params)?.1)
    }

    fn query_row(&self, sql: &str, params: &[Param]) -> Result<Row, Error> {
        let (rows, _) = self.query(sql, params)?;
        rows.into_iter().next().ok_or_else(Error::not_found)
    }

    fn simple(&self, sql: &str) -> Result<u64, Error> {
        self.client.lock().unwrap().simple(sql)
    }

    fn transaction<T>(&self, operation: impl FnOnce(&Tx) -> Result<T, Error>) -> Result<T, Error> {
        self.simple("BEGIN")?;
        let tx = Tx { store: self };
        match operation(&tx) {
            Ok(value) => match self.simple("COMMIT") {
                Ok(_) => Ok(value),
                Err(err) => {
                    let _ = self.simple("ROLLBACK");
                    Err(err)
                }
            },
            Err(err) => {
                let _ = self.simple("ROLLBACK");
                Err(err)
            }
        }
    }

    fn check_grant_key(&self) -> Result<(), Error> {
        let row = self.query_row(
            "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='grant_key_check'",
            &[],
        )?;
        if row.integer(0)? == 0 {
            return Ok(());
        }
        let found = self.query("SELECT ciphertext FROM grant_key_check WHERE id=1", &[])?;
        let Some(row) = found.0.into_iter().next() else {
            return self.reject_unkeyed_identity_credentials();
        };
        self.validate_grant_key(&row.bytea(0)?)
    }

    fn reject_unkeyed_identity_credentials(&self) -> Result<(), Error> {
        let row = self.query_row(
            "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='identity_connections'",
            &[],
        )?;
        if row.integer(0)? == 0 {
            return Ok(());
        }
        let row = self.query_row(
            "SELECT count(*) FROM identity_connections WHERE octet_length(credential)>0",
            &[],
        )?;
        if row.integer(0)? != 0 {
            return Err(Error::internal(
                "identity credential encryption key missing or incorrect",
            ));
        }
        Ok(())
    }

    fn validate_grant_key(&self, ciphertext: &[u8]) -> Result<(), Error> {
        let grants = self.grants.as_ref().ok_or_else(|| {
            Error::internal("identity credential encryption key missing or incorrect")
        })?;
        let plain = grants.open(ciphertext, KEY_BINDING)?;
        if plain != KEY_BINDING.as_bytes() {
            return Err(Error::internal(
                "identity credential encryption key missing or incorrect",
            ));
        }
        Ok(())
    }

    fn initialize_grant_key(&self) -> Result<(), Error> {
        let grants = self.grants.as_ref().ok_or_else(|| {
            Error::internal("identity credential encryption key missing or incorrect")
        })?;
        let sealed = grants.seal(KEY_BINDING.as_bytes(), KEY_BINDING);
        self.exec(
            "INSERT INTO grant_key_check(id,ciphertext) VALUES(1,?) ON CONFLICT(id) DO NOTHING",
            &[Param::bytea(&sealed)],
        )?;
        self.check_grant_key()
    }

    fn initialize_schema(&self) -> Result<(), Error> {
        self.transaction(|tx| {
            let version = tx.load_schema_version()?;
            if version == 0 {
                // DDL carries no parameters; it bypasses placeholder binding.
                for statement in schema::STATEMENTS {
                    tx.store.simple(statement).map_err(|e| {
                        Error::internal(format!("create current database schema: {e}"))
                    })?;
                }
            } else if version != schema::SCHEMA_VERSION {
                return Err(Error::internal(
                    "database schema differs from this application",
                ));
            }
            tx.verify_required_columns()?;
            for (name, table) in schema::VERIFY_TRIGGERS {
                tx.verify_trigger(name, table)?;
            }
            Ok(())
        })
    }

    fn grants(&self) -> Result<&GrantCipher, Error> {
        self.grants.as_ref().ok_or_else(|| {
            Error::internal("identity credential encryption key missing or incorrect")
        })
    }

    pub fn save_connection(&self, connection: &Connection, credential: &[u8]) -> Result<(), Error> {
        if self.grants.is_none() {
            return Err(Error::internal(
                "identity credential encryption key missing or incorrect",
            ));
        }
        if !provider_valid(&connection.provider_id)
            || !credential_valid(credential)
            || connection.owner_id <= 0
            || connection.id.is_empty()
            || connection.generation <= 0
        {
            return Err(Error::denied("identity authority denied"));
        }
        let data = serde_json::to_string(connection)?;
        let sealed = self.grants()?.seal(
            credential,
            &identity_binding(&connection.id, connection.generation),
        );
        self.transaction(|tx| {
            tx.exec(
                "INSERT INTO identity_connections(id,owner_id,generation,state,data,credential) VALUES(?,?,?,?,?,?)",
                &[
                    Param::text(&connection.id),
                    Param::int(connection.owner_id),
                    Param::int(connection.generation),
                    Param::text(&connection.state),
                    Param::text(&data),
                    Param::bytea(&sealed),
                ],
            )?;
            tx.append_event(&Event {
                id: 0,
                time: UnixTime::now(),
                action: "connected".to_string(),
                owner_id: connection.owner_id,
                actor_id: connection.owner_id,
                connection_id: connection.id.clone(),
                lease_id: String::new(),
                project_id: String::new(),
                grant_id: String::new(),
                execution_id: String::new(),
                kind: String::new(),
                generation: connection.generation,
            })
        })
    }

    pub fn connection(&self, id: &str) -> Result<Connection, Error> {
        let row = self.query_row(
            "SELECT data FROM identity_connections WHERE id=?",
            &[Param::text(id)],
        )?;
        Ok(serde_json::from_str(row.text(0)?)?)
    }

    pub fn credential(&self, connection: &Connection) -> Result<Vec<u8>, Error> {
        let row = self.query_row(
            "SELECT credential FROM identity_connections WHERE id=? AND generation=? AND state='ready'",
            &[Param::text(&connection.id), Param::int(connection.generation)],
        )?;
        self.grants()?.open(
            &row.bytea(0)?,
            &identity_binding(&connection.id, connection.generation),
        )
    }

    pub fn connections(&self, owner: i64) -> Result<Vec<Connection>, Error> {
        let (rows, _) = self.query(
            "SELECT data FROM identity_connections WHERE owner_id=? ORDER BY id",
            &[Param::int(owner)],
        )?;
        rows.iter()
            .map(|r| Ok(serde_json::from_str(r.text(0)?)?))
            .collect()
    }

    pub fn available(&self, actor: i64, project: &str) -> Result<Vec<Connection>, Error> {
        let (rows, _) = self.query(
            "SELECT c.data FROM identity_connections c WHERE c.state='ready' AND (c.owner_id=? OR EXISTS(SELECT 1 FROM identity_grants g WHERE g.connection_id=c.id AND g.user_id=? AND g.project_id=? AND NOT g.revoked)) ORDER BY c.id",
            &[Param::int(actor), Param::int(actor), Param::text(project)],
        )?;
        let mut out = Vec::with_capacity(rows.len());
        for row in &rows {
            let mut connection: Connection = serde_json::from_str(row.text(0)?)?;
            if connection.owner_id != actor {
                connection.email.clear();
            }
            out.push(connection);
        }
        Ok(out)
    }

    // IdentityState denies admission before callers attempt native termination.
    pub fn set_state(&self, connection: &Connection, state: &str) -> Result<(), Error> {
        let mut updated = connection.clone();
        updated.state = state.to_string();
        let data = serde_json::to_string(&updated)?;
        self.transaction(|tx| {
            let count = tx.exec(
                "UPDATE identity_connections SET state=?,data=?,credential=CASE WHEN ?='revoked' THEN '\\x'::bytea ELSE credential END WHERE id=? AND generation=?",
                &[
                    Param::text(state),
                    Param::text(&data),
                    Param::text(state),
                    Param::text(&connection.id),
                    Param::int(connection.generation),
                ],
            )?;
            changed(count)?;
            tx.append_event(&Event {
                id: 0,
                time: UnixTime::now(),
                action: state.to_string(),
                owner_id: connection.owner_id,
                actor_id: connection.owner_id,
                connection_id: connection.id.clone(),
                lease_id: String::new(),
                project_id: String::new(),
                grant_id: String::new(),
                execution_id: String::new(),
                kind: String::new(),
                generation: connection.generation,
            })
        })
    }

    pub fn save_grant(&self, grant: &Grant) -> Result<(), Error> {
        let data = serde_json::to_string(grant)?;
        self.transaction(|tx| {
            tx.exec(
                "INSERT INTO identity_grants(id,connection_id,user_id,project_id,revision,revoked,data) VALUES(?,?,?,?,?,?,?)",
                &[
                    Param::text(&grant.id),
                    Param::text(&grant.connection_id),
                    Param::int(grant.user_id),
                    Param::text(&grant.project_id),
                    Param::int(grant.revision),
                    Param::boolean(grant.revoked),
                    Param::text(&data),
                ],
            )?;
            tx.append_event(&Event {
                id: 0,
                time: UnixTime::now(),
                action: "grant_created".to_string(),
                owner_id: 0,
                actor_id: 0,
                connection_id: grant.connection_id.clone(),
                lease_id: String::new(),
                project_id: grant.project_id.clone(),
                grant_id: grant.id.clone(),
                execution_id: String::new(),
                kind: String::new(),
                generation: 0,
            })
        })
    }

    pub fn grant(&self, id: &str) -> Result<Grant, Error> {
        let row = self.query_row(
            "SELECT data FROM identity_grants WHERE id=?",
            &[Param::text(id)],
        )?;
        Ok(serde_json::from_str(row.text(0)?)?)
    }

    pub fn grants_for(&self, id: &str) -> Result<Vec<Grant>, Error> {
        let (rows, _) = self.query(
            "SELECT data FROM identity_grants WHERE connection_id=? ORDER BY id",
            &[Param::text(id)],
        )?;
        rows.iter()
            .map(|r| Ok(serde_json::from_str(r.text(0)?)?))
            .collect()
    }

    pub fn revoke_grant(&self, grant: &Grant) -> Result<(), Error> {
        let mut updated = grant.clone();
        updated.revoked = true;
        updated.revision += 1;
        let data = serde_json::to_string(&updated)?;
        let previous = updated.revision - 1;
        self.transaction(|tx| {
            let count = tx.exec(
                "UPDATE identity_grants SET revoked=TRUE,revision=?,data=? WHERE id=? AND revision=?",
                &[Param::int(updated.revision), Param::text(&data), Param::text(&grant.id), Param::int(previous)],
            )?;
            changed(count)?;
            tx.append_event(&Event {
                id: 0,
                time: UnixTime::now(),
                action: "grant_revoked".to_string(),
                owner_id: 0,
                actor_id: 0,
                connection_id: grant.connection_id.clone(),
                lease_id: String::new(),
                project_id: grant.project_id.clone(),
                grant_id: grant.id.clone(),
                execution_id: String::new(),
                kind: String::new(),
                generation: 0,
            })
        })
    }

    pub fn leases(&self) -> Result<Vec<Lease>, Error> {
        let (rows, _) = self.query("SELECT data FROM identity_leases ORDER BY id", &[])?;
        rows.iter()
            .map(|r| Ok(serde_json::from_str(r.text(0)?)?))
            .collect()
    }

    pub fn lease(&self, id: &str) -> Result<Lease, Error> {
        let row = self.query_row(
            "SELECT data FROM identity_leases WHERE id=?",
            &[Param::text(id)],
        )?;
        Ok(serde_json::from_str(row.text(0)?)?)
    }

    pub fn reserve(&self, lease: &Lease) -> Result<(), Error> {
        if !provider_valid(&lease.provider_id) {
            return Err(Error::denied("identity authority denied"));
        }
        let data = serde_json::to_string(lease)?;
        self.transaction(|tx| {
            let count = tx.exec(
                "INSERT INTO identity_leases(id,connection_id,data) SELECT ?,id,? FROM identity_connections WHERE id=? AND generation=? AND state='ready' AND data->>'provider_id'=? AND (?='muse' OR NOT EXISTS(SELECT 1 FROM identity_leases WHERE connection_id=?)) AND (?='' OR EXISTS(SELECT 1 FROM identity_grants WHERE id=? AND connection_id=? AND user_id=? AND project_id=? AND revision=? AND revoked=FALSE))",
                &[
                    Param::text(&lease.id),
                    Param::text(&data),
                    Param::text(&lease.connection_id),
                    Param::int(lease.generation),
                    Param::text(&lease.provider_id),
                    Param::text(&lease.provider_id),
                    Param::text(&lease.connection_id),
                    Param::text(&lease.grant_id),
                    Param::text(&lease.grant_id),
                    Param::text(&lease.connection_id),
                    Param::int(lease.actor_id),
                    Param::text(&lease.project_id),
                    Param::int(lease.grant_revision),
                ],
            )?;
            if count != 1 {
                return Err(Error::busy());
            }
            tx.append_event(&lease_event(lease, "reserved"))
        })
    }

    pub fn register(&self, lease: &Lease) -> Result<(), Error> {
        let data = serde_json::to_string(lease)?;
        self.transaction(|tx| {
            let count = tx.exec(
                "UPDATE identity_leases SET data=? WHERE id=? AND (data->'binding') IS NULL",
                &[Param::text(&data), Param::text(&lease.id)],
            )?;
            changed(count)?;
            tx.append_event(&lease_event(lease, "registered"))
        })
    }

    // IdentityReturn atomically accepts maintained bytes and releases one generation.
    pub fn return_lease(&self, lease: &Lease, credential: &[u8]) -> Result<(), Error> {
        if !credential_valid(credential) {
            return Err(Error::uncertain());
        }
        self.transaction(|tx| {
            tx.maintain_credential(lease, credential)?;
            let count = tx.exec(
                "DELETE FROM identity_leases WHERE id=? AND connection_id=? AND (data->>'generation')::bigint=?",
                &[Param::text(&lease.id), Param::text(&lease.connection_id), Param::int(lease.generation)],
            )?;
            changed(count)?;
            tx.append_event(&lease_event(lease, "returned"))
        })
    }

    // IdentityExecution returns the retained acquisition identity for one
    // (kind, execution_id). The record outlives lease return/deletion.
    pub fn execution(&self, kind: &str, execution_id: &str) -> Result<Execution, Error> {
        let row = self.query_row(
            "SELECT data FROM identity_executions WHERE kind=? AND execution_id=?",
            &[Param::text(kind), Param::text(execution_id)],
        )?;
        Ok(serde_json::from_str(row.text(0)?)?)
    }

    // IdentityAdmitExecution records one execution identity. Repeating the
    // same identity returns the recorded row; callers compare digests to
    // detect a changed request for that ID.
    pub fn admit_execution(&self, execution: &Execution) -> Result<(Execution, bool), Error> {
        execution.validate()?;
        let data = serde_json::to_string(execution)?;
        let count = self.exec(
            "INSERT INTO identity_executions(kind,execution_id,state,lease_id,data) VALUES(?,?,?,?,?)\n\t\tON CONFLICT(kind,execution_id) DO NOTHING",
            &[
                Param::text(&execution.kind),
                Param::text(&execution.execution_id),
                Param::text(&execution.state),
                Param::text(&execution.lease_id),
                Param::text(&data),
            ],
        )?;
        if count == 1 {
            return Ok((execution.clone(), true));
        }
        Ok((
            self.execution(&execution.kind, &execution.execution_id)?,
            false,
        ))
    }

    // IdentityObserveExecution advances an execution's state, lease and
    // binding. The acquisition digest is additionally guarded by a schema
    // trigger, and a terminal execution can never leave that state.
    pub fn observe_execution(&self, execution: &Execution) -> Result<(), Error> {
        execution.validate()?;
        let data = serde_json::to_string(execution)?;
        let count = self.exec(
            "UPDATE identity_executions SET state=?,lease_id=?,data=? WHERE kind=? AND execution_id=?",
            &[
                Param::text(&execution.state),
                Param::text(&execution.lease_id),
                Param::text(&data),
                Param::text(&execution.kind),
                Param::text(&execution.execution_id),
            ],
        )?;
        if count != 1 {
            return Err(Error::not_found());
        }
        Ok(())
    }

    pub fn forget_lease(&self, id: &str) -> Result<(), Error> {
        self.transaction(|tx| {
            let row = tx.query_row(
                "SELECT data FROM identity_leases WHERE id=?",
                &[Param::text(id)],
            )?;
            let lease: Lease = serde_json::from_str(row.text(0)?)?;
            tx.exec("DELETE FROM identity_leases WHERE id=?", &[Param::text(id)])?;
            tx.append_event(&lease_event(&lease, "reconciled"))
        })
    }

    pub fn events(&self, owner: i64, id: &str) -> Result<Vec<Event>, Error> {
        if owner <= 0 {
            return Err(Error::denied("identity authority denied"));
        }
        let (rows, _) = self.query(
            "SELECT id,data FROM identity_events WHERE owner_id=? AND connection_id=? ORDER BY id DESC LIMIT 200",
            &[Param::int(owner), Param::text(id)],
        )?;
        let mut out = Vec::with_capacity(rows.len());
        for row in &rows {
            let id = row.integer(0)?;
            let mut event: Event = serde_json::from_str(row.text(1)?)?;
            event.id = id;
            out.push(event);
        }
        Ok(out)
    }
}

impl<'a> Tx<'a> {
    fn query(&self, sql: &str, params: &[Param]) -> Result<(Vec<Row>, u64), Error> {
        self.store.query(sql, params)
    }

    fn exec(&self, sql: &str, params: &[Param]) -> Result<u64, Error> {
        self.store.exec(sql, params)
    }

    fn query_row(&self, sql: &str, params: &[Param]) -> Result<Row, Error> {
        self.store.query_row(sql, params)
    }

    fn load_schema_version(&self) -> Result<i64, Error> {
        let row = self.query_row(
            "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='schema_version'",
            &[],
        )?;
        if row.integer(0)? == 0 {
            let row = self.query_row(
                "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'",
                &[],
            )?;
            if row.integer(0)? != 0 {
                return Err(Error::internal("refusing an unversioned nonempty database"));
            }
            return Ok(0);
        }
        let row = self.query_row(
            "SELECT count(*),min(version),max(version) FROM schema_version",
            &[],
        )?;
        // min/max arrive NULL only on an empty table, where count is 0.
        let count = row.integer(0)?;
        let minimum = row.fields[1]
            .as_ref()
            .and_then(|b| std::str::from_utf8(b).ok()?.parse::<i64>().ok());
        let maximum = row.fields[2]
            .as_ref()
            .and_then(|b| std::str::from_utf8(b).ok()?.parse::<i64>().ok());
        match (count, minimum, maximum) {
            (1, Some(min), Some(max)) if min >= 1 && min == max => Ok(min),
            _ => Err(Error::internal("invalid database schema version record")),
        }
    }

    fn verify_required_columns(&self) -> Result<(), Error> {
        for query in schema::VERIFY_QUERIES {
            self.query(query, &[])
                .map_err(|_| Error::internal("database schema is incomplete"))?;
        }
        Ok(())
    }

    fn verify_trigger(&self, name: &str, table: &str) -> Result<(), Error> {
        let row = self.query_row(
            "SELECT count(*) FROM information_schema.triggers WHERE trigger_name=? AND event_object_table=?",
            &[Param::text(name), Param::text(table)],
        )?;
        if row.integer(0)? != 1 {
            return Err(Error::internal("database schema is incomplete"));
        }
        Ok(())
    }

    fn append_event(&self, event: &Event) -> Result<(), Error> {
        let mut event = event.clone();
        if event.owner_id == 0 {
            let row = self.query_row(
                "SELECT owner_id,generation FROM identity_connections WHERE id=?",
                &[Param::text(&event.connection_id)],
            )?;
            event.owner_id = row.integer(0)?;
            if event.generation == 0 {
                event.generation = row.integer(1)?;
            }
        }
        if event.actor_id == 0 {
            event.actor_id = event.owner_id;
        }
        event.time = UnixTime::now();
        let data = serde_json::to_string(&event)?;
        self.exec(
            "INSERT INTO identity_events(owner_id,connection_id,data) VALUES(?,?,?)",
            &[
                Param::int(event.owner_id),
                Param::text(&event.connection_id),
                Param::text(&data),
            ],
        )?;
        Ok(())
    }

    fn maintain_credential(&self, lease: &Lease, credential: &[u8]) -> Result<(), Error> {
        use crate::wire::CODEX;
        let found = self.query(
            "SELECT data FROM identity_connections WHERE id=? AND generation=? AND state='ready'",
            &[
                Param::text(&lease.connection_id),
                Param::int(lease.generation),
            ],
        )?;
        let Some(row) = found.0.into_iter().next() else {
            return Err(Error::stale());
        };
        let mut connection: Connection = serde_json::from_str(row.text(0)?)?;
        let rotated = |id: &str| id == CODEX;
        if connection.provider_id != lease.provider_id || !rotated(&lease.provider_id) {
            return Err(Error::denied("identity authority denied"));
        }
        connection.generation += 1;
        let data = serde_json::to_string(&connection)?;
        let sealed = self.store.grants()?.seal(
            credential,
            &identity_binding(&connection.id, connection.generation),
        );
        let count = self.exec(
            "UPDATE identity_connections SET generation=?,data=?,credential=? WHERE id=? AND generation=?",
            &[
                Param::int(connection.generation),
                Param::text(&data),
                Param::bytea(&sealed),
                Param::text(&connection.id),
                Param::int(lease.generation),
            ],
        )?;
        changed(count)
    }
}

fn changed(count: u64) -> Result<(), Error> {
    if count != 1 {
        return Err(Error::stale());
    }
    Ok(())
}

fn lease_event(lease: &Lease, action: &str) -> Event {
    Event {
        id: 0,
        time: UnixTime::now(),
        action: action.to_string(),
        owner_id: 0,
        actor_id: lease.actor_id,
        connection_id: lease.connection_id.clone(),
        lease_id: lease.id.clone(),
        project_id: lease.project_id.clone(),
        grant_id: lease.grant_id.clone(),
        execution_id: lease.execution_id.clone(),
        kind: lease.kind.clone(),
        generation: lease.generation,
    }
}

#[derive(Debug, Clone)]
enum Param {
    Text(String),
    Int(i64),
    Boolean(bool),
    Bytea(Vec<u8>),
}

impl Param {
    fn text(value: &str) -> Param {
        Param::Text(value.to_string())
    }

    fn int(value: i64) -> Param {
        Param::Int(value)
    }

    fn boolean(value: bool) -> Param {
        Param::Boolean(value)
    }

    fn bytea(value: &[u8]) -> Param {
        Param::Bytea(value.to_vec())
    }

    fn encode(&self) -> Option<Vec<u8>> {
        Some(match self {
            Param::Text(value) => value.as_bytes().to_vec(),
            Param::Int(value) => value.to_string().into_bytes(),
            Param::Boolean(true) => b"TRUE".to_vec(),
            Param::Boolean(false) => b"FALSE".to_vec(),
            // Text-format bytea uses hex encoding; the stored bytes match
            // the driver's binary encoding exactly.
            Param::Bytea(value) => {
                let mut out = Vec::with_capacity(2 + value.len() * 2);
                out.extend_from_slice(b"\\x");
                for byte in value {
                    out.extend_from_slice(format!("{byte:02x}").as_bytes());
                }
                out
            }
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bind_rewrites_placeholders_outside_literals() {
        assert_eq!(bind("SELECT ?, ?",), "SELECT $1, $2");
        assert_eq!(
            bind("WHERE state='ready' AND id=?"),
            "WHERE state='ready' AND id=$1"
        );
        assert_eq!(bind("VALUES('it''s ?')"), "VALUES('it''s ?')");
        // The revoked-credential CASE keeps its quoted literal intact.
        let query =
            "credential=CASE WHEN ?='revoked' THEN '\\x'::bytea ELSE credential END WHERE id=?";
        assert_eq!(
            bind(query),
            "credential=CASE WHEN $1='revoked' THEN '\\x'::bytea ELSE credential END WHERE id=$2"
        );
    }

    #[test]
    fn bytea_params_use_hex_text_form() {
        assert_eq!(Param::bytea(&[0xde, 0xad]).encode().unwrap(), b"\\xdead");
    }
}
