//! Durable importer acquisition custody (R06). The release image importer
//! runs as a process-group leader with Podman descendants; this record
//! binds each acquisition before launch and proves its retirement after.
//! Recover, rollback and boot consume the same custody state, but
//! recovery is observation-only: it retires custody only when no owned
//! descendant can still execute (provably empty group, or a reboot since
//! binding), and otherwise fences with precise attention. A signal is
//! sent only by a live in-process owner holding its unreaped-child
//! anchor; persisted recovery never signals a group from a saved
//! integer, however well it matches, and never turns a fence into an
//! automatic retry or success.
use std::fs::{self, OpenOptions};
use std::io::Write;
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::path::Path;
use std::time::Instant;

use serde::{Deserialize, Serialize};

pub(crate) const CUSTODY: &str = "importer-custody.json";
const SCRATCH: &str = "importer-custody.next";
const FORMAT: u32 = 1;
const CLOSED: &str = "importer custody uncertain; inspect or recover this operation";

const ACQUIRED: &str = "acquired";
const LAUNCHED: &str = "launched";
const RETIRED: &str = "retired";

fn is_hex64(value: &str) -> bool {
    value.len() == 64 && value.bytes().all(|b| b.is_ascii_hexdigit())
}

/// current_boot_id reads the kernel boot identity: two reads in one boot
/// always match, and no process predates a reboot. Recovery compares the
/// bound boot id before trusting any pid.
fn current_boot_id() -> Result<String, String> {
    let id = fs::read_to_string("/proc/sys/kernel/random/boot_id").map_err(|_| CLOSED)?;
    let id = id.trim().to_string();
    if id.is_empty() || id.len() > 128 {
        return Err(CLOSED.into());
    }
    Ok(id)
}

/// ImporterCustody binds one image acquisition: the exact payload digest
/// and argv admitted before launch, the boot identity and process group
/// with leader identity bound at launch, and the acquisition state.
/// `pgid` and `leader_starttime` are zero until launch binds them.
#[derive(Debug, Clone, Deserialize, Serialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub(crate) struct ImporterCustody {
    format: u32,
    payload_digest: String,
    argv: Vec<String>,
    boot_id: String,
    pgid: i32,
    leader_starttime: u64,
    state: String,
}

impl ImporterCustody {
    fn validate(&self) -> Result<(), String> {
        if self.format != FORMAT {
            return Err(CLOSED.into());
        }
        if !is_hex64(&self.payload_digest) {
            return Err(CLOSED.into());
        }
        if self.argv.is_empty()
            || self.argv.len() > 32
            || self.argv.iter().any(|a| a.is_empty() || a.len() > 1024)
        {
            return Err(CLOSED.into());
        }
        if self.boot_id.is_empty() || self.boot_id.len() > 128 {
            return Err(CLOSED.into());
        }
        match self.state.as_str() {
            ACQUIRED => {
                if self.pgid != 0 || self.leader_starttime != 0 {
                    return Err(CLOSED.into());
                }
            }
            // A reboot can retire an acquisition before it ever obtained
            // a process identity. Preserve its zero identity on disk.
            RETIRED if self.pgid == 0 && self.leader_starttime == 0 => {}
            LAUNCHED | RETIRED => {
                if self.pgid <= 1 {
                    return Err(CLOSED.into());
                }
            }
            _ => return Err(CLOSED.into()),
        }
        Ok(())
    }
}

/// leader_starttime reads one process's starttime (field 22 of
/// `/proc/<pid>/stat`, ticks since boot). The pid plus starttime pair
/// identifies the exact process incarnation: a reused pid shows a
/// different starttime, and a dead pid shows no entry at all. Reading is
/// harmless for any pid; signal authority stays pinned separately
/// (`retire_group` and custody validation refuse groups 0 and 1).
pub(crate) fn leader_starttime(pid: i32) -> Result<u64, String> {
    if pid <= 0 {
        return Err(CLOSED.into());
    }
    let stat = fs::read_to_string(format!("/proc/{pid}/stat")).map_err(|_| CLOSED)?;
    let after = stat.rsplit_once(") ").map(|(_, rest)| rest).ok_or(CLOSED)?;
    after
        .split_whitespace()
        .nth(19)
        .and_then(|field| field.parse::<u64>().ok())
        .ok_or_else(|| CLOSED.into())
}

