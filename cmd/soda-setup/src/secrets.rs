use std::fs::{self, OpenOptions};
use std::io::Write;
use std::os::unix::fs::{DirBuilderExt, OpenOptionsExt, PermissionsExt};
use std::path::Path;

use crate::system::{chown_path, euid, FORGEJO_DB_USER, POSTGRES_SOCKET_DIR, SODA_SERVICE_GROUP};

fn read_random_32() -> Result<[u8; 32], String> {
    let mut raw = [0u8; 32];
    getrandom::fill(&mut raw).map_err(|e| format!("generate database password: {e}"))?;
    Ok(raw)
}

pub(crate) fn hex_encode(raw: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut out = String::with_capacity(raw.len() * 2);
    for byte in raw {
        out.push(HEX[(byte >> 4) as usize] as char);
        out.push(HEX[(byte & 0x0f) as usize] as char);
    }
    out
}

pub(crate) fn base64_encode(raw: &[u8]) -> String {
    const ALPHABET: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::with_capacity(raw.len().div_ceil(3) * 4);
    for chunk in raw.chunks(3) {
        let mut n: u32 = 0;
        for (i, byte) in chunk.iter().enumerate() {
            n |= (*byte as u32) << (16 - 8 * i);
        }
        out.push(ALPHABET[(n >> 18) as usize & 63] as char);
        out.push(ALPHABET[(n >> 12) as usize & 63] as char);
        if chunk.len() > 1 {
            out.push(ALPHABET[(n >> 6) as usize & 63] as char);
        } else {
            out.push('=');
        }
        if chunk.len() > 2 {
            out.push(ALPHABET[n as usize & 63] as char);
        } else {
            out.push('=');
        }
    }
    out
}

fn write_secret_file(path: &Path, value: &str) -> Result<(), String> {
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|e| format!("create {}: {e}", path.display()))?;
    if let Err(e) = file.write_all(format!("{value}\n").as_bytes()) {
        let _ = fs::remove_file(path);
        return Err(e.to_string());
    }
    drop(file);
    Ok(())
}

// reuse_postgres_secrets adopts a complete first-boot credential set so the
// operator flow never regenerates live database passwords. It returns the
// DSN path when all four files exist and agree, an error when they exist
// but disagree, and None when anything is missing (the caller then creates
// every file O_EXCL, so partial sets still fail instead of mixing).
pub(crate) fn reuse_postgres_secrets(dir: &Path) -> Result<Option<std::path::PathBuf>, String> {
    let mut reads = std::collections::HashMap::new();
    for name in ["super.passwd", "forgejo.passwd", "soda.passwd", "soda.dsn"] {
        match fs::read(dir.join(name)) {
            Ok(data) => {
                reads.insert(name, String::from_utf8_lossy(&data).trim().to_string());
            }
            Err(_) => return Ok(None),
        }
    }
    let empty = String::new();
    let soda_pw = reads.get("soda.passwd").unwrap_or(&empty);
    let dsn = reads.get("soda.dsn").unwrap_or(&empty);
    if soda_pw.is_empty() || !dsn.contains(soda_pw) {
        return Err(format!(
            "database secrets in {} exist but disagree; inspect them before setup",
            dir.display()
        ));
    }
    Ok(Some(dir.join("soda.dsn")))
}

// provision_postgres_secrets generates the database credential files: one
// hex password per role plus the soda connection URL. Files land O_EXCL so
// a pre-existing secret is never overwritten; on failure this run removes
// only the files it created, preserving anything already there. Ownership
// fixes apply at creation only; pre-existing files keep their metadata.
pub(crate) fn provision_postgres_secrets(
    dir: &Path,
) -> Result<(Vec<std::path::PathBuf>, std::path::PathBuf), String> {
    fs::DirBuilder::new()
        .recursive(true)
        .mode(0o700)
        .create(dir)
        .map_err(|e| e.to_string())?;
    if let Some(reuse) = reuse_postgres_secrets(dir)? {
        return Ok((Vec::new(), reuse));
    }
    let mut created: Vec<std::path::PathBuf> = Vec::new();
    let mut passwords = std::collections::HashMap::new();
    let mut failure: Option<String> = None;
    for role in ["super", "forgejo", "soda"] {
        let raw = match read_random_32() {
            Ok(raw) => raw,
            Err(err) => {
                failure = Some(err);
                break;
            }
        };
        let pw = hex_encode(&raw);
        let path = dir.join(format!("{role}.passwd"));
        if let Err(err) = write_secret_file(&path, &pw) {
            failure = Some(err);
            break;
        }
        if role == "forgejo" && euid() == 0 {
            if let Err(err) = chown_path(&path, FORGEJO_DB_USER, FORGEJO_DB_USER) {
                let _ = fs::remove_file(&path);
                failure = Some(err.to_string());
                break;
            }
        }
        created.push(path);
        passwords.insert(role, pw);
    }
    let mut dsn_path = std::path::PathBuf::new();
    if failure.is_none() {
        dsn_path = dir.join("soda.dsn");
        let dsn = format!(
            "postgres://soda:{}@/soda?host={POSTGRES_SOCKET_DIR}&sslmode=disable",
            passwords.get("soda").cloned().unwrap_or_default()
        );
        match write_secret_file(&dsn_path, &dsn) {
            Ok(()) => {
                created.push(dsn_path.clone());
                let chmod_err =
                    fs::set_permissions(&dsn_path, fs::Permissions::from_mode(0o640)).err();
                if let Some(err) = chmod_err {
                    failure = Some(err.to_string());
                } else if euid() == 0 {
                    if let Err(err) = chown_path(&dsn_path, 0, SODA_SERVICE_GROUP) {
                        failure = Some(err.to_string());
                    }
                }
            }
            Err(err) => {
                failure = Some(err);
            }
        }
    }
    if let Some(err) = failure {
        for path in &created {
            let _ = fs::remove_file(path);
        }
        return Err(err);
    }
    Ok((created, dsn_path))
}
