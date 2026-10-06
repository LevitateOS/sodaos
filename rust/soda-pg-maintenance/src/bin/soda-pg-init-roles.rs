//! Idempotently create the appliance PostgreSQL roles and databases.
//! Runs as root: reads mode-0600 password files and applies them over
//! container-local peer auth. Missing roles/databases are created; existing
//! role passwords are reset to the files, so a replaced data dir or rotated
//! file converges on the next boot. Role and database share one name
//! (default roles: forgejo soda).
//!
//! Environment overrides (used by tests; production uses the defaults):
//!   SODA_PG_CONTAINER     container name (default soda-postgres)
//!   SODA_PG_PASSWORD_DIR  dir holding <role>.passwd files (default /etc/soda/postgres)
//!   SODA_PG_DATABASES     space-separated roles/databases (default "forgejo soda")
//!
//! Rust port of appliance/bin/soda-pg-init-roles. Std-only (`cargo build
//! --offline`). Exit codes and messages match the shell.

use soda_pg_maintenance::{delivery_exit, escape_literal, init_role_sql, valid_db_name};
use std::env;
use std::fs;
use std::io::Write;
use std::process::{Command, Stdio};
use std::thread;
use std::time::{Duration, Instant};

fn main() {
    std::process::exit(run());
}

fn run() -> i32 {
    let container = env::var("SODA_PG_CONTAINER").unwrap_or_else(|_| "soda-postgres".to_string());
    let pwdir =
        env::var("SODA_PG_PASSWORD_DIR").unwrap_or_else(|_| "/etc/soda/postgres".to_string());
    let databases = env::var("SODA_PG_DATABASES").unwrap_or_else(|_| "forgejo soda".to_string());

    let deadline = Instant::now() + Duration::from_secs(180);
    loop {
        let ready = Command::new("podman")
            .args(["exec", "-u", "postgres", &container, "pg_isready", "-q"])
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status()
            .is_ok_and(|s| s.success());
        if ready {
            break;
        }
        if Instant::now() >= deadline {
            eprintln!("container {container} never became ready");
            return 1;
        }
        thread::sleep(Duration::from_secs(2));
    }

    let mut sql = String::new();
    for db in databases.split_whitespace() {
        if !valid_db_name(db) {
            eprintln!("refusing database name: {db}");
            return 2;
        }
        let pwfile = format!("{pwdir}/{db}.passwd");
        // `[ -s ]`: missing or empty is "missing"; an unreadable file fails
        // the later read and reports "empty", as `cat` would.
        if !fs::metadata(&pwfile).is_ok_and(|info| info.len() > 0) {
            eprintln!("missing password file: {pwfile}");
            return 1;
        }
        let password = fs::read_to_string(&pwfile).unwrap_or_default();
        // `$(...)` strips trailing newlines; an empty result is refused.
        if password.trim_end_matches('\n').is_empty() {
            eprintln!("empty password file: {pwfile}");
            return 1;
        }
        let password = password.trim_end_matches('\n');
        sql.push_str(&init_role_sql(db, &escape_literal(password)));
    }

    let mut child = match Command::new("podman")
        .args([
            "exec",
            "-i",
            "-u",
            "postgres",
            &container,
            "psql",
            "-v",
            "ON_ERROR_STOP=1",
            "-f",
            "-",
        ])
        .stdin(Stdio::piped())
        .stdout(Stdio::null())
        .spawn()
    {
        Ok(child) => child,
        Err(e) => {
            eprintln!("{e}");
            return 1;
        }
    };
    // A heredoc appends one trailing newline.
    let input = format!("{sql}\n");
    let wrote = child
        .stdin
        .take()
        .is_some_and(|mut stdin| stdin.write_all(input.as_bytes()).is_ok());
    let code = match child.wait() {
        Ok(status) => delivery_exit(wrote, status),
        Err(e) => {
            eprintln!("{e}");
            1
        }
    };
    if code != 0 {
        return code;
    }

    println!("roles ready: {databases}");
    0
}
