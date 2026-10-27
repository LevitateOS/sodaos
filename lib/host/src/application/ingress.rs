//! Fixed private application sites in the existing appliance Caddy configuration.
//!
//! FJ-847: the routes directory is opened once into a validated descriptor
//! and every staging/publication/read step resolves through it with
//! `O_NOFOLLOW` semantics. A symlink swapped into the routes path between a
//! check and a use cannot redirect staging or publication: the anchored
//! operations fail closed on the descriptor.
use super::wire::Spec;
use std::io::Read;
use std::os::fd::OwnedFd;
use std::path::{Path, PathBuf};

/// Leaf only: the routes tree lives under the single shared storage root, so
/// the fixture redirect (`SODA_APPLICATIONS_ROOT`) isolates ingress with
/// setup, runtime data, and snapshots. Production keeps the fixed appliance
/// path when the variable is unset; a fixture reach without a pre-created
/// private operator routes dir fails closed in `directory()` below.
const ROUTES_LEAF: &str = "routes";

fn routes_root() -> Result<PathBuf, String> {
    Ok(super::storage_root()?.join(ROUTES_LEAF))
}

fn body(spec: &Spec) -> Result<String, String> {
    spec.validate()?;
    Ok(format!("{} {{\n\tbind {{$SODA_BIND}}\n\ttls {{$SODA_TLS:/etc/soda/tls/cert.pem /etc/soda/tls/key.pem}}\n\treverse_proxy 127.0.0.1:{}\n}}\n", spec.origin, spec.port))
}

fn anchor(root: &Path, owner: u32) -> Result<OwnedFd, String> {
    // FJ-847: the routes directory is validated BEFORE use and every later
    // step in this file resolves through the returned descriptor. A symlink
    // swapped into `root` after this point cannot move staging, publication,
    // or the read-back: those never re-resolve the path.
    let fd =
        super::home::open_validated_dir(root, owner, 0o077, "application routes are unavailable")
            .map_err(|_| {
            "application routes require a private canonical operator directory".to_string()
        })?;
    super::home::fd_matches_path(root, &fd).map_err(|_| {
        "application routes require a private canonical operator directory".to_string()
    })?;
    Ok(fd)
}

fn current_at(dir: &OwnedFd, name: &str, expected: &[u8], owner: u32) -> Result<(), String> {
    let file = super::home::open_child_opt(dir, name, false, "application route is unavailable")?
        .ok_or("application route is unavailable")?;
    let stat = super::home::stat_opened(&file, "application route is unavailable")?;
    if (stat.st_mode as u32 & libc::S_IFMT) != libc::S_IFREG
        || stat.st_uid != owner
        || (stat.st_mode as u32 & 0o077) != 0
        || stat.st_nlink != 1
        || stat.st_size < 0
        || stat.st_size as u64 != expected.len() as u64
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

fn install_at(root: &Path, spec: &Spec, owner: u32) -> Result<(), String> {
    let content = body(spec)?;
    let dir = anchor(root, owner)?;
    let name = format!("{}.caddy", spec.id);
    if super::home::open_child_opt(&dir, &name, false, "application route is unavailable")?
        .is_some()
    {
        return current_at(&dir, &name, content.as_bytes(), owner);
    }
    let mut nonce = [0u8; 12];
    getrandom::fill(&mut nonce).map_err(|_| "application route randomness is unavailable")?;
    let pending = format!(".{}.{}.pending", spec.id, crate::sha256::hex_lower(&nonce));
    let staged = super::home::open_staged_file(
        &dir,
        &pending,
        owner,
        "application route staging failed",
        "application route staging failed",
    )?;
    super::home::write_full_and_sync(
        &dir,
        &pending,
        &staged,
        content.as_bytes(),
        "application route staging failed",
        "application route staging failed",
    )?;
    drop(staged);
    // An atomic hard link publishes complete bytes without replacing an
    // existing route. The pending name does not match Caddy's import glob.
    // Both names resolve through the validated routes descriptor.
    super::home::link_staged(
        &dir,
        &pending,
        &name,
        "application route already exists or publication failed",
        "application route directory sync failed",
    )?;
    current_at(&dir, &name, content.as_bytes(), owner)
}

pub(super) fn install(spec: &Spec) -> Result<(), String> {
    install_at(&routes_root()?, spec, 0)
}

pub(super) fn verify(spec: &Spec) -> Result<(), String> {
    let content = body(spec)?;
    let dir = anchor(&routes_root()?, 0)?;
    let name = format!("{}.caddy", spec.id);
    current_at(&dir, &name, content.as_bytes(), 0)
}
