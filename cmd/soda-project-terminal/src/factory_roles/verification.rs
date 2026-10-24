//! One fixed approved check per native run/manifest/incarnation. Retained
//! requests are never rerun; source and the preparation checkout stay intact.

use super::{account, execution, fsx, ops_approve, records, sha, validate};
use crate::emit::str_value;
use crate::error::{fail, Error};
use crate::state_json::StateValue;
use serde::Serialize;
use sha2::{Digest, Sha256};
use std::io::Read;
use std::os::unix::fs::{DirBuilderExt, MetadataExt};
use std::path::{Path, PathBuf};

const KEYS: &[&str] = &[
    "id",
    "run_id",
    "invocation_id",
    "manifest_digest",
    "check_digest",
    "source_commit",
    "deadline",
];
// The owning Go delivery producer caps every verification root at
// MaxAppVerificationRunMinutes; owner increases change the aggregate only.
const MAX_CHECK_ACTIVE: std::time::Duration = std::time::Duration::from_secs(30 * 60);

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
pub(crate) struct Request {
    pub id: String,
    pub run_id: String,
    pub invocation_id: String,
    pub manifest_digest: String,
    pub check_digest: String,
    pub source_commit: String,
    pub deadline: String,
}

fn native_id(value: &str) -> bool {
    value.len() == 32
        && value
            .bytes()
            .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
}

impl Request {
    pub(crate) fn parse(value: &StateValue, wire: bool) -> Result<Self, Error> {
        let mut keys = KEYS.to_vec();
        if wire {
            keys.push("op");
        }
        if !validate::as_object(value).is_some_and(|entries| validate::key_set(entries, &keys)) {
            return fail("unsupported verification request");
        }
        let get = |key| {
            value
                .get(key)
                .and_then(StateValue::as_str)
                .unwrap_or("")
                .to_string()
        };
        let r = Self {
            id: get("id"),
            run_id: get("run_id"),
            invocation_id: get("invocation_id"),
            manifest_digest: get("manifest_digest"),
            check_digest: get("check_digest"),
            source_commit: get("source_commit"),
            deadline: get("deadline"),
        };
        if !validate::is_id(&r.id)
            || !native_id(&r.run_id)
            || !native_id(&r.invocation_id)
            || !validate::is_digest(&r.manifest_digest)
            || !validate::is_digest(&r.check_digest)
            || !validate::is_commit(&r.source_commit)
            || soda_wire_time::parse_nanos(&r.deadline).is_none()
        {
            return fail("invalid verification identity");
        }
        Ok(r)
    }

    pub fn value(&self) -> StateValue {
        StateValue::parse(&serde_json::to_string(self).expect("verification strings serialize"))
            .expect("verification JSON parses")
    }
}

pub(crate) fn directory(ctx: &crate::Ctx, r: &Request) -> Result<PathBuf, Error> {
    Ok(fsx::prep_dir(ctx, &r.id)?
        .join("verifications")
        .join(&r.run_id))
}

fn saved(ctx: &crate::Ctx, dir: &Path, r: &Request) -> Result<(), Error> {
    fsx::owned_dir(ctx, dir, 0o755)?;
    let old = fsx::read_json(ctx, &dir.join("request.json"), 4096)?;
    if Request::parse(&old, false)? != *r {
        return fail("verification identity was already bound differently");
    }
    Ok(())
}

fn create_request(ctx: &crate::Ctx, dir: &Path, r: &Request) -> Result<(), Error> {
    let prep = fsx::prep_dir(ctx, &r.id)?;
    fsx::owned_dir(ctx, &prep, 0o755)?;
    let parent = prep.join("verifications");
    fsx::mkdir_p(&parent, 0o755)?;
    fsx::owned_dir(ctx, &parent, 0o755)?;
    std::fs::DirBuilder::new()
        .mode(0o755)
        .create(dir)
        .map_err(Error::classify)?;
    fsx::write_new(
        &dir.join("request.json"),
        crate::emit::dumps_default(&r.value()).as_bytes(),
        0o644,
    )?;
    super::verification_execution::sync_dir(dir)?;
    super::verification_execution::sync_dir(&parent)?;
    super::verification_execution::sync_dir(&prep)
}

