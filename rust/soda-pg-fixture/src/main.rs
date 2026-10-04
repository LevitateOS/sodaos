//! Ephemeral PostgreSQL 17 for Soda/Forgejo tests (A10 runtime fixture).
//! Same pinned image and role==database shape as the appliance cluster, with
//! random per-run passwords that never appear in argv, stdout or logs.
//!
//! ```sh
//! eval "$(soda-pg-fixture start)"   # prints KEY=VALUE assignments
//! # ... run tests against $SODA_PG_HOST:$SODA_PG_PORT ...
//! soda-pg-fixture stop "$SODA_PG_CONTAINER"
//! ```
//!
//! Start output (passwords exposed as file paths only):
//!   SODA_PG_CONTAINER  container name for stop
//!   SODA_PG_HOST       127.0.0.1
//!   SODA_PG_PORT       mapped loopback port
//!   SODA_PG_DATABASES  space-separated databases created (default "forgejo soda")
//!   SODA_PG_DIR        secret dir holding <role>.passwd (mode 0600); caller removes it
//!   SODA_PG_SUPER_PASSWORD_FILE  postgres superuser password file
//!
//! SODA_PG_FIXTURE_DIR reuses a caller-owned dir instead of mktemp.
//! SODA_PG_DATABASES overrides the role/database list (names: [a-z0-9_]+).
//! Exit 3 when the container engine or image is unavailable: callers skip.
//!
//! Rust port of scripts/pg-fixture.sh. Std-only so the tree builds and tests
//! with zero network (`cargo build --offline`). Behavior, exit codes and
//! output lines match the shell original exactly.

use std::env;
use std::fs::{self, OpenOptions};
use std::io::{Read, Write};
use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::thread;
use std::time::{Duration, Instant};

// Must match Image= in appliance/services/soda-postgres.container.
const IMAGE: &str =
    "docker.io/library/postgres:17@sha256:67f41722b7a8cbdb868a44a4995c846eddfdc2973bccb291ce937dce88ad5675";
const READY_TIMEOUT: Duration = Duration::from_secs(90);

fn main() {
    std::process::exit(run());
}

fn run() -> i32 {
    // `command -v podman` first: a missing engine is skip (3), like the script.
    if !have_podman() {
        eprintln!("podman unavailable");
        return 3;
    }
    let argv: Vec<String> = env::args().collect();
    match argv.get(1).map(String::as_str) {
        Some("start") => cmd_start(),
        Some("stop") => {
            let name = argv.get(2).map(String::as_str).unwrap_or("");
            if name.is_empty() {
                eprintln!("container name required");
                return 2;
            }
            // `exec podman rm -f`: stdout discarded, status propagated.
            match Command::new("podman")
                .args(["rm", "-f", name])
                .stdout(Stdio::null())
                .status()
            {
                Ok(status) => status.code().unwrap_or(1),
                Err(_) => 1,
            }
        }
        _ => {
            let prog = argv
                .first()
                .map(String::as_str)
                .unwrap_or("soda-pg-fixture");
            eprintln!("usage: {prog} {{start|stop <container>}}");
            2
        }
    }
}

fn is_executable(path: &Path) -> bool {
    fs::metadata(path).is_ok_and(|info| info.is_file() && info.permissions().mode() & 0o111 != 0)
}

fn have_podman() -> bool {
    env::var_os("PATH")
        .is_some_and(|paths| env::split_paths(&paths).any(|dir| is_executable(&dir.join("podman"))))
}

/// Shell `case "$db" in ''|*[!a-z0-9_]*|'pg_'*)` refusal, inverted.
fn valid_db_name(db: &str) -> bool {
    !db.is_empty()
        && db
            .bytes()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'_')
        && !db.starts_with("pg_")
}

fn read_urandom(buf: &mut [u8]) -> Result<(), String> {
    fs::File::open("/dev/urandom")
        .and_then(|mut f| f.read_exact(buf))
        .map_err(|e| format!("/dev/urandom: {e}"))
}

