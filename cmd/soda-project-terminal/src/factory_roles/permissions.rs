//! P7 native caller/worker attenuation + OS permission policy (FW34).
//!
//! Classification and staging helpers for the D14 P protections: seven
//! profiles over the two native role accounts, parent-only tool refusal
//! for worker-bound callers, exact child inheritance, the fixed read-only
//! IPC mount spec, managed-config digest plus repo-override refusal,
//! capability/provider credential separation, per-profile shell-write
//! policy, and journal/symlink forgery rejection.
//!
//! Enforcement is kernel/file-mode based (read-only trees, `O_NOFOLLOW`,
//! `0600` secrets, `O_EXCL` journals). Production wiring of this policy
//! into staging/start-marker rides FW39; checkpoint consumption rides
//! FW42. This module stages and classifies only.

use crate::error::{fail, Error};
use crate::sha;
use std::fs::OpenOptions;
use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};
use std::path::Path;

pub const PROFILES: [&str; 7] = [
    "product_lead",
    "architect",
    "builder",
    "inspector",
    "fixer",
    "strategist",
    "test_pilot",
];

/// Profiles with OS-enforced read-only source plus disposable scratch.
pub const READONLY_PROFILES: [&str; 5] = [
    "product_lead",
    "architect",
    "inspector",
    "strategist",
    "test_pilot",
];

/// Profiles that may edit assigned source (never protected paths).
pub const WRITER_PROFILES: [&str; 2] = ["builder", "fixer"];

/// Native execution identity per profile (D01/D14): `review` and
/// `verify` run as the reviewer account; every other phase runs as the
/// coder account. Profile to phase: product_lead/decompose,
/// architect/investigate, builder/implement, fixer/correct and
/// strategist/reassess-or-repair run as soda-coder;
/// inspector/review and test_pilot/verify run as soda-reviewer.
/// Profile names never become seven native actors and no new account
/// roles are introduced.
pub fn profile_role(profile: &str) -> Result<&'static str, Error> {
    match profile {
        "product_lead" | "architect" | "builder" | "fixer" | "strategist" => Ok("soda-coder"),
        "inspector" | "test_pilot" => Ok("soda-reviewer"),
        _ => fail("unsupported factory profile"),
    }
}

pub fn is_readonly(profile: &str) -> Result<bool, Error> {
    match profile {
        _ if READONLY_PROFILES.contains(&profile) => Ok(true),
        _ if WRITER_PROFILES.contains(&profile) => Ok(false),
        _ => fail("unsupported factory profile"),
    }
}

/// Reads any bound caller may invoke, parent or worker.
pub const READ_TOOLS: [&str; 4] = [
    "factory.context",
    "factory.evidence",
    "factory.status",
    "factory.receipt",
];

/// Mutations only a parent may submit. A worker reports to its native
/// parent and cannot submit parent completion.
pub const PARENT_ONLY_TOOLS: [&str; 10] = [
    "factory.finish_incomplete",
    "factory.finish_product_plan",
    "factory.finish_issue_plan",
    "factory.finish_implementation",
    "factory.finish_review",
    "factory.finish_correction",
    "factory.finish_reassessment",
    "factory.finish_app_verification",
    "factory.submit_task_output",
    "factory.request_handoff",
];

/// Trusted native caller identity (T49): the parent run or a worker,
/// attested from the kernel-reported effective UID at the enforcement
/// point. The native adapter runs the parent outside the role account
/// and each worker setuid'd into its role account (`run_as_role`), so
/// an effective UID equal to the worker role account attests the
/// worker and anything else is the parent. A caller-supplied bool,
/// role string, child ID or environment variable proves nothing (D14)
/// and is not accepted here: the only input besides the live euid is
/// the host-recorded role UID.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Caller {
    Parent,
    Worker,
}

impl Caller {
    /// Attest the caller from the current effective UID against the
    /// worker role account UID recorded by the host.
    pub fn attest(role_uid: u32) -> Caller {
        // SAFETY: geteuid takes no arguments and always succeeds.
        if unsafe { libc::geteuid() } == role_uid {
            Caller::Worker
        } else {
            Caller::Parent
        }
    }
}

