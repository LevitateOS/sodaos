// Store schema bootstrap and key binding, extracted from store.rs (A05.M).
use super::store::{Param, Store, Tx};
use crate::crypto::{GrantCipher, KEY_BINDING};
use crate::schema;
use crate::wire::Error;

impl Store {
    pub(crate) fn check_grant_key(&self) -> Result<(), Error> {
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

    pub(crate) fn reject_unkeyed_identity_credentials(&self) -> Result<(), Error> {
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
            &[Param::bytea(&sealed)],
        )?;
        self.check_grant_key()
    }

    pub(crate) fn initialize_schema(&self) -> Result<(), Error> {
        self.transaction(|tx| {
            let version = tx.load_schema_version()?;
            if version == 0 {
                // DDL carries no parameters.
                for statement in schema::STATEMENTS {
                    tx.simple(statement).map_err(|e| {
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

    pub(crate) fn grants(&self) -> Result<&GrantCipher, Error> {
        self.grants.as_ref().ok_or_else(|| {
            Error::internal("identity credential encryption key missing or incorrect")
        })
    }
}

impl<'a> Tx<'a> {
    pub(crate) fn load_schema_version(&self) -> Result<i64, Error> {
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
        let minimum = row.nullable_integer(1)?;
        let maximum = row.nullable_integer(2)?;
        match (count, minimum, maximum) {
            (1, Some(min), Some(max)) if min >= 1 && min == max => Ok(min),
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
            &[Param::text(name), Param::text(table)],
        )?;
        if row.integer(0)? != 1 {
            return Err(Error::internal("database schema is incomplete"));
        }
        Ok(())
    }
}
