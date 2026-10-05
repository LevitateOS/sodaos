// Identity store surface over PostgreSQL, mirroring
// internal/store/{identity,identity_events,grants,schema}.go: the same
// statements (with the same `?` placeholder binding), the same row JSON,
// the same AES-GCM credential custody and the same schema bootstrap.

use crate::crypto::GrantCipher;
use crate::pg::{Client as PgClient, Dsn, Row};
use crate::wire::Error;
use std::sync::Mutex;

pub(crate) fn identity_binding(connection_id: &str, generation: i64) -> String {
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
    pub(crate) client: Mutex<PgClient>,
    pub(crate) grants: Option<GrantCipher>,
}

pub struct Tx<'a> {
    pub(crate) store: &'a Store,
}

impl Store {
    pub fn open_encrypted(dsn: &str, key: &[u8]) -> Result<Store, Error> {
        let cipher = GrantCipher::new(key)?;
        Self::open(dsn, Some(cipher))
    }

    pub(crate) fn open(dsn: &str, grants: Option<GrantCipher>) -> Result<Store, Error> {
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

    pub(crate) fn query(&self, sql: &str, params: &[Param]) -> Result<(Vec<Row>, u64), Error> {
        let encoded: Vec<Option<Vec<u8>>> = params.iter().map(|p| p.encode()).collect();
        let refs: Vec<Option<&[u8]>> = encoded.iter().map(|o| o.as_deref()).collect();
        self.client.lock().unwrap().query(&bind(sql), &refs)
    }

    pub(crate) fn exec(&self, sql: &str, params: &[Param]) -> Result<u64, Error> {
        Ok(self.query(sql, params)?.1)
    }

    pub(crate) fn query_row(&self, sql: &str, params: &[Param]) -> Result<Row, Error> {
        let (rows, _) = self.query(sql, params)?;
        rows.into_iter().next().ok_or_else(Error::not_found)
    }

    pub(crate) fn simple(&self, sql: &str) -> Result<u64, Error> {
        self.client.lock().unwrap().simple(sql)
    }

    pub(crate) fn transaction<T>(
        &self,
        operation: impl FnOnce(&Tx) -> Result<T, Error>,
    ) -> Result<T, Error> {
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
}

impl<'a> Tx<'a> {
    pub(crate) fn query(&self, sql: &str, params: &[Param]) -> Result<(Vec<Row>, u64), Error> {
        self.store.query(sql, params)
    }

    pub(crate) fn exec(&self, sql: &str, params: &[Param]) -> Result<u64, Error> {
        self.store.exec(sql, params)
    }

    pub(crate) fn query_row(&self, sql: &str, params: &[Param]) -> Result<Row, Error> {
        self.store.query_row(sql, params)
    }
}

pub(crate) fn changed(count: u64) -> Result<(), Error> {
    if count != 1 {
        return Err(Error::stale());
    }
    Ok(())
}

#[derive(Debug, Clone)]
pub(crate) enum Param {
    Text(String),
    Int(i64),
    Boolean(bool),
    Bytea(Vec<u8>),
}

impl Param {
    pub(crate) fn text(value: &str) -> Param {
        Param::Text(value.to_string())
    }

    pub(crate) fn int(value: i64) -> Param {
        Param::Int(value)
    }

    pub(crate) fn boolean(value: bool) -> Param {
        Param::Boolean(value)
    }

    pub(crate) fn bytea(value: &[u8]) -> Param {
        Param::Bytea(value.to_vec())
    }

    pub(crate) fn encode(&self) -> Option<Vec<u8>> {
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