/// Caller attenuation (T49): natively-attested workers are refused
/// every parent-only tool; unknown tools refuse for any caller.
pub fn admit_tool(tool: &str, caller: Caller) -> Result<(), Error> {
    if READ_TOOLS.contains(&tool) {
        return Ok(());
    }
    if PARENT_ONLY_TOOLS.contains(&tool) {
        if caller == Caller::Worker {
            return fail("worker cannot submit parent completion");
        }
        return Ok(());
    }
    fail("unsupported factory tool")
}

/// Exact admitted execution identity a child must retain (T49).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ChildSpec {
    pub model: String,
    pub reasoning: String,
    pub contributor: bool,
    pub account: String,
    pub deadline: String,
    pub extra_lease: bool,
}

/// The child inherits the exact admitted profile/account/deadline with no
/// extra lease; any deviation refuses.
pub fn check_inheritance(parent: &ChildSpec, child: &ChildSpec) -> Result<(), Error> {
    if child.extra_lease {
        return fail("worker shares no extra lease");
    }
    if parent.model != child.model
        || parent.reasoning != child.reasoning
        || parent.contributor != child.contributor
        || parent.account != child.account
        || parent.deadline != child.deadline
    {
        return fail("worker identity must match the admitted run");
    }
    Ok(())
}

/// One Project-side mount visible to a worker.
pub struct MountEntry {
    pub path: String,
    pub readonly: bool,
    pub exposes_secrets: bool,
}

fn is_broad_mount(path: &str) -> bool {
    matches!(path, "/" | "/var" | "/var/lib/soda" | "/home" | "/var/lib")
}

fn is_foreign_socket(path: &str) -> bool {
    path.contains("extension")
        || path.contains("operator")
        || path.contains("broker")
        || path.contains("host-operation")
}

/// Fixed IPC mount spec (T49/T50): the dedicated IPC directory mounted
/// read-only exposing exactly the Factory socket; no secrets, no broad
/// mounts, no foreign sockets, never read-write.
pub fn check_mount(entries: &[MountEntry]) -> Result<(), Error> {
    let mut saw_ipc = false;
    for entry in entries {
        if is_broad_mount(&entry.path) {
            return fail("broad factory mount refused");
        }
        if is_foreign_socket(&entry.path) {
            return fail("foreign factory socket refused");
        }
        if entry.exposes_secrets {
            return fail("factory mount exposes secrets");
        }
        if entry.path == crate::mcp::IPC_DIR_NAME || entry.path.ends_with("/factory-ipc") {
            if !entry.readonly {
                return fail("factory IPC mount must be read-only");
            }
            saw_ipc = true;
        }
    }
    if !saw_ipc {
        return fail("factory IPC mount missing");
    }
    Ok(())
}

/// Managed MCP configuration digest check: the staged bytes must equal
/// the trusted digest recorded before the start marker.
pub fn verify_config_digest(expected_hex: &str, staged: &[u8]) -> Result<(), Error> {
    use sha2::{Digest, Sha256};
    if expected_hex.len() != 64 {
        return fail("unsupported config digest");
    }
    let mut hasher = Sha256::new();
    hasher.update(staged);
    if sha::hex(&hasher.finalize()) != expected_hex {
        return fail("managed config digest mismatch");
    }
    Ok(())
}

/// Repository-supplied MCP overrides are never honored (T50 config).
pub fn repo_config_refused(_path: &Path) -> Result<(), Error> {
    fail("repository MCP overrides are ignored")
}

/// Stage one managed config/capability file, helper-owned read-only.
pub fn stage_managed(path: &Path, bytes: &[u8]) -> Result<(), Error> {
    crate::fsx::write_new(path, bytes, 0o444)
}

/// Capability credential path stays inside the caller's own role scope;
/// cross-role (parent/other) credential paths refuse.
pub fn credential_path_for(
    credentials: &Path,
    role: &str,
    name: &str,
) -> Result<std::path::PathBuf, Error> {
    crate::validate::check_role_str(role)?;
    if !crate::validate::is_name(name) {
        return fail("unsupported credential identity");
    }
    Ok(credentials.join(role).join(name))
}

