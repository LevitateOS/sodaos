// Store schema bootstrap and key binding, extracted from store.rs (A05.M).
use super::store::{Store, Tx};
use crate::crypto::{GrantCipher, KEY_BINDING};
use crate::pg::pg_error;
use crate::schema;
use crate::wire::Error;
use std::sync::Mutex;
use tokio_postgres::types::ToSql;

// SCHEMA_OPEN serializes schema opens inside this process. Two concurrent
// opens of a retained 68 database must not both attempt the upgrade CREATE:
// the second opener rereads the committed 69 marker instead of failing.
// Cross-process losers are covered by the version reread below, which runs
// after the losing transaction rolls back.
static SCHEMA_OPEN: Mutex<()> = Mutex::new(());

impl Store {
    pub(crate) fn check_grant_key(&self) -> Result<(), Error> {
        let row = self.query_row(
            "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='grant_key_check'",
            &[],
        )?;
        if row.try_get::<_, i64>(0).map_err(pg_error)? == 0 {
            return Ok(());
        }
        let found = self.query("SELECT ciphertext FROM grant_key_check WHERE id=1", &[])?;
        let Some(row) = found.0.into_iter().next() else {
            return self.reject_unkeyed_identity_credentials();
        };
        self.validate_grant_key(row.try_get::<_, Vec<u8>>(0).map_err(pg_error)?.as_slice())
    }

    pub(crate) fn reject_unkeyed_identity_credentials(&self) -> Result<(), Error> {
        let row = self.query_row(
            "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='identity_connections'",
            &[],
        )?;
        if row.try_get::<_, i64>(0).map_err(pg_error)? == 0 {
            return Ok(());
        }
        let row = self.query_row(
            "SELECT count(*) FROM identity_connections WHERE octet_length(credential)>0",
            &[],
        )?;
        if row.try_get::<_, i64>(0).map_err(pg_error)? != 0 {
            return Err(Error::internal(
                "identity credential encryption key missing or incorrect",
            ));
        }
        Ok(())
    }

