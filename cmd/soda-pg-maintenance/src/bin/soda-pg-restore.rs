//! Restore appliance PostgreSQL databases from a soda-pg-backup run.
//! Usage: soda-pg-restore --yes <backup-dir> [db ...]
//! Default database list is every *.dump in the run. Missing databases are
//! recreated owned by their same-name role (the appliance role==db convention).
//!
//! Stop Forgejo and soda-host before restoring, and restore globals only onto
//! a fresh cluster (first init already creates the standard roles):
//!   soda-pg-restore --yes --globals <backup-dir>
//!
//! Same SODA_PG_CONTAINER override as soda-pg-backup. Authentication is
//! container-local peer auth; passwords never appear in argv or logs.
//!
//! Rust port of appliance/bin/soda-pg-restore. Std-only (`cargo build
//! --offline`). Exit codes and messages match the shell.

use soda_pg_maintenance::{db_exists_sql, delivery_exit, valid_db_name};
use std::env;
use std::fs;
use std::path::Path;
use std::process::{Command, Stdio};

fn main() {
    std::process::exit(run());
}

fn run() -> i32 {
    let container = env::var("SODA_PG_CONTAINER").unwrap_or_else(|_| "soda-postgres".to_string());
    let argv: Vec<String> = env::args().collect();
    let mut args = argv.iter().skip(1);
    if args.next().map(String::as_str) != Some("--yes") {
        eprintln!(
            "refusing without --yes: stop forgejo.service and soda-host first, then repeat with --yes"
        );
        return 2;
    }
    let rest: Vec<&str> = args.map(String::as_str).collect();
    let (globals_only, rest) = match rest.first() {
        Some(&"--globals") => (true, &rest[1..]),
        _ => (false, &rest[..]),
    };
    let Some(&run) = rest.first() else {
        eprintln!("backup dir required");
        return 2;
    };
    if !Path::new(run).is_dir() {
        eprintln!("backup dir required");
        return 2;
    }
    let explicit: Vec<&str> = rest[1..].to_vec();

    if globals_only {
        let globals = format!("{run}/globals.sql");
        if !is_nonempty(&globals) {
            eprintln!("no globals.sql in {run}");
            return 1;
        }
        let sql = match fs::read(&globals) {
            Ok(sql) => sql,
            Err(_) => {
                eprintln!("no globals.sql in {run}");
                return 1;
            }
        };
        if let Err(code) = psql_stdin(&container, &sql) {
            return code;
        }
        println!("globals restored from {run}");
        return 0;
    }

    let dbs: Vec<String> = if explicit.is_empty() {
        match dump_names(run) {
            Some(dbs) if !dbs.is_empty() => dbs,
            _ => {
                eprintln!("no database dumps in {run}");
                return 1;
            }
        }
    } else {
        explicit.iter().map(|s| s.to_string()).collect()
    };

    let stage = format!("soda-restore-{}", std::process::id());
    for db in &dbs {
        if !valid_db_name(db) {
            eprintln!("refusing database name: {db}");
            return 2;
        }
        let dump = format!("{run}/{db}.dump");
        if !is_nonempty(&dump) {
            eprintln!("no dump for {db} in {run}");
            return 1;
        }
        if !db_exists(&container, db) {
            if let Err(code) = createdb(&container, db) {
                return code;
            }
        }
        let staged = format!("/tmp/{stage}-{db}.dump");
        // podman cp stages the file root-owned; hand it to postgres for
        // restore and cleanup so no world-writable staging is needed.
        if let Err(code) = podman_status(&["cp", &dump, &format!("{container}:{staged}")]) {
            return code;
        }
        if let Err(code) =
            podman_status(&["exec", &container, "chown", "postgres:postgres", &staged])
        {
            return code;
        }
        if podman_status(&[
            "exec",
            "-u",
            "postgres",
            &container,
            "pg_restore",
            &format!("--dbname={db}"),
            "--clean",
            "--if-exists",
            &staged,
        ])
        .is_err()
        {
            let _ = podman_status(&["exec", "-u", "postgres", &container, "rm", "-f", &staged]);
            eprintln!("restore of {db} failed");
            return 1;
        }
        let _ = podman_status(&["exec", "-u", "postgres", &container, "rm", "-f", &staged]);
        println!("restored {db} from {run}");
    }
    0
}

/// `[ -s ]`: exists with size over any file type.
fn is_nonempty(path: &str) -> bool {
    fs::metadata(path).is_ok_and(|info| info.len() > 0)
}

/// `podman ...` with inherited stdio, like the shell. `set -e` failures
/// abort with the command's own status; represent that as Err.
fn podman_status(args: &[&str]) -> Result<(), i32> {
    match Command::new("podman").args(args).status() {
        Ok(status) if status.success() => Ok(()),
        Ok(status) => Err(status.code().unwrap_or(1)),
        Err(_) => Err(1),
    }
}

fn psql_stdin(container: &str, sql: &[u8]) -> Result<(), i32> {
    use std::io::Write;
    let mut child = Command::new("podman")
        .args([
            "exec",
            "-i",
            "-u",
            "postgres",
            container,
            "psql",
            "-v",
            "ON_ERROR_STOP=1",
            "-f",
            "-",
        ])
        .stdin(Stdio::piped())
        .stdout(Stdio::null())
        .spawn()
        .map_err(|_| 1)?;
    let wrote = child
        .stdin
        .take()
        .is_some_and(|mut stdin| stdin.write_all(sql).is_ok());
    let code = match child.wait() {
        Ok(status) => delivery_exit(wrote, status),
        Err(_) => 1,
    };
    if code == 0 {
        Ok(())
    } else {
        Err(code)
    }
}

/// Alphabetical `*.dump` basenames, like the shell glob.
fn dump_names(run: &str) -> Option<Vec<String>> {
    let mut names: Vec<String> = fs::read_dir(run)
        .ok()?
        .filter_map(|e| e.ok())
        .map(|e| e.file_name().to_string_lossy().into_owned())
        .filter(|name| name.ends_with(".dump"))
        .map(|name| name.trim_end_matches(".dump").to_string())
        .collect();
    names.sort();
    Some(names)
}

/// `psql -tAc` existence probe; `$(...)` strips trailing newlines.
fn db_exists(container: &str, db: &str) -> bool {
    Command::new("podman")
        .args([
            "exec",
            "-u",
            "postgres",
            container,
            "psql",
            "-tAc",
            &db_exists_sql(db),
        ])
        .output()
        .is_ok_and(|out| {
            out.status.success()
                && String::from_utf8_lossy(&out.stdout).trim_end_matches('\n') == "1"
        })
}

fn createdb(container: &str, db: &str) -> Result<(), i32> {
    match Command::new("podman")
        .args([
            "exec",
            "-u",
            "postgres",
            container,
            "createdb",
            &format!("--owner={db}"),
            db,
        ])
        .stdout(Stdio::null())
        .status()
    {
        Ok(status) if status.success() => Ok(()),
        Ok(status) => Err(status.code().unwrap_or(1)),
        Err(_) => Err(1),
    }
}
