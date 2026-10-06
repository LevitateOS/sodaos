// PostgreSQL DSN parsing, extracted from pg.rs (A-R02a).
use crate::wire::Error;
use std::collections::HashMap;

pub struct Dsn {
    pub(crate) user: String,
    pub(crate) password: String,
    pub(crate) host: String,
    pub(crate) port: u16,
    pub(crate) database: String,
}

impl Dsn {
    pub fn parse(dsn: &str) -> Result<Dsn, Error> {
        let dsn = dsn.trim();
        if dsn.is_empty() {
            return Err(Error::internal("postgres connection string is required"));
        }
        let rest = dsn
            .strip_prefix("postgres://")
            .or_else(|| dsn.strip_prefix("postgresql://"))
            .ok_or_else(|| Error::internal("postgres connection string is required"))?;
        let (authority, path_query) = match rest.find('/') {
            Some(i) => (&rest[..i], &rest[i + 1..]),
            None => (rest, ""),
        };
        let (path, query) = match path_query.find('?') {
            Some(i) => (&path_query[..i], &path_query[i + 1..]),
            None => (path_query, ""),
        };
        let (credentials, hostport) = match authority.rfind('@') {
            Some(i) => (&authority[..i], &authority[i + 1..]),
            None => ("", authority),
        };
        let (user, password) = match credentials.find(':') {
            Some(i) => (&credentials[..i], &credentials[i + 1..]),
            None => (credentials, ""),
        };
        let mut params = HashMap::new();
        for pair in query.split('&') {
            if pair.is_empty() {
                continue;
            }
            let (key, value) = match pair.find('=') {
                Some(i) => (&pair[..i], &pair[i + 1..]),
                None => (pair, ""),
            };
            params.insert(percent_decode(key)?, percent_decode(value)?);
        }
        if let Some(mode) = params.get("sslmode") {
            if mode != "disable" {
                return Err(Error::internal(format!(
                    "unsupported postgres sslmode {mode:?}"
                )));
            }
        }
        let (mut host, mut port) = match hostport.rfind(':') {
            Some(i) => (
                &hostport[..i],
                hostport[i + 1..].parse::<u16>().unwrap_or(5432),
            ),
            None => (hostport, 5432),
        };
        if let Some(value) = params.get("host") {
            host = value;
        }
        if let Some(value) = params.get("port") {
            port = value
                .parse::<u16>()
                .map_err(|_| Error::internal("invalid postgres port"))?;
        }
        let mut database = percent_decode(path)?;
        if let Some(value) = params.get("dbname") {
            database = value.clone();
        }
        if database.is_empty() || user.is_empty() {
            return Err(Error::internal("postgres connection string is required"));
        }
        Ok(Dsn {
            user: percent_decode(user)?,
            password: percent_decode(password)?,
            host: host.to_string(),
            port,
            database,
        })
    }
}

fn percent_decode(input: &str) -> Result<String, Error> {
    let bytes = input.as_bytes();
    let mut out = Vec::with_capacity(bytes.len());
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'%' {
            if i + 2 >= bytes.len() {
                return Err(Error::internal("invalid postgres connection string"));
            }
            let hex = std::str::from_utf8(&bytes[i + 1..i + 3])
                .map_err(|_| Error::internal("invalid postgres connection string"))?;
            let byte = u8::from_str_radix(hex, 16)
                .map_err(|_| Error::internal("invalid postgres connection string"))?;
            out.push(byte);
            i += 3;
        } else {
            out.push(bytes[i]);
            i += 1;
        }
    }
    String::from_utf8(out).map_err(|_| Error::internal("invalid postgres connection string"))
}
