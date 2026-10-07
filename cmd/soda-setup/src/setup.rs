use std::fs::{self, OpenOptions};
use std::io::{self, Write};
use std::os::unix::fs::{DirBuilderExt, OpenOptionsExt};
use std::path::Path;

use crate::config::encode_dashboard_config;
use crate::forgejo::{forgejo_get_user, forgejo_revoke_token};
use crate::origin::{admit_setup_paths, credential};
use crate::secrets::{base64_encode, provision_postgres_secrets};

pub(crate) fn setup(
    external: &str,
    internal: &str,
    token_path: &str,
    out: &Path,
    pg_dir: &Path,
    stdout: &mut dyn Write,
) -> Result<(), String> {
    admit_setup_paths(external, internal, out)?;
    let token = credential(token_path)?;
    let user = forgejo_get_user(internal.trim_end_matches('/'), &token)?;
    if !user.admin {
        return Err("forgejo operator token is not a site administrator".to_string());
    }
    let dir = out
        .parent()
        .ok_or_else(|| "out must be absolute".to_string())?;
    fs::DirBuilder::new()
        .recursive(true)
        .mode(0o700)
        .create(dir)
        .map_err(|e| e.to_string())?;
    let key_path = dir.join("grant-key");
    let key_raw =
        read_random_32_inner().map_err(|e| format!("generate identity encryption key: {e}"))?;
    write_setup_secret(&key_path, &base64_encode(&key_raw))?;
    let (pg_created, dsn_path) = match provision_postgres_secrets(pg_dir) {
        Ok(result) => result,
        Err(err) => {
            let _ = fs::remove_file(&key_path);
            return Err(err);
        }
    };
    let config = encode_dashboard_config(
        "127.0.0.1:8080",
        external.trim_end_matches('/'),
        internal.trim_end_matches('/'),
        &dsn_path.to_string_lossy(),
        "/run/soda/host.sock",
        "",
        &key_path.to_string_lossy(),
        user.id,
    );
    if let Err(err) = write_setup_config(out, &config) {
        // This run created the key and database secrets through O_EXCL, so
        // no other setup owns them; remove them so a retry is not blocked
        // by our own partial state.
        let _ = fs::remove_file(&key_path);
        for path in &pg_created {
            let _ = fs::remove_file(path);
        }
        return Err(err);
    }
    // Revoke only after the configuration is durable: earlier failures keep
    // the bootstrap token so the operator can retry with it.
    if let Err(err) = forgejo_revoke_token(internal.trim_end_matches('/'), &token) {
        return Err(format!(
            "dashboard configuration written, but bootstrap-token revocation was not confirmed; inspect Forgejo Settings > Applications and revoke the token if it is still active, then continue activation with the installed soda-activate command: {err}"
        ));
    }
    let _ = writeln!(
        stdout,
        "Dashboard configuration created and the bootstrap token revoked. Set native soda service ownership before enabling the dashboard. No host privilege was granted to a Forgejo user."
    );
    Ok(())
}

fn read_random_32_inner() -> io::Result<[u8; 32]> {
    let mut raw = [0u8; 32];
    getrandom::fill(&mut raw).map_err(io::Error::other)?;
    Ok(raw)
}

fn write_setup_secret(path: &Path, value: &str) -> Result<(), String> {
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|e| format!("create credential encryption key: {e}"))?;
    let data = format!("{value}\n");
    file.write_all(data.as_bytes()).map_err(|e| e.to_string())?;
    drop(file);
    Ok(())
}

fn write_setup_config(out: &Path, config: &[u8]) -> Result<(), String> {
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(out)
        .map_err(|e| e.to_string())?;
    file.write_all(config).map_err(|e| e.to_string())?;
    drop(file);
    Ok(())
}