/// `mktemp -d "${TMPDIR:-/tmp}/soda-pg-fixture.XXXXXX"` (mode 0700), or the
/// caller-owned SODA_PG_FIXTURE_DIR created like `mkdir -p`.
fn secret_dir() -> Result<PathBuf, String> {
    if let Ok(dir) = env::var("SODA_PG_FIXTURE_DIR") {
        let path = PathBuf::from(&dir);
        fs::create_dir_all(&path).map_err(|e| format!("{dir}: {e}"))?;
        return Ok(path);
    }
    let base = env::var("TMPDIR").unwrap_or_else(|_| "/tmp".to_string());
    for _ in 0..100 {
        let mut rand = [0u8; 6];
        read_urandom(&mut rand)?;
        let suffix: String = rand
            .iter()
            .map(|b| {
                const ALPHABET: &[u8] =
                    b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
                ALPHABET[(b % 62) as usize] as char
            })
            .collect();
        let cand = PathBuf::from(format!("{base}/soda-pg-fixture.{suffix}"));
        match fs::create_dir(&cand) {
            Ok(()) => {
                fs::set_permissions(&cand, fs::Permissions::from_mode(0o700))
                    .map_err(|e| format!("{}: {e}", cand.display()))?;
                return Ok(cand);
            }
            Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(e) => return Err(format!("{}: {e}", cand.display())),
        }
    }
    Err("mktemp: too many attempts".to_string())
}

/// `head -c 33 /dev/urandom | od -An -tx1 | tr -d ' \n'`: 66 lowercase hex
/// chars, no trailing newline, then `chmod 600`.
fn write_password(dir: &Path, role: &str) -> Result<(), String> {
    let mut bytes = [0u8; 33];
    read_urandom(&mut bytes)?;
    let hex: String = bytes.iter().map(|b| format!("{b:02x}")).collect();
    let path = dir.join(format!("{role}.passwd"));
    OpenOptions::new()
        .write(true)
        .create(true)
        .truncate(true)
        .mode(0o600)
        .open(&path)
        .and_then(|mut f| f.write_all(hex.as_bytes()))
        .map_err(|e| format!("{}: {e}", path.display()))?;
    fs::set_permissions(&path, fs::Permissions::from_mode(0o600))
        .map_err(|e| format!("{}: {e}", path.display()))
}

fn role_sql(db: &str, password: &str) -> String {
    format!("CREATE ROLE \"{db}\" LOGIN PASSWORD '{password}'; CREATE DATABASE \"{db}\" OWNER \"{db}\";")
}

/// First line of `podman port`, minus `${port##*:}`.
fn first_port(output: &str) -> String {
    let line = output.lines().next().unwrap_or("");
    match line.rfind(':') {
        Some(i) => line[i + 1..].to_string(),
        None => line.to_string(),
    }
}

/// The EXIT trap after a successful `podman run`: best-effort `rm -f`.
fn cleanup(name: &str) {
    let _ = Command::new("podman")
        .args(["rm", "-f", name])
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .status();
}

