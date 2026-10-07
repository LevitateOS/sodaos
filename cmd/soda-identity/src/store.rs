use crate::crypto::GrantCipher;
use crate::pg::{Connection as PgConnection, Dsn, Row};
use crate::pg_query::{self, Outcome};
use crate::wire::Error;
use std::cell::{Cell, RefCell};
use std::sync::{Mutex, MutexGuard, TryLockError};
use std::time::Duration;
use tokio::runtime::{Builder, Runtime};
use tokio::time::{timeout_at, Instant};
use tokio_postgres::types::ToSql;

const OPERATION_BUDGET: Duration = Duration::from_secs(30);
const CLEANUP_BUDGET: Duration = Duration::from_millis(250);

pub(crate) fn identity_binding(connection_id: &str, generation: i64) -> String {
    format!("soda/identity/{connection_id}/{generation}")
}

pub struct Store {
    client: Mutex<Option<PgConnection>>,
    runtime: Runtime,
    dsn: Dsn,
    operation_budget: Duration,
    pub(crate) grants: Option<GrantCipher>,
}

pub struct Tx<'a> {
    pub(crate) store: &'a Store,
    connection: RefCell<&'a mut Option<PgConnection>>,
    deadline: Instant,
    failed: Cell<bool>,
}

impl Store {
    pub fn open_encrypted(dsn: &str, key: &[u8]) -> Result<Store, Error> {
        let cipher = GrantCipher::new(key)?;
        Self::open(dsn, Some(cipher))
    }

    pub(crate) fn open(dsn: &str, grants: Option<GrantCipher>) -> Result<Store, Error> {
        let dsn = Dsn::parse(dsn)?;
        let runtime = Builder::new_current_thread()
            .enable_all()
            .build()
            .map_err(Error::from)?;
        let store = Store {
            client: Mutex::new(None),
            runtime,
            dsn,
            operation_budget: OPERATION_BUDGET,
            grants,
        };
        store.query("SELECT 1", &[])?;
        store.check_grant_key()?;
        store.initialize_schema()?;
        if store.grants.is_some() {
            store.initialize_grant_key()?;
        }
        Ok(store)
    }

    pub(crate) fn query(
        &self,
        sql: &str,
        params: &[&(dyn ToSql + Sync)],
    ) -> Result<(Vec<Row>, u64), Error> {
        let (mut connection, deadline, _) = self.begin_operation()?;
        self.query_on(&mut connection, deadline, sql, params)
    }

    pub(crate) fn exec(&self, sql: &str, params: &[&(dyn ToSql + Sync)]) -> Result<u64, Error> {
        let (mut connection, deadline, _) = self.begin_operation()?;
        self.exec_on(&mut connection, deadline, sql, params)
    }

    pub(crate) fn query_row(
        &self,
        sql: &str,
        params: &[&(dyn ToSql + Sync)],
    ) -> Result<Row, Error> {
        let (rows, _) = self.query(sql, params)?;
        rows.into_iter().next().ok_or_else(Error::not_found)
    }

    #[cfg(test)]
    pub(crate) fn simple(&self, sql: &str) -> Result<u64, Error> {
        let (mut connection, deadline, work_deadline) = self.begin_operation()?;
        run_simple(self, &mut connection, sql, work_deadline, deadline).map(|()| 0)
    }

