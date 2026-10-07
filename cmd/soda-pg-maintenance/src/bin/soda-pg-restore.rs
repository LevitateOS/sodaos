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
        // Keep the stage container-local: the host cannot own this pathname
        // or guarantee its cleanup after podman has copied into the container.
        let stage = match create_remote_stage(&container) {
            Ok(path) => path,
            Err(code) => return code,
        };
        if let Err(error) = restore_dump_at_with(&container, &dump, db, &stage, podman_status) {
            if error.restore_failed {
                eprintln!("restore of {db} failed");
            }
            if error.cleanup_failed {
                eprintln!("cannot remove container restore staging {stage}");
            }
            return error.exit_code;
        }
        println!("restored {db} from {run}");
    }
    0
}

/// Ask the container to create a private, exclusive stage directory. The
/// returned path is constrained to one fixed `/tmp` leaf before it is used.
fn create_remote_stage(container: &str) -> Result<String, i32> {
    let output = Command::new("podman")
        .args([
            "exec",
            container,
            "mktemp",
            "-d",
            "/tmp/soda-restore.XXXXXXXX",
        ])
        .output()
        .map_err(|_| 1)?;
    if !output.status.success() {
        return Err(output.status.code().unwrap_or(1));
    }
    let output = String::from_utf8_lossy(&output.stdout);
    let path = output.strip_suffix('\n').unwrap_or(&output).to_owned();
    if !valid_remote_stage_path(&path) {
        return Err(1);
    }
    Ok(path)
}

fn valid_remote_stage_path(path: &str) -> bool {
    path.strip_prefix("/tmp/soda-restore.")
        .is_some_and(|leaf| leaf.len() == 8 && leaf.bytes().all(|b| b.is_ascii_alphanumeric()))
}

#[derive(Debug, PartialEq, Eq)]
struct RestoreStageError {
    exit_code: i32,
    restore_failed: bool,
    cleanup_failed: bool,
}

/// Copy, transfer, restore, and always remove one container-owned stage.
/// The command seam keeps lifecycle behavior testable without starting Podman.
fn restore_dump_at_with(
    container: &str,
    dump: &str,
    db: &str,
    stage: &str,
    mut command: impl FnMut(&[&str]) -> Result<(), i32>,
) -> Result<(), RestoreStageError> {
    let staged = format!("{stage}/{db}.dump");
    let operation = (|| -> Result<(), RestoreStageError> {
        command(&["cp", dump, &format!("{container}:{staged}")]).map_err(|exit_code| {
            RestoreStageError {
                exit_code,
                restore_failed: false,
                cleanup_failed: false,
            }
        })?;
        command(&["exec", container, "chown", "postgres:postgres", stage]).map_err(
            |exit_code| RestoreStageError {
                exit_code,
                restore_failed: false,
                cleanup_failed: false,
            },
        )?;
        command(&["exec", container, "chown", "postgres:postgres", &staged]).map_err(
            |exit_code| RestoreStageError {
                exit_code,
                restore_failed: false,
                cleanup_failed: false,
            },
        )?;
        command(&[
            "exec",
            "-u",
            "postgres",
            container,
            "pg_restore",
            &format!("--dbname={db}"),
            "--clean",
            "--if-exists",
            &staged,
        ])
        .map_err(|_| RestoreStageError {
            exit_code: 1,
            restore_failed: true,
            cleanup_failed: false,
        })
    })();
    let cleanup = command(&["exec", container, "rm", "-rf", stage]);
    match operation {
        Ok(()) => cleanup.map_err(|exit_code| RestoreStageError {
            exit_code,
            restore_failed: false,
            cleanup_failed: true,
        }),
        Err(mut error) => {
            error.cleanup_failed = cleanup.is_err();
            Err(error)
        }
    }
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn remote_stage_requires_one_fixed_safe_leaf() {
        assert!(valid_remote_stage_path("/tmp/soda-restore.A1b2C3d4"));
        for bad in [
            "",
            "/tmp/soda-restore.short",
            "/tmp/soda-restore.A1b2C3d4/child",
            "/tmp/soda-restore.A1b2C3d4/../other",
            "/tmp/soda-restore.A1b2C3d4\n/other",
            "/tmp/other.A1b2C3d4",
            "/tmp/soda-restore.A1b2C3!4",
        ] {
            assert!(!valid_remote_stage_path(bad), "{bad:?}");
        }
    }

    #[test]
    fn failed_copy_chown_or_restore_always_cleans_the_same_stage() {
        for failed_operation in 0..4 {
            let mut calls = Vec::new();
            let result = restore_dump_at_with(
                "postgres-container",
                "/backup/soda.dump",
                "soda",
                "/tmp/soda-restore.A1b2C3d4",
                |args| {
                    calls.push(args.iter().map(|arg| (*arg).to_owned()).collect::<Vec<_>>());
                    if calls.len() - 1 == failed_operation {
                        Err(23)
                    } else {
                        Ok(())
                    }
                },
            );
            let error = result.unwrap_err();
            assert_eq!(error.cleanup_failed, false);
            assert_eq!(error.restore_failed, failed_operation == 3);
            assert_eq!(error.exit_code, if failed_operation == 3 { 1 } else { 23 });
            let expected_cleanup = [
                "exec",
                "postgres-container",
                "rm",
                "-rf",
                "/tmp/soda-restore.A1b2C3d4",
            ]
            .map(str::to_owned);
            assert_eq!(
                calls.last().unwrap().as_slice(),
                expected_cleanup.as_slice()
            );
        }
    }

    #[test]
    fn cleanup_failure_cannot_report_success_and_success_cleans_before_return() {
        let mut calls = Vec::new();
        let result = restore_dump_at_with(
            "postgres-container",
            "/backup/soda.dump",
            "soda",
            "/tmp/soda-restore.A1b2C3d4",
            |args| {
                calls.push(args.iter().map(|arg| (*arg).to_owned()).collect::<Vec<_>>());
                if calls.len() == 5 {
                    Err(44)
                } else {
                    Ok(())
                }
            },
        );
        assert_eq!(
            result.unwrap_err(),
            RestoreStageError {
                exit_code: 44,
                restore_failed: false,
                cleanup_failed: true,
            }
        );
        let expected_cleanup = [
            "exec",
            "postgres-container",
            "rm",
            "-rf",
            "/tmp/soda-restore.A1b2C3d4",
        ]
        .map(str::to_owned);
        assert_eq!(
            calls.last().unwrap().as_slice(),
            expected_cleanup.as_slice()
        );

        let mut success_calls = Vec::new();
        assert!(restore_dump_at_with(
            "postgres-container",
            "/backup/soda.dump",
            "soda",
            "/tmp/soda-restore.A1b2C3d4",
            |args| {
                success_calls.push(args.iter().map(|arg| (*arg).to_owned()).collect::<Vec<_>>());
                Ok(())
            },
        )
        .is_ok());
        assert_eq!(
            success_calls.last().unwrap().as_slice(),
            expected_cleanup.as_slice()
        );
    }
}