fn snapshot_bytes(ctx: &crate::Ctx, path: &Path, limit: usize) -> Result<Vec<u8>, Error> {
    let file = fsx::open_ro(path, true)?;
    let meta = file.metadata().map_err(Error::classify)?;
    if !meta.is_file()
        || meta.uid() != ctx.priv_uid()
        || meta.nlink() != 1
        || meta.mode() & 0o022 != 0
    {
        return fail("unsafe approved verification input");
    }
    let mut bytes = Vec::new();
    file.take(limit as u64 + 1)
        .read_to_end(&mut bytes)
        .map_err(Error::classify)?;
    if bytes.len() > limit {
        return fail("oversized approved verification input");
    }
    Ok(bytes)
}

fn approved(ctx: &crate::Ctx, r: &Request) -> Result<Vec<StateValue>, Error> {
    let prep = fsx::prep_dir(ctx, &r.id)?;
    fsx::owned_dir(ctx, &prep, 0o755)?;
    fsx::refuse_barred(ctx, &prep)?;
    let fields = ops_approve::ReqFields::from_stored(&fsx::read_json(
        ctx,
        &prep.join("request.json"),
        4096,
    )?)?;
    if fields.role != "soda-reviewer"
        || fields.source_commit != r.source_commit
        || records::phase_of(ctx, &prep)?.0 != "ready"
    {
        return fail("verification requires its ready integrated reviewer preparation");
    }
    let snapshot = prep.join("snapshot");
    fsx::owned_dir(ctx, &snapshot, 0o755)?;
    let mut files = Vec::new();
    for entry in std::fs::read_dir(&snapshot).map_err(Error::classify)? {
        let entry = entry.map_err(Error::classify)?;
        let name = entry.file_name().to_string_lossy().into_owned();
        if name == "source.bundle" {
            continue;
        }
        validate::check_name_str(&name)?;
        files.push((
            name,
            snapshot_bytes(ctx, &entry.path(), ops_approve::MAX_APPROVED_FILE_SIZE)?,
        ));
        if files.len() > ops_approve::MAX_APPROVED_FILES {
            return fail("oversized verification snapshot");
        }
    }
    if files.iter().map(|(_, bytes)| bytes.len()).sum::<usize>() > ops_approve::MAX_APPROVED_TOTAL
        || sha::approved_digest(&files) != fields.setup_digest
    {
        return fail("approved verification snapshot changed");
    }
    let check = files
        .iter()
        .find(|(name, _)| name == crate::CHECK_ENTRY)
        .ok_or_else(|| Error::fail("approved check is absent"))?;
    if sha::hex(&Sha256::digest(&check.1)) != r.check_digest {
        return fail("approved verification check digest differs");
    }
    let tools = fsx::read_json(ctx, &prep.join("tools.json"), 65536)?;
    if tools.get("missing").and_then(StateValue::as_str) != Some("")
        || tools
            .get("verified")
            .and_then(|v| v.get("refusal"))
            .and_then(StateValue::as_str)
            .unwrap_or("")
            != ""
    {
        return fail("verification tool admission is unavailable");
    }
    let values = match tools.get("tools") {
        Some(StateValue::Array(values)) if values.len() <= records::MAX_TOOLS => values,
        _ => return fail("invalid verification tools"),
    };
    for tool in values {
        records::check_tool_entry(tool)?;
    }
    Ok(values.clone())
}

pub(crate) fn result(ctx: &crate::Ctx, dir: &Path, r: &Request) -> Result<StateValue, Error> {
    saved(ctx, dir, r)?;
    let finished = match fsx::read_json(ctx, &dir.join("finished.json"), 4096) {
        Ok(v) => Some(v),
        Err(Error::Missing) => None,
        Err(e) => return Err(e),
    };
    let stopped = fsx::lexists(&dir.join("stopped.json"));
    let supervisor = super::verification_execution::supervisor_alive(ctx, dir)?;
    let child = fsx::lexists(&dir.join("active-child.json"));
    let cleaned = fsx::lexists(&dir.join("fixture.json"))
        && !fsx::lexists(&dir.join("work"))
        && !fsx::lexists(&dir.join("home"));
    let never_started = fsx::lexists(&dir.join("not-started.json"))
        && !["started.json", "fixture.json", "work", "home"]
            .iter()
            .any(|name| fsx::lexists(&dir.join(name)));
    let retirement =
        if !supervisor && !child && (finished.is_some() || stopped) && (cleaned || never_started) {
            "confirmed"
        } else {
            "unconfirmed"
        };
    let exit = |key| {
        finished
            .as_ref()
            .and_then(|v| v.get(key))
            .cloned()
            .unwrap_or(StateValue::Null)
    };
    let phase = if stopped {
        "stopped"
    } else if finished.is_some() && retirement == "confirmed" {
        if exit("setup_exit") == StateValue::Number("0".into())
            && exit("check_exit") == StateValue::Number("0".into())
        {
            "completed"
        } else {
            "failed"
        }
    } else if supervisor {
        "running"
    } else {
        "uncertain"
    };
    let raw = match snapshot_bytes(ctx, &dir.join("check.log"), crate::LOG_CAP + 1024) {
        Ok(v) => v,
        Err(Error::Missing) => Vec::new(),
        Err(e) => return Err(e),
    };
    let kept = &raw[..raw.len().min(crate::LOG_CAP)];
    let mut output = String::from_utf8_lossy(kept).into_owned();
    while output.len() > crate::LOG_CAP {
        output.pop();
    }
    let mut fields = match r.value() {
        StateValue::Object(fields) => fields,
        _ => unreachable!(),
    };
    fields.extend([
        ("phase".into(), str_value(phase)),
        ("setup_exit".into(), exit("setup_exit")),
        ("check_exit".into(), exit("check_exit")),
        (
            "output_digest".into(),
            str_value(&sha::hex(&Sha256::digest(kept))),
        ),
        ("output".into(), str_value(&output)),
        (
            "truncated".into(),
            StateValue::Bool(raw.len() > crate::LOG_CAP),
        ),
        ("retirement".into(), str_value(retirement)),
    ]);
    Ok(StateValue::Object(fields))
}

