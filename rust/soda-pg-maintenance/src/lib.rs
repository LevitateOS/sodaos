//! Shared helpers for the appliance PostgreSQL maintenance tools
//! (soda-pg-backup, soda-pg-restore, soda-pg-init-roles). Rust ports of
//! appliance/bin/soda-pg-*. Std-only so the tree builds without vendoring
//! (`cargo build`).

use std::time::{SystemTime, UNIX_EPOCH};

/// Shell `case "$db" in ''|*[!a-z0-9_]*|'pg_'*)` refusal, inverted.
pub fn valid_db_name(db: &str) -> bool {
    !db.is_empty()
        && db
            .bytes()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'_')
        && !db.starts_with("pg_")
}

/// SQL-quote doubling for `PASSWORD '...'` literals.
pub fn escape_literal(password: &str) -> String {
    password.replace('\'', "''")
}

/// Idempotent role/database provisioning block for one database, exactly as
/// the shell emits it (trailing newline included; newlines matter because
/// `\gexec` consumes the rest of its line).
pub fn init_role_sql(db: &str, escaped_password: &str) -> String {
    format!(
        "DO $$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='{db}') THEN CREATE ROLE \"{db}\" LOGIN; END IF; END $$;\n\
         ALTER ROLE \"{db}\" WITH LOGIN PASSWORD '{escaped_password}';\n\
         SELECT 'CREATE DATABASE \"{db}\" OWNER \"{db}\"' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname='{db}')\\gexec\n"
    )
}

/// `SELECT 1` existence probe used before `createdb --owner`.
pub fn db_exists_sql(db: &str) -> String {
    format!("SELECT 1 FROM pg_database WHERE datname='{db}'")
}

/// Days since 1970-01-01 to (year, month, day).
fn civil_from_days(z: i64) -> (i64, u32, u32) {
    let z = z + 719468;
    let era = if z >= 0 { z } else { z - 146096 } / 146097;
    let doe = (z - era * 146097) as u64; // [0, 146096]
    let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365; // [0, 399]
    let y = yoe as i64 + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100); // [0, 365]
    let mp = (5 * doy + 2) / 153; // [0, 11]
    let d = (doy - (153 * mp + 2) / 5 + 1) as u32; // [1, 31]
    let m = if mp < 10 { mp + 3 } else { mp - 9 } as u32; // [1, 12]
    (if m <= 2 { y + 1 } else { y }, m, d)
}

/// `date -u +%Y%m%dT%H%M%SZ` for a unix timestamp.
pub fn stamp_from_unix(secs: i64) -> String {
    let days = secs.div_euclid(86400);
    let rem = secs.rem_euclid(86400);
    let (y, m, d) = civil_from_days(days);
    format!(
        "{y:04}{m:02}{d:02}T{:02}{:02}{:02}Z",
        rem / 3600,
        rem % 3600 / 60,
        rem % 60
    )
}

/// Current UTC stamp, or None when the clock is unreadable.
pub fn utc_stamp() -> Option<String> {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .ok()
        .and_then(|t| i64::try_from(t.as_secs()).ok())
        .map(stamp_from_unix)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn db_names_match_shell_case() {
        for ok in ["forgejo", "soda", "pg", "a1_"] {
            assert!(valid_db_name(ok), "{ok}");
        }
        for bad in ["", "Foo", "a-b", "pg_x", "a b"] {
            assert!(!valid_db_name(bad), "{bad}");
        }
    }

    #[test]
    fn literal_escapes_quotes_only() {
        assert_eq!(escape_literal("q'uote$d\\"), "q''uote$d\\");
        assert_eq!(escape_literal("plain"), "plain");
    }

    #[test]
    fn init_sql_shape() {
        assert_eq!(
            init_role_sql("soda", "q''uote"),
            "DO $$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='soda') THEN CREATE ROLE \"soda\" LOGIN; END IF; END $$;\n\
             ALTER ROLE \"soda\" WITH LOGIN PASSWORD 'q''uote';\n\
             SELECT 'CREATE DATABASE \"soda\" OWNER \"soda\"' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname='soda')\\gexec\n"
        );
    }

    #[test]
    fn stamps_match_date_utc() {
        assert_eq!(stamp_from_unix(0), "19700101T000000Z");
        assert_eq!(stamp_from_unix(1760000000), "20251009T085320Z");
        assert_eq!(stamp_from_unix(951782400), "20000229T000000Z");
        assert_eq!(stamp_from_unix(1582991999), "20200229T155959Z");
    }
}