    pub(crate) fn transaction<T>(
        &self,
        operation: impl FnOnce(&Tx<'_>) -> Result<T, Error>,
    ) -> Result<T, Error> {
        let (mut connection, deadline, work_deadline) = self.begin_operation()?;
        self.connect_if_needed(&mut connection, work_deadline)?;
        if let Err(error) = run_simple(self, &mut connection, "BEGIN", work_deadline, deadline) {
            self.discard(&mut connection, deadline);
            return Err(error);
        }

        let tx = Tx {
            store: self,
            connection: RefCell::new(&mut connection),
            deadline,
            failed: Cell::new(false),
        };
        let result = operation(&tx);
        let failed = tx.failed.get();
        drop(tx);

        match result {
            Ok(value) => {
                if failed || connection.is_none() {
                    if connection.is_some()
                        && run_simple(self, &mut connection, "ROLLBACK", work_deadline, deadline)
                            .is_err()
                    {
                        self.discard(&mut connection, deadline);
                    }
                    return Err(Error::internal("postgres transaction failed"));
                }
                match run_simple(self, &mut connection, "COMMIT", work_deadline, deadline) {
                    Ok(()) => Ok(value),
                    Err(error) => {
                        // COMMIT may have reached the server even when its reply
                        // was lost. Discard this session and never retry it.
                        self.discard(&mut connection, deadline);
                        Err(error)
                    }
                }
            }
            Err(error) => {
                if connection.is_some() {
                    if run_simple(self, &mut connection, "ROLLBACK", work_deadline, deadline)
                        .is_err()
                    {
                        self.discard(&mut connection, deadline);
                    }
                }
                Err(error)
            }
        }
    }

    fn begin_operation(
        &self,
    ) -> Result<(MutexGuard<'_, Option<PgConnection>>, Instant, Instant), Error> {
        let deadline = Instant::now() + self.operation_budget;
        let work_deadline = deadline - CLEANUP_BUDGET;
        loop {
            match self.client.try_lock() {
                Ok(guard) => return Ok((guard, deadline, work_deadline)),
                Err(TryLockError::Poisoned(_)) => {
                    return Err(Error::internal("postgres connection lock poisoned"));
                }
                Err(TryLockError::WouldBlock) if Instant::now() < work_deadline => {
                    std::thread::sleep(Duration::from_millis(2));
                }
                Err(TryLockError::WouldBlock) => {
                    return Err(Error::internal("postgres operation deadline exceeded"));
                }
            }
        }
    }

    fn connect_if_needed(
        &self,
        connection: &mut Option<PgConnection>,
        work_deadline: Instant,
    ) -> Result<(), Error> {
        if connection.is_some() {
            return Ok(());
        }
        match self
            .runtime
            .block_on(async { timeout_at(work_deadline, PgConnection::connect(&self.dsn)).await })
        {
            Ok(Ok(connected)) => {
                *connection = Some(connected);
                Ok(())
            }
            Ok(Err(error)) => Err(error),
            Err(_) => Err(Error::internal("postgres operation deadline exceeded")),
        }
    }

    fn discard(&self, connection: &mut Option<PgConnection>, deadline: Instant) {
        let Some(owned) = connection.take() else {
            return;
        };
        let PgConnection { client, mut driver } = owned;
        drop(client);
        driver.abort();
        let _ = self.runtime.block_on(async {
            let _ = timeout_at(deadline, &mut driver).await;
        });
    }

    #[cfg(test)]
    pub(crate) fn set_operation_budget_for_test(&mut self, budget: Duration) {
        self.operation_budget = budget;
    }
}

impl Drop for Store {
    fn drop(&mut self) {
        let connection = match self.client.get_mut() {
            Ok(connection) => connection.take(),
            Err(poisoned) => poisoned.into_inner().take(),
        };
        let Some(PgConnection { client, mut driver }) = connection else {
            return;
        };
        drop(client);
        let deadline = Instant::now() + CLEANUP_BUDGET;
        self.runtime.block_on(async {
            let graceful_deadline = deadline - Duration::from_millis(50);
            if timeout_at(graceful_deadline, &mut driver).await.is_err() {
                driver.abort();
                let _ = timeout_at(deadline, &mut driver).await;
            }
        });
    }
}

impl Tx<'_> {
    pub(crate) fn query(
        &self,
        sql: &str,
        params: &[&(dyn ToSql + Sync)],
    ) -> Result<(Vec<Row>, u64), Error> {
        if self.failed.get() {
            return Err(Error::internal("postgres transaction already failed"));
        }
        let mut connection = self.connection.borrow_mut();
        if (**connection).is_none() {
            self.failed.set(true);
            return Err(Error::internal(
                "postgres transaction connection unavailable",
            ));
        }
        let result = self
            .store
            .query_on(&mut **connection, self.deadline, sql, params);
        if result.is_err() {
            self.failed.set(true);
        }
        result
    }

    pub(crate) fn exec(&self, sql: &str, params: &[&(dyn ToSql + Sync)]) -> Result<u64, Error> {
        if self.failed.get() {
            return Err(Error::internal("postgres transaction already failed"));
        }
        let mut connection = self.connection.borrow_mut();
        if (**connection).is_none() {
            self.failed.set(true);
            return Err(Error::internal(
                "postgres transaction connection unavailable",
            ));
        }
        let result = self
            .store
            .exec_on(&mut **connection, self.deadline, sql, params);
        if result.is_err() {
            self.failed.set(true);
        }
        result
    }