/// Worker credential gate (T49): a worker-bound caller may only resolve
/// credential paths inside its own role scope, never the parent scope or
/// another role's scope.
pub fn check_worker_credential(
    worker_role: &str,
    path: &Path,
    credentials: &Path,
) -> Result<(), Error> {
    crate::validate::check_role_str(worker_role)?;
    let scope = credentials.join(worker_role);
    if path.starts_with(&scope) {
        Ok(())
    } else {
        fail("worker credential scope refused")
    }
}

/// True when a worker environment leaks the parent-only secret bytes.
pub fn env_leaks_secret(env: &[(String, String)], parent_secret: &str) -> bool {
    if parent_secret.is_empty() {
        return false;
    }
    env.iter().any(|(_, value)| value.contains(parent_secret))
}

/// Protected names no profile may write through shell tools (T50):
/// Git metadata, Factory journals/receipts/tombstones, credentials,
/// managed MCP configuration and the IPC mount.
pub const PROTECTED_NAMES: [&str; 11] = [
    ".git",
    "credentials",
    "factory-ipc",
    "journal",
    "mcp.json",
    "factory-mcp.json",
    "finished.json",
    "started.json",
    "request.json",
    "stopped.json",
    "journal.log",
];

pub fn is_protected(target: &Path) -> bool {
    target.components().any(|component| {
        component
            .as_os_str()
            .to_str()
            .is_some_and(|name| PROTECTED_NAMES.contains(&name))
    })
}

/// Per-profile shell-write policy (T50). Scratch is disposable for every
/// profile. Readonly profiles write nowhere else. Writers edit assigned
/// source but never protected paths.
pub fn classify_write(
    profile: &str,
    target: &Path,
    assigned: &Path,
    scratch: &Path,
) -> Result<(), Error> {
    let readonly = is_readonly(profile)?;
    if target.starts_with(scratch) {
        return Ok(());
    }
    if readonly {
        return fail("readonly profile cannot write source");
    }
    if is_protected(target) {
        return fail("protected factory path is not writable");
    }
    if target.starts_with(assigned) {
        return Ok(());
    }
    fail("write outside assigned source refused")
}

/// Stage a read-only execution tree (T50 mechanism): symlinks refuse the
/// staging itself, then write bits are removed (dirs `0555`, files
/// `0444`) so the kernel denies shell writes for any profile.
pub fn stage_readonly_tree(root: &Path) -> Result<(), Error> {
    let meta = std::fs::symlink_metadata(root).map_err(Error::classify)?;
    if meta.file_type().is_symlink() {
        return fail("refused symlinked staging");
    }
    if meta.is_dir() {
        for entry in std::fs::read_dir(root).map_err(Error::classify)? {
            let entry = entry.map_err(Error::classify)?;
            stage_readonly_tree(&entry.path())?;
        }
        std::fs::set_permissions(root, std::fs::Permissions::from_mode(0o555))
            .map_err(Error::classify)?;
    } else {
        std::fs::set_permissions(root, std::fs::Permissions::from_mode(0o444))
            .map_err(Error::classify)?;
    }
    Ok(())
}

/// Bounded journal size for trusted reads (64 KiB, D14 result bound).
pub const MAX_JOURNAL: usize = 64 * 1024;

/// Open a journal with `O_NOFOLLOW`: a symlinked (forged) journal path
/// fails here before any byte is trusted (T50).
pub fn open_journal_no_follow(path: &Path) -> Result<Vec<u8>, Error> {
    use std::io::Read;
    let file = OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW)
        .open(path)
        .map_err(Error::classify)?;
    let mut raw = Vec::new();
    file.take((MAX_JOURNAL as u64) + 1)
        .read_to_end(&mut raw)
        .map_err(|err| Error::io("read", &err))?;
    if raw.len() > MAX_JOURNAL {
        return fail("oversized journal");
    }
    Ok(raw)
}

/// A journal is trusted only under its recorded digest; forged bytes
/// refuse even when the path is genuine (T50).
pub fn verify_journal(path: &Path, expected_hex: &str) -> Result<Vec<u8>, Error> {
    use sha2::{Digest, Sha256};
    let raw = open_journal_no_follow(path)?;
    let mut hasher = Sha256::new();
    hasher.update(&raw);
    if sha::hex(&hasher.finalize()) != expected_hex {
        return fail("journal digest mismatch");
    }
    Ok(raw)
}