fn load_raw(directory: &Path) -> Result<Option<ImporterCustody>, String> {
    match fs::read(directory.join(CUSTODY)) {
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(None),
        Err(_) => Err(CLOSED.into()),
        Ok(bytes) => {
            if bytes.len() > 1 << 20 {
                return Err(CLOSED.into());
            }
            let record: ImporterCustody =
                soda_host::json::decode_strict_as(&bytes).map_err(|_| CLOSED)?;
            record.validate()?;
            Ok(Some(record))
        }
    }
}

/// persist writes one fresh acquisition atomically, mirroring the
/// operation journal: an unsafe scratch entry is never replaced, the
/// payload commits through a rename, and the directory fsyncs. A
/// non-retired record already on disk fences the write: live custody is
/// never overwritten, only retired custody is superseded by the next
/// acquisition.
fn persist(directory: &Path, record: &ImporterCustody, owner: u32) -> Result<(), String> {
    record.validate()?;
    if let Some(existing) = load_raw(directory)? {
        if existing.state != RETIRED {
            return Err(CLOSED.into());
        }
    }
    write_replace(directory, record, owner)
}

/// acquire binds one acquisition before importer launch: the exact payload
/// digest and argv. A non-retired record fences; a retired record is
/// superseded by the fresh acquisition.
pub(crate) fn acquire(
    directory: &Path,
    payload_digest: &str,
    argv: &[String],
    owner: u32,
) -> Result<(), String> {
    persist(
        directory,
        &ImporterCustody {
            format: FORMAT,
            payload_digest: payload_digest.into(),
            argv: argv.to_vec(),
            boot_id: current_boot_id()?,
            pgid: 0,
            leader_starttime: 0,
            state: ACQUIRED.into(),
        },
        owner,
    )
}

/// launched binds the spawned group to a just-acquired invocation. It
/// must run while the child handle is still owned (before the wait loop),
/// so the recorded pgid cannot have been reused yet. The invocation must
/// match the acquired one exactly.
pub(crate) fn launched(
    directory: &Path,
    payload_digest: &str,
    argv: &[String],
    pgid: i32,
    owner: u32,
) -> Result<(), String> {
    let current = load_raw(directory)?.ok_or(CLOSED)?;
    if current.state != ACQUIRED
        || current.payload_digest != payload_digest
        || current.argv != argv
        || current.boot_id != current_boot_id()?
    {
        return Err(CLOSED.into());
    }
    let starttime = leader_starttime(pgid)?;
    let mut next = current;
    next.pgid = pgid;
    next.leader_starttime = starttime;
    next.state = LAUNCHED.into();
    // Bypass the live-custody fence: this is the acquired record's own
    // launch transition, verified above. Persist directly.
    next.validate()?;
    write_replace(directory, &next, owner)
}

/// retired records a group-verified retirement. The caller must have
/// observed the empty group (the wait loop verifies it before success);
/// this is the durable mark recovery consumes.
pub(crate) fn retired(directory: &Path, payload_digest: &str, owner: u32) -> Result<(), String> {
    let current = load_raw(directory)?.ok_or(CLOSED)?;
    if current.state != LAUNCHED || current.payload_digest != payload_digest {
        return Err(CLOSED.into());
    }
    let mut next = current;
    next.state = RETIRED.into();
    next.validate()?;
    write_replace(directory, &next, owner)
}

