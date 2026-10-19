//! Fixed private application sites in the existing appliance Caddy configuration.
use super::wire::Spec;
use std::fs::{self, File, OpenOptions};
use std::io::{Read, Write};
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::path::{Path, PathBuf};

const ROUTES: &str = "/home/soda-applications/routes";

fn body(spec: &Spec) -> Result<String, String> {
    spec.validate()?;
    Ok(format!("{} {{\n\tbind {{$SODA_BIND}}\n\ttls {{$SODA_TLS:/etc/soda/tls/cert.pem /etc/soda/tls/key.pem}}\n\treverse_proxy 127.0.0.1:{}\n}}\n", spec.origin, spec.port))
}

fn directory(path: &Path, owner: u32) -> Result<(), String> {
    let meta = fs::symlink_metadata(path).map_err(|_| "application routes are unavailable")?;
    if !meta.is_dir()
        || meta.file_type().is_symlink()
        || meta.uid() != owner
        || meta.mode() & 0o077 != 0
        || fs::canonicalize(path).map_err(|_| "application routes are unavailable")? != path
    {
        return Err("application routes require a private canonical operator directory".into());
    }
    Ok(())
}

fn current(path: &Path, expected: &[u8], owner: u32) -> Result<(), String> {
    let file = OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK)
        .open(path)
        .map_err(|_| "application route is unavailable")?;
    let meta = file
        .metadata()
        .map_err(|_| "application route is unavailable")?;
    if !meta.is_file()
        || meta.uid() != owner
        || meta.mode() & 0o077 != 0
        || meta.nlink() != 1
        || meta.len() != expected.len() as u64
    {
        return Err("application route ownership or content differs".into());
    }
    let mut bytes = Vec::new();
    file.take(expected.len() as u64 + 1)
        .read_to_end(&mut bytes)
        .map_err(|_| "application route is unavailable")?;
    if bytes != expected {
        return Err("application route differs from admitted origin or backend".into());
    }
    Ok(())
}

fn path(root: &Path, spec: &Spec, owner: u32) -> Result<(PathBuf, String), String> {
    let content = body(spec)?;
    directory(root, owner)?;
    Ok((root.join(format!("{}.caddy", spec.id)), content))
}

fn install_at(root: &Path, spec: &Spec, owner: u32) -> Result<(), String> {
    let (route, content) = path(root, spec, owner)?;
    match fs::symlink_metadata(&route) {
        Ok(_) => return current(&route, content.as_bytes(), owner),
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {}
        Err(_) => return Err("application route is unavailable".into()),
    }
    let mut nonce = [0u8; 12];
    getrandom::fill(&mut nonce).map_err(|_| "application route randomness is unavailable")?;
    let pending = root.join(format!(
        ".{}.{}.pending",
        spec.id,
        crate::sha256::hex_lower(&nonce)
    ));
    let result = (|| {
        let mut file = OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .custom_flags(libc::O_NOFOLLOW)
            .open(&pending)
            .map_err(|_| "application route staging failed")?;
        file.write_all(content.as_bytes())
            .map_err(|_| "application route staging failed")?;
        file.sync_all()
            .map_err(|_| "application route staging failed")?;
        // An atomic hard link publishes complete bytes without replacing an
        // existing route. The pending name does not match Caddy's import glob.
        fs::hard_link(&pending, &route)
            .map_err(|_| "application route already exists or publication failed")?;
        fs::remove_file(&pending).map_err(|_| "application route publication failed")?;
        File::open(root)
            .and_then(|file| file.sync_all())
            .map_err(|_| "application route directory sync failed")?;
        current(&route, content.as_bytes(), owner)
    })();
    let _ = fs::remove_file(&pending);
    result
}

pub(super) fn install(spec: &Spec) -> Result<(), String> {
    install_at(&super::home_path(Path::new(ROUTES))?, spec, 0)
}

pub(super) fn verify(spec: &Spec) -> Result<(), String> {
    let (route, content) = path(&super::home_path(Path::new(ROUTES))?, spec, 0)?;
    current(&route, content.as_bytes(), 0)
}