pub(crate) fn start(ctx: &crate::Ctx, data: &StateValue) -> Result<StateValue, Error> {
    let r = Request::parse(data, true)?;
    let dir = directory(ctx, &r)?;
    if fsx::lexists(&dir) {
        return result(ctx, &dir, &r);
    }
    let tools = approved(ctx, &r)?;
    let deadline = execution::deadline_instant(&r.deadline)?;
    if deadline.saturating_duration_since(std::time::Instant::now()) > MAX_CHECK_ACTIVE {
        return fail("verification deadline exceeds one run");
    }
    create_request(ctx, &dir, &r)?;
    let account = account::role_record(ctx, "soda-reviewer")?
        .ok_or_else(|| Error::fail("verification reviewer account is unavailable"))?;
    for name in ["work", "home", "home/tmp"] {
        let path = dir.join(name);
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&path)
            .map_err(Error::classify)?;
        fsx::chown(&path, account.uid, account.gid)?;
    }
    super::verification_execution::launch(ctx, &dir, &r, &tools, &account)?;
    result(ctx, &dir, &r)
}

pub(crate) fn inspect(ctx: &crate::Ctx, data: &StateValue) -> Result<StateValue, Error> {
    let r = Request::parse(data, true)?;
    result(ctx, &directory(ctx, &r)?, &r)
}

