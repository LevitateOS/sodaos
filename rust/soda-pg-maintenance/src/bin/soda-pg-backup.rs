//! Consistent backup of the appliance PostgreSQL databases.
//! Dumps globals plus each listed database (default: forgejo soda) into a new
//! timestamped directory, verifies the archives, then rotates old runs.
//! Authentication is container-local peer auth via podman exec; no password
//! appears in argv, env, logs or the backup itself.
//!
//! Environment overrides (used by tests; production uses the defaults):
//!   SODA_PG_CONTAINER  container name (default soda-postgres)
//!   SODA_PG_BACKUP_DIR backup root (default /var/lib/soda/backups/postgres)
//!   SODA_PG_DATABASES  space-separated databases (default "forgejo soda")
//!   SODA_PG_KEEP       runs to retain (default 7)
//!
//! Rust port of appliance/bin/soda-pg-backup. Std-only (`cargo build
//! --offline`). Exit codes, messages and rotation behavior match the shell.

use soda_pg_maintenance::utc_stamp;
use std::env;
use std::fs::{self, File};
use std::path::Path;
use std::process::Command;

fn main() {
    std::process::exit(run());
}

fn run() -> i32 {
    let container = env::var("SODA_PG_CONTAINER").unwrap_or_else(|_| "soda-postgres".to_string());
    let backup_root = env::var("SODA_PG_BACKUP_DIR")
        .unwrap_or_else(|_| "/var/lib/soda/backups/postgres".to_string());
    let databases = env::var("SODA_PG_DATABASES").unwrap_or_else(|_| "forgejo soda".to_string());
    let keep_raw = env::var("SODA_PG_KEEP").unwrap_or_else(|_| "7".to_string());

    // `case "$keep" in (*[!0-9]*|'')` then `-ge 1`.
    let keep_valid = !keep_raw.is_empty()
        && keep_raw.bytes().all(|b| b.is_ascii_digit())
        && keep_raw.parse::<i64>().is_ok_and(|n| n >= 1);
    if !keep_valid {
        eprintln!("SODA_PG_KEEP must be a positive integer");
        return 2;
    }
    let keep: usize = keep_raw.parse().unwrap_or(usize::MAX);

    let Some(stamp) = utc_stamp() else {
        eprintln!("cannot read clock");
        return 1;
    };
    let pid = std::process::id();
    let work = format!("{backup_root}/.in-progress-{stamp}-{pid}");
    let final_dir = format!("{backup_root}/{stamp}");
    if let Err(e) = fs::create_dir_all(&work) {
        eprintln!("{work}: {e}");
        return 1;
    }
    // The EXIT trap owns the work dir until the run is published.
    let cleanup = |work: &str| {
        let _ = fs::remove_dir_all(work);
    };

    // One pg_dump per database, each a single consistent snapshot; globals
    // hold roles and permissions shared by both databases.
    if let Err(code) = dump_to_file(
        &container,
        &["pg_dumpall", "--globals-only"],
        &format!("{work}/globals.sql"),
    ) {
        cleanup(&work);
        return code;
    }
    for db in databases.split_whitespace() {
        let dump = format!("{work}/{db}.dump");
        if let Err(code) = dump_to_file(&container, &["pg_dump", "--format=custom", db], &dump) {
            cleanup(&work);
            return code;
        }
    }

    // Verify before publishing: every archive must be a non-empty
    // custom-format dump (PGDMP magic) and globals must hold both roles.
    for db in databases.split_whitespace() {
        let dump = format!("{work}/{db}.dump");
        if !fs::metadata(&dump).is_ok_and(|info| info.len() > 0) {
            eprintln!("empty dump for {db}");
            cleanup(&work);
            return 1;
        }
        // `head -c 5`: only the magic is read, never the whole archive.
        if !has_pgdmp_magic(&dump) {
            eprintln!("bad archive magic for {db}");
            cleanup(&work);
            return 1;
        }
    }
    let globals = fs::read_to_string(format!("{work}/globals.sql")).unwrap_or_default();
    if !globals.contains("CREATE ROLE forgejo") {
        eprintln!("globals lack forgejo role");
        cleanup(&work);
        return 1;
    }
    if !globals.contains("CREATE ROLE soda") {
        eprintln!("globals lack soda role");
        cleanup(&work);
        return 1;
    }

    if let Err(e) = fs::rename(&work, &final_dir) {
        eprintln!("{work}: {e}");
        cleanup(&work);
        return 1;
    }
    // Trap disarmed: rotate only after the new run is published; never
    // delete the run just made.
    rotate(&backup_root, keep);

    println!("backup complete: {final_dir}");
    0
}

/// `podman exec -u postgres <container> <args...> >file`, streaming stdout
/// to the file with stderr inherited. `set -e`: a failing dump aborts the
/// run with the command's own status.
fn dump_to_file(container: &str, args: &[&str], file: &str) -> Result<(), i32> {
    let out = File::create(file).map_err(|_| 1)?;
    match Command::new("podman")
        .arg("exec")
        .arg("-u")
        .arg("postgres")
        .arg(container)
        .args(args)
        .stdout(out)
        .status()
    {
        Ok(status) if status.success() => Ok(()),
        Ok(status) => Err(status.code().unwrap_or(1)),
        Err(_) => Err(1),
    }
}

fn has_pgdmp_magic(dump: &str) -> bool {
    use std::io::Read;
    let mut head = [0u8; 5];
    File::open(dump)
        .and_then(|mut f| f.read_exact(&mut head))
        .is_ok()
        && head == *b"PGDMP"
}

/// `ls -1 | grep -E '^[0-9]{8}T[0-9]{6}Z$' | sort`, dropping the oldest
/// while more than `keep` runs exist.
fn rotate(backup_root: &str, keep: usize) {
    let mut runs: Vec<String> = fs::read_dir(backup_root)
        .map(|entries| {
            entries
                .filter_map(|e| e.ok())
                .map(|e| e.file_name().to_string_lossy().into_owned())
                .filter(|name| is_run_name(name))
                .collect()
        })
        .unwrap_or_default();
    runs.sort();
    while runs.len() > keep {
        let victim = runs.remove(0);
        let _ = fs::remove_dir_all(Path::new(backup_root).join(&victim));
    }
}

/// `^[0-9]{8}T[0-9]{6}Z$`: 8+1+6+1 = 16 chars.
fn is_run_name(name: &str) -> bool {
    let bytes = name.as_bytes();
    bytes.len() == 16
        && bytes[..8].iter().all(|b| b.is_ascii_digit())
        && bytes[8] == b'T'
        && bytes[9..15].iter().all(|b| b.is_ascii_digit())
        && bytes[15] == b'Z'
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn run_names_match_shell_grep() {
        assert!(is_run_name("20261004T162421Z"));
        for bad in [
            "",
            "20261004T162421",   // short
            "20261004T162421ZZ", // long
            "2026100T162421Z",   // short date
            "20261004t162421Z",  // lowercase t
            "20261004T16242ZZ",  // non-digit time
            ".in-progress-20261004T162421Z-123",
            "globals.sql",
        ] {
            assert!(!is_run_name(bad), "{bad:?}");
        }
    }
}
