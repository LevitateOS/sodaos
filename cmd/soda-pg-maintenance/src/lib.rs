//! Shared helpers for the appliance PostgreSQL maintenance tools
//! (soda-pg-backup, soda-pg-restore, soda-pg-init-roles). Rust ports of
//! appliance/bin/soda-pg-*. Std-only so the tree builds without vendoring
//! (`cargo build`).

use std::process::ExitStatus;
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

/// `date -u +%Y%m%dT%H%M%SZ` for a unix timestamp.
pub fn stamp_from_unix(secs: i64) -> Option<String> {
    soda_wire_time::compact_utc(secs)
}

/// Exit code for a reaped SQL-delivery child (O01-F1/O06-F1). The caller
/// owns the child, writes its stdin, reaps exactly once, and maps the
/// outcome here: a nonzero child status is preserved, but a failed input
/// write is a failure even when the child itself exited zero.
pub fn delivery_exit(wrote: bool, status: ExitStatus) -> i32 {
    match (wrote, status.code()) {
        (true, Some(0)) => 0,
        (_, Some(code)) if code != 0 => code,
        _ => 1,
    }
}

/// Current UTC stamp, or None when the clock is unreadable.
pub fn utc_stamp() -> Option<String> {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .ok()
        .and_then(|t| i64::try_from(t.as_secs()).ok())
        .and_then(stamp_from_unix)
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

    #[cfg(unix)]
    #[test]
    fn delivery_exit_maps_failed_write_and_child_status() {
        use std::os::unix::process::ExitStatusExt;
        let exited = |code: i32| ExitStatus::from_raw(code << 8);
        // O01-F1/O06-F1: failed input delivery plus child exit zero is a
        // failure, never success.
        assert_eq!(delivery_exit(false, exited(0)), 1);
        assert_eq!(delivery_exit(true, exited(0)), 0);
        // Nonzero child errors are preserved either way.
        assert_eq!(delivery_exit(true, exited(3)), 3);
        assert_eq!(delivery_exit(false, exited(3)), 3);
        // Signalled children carry no code; report generic failure.
        assert_eq!(delivery_exit(true, ExitStatus::from_raw(9)), 1);
        assert_eq!(delivery_exit(false, ExitStatus::from_raw(9)), 1);
    }

    #[test]
    fn stamps_match_date_utc() {
        assert_eq!(stamp_from_unix(0).as_deref(), Some("19700101T000000Z"));
        assert_eq!(
            stamp_from_unix(1760000000).as_deref(),
            Some("20251009T085320Z")
        );
        assert_eq!(
            stamp_from_unix(951782400).as_deref(),
            Some("20000229T000000Z")
        );
        assert_eq!(
            stamp_from_unix(1582991999).as_deref(),
            Some("20200229T155959Z")
        );
        assert_eq!(stamp_from_unix(i64::MIN), None);
    }
}