/// write_replace commits one verified custody transition over the live
/// record. Only the acquire/launch/retire transitions above may call it,
/// each after checking the current state.
fn write_replace(directory: &Path, record: &ImporterCustody, owner: u32) -> Result<(), String> {
    record.validate()?;
    let destination = directory.join(SCRATCH);
    match fs::symlink_metadata(&destination) {
        Ok(info) => {
            if !info.is_file()
                || info.uid() != owner
                || info.mode() & 0o022 != 0
                || info.len() > 1 << 20
            {
                return Err(CLOSED.into());
            }
            fs::remove_file(&destination).map_err(|_| CLOSED)?;
        }
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => {}
        Err(_) => return Err(CLOSED.into()),
    }
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .custom_flags(libc::O_NOFOLLOW | libc::O_CLOEXEC)
        .open(&destination)
        .map_err(|_| CLOSED)?;
    file.write_all(&serde_json::to_vec(record).map_err(|_| CLOSED)?)
        .and_then(|_| file.sync_all())
        .map_err(|_| CLOSED)?;
    fs::rename(&destination, directory.join(CUSTODY)).map_err(|_| CLOSED)?;
    fs::File::open(directory)
        .and_then(|f| f.sync_all())
        .map_err(|_| CLOSED.into())
}

/// reconcile consumes custody on the recover, rollback and boot paths. An
/// absent or retired record proceeds. An acquired-but-never-launched
/// record fences: a group may exist with no recorded pgid, and nothing
/// safe can be signaled. A launched record retires only with its leader
/// identity pinned; a lost or reused identity with a live group fences
/// without signaling, while a provably empty group retires silently.
/// leader_note classifies the recorded leader for precise attention:
/// gone, still ours, or replaced by reuse.
fn leader_note(current: &ImporterCustody) -> String {
    match leader_starttime(current.pgid) {
        Err(_) => format!("leader {} gone", current.pgid),
        Ok(starttime) if starttime == current.leader_starttime => {
            format!("leader {} still ours", current.pgid)
        }
        Ok(_) => format!("leader {} replaced by reuse", current.pgid),
    }
}

/// member_note names up to eight executing group members for precise
/// attention, so inspection never has to guess which processes fence.
fn member_note(members: &[i32]) -> String {
    let shown: Vec<String> = members.iter().take(8).map(|m| m.to_string()).collect();
    if members.len() > 8 {
        format!("members {} and {} more", shown.join(","), members.len() - 8)
    } else {
        format!("members {}", shown.join(","))
    }
}

pub(crate) fn reconcile(directory: &Path, owner: u32, deadline: Instant) -> Result<(), String> {
    let _ = deadline;
    let current = match load_raw(directory)? {
        None => return Ok(()),
        Some(record) => record,
    };
    match current.state.as_str() {
        RETIRED => Ok(()),
        ACQUIRED | LAUNCHED => reconcile_outstanding(directory, &current, owner),
        _ => Err(CLOSED.into()),
    }
}

/// reconcile_outstanding consumes one outstanding acquisition without
/// signaling anything. A reboot since binding proves no bound process can
/// still execute, so the custody retires. Otherwise an acquired record is
/// an uncertain crash prefix and always fences, while a launched record
/// retires only over a provably empty group; a live group fences with its
/// observed members named, and the operator retires those survivors
/// before recovery proceeds. Recovery never fabricates signal authority
/// it does not hold.
fn reconcile_outstanding(
    directory: &Path,
    current: &ImporterCustody,
    owner: u32,
) -> Result<(), String> {
    if current.boot_id != current_boot_id()? {
        let mut next = current.clone();
        next.state = RETIRED.into();
        return write_replace(directory, &next, owner);
    }
    if current.state == ACQUIRED {
        return Err(format!(
            "{}: acquisition recorded but never launched in this boot; no group can be named",
            CLOSED
        ));
    }
    let members = crate::command::group_executing_members(current.pgid).map_err(|_| CLOSED)?;
    if members.is_empty() {
        let mut next = current.clone();
        next.state = RETIRED.into();
        return write_replace(directory, &next, owner);
    }
    Err(format!(
        "{}: pgid {} ({}; {}) still executes; retire the observed processes before recovery",
        CLOSED,
        current.pgid,
        leader_note(current),
        member_note(&members),
    ))
}