    pub(crate) fn query_row(
        &self,
        sql: &str,
        params: &[&(dyn ToSql + Sync)],
    ) -> Result<Row, Error> {
        let (rows, _) = self.query(sql, params)?;
        match rows.into_iter().next() {
            Some(row) => Ok(row),
            None => {
                self.failed.set(true);
                Err(Error::not_found())
            }
        }
    }

    pub(crate) fn simple(&self, sql: &str) -> Result<u64, Error> {
        if self.failed.get() {
            return Err(Error::internal("postgres transaction already failed"));
        }
        let mut connection = self.connection.borrow_mut();
        if (**connection).is_none() {
            self.failed.set(true);
            return Err(Error::internal(
                "postgres transaction connection unavailable",
            ));
        }
        let deadline = self.deadline;
        let work_deadline = deadline - CLEANUP_BUDGET;
        let result =
            run_simple(self.store, &mut **connection, sql, work_deadline, deadline).map(|()| 0);
        if result.is_err() {
            self.failed.set(true);
        }
        result
    }
}

impl Store {
    fn query_on(
        &self,
        connection: &mut Option<PgConnection>,
        deadline: Instant,
        sql: &str,
        params: &[&(dyn ToSql + Sync)],
    ) -> Result<(Vec<Row>, u64), Error> {
        let work_deadline = deadline - CLEANUP_BUDGET;
        self.connect_if_needed(connection, work_deadline)?;
        let client = &connection.as_ref().expect("connected client").client;
        match self.runtime.block_on(async {
            pg_query::bounded(client, client.query(sql, params), work_deadline, deadline).await
        }) {
            Outcome::Complete(Ok(rows)) => Ok((rows, 0)),
            Outcome::Complete(Err(error)) => {
                if connection.as_ref().is_some_and(|c| c.client.is_closed()) {
                    self.discard(connection, deadline);
                }
                Err(error)
            }
            Outcome::TimedOut => {
                self.discard(connection, deadline);
                Err(Error::internal("postgres operation deadline exceeded"))
            }
        }
    }

    fn exec_on(
        &self,
        connection: &mut Option<PgConnection>,
        deadline: Instant,
        sql: &str,
        params: &[&(dyn ToSql + Sync)],
    ) -> Result<u64, Error> {
        let work_deadline = deadline - CLEANUP_BUDGET;
        self.connect_if_needed(connection, work_deadline)?;
        let client = &connection.as_ref().expect("connected client").client;
        match self.runtime.block_on(async {
            pg_query::bounded(client, client.execute(sql, params), work_deadline, deadline).await
        }) {
            Outcome::Complete(Ok(count)) => Ok(count),
            Outcome::Complete(Err(error)) => {
                if connection.as_ref().is_some_and(|c| c.client.is_closed()) {
                    self.discard(connection, deadline);
                }
                Err(error)
            }
            Outcome::TimedOut => {
                self.discard(connection, deadline);
                Err(Error::internal("postgres operation deadline exceeded"))
            }
        }
    }
}

fn run_simple(
    store: &Store,
    connection: &mut Option<PgConnection>,
    sql: &str,
    work_deadline: Instant,
    deadline: Instant,
) -> Result<(), Error> {
    store.connect_if_needed(connection, work_deadline)?;
    let client = &connection.as_ref().expect("connected client").client;
    match store.runtime.block_on(async {
        pg_query::bounded(client, client.batch_execute(sql), work_deadline, deadline).await
    }) {
        Outcome::Complete(Ok(())) => Ok(()),
        Outcome::Complete(Err(error)) => {
            if connection.as_ref().is_some_and(|c| c.client.is_closed()) {
                store.discard(connection, deadline);
            }
            Err(error)
        }
        Outcome::TimedOut => {
            store.discard(connection, deadline);
            Err(Error::internal("postgres operation deadline exceeded"))
        }
    }
}

pub(crate) fn schema_integer(value: i64) -> Result<i32, Error> {
    i32::try_from(value).map_err(|_| Error::internal("postgres integer parameter is out of range"))
}

pub(crate) fn changed(count: u64) -> Result<(), Error> {
    if count != 1 {
        return Err(Error::stale());
    }
    Ok(())
}