fn cmd_start() -> i32 {
    // Unset -> default; explicitly empty -> bare cluster (superuser only).
    let databases = env::var("SODA_PG_DATABASES").unwrap_or_else(|_| "forgejo soda".to_string());
    let list: Vec<&str> = databases.split_whitespace().collect();
    for db in &list {
        if !valid_db_name(db) {
            eprintln!("refusing database name: {db}");
            return 2;
        }
    }

    let have_image = Command::new("podman")
        .args(["image", "exists", IMAGE])
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .status()
        .is_ok_and(|s| s.success());
    if !have_image {
        let pulled = Command::new("podman")
            .args(["pull", IMAGE])
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status()
            .is_ok_and(|s| s.success());
        if !pulled {
            eprintln!("postgres fixture image unavailable");
            return 3;
        }
    }

    let dir = match secret_dir() {
        Ok(dir) => dir,
        Err(e) => {
            eprintln!("{e}");
            return 1;
        }
    };
    let mut roles = vec!["postgres"];
    roles.extend(list.iter().copied());
    for role in &roles {
        if let Err(e) = write_password(&dir, role) {
            eprintln!("{e}");
            return 1;
        }
    }

    let mut rand = [0u8; 2];
    if let Err(e) = read_urandom(&mut rand) {
        eprintln!("{e}");
        return 1;
    }
    // `soda-pg-test-$$-$RANDOM`.
    let random = u16::from_ne_bytes(rand) % 32768;
    let name = format!("soda-pg-test-{}-{random}", std::process::id());
    let super_secret = format!(
        "{}/postgres.passwd:/run/secrets/soda-pg-super:ro,z",
        dir.display()
    );
    match Command::new("podman")
        .args([
            "run",
            "-d",
            "--rm",
            "--name",
            &name,
            "--pull=never",
            "-p",
            "127.0.0.1::5432",
            "-v",
            &super_secret,
            "-e",
            "POSTGRES_PASSWORD_FILE=/run/secrets/soda-pg-super",
            IMAGE,
        ])
        .stdout(Stdio::null())
        .status()
    {
        Ok(status) if status.success() => {}
        // `set -e`: the failing command's status is the script's status.
        // The EXIT trap is not armed yet, so no cleanup here.
        Ok(status) => return status.code().unwrap_or(1),
        Err(e) => {
            eprintln!("podman run: {e}");
            return 1;
        }
    }

    let deadline = Instant::now() + READY_TIMEOUT;
    loop {
        let ready = Command::new("podman")
            .args(["exec", "-u", "postgres", &name, "pg_isready", "-q"])
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status()
            .is_ok_and(|s| s.success());
        if ready {
            break;
        }
        if Instant::now() >= deadline {
            eprintln!("postgres fixture never became ready");
            cleanup(&name);
            return 1;
        }
        thread::sleep(Duration::from_secs(1));
    }

    let mut sql = String::new();
    for db in &list {
        let password = match fs::read_to_string(dir.join(format!("{db}.passwd"))) {
            Ok(password) => password,
            Err(e) => {
                eprintln!("{e}");
                cleanup(&name);
                return 1;
            }
        };
        // `$(...)` strips trailing newlines; the file holds bare hex.
        sql.push_str(&role_sql(db, password.trim_end_matches('\n')));
    }
    let mut child = match Command::new("podman")
        .args([
            "exec",
            "-i",
            "-u",
            "postgres",
            &name,
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
            cleanup(&name);
            return 1;
        }
    };
    // A heredoc appends one trailing newline.
    let input = format!("{sql}\n");
    let wrote = child
        .stdin
        .take()
        .is_some_and(|mut stdin| stdin.write_all(input.as_bytes()).is_ok());
    match (wrote, child.wait()) {
        (true, Ok(status)) if status.success() => {}
        (_, Ok(status)) => {
            cleanup(&name);
            return status.code().unwrap_or(1);
        }
        (_, Err(e)) => {
            eprintln!("{e}");
            cleanup(&name);
            return 1;
        }
    }

    // Trap disarmed: from here failures exit without cleanup, like the script.
    let port = match Command::new("podman")
        .args(["port", &name, "5432"])
        .output()
    {
        Ok(output) if output.status.success() => {
            first_port(&String::from_utf8_lossy(&output.stdout))
        }
        Ok(output) => return output.status.code().unwrap_or(1),
        Err(e) => {
            eprintln!("{e}");
            return 1;
        }
    };
    println!("SODA_PG_CONTAINER={name}");
    println!("SODA_PG_HOST=127.0.0.1");
    println!("SODA_PG_PORT={port}");
    println!("SODA_PG_DATABASES=\"{databases}\"");
    println!("SODA_PG_DIR={}", dir.display());
    println!(
        "SODA_PG_SUPER_PASSWORD_FILE={}/postgres.passwd",
        dir.display()
    );
    0
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn db_names_match_shell_case() {
        for ok in ["forgejo", "soda", "pg", "a", "0", "a1_", "_x", "pgx"] {
            assert!(valid_db_name(ok), "{ok}");
        }
        for bad in ["", "Foo", "a-b", "a.b", "a b", "pg_x", "pg_", "é"] {
            assert!(!valid_db_name(bad), "{bad}");
        }
    }

    #[test]
    fn port_takes_first_line_after_last_colon() {
        assert_eq!(first_port("0.0.0.0:54321\n"), "54321");
        assert_eq!(first_port("127.0.0.1:123\n127.0.0.1:456\n"), "123");
        assert_eq!(first_port("5432"), "5432");
        assert_eq!(first_port(""), "");
    }

    #[test]
    fn role_sql_shape() {
        assert_eq!(
            role_sql("soda", "ab12"),
            "CREATE ROLE \"soda\" LOGIN PASSWORD 'ab12'; CREATE DATABASE \"soda\" OWNER \"soda\";"
        );
    }
}