    pub(crate) fn validate_grant_key(&self, ciphertext: &[u8]) -> Result<(), Error> {
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

    pub(crate) fn initialize_grant_key(&self) -> Result<(), Error> {
        let grants = self.grants.as_ref().ok_or_else(|| {
            Error::internal("identity credential encryption key missing or incorrect")
        })?;
        let sealed = grants.seal(KEY_BINDING.as_bytes(), KEY_BINDING);
        self.exec(
            "INSERT INTO grant_key_check(id,ciphertext) VALUES(1,$1) ON CONFLICT(id) DO NOTHING",
            &[&sealed as &(dyn ToSql + Sync)],
        )?;
        self.check_grant_key()
    }

    pub(crate) fn initialize_schema(&self) -> Result<(), Error> {
        let _open = SCHEMA_OPEN
            .lock()
            .map_err(|_| Error::internal("schema open lock poisoned"))?;
        match self.transaction(|tx| {
            Self::open_current_schema(tx)?;
            Ok(())
        }) {
            Ok(()) => Ok(()),
            Err(upgrade) => self.reopen_after_lost_upgrade_race(upgrade),
        }
    }

    fn open_current_schema(tx: &Tx<'_>) -> Result<(), Error> {
        let mut version = tx.load_schema_version()?;
        if version == 0 {
            // DDL carries no parameters.
            for statement in schema::STATEMENTS {
                tx.simple(statement)
                    .map_err(|e| Error::internal(format!("create current database schema: {e}")))?;
            }
        } else if version == 68 {
            tx.migrate_schema_68_to_69()?;
            version = 69;
        } else if version != 69 && version != schema::SCHEMA_VERSION {
            return Err(Error::internal(
                "database schema differs from this application",
            ));
        }
        if version == 69 {
            for statement in schema::UPGRADE_69_TO_70 {
                tx.simple(statement)
                    .map_err(|e| Error::internal(format!("migrate feed posts: {e}")))?;
            }
        }
        Self::verify_current_schema(tx)
    }

    fn verify_current_schema(tx: &Tx<'_>) -> Result<(), Error> {
        tx.verify_required_columns()?;
        for (name, table) in schema::VERIFY_TRIGGERS {
            tx.verify_trigger(name, table)?;
        }
        Ok(())
    }

    /// Reread the schema marker after a failed open. When a concurrent
    /// opener committed the 68 to 69 upgrade meanwhile, the database is now
    /// current: it still faces the full verification before the open
    /// succeeds. Any other outcome returns the original failure, so genuine
    /// corruption keeps its refusal signal and the unmigrated database is
    /// left untouched.
    fn reopen_after_lost_upgrade_race(&self, upgrade: Error) -> Result<(), Error> {
        let reread = self.transaction(|tx| {
            if tx.load_schema_version()? != schema::SCHEMA_VERSION {
                return Err(Error::internal("database schema changed under open"));
            }
            Self::verify_current_schema(tx)?;
            Ok(())
        });
        match reread {
            Ok(()) => Ok(()),
            Err(_) => Err(upgrade),
        }
    }

    pub(crate) fn grants(&self) -> Result<&GrantCipher, Error> {
        self.grants.as_ref().ok_or_else(|| {
            Error::internal("identity credential encryption key missing or incorrect")
        })
    }
}

impl<'a> Tx<'a> {
    /// Upgrade a genuine 68 database in place: a 68 marker beside an
    /// already-present renewals journal is shape corruption, refused
    /// without mutation like any other foreign version.
    pub(crate) fn migrate_schema_68_to_69(&self) -> Result<(), Error> {
        let row = self.query_row(
            "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='application_setup_renewals' AND table_type='BASE TABLE'",
            &[],
        )?;
        if row.try_get::<_, i64>(0).map_err(pg_error)? != 0 {
            return Err(Error::internal(
                "database schema differs from this application",
            ));
        }
        for statement in schema::UPGRADE_68_TO_69 {
            self.simple(statement)
                .map_err(|e| Error::internal(format!("migrate database schema 68 to 69: {e}")))?;
        }
        Ok(())
    }

    pub(crate) fn load_schema_version(&self) -> Result<i64, Error> {
        let row = self.query_row(
            "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='schema_version'",
            &[],
        )?;
        if row.try_get::<_, i64>(0).map_err(pg_error)? == 0 {
            let row = self.query_row(
                "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'",
                &[],
            )?;
            if row.try_get::<_, i64>(0).map_err(pg_error)? != 0 {
                return Err(Error::internal("refusing an unversioned nonempty database"));
            }
            return Ok(0);
        }
        let row = self.query_row(
            "SELECT count(*),min(version),max(version) FROM schema_version",
            &[],
        )?;
        // min/max arrive NULL only on an empty table, where count is 0.
        let count = row.try_get::<_, i64>(0).map_err(pg_error)?;
        let minimum = row.try_get::<_, Option<i32>>(1).map_err(pg_error)?;
        let maximum = row.try_get::<_, Option<i32>>(2).map_err(pg_error)?;
        match (count, minimum, maximum) {
            (1, Some(min), Some(max)) if min >= 1 && min == max => Ok(i64::from(min)),
            _ => Err(Error::internal("invalid database schema version record")),
        }
    }

    pub(crate) fn verify_required_columns(&self) -> Result<(), Error> {
        for query in schema::VERIFY_QUERIES {
            self.query(query, &[])
                .map_err(|_| Error::internal("database schema is incomplete"))?;
        }
        Ok(())
    }

    pub(crate) fn verify_trigger(&self, name: &str, table: &str) -> Result<(), Error> {
        let row = self.query_row(
            "SELECT count(*) FROM information_schema.triggers WHERE trigger_name=$1 AND event_object_table=$2",
            &[
                &name as &(dyn ToSql + Sync),
                &table as &(dyn ToSql + Sync),
            ],
        )?;
        if row.try_get::<_, i64>(0).map_err(pg_error)? != 1 {
            return Err(Error::internal("database schema is incomplete"));
        }
        Ok(())
    }
}