pub(crate) fn stop(ctx: &crate::Ctx, data: &StateValue) -> Result<StateValue, Error> {
    let r = Request::parse(data, true)?;
    let dir = directory(ctx, &r)?;
    let lock = execution::state_lock(ctx)?;
    if !fsx::lexists(&dir) {
        // Host may have durably admitted pending before its start reached us.
        // The same lock fences a delayed start until this exact stopped tuple
        // is durable. Absence alone is never used as a retirement receipt.
        create_request(ctx, &dir, &r)?;
        fsx::write_new(
            &dir.join("not-started.json"),
            b"{\"not_started\":true}",
            0o644,
        )?;
        fsx::write_new(&dir.join("stopped.json"), b"{\"stopped\":true}", 0o644)?;
        super::verification_execution::sync_dir(&dir)?;
        return result(ctx, &dir, &r);
    }
    saved(ctx, &dir, &r)?;
    let untouched = ![
        "started.json",
        "active-child.json",
        "finished.json",
        "fixture.json",
        "work",
        "home",
        "clone.log",
        "checkout.log",
        "setup.log",
        "check.log",
    ]
    .iter()
    .any(|name| fsx::lexists(&dir.join(name)));
    if untouched || fsx::lexists(&dir.join("not-started.json")) {
        if !fsx::lexists(&dir.join("not-started.json")) {
            fsx::write_new(
                &dir.join("not-started.json"),
                b"{\"not_started\":true}",
                0o644,
            )?;
        }
        let marker = fsx::read_json(ctx, &dir.join("not-started.json"), 1024)?;
        if !validate::as_object(&marker).is_some_and(|e| validate::key_set(e, &["not_started"]))
            || marker.get("not_started").and_then(StateValue::as_bool) != Some(true)
            || [
                "started.json",
                "active-child.json",
                "fixture.json",
                "work",
                "home",
            ]
            .iter()
            .any(|name| fsx::lexists(&dir.join(name)))
        {
            return fail("verification absence fence is uncertain");
        }
        match fsx::write_new(&dir.join("stopped.json"), b"{\"stopped\":true}", 0o644) {
            Ok(()) | Err(Error::Exists) => {}
            Err(e) => return Err(e),
        }
        super::verification_execution::sync_dir(&dir)?;
        return result(ctx, &dir, &r);
    }
    // A terminal receipt is immutable. Its job already retired role children
    // and removed the recorded fixture before writing this durable result.
    if fsx::lexists(&dir.join("finished.json")) && !fsx::lexists(&dir.join("active-child.json")) {
        drop(lock);
        if !super::verification_execution::stop_supervisor(ctx, &dir)? {
            return fail("verification retirement is unconfirmed");
        }
        return result(ctx, &dir, &r);
    }
    match fsx::write_new(&dir.join("stopped.json"), b"{\"stopped\":true}", 0o644) {
        Ok(()) | Err(Error::Exists) => {}
        Err(e) => return Err(e),
    }
    super::verification_execution::sync_dir(&dir)?;
    let child = execution::retire_active_role_group(ctx, &dir, "soda-reviewer")?;
    drop(lock);
    // Let the supervising owner join/reap its role child before terminating
    // that owner. If it cannot finish, positive process identity is required.
    let until = std::time::Instant::now() + std::time::Duration::from_secs(1);
    while child == "confirmed"
        && fsx::lexists(&dir.join("active-child.json"))
        && std::time::Instant::now() < until
    {
        std::thread::sleep(std::time::Duration::from_millis(20));
    }
    let supervisor = if child == "confirmed" {
        super::verification_execution::stop_supervisor(ctx, &dir)?
    } else {
        false
    };
    if supervisor {
        let _lock = execution::state_lock(ctx)?;
        if execution::retire_active_role_group(ctx, &dir, "soda-reviewer")? != "confirmed" {
            return fail("verification child retirement is unconfirmed");
        }
        match std::fs::remove_file(dir.join("active-child.json")) {
            Ok(()) => {}
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {}
            Err(e) => return Err(Error::classify(e)),
        }
        super::verification_execution::cleanup(ctx, &dir)?;
    }
    let mut out = result(ctx, &dir, &r)?;
    if let StateValue::Object(fields) = &mut out {
        if !supervisor {
            if let Some((_, value)) = fields.iter_mut().find(|(key, _)| key == "retirement") {
                *value = str_value("unconfirmed");
            }
        }
    }
    Ok(out)
}

pub(crate) fn blocks_release(ctx: &crate::Ctx, prep: &Path) -> Result<bool, Error> {
    let parent = prep.join("verifications");
    if !fsx::lexists(&parent) {
        return Ok(false);
    }
    fsx::owned_dir(ctx, &parent, 0o755)?;
    for entry in std::fs::read_dir(&parent).map_err(Error::classify)? {
        let dir = entry.map_err(Error::classify)?.path();
        let request = Request::parse(
            &fsx::read_json(ctx, &dir.join("request.json"), 4096)?,
            false,
        )?;
        if directory(ctx, &request)? != dir {
            return fail("unknown verification directory");
        }
        let state = result(ctx, &dir, &request)?;
        if state.get("retirement").and_then(StateValue::as_str) != Some("confirmed") {
            return Ok(true);
        }
    }
    Ok(false)
}

pub(crate) fn stop_all(ctx: &crate::Ctx, prep: &Path) -> Result<bool, Error> {
    let parent = prep.join("verifications");
    if !fsx::lexists(&parent) {
        return Ok(true);
    }
    fsx::owned_dir(ctx, &parent, 0o755)?;
    let mut confirmed = true;
    for entry in std::fs::read_dir(&parent).map_err(Error::classify)? {
        let dir = entry.map_err(Error::classify)?.path();
        let request = Request::parse(
            &fsx::read_json(ctx, &dir.join("request.json"), 4096)?,
            false,
        )?;
        if directory(ctx, &request)? != dir {
            return fail("unknown verification directory");
        }
        let mut wire = match request.value() {
            StateValue::Object(fields) => fields,
            _ => unreachable!(),
        };
        wire.push(("op".into(), str_value("verify_stop")));
        let state = stop(ctx, &StateValue::Object(wire))?;
        confirmed &= state.get("retirement").and_then(StateValue::as_str) == Some("confirmed");
    }
    Ok(confirmed)
}
