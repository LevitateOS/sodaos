//! `record` and `start`: tool-evidence intake plus supervised launch.

use super::fsx::{hold_json, hold_state};
use super::ops_inspect::group_alive;
use crate::emit::{self, obj, str_value};
use crate::error::{fail, Error};
use crate::fsx;
use crate::ops_approve::ReqFields;
use crate::ops_inspect;
use crate::state_json::StateValue;
use crate::validate;
use std::os::unix::fs::MetadataExt;
use std::path::Path;

pub const MAX_TOOLS: usize = 8;
pub const MAX_TOOL_TEXT: usize = 256;

fn char_len(text: &str) -> usize {
    text.chars().count()
}

pub fn check_tool_entry(tool: &StateValue) -> Result<(), Error> {
    let ok = validate::as_object(tool).is_some_and(|entries| {
        if !validate::key_set(entries, &["name", "path", "version"]) {
            return false;
        }
        let get = |key: &str| tool.get(key).and_then(|v| v.as_str());
        let (Some(_name), Some(path), Some(version)) = (get("name"), get("path"), get("version"))
        else {
            return false;
        };
        // Only path and version are length-capped; names are unbounded.
        char_len(path) <= MAX_TOOL_TEXT && char_len(version) <= MAX_TOOL_TEXT
    });
    if ok {
        Ok(())
    } else {
        fail("unsupported tool evidence")
    }
}

pub fn check_verified(verified: &StateValue) -> Result<(), Error> {
    let entries = match validate::as_object(verified) {
        Some(entries) => entries,
        None => return fail("unsupported launcher evidence"),
    };
    let mut unique: Vec<&str> = Vec::new();
    for (key, _) in entries {
        if !unique.contains(&key.as_str()) {
            unique.push(key);
        }
    }
    let allowed = ["uid", "login", "groups", "refusal"];
    if unique.iter().any(|key| !allowed.contains(key)) {
        return fail("unsupported launcher evidence");
    }
    for required in ["uid", "login", "groups"] {
        if !unique.contains(&required) {
            return fail("unsupported launcher evidence");
        }
    }
    match verified.get("refusal") {
        None => Ok(()),
        Some(value) => match value.as_str() {
            Some(text) if char_len(text) <= MAX_TOOL_TEXT => Ok(()),
            _ => fail("unsupported launcher refusal"),
        },
    }
}

pub fn do_record(ctx: &crate::Ctx, data: &StateValue) -> Result<StateValue, Error> {
    if !validate::as_object(data)
        .is_some_and(|e| validate::key_set(e, &["op", "id", "tools", "missing", "verified"]))
    {
        return fail("unsupported record request");
    }
    let pid = validate::check_id(data.get("id").unwrap_or(&StateValue::Null))?.to_string();
    let tools = data.get("tools").unwrap_or(&StateValue::Null);
    let items = match tools {
        StateValue::Array(items) if items.len() <= MAX_TOOLS => items,
        _ => return fail("unsupported tool evidence"),
    };
    for tool in items {
        check_tool_entry(tool)?;
    }
    let missing = match data.get("missing").and_then(|v| v.as_str()) {
        Some(text) if char_len(text) <= MAX_TOOL_TEXT => text,
        _ => return fail("unsupported missing requirement"),
    };
    let verified = data.get("verified").unwrap_or(&StateValue::Null);
    check_verified(verified)?;
    fsx::ensure_layout(ctx)?;
    let directory = fsx::prep_dir(ctx, &pid)?;
    ops_inspect::refuse_barred(ctx, &directory)?;
    if !fsx::lexists(&directory.join("request.json")) {
        return fail("unknown preparation identity");
    }
    if fsx::lexists(&directory.join("finished.json")) {
        return fail("preparation already finished");
    }
    let payload = emit::dumps_default(&obj(vec![
        ("tools", tools.clone()),
        ("missing", str_value(missing)),
        ("verified", verified.clone()),
    ]));
    let target = directory.join("tools.json");
    match fsx::write_new(&target, payload.as_bytes(), 0o644) {
        Ok(()) => {}
        Err(Error::Exists) => {
            let saved = fsx::read_json(ctx, &target, 65536)?;
            let fresh = StateValue::parse(&payload).expect("emitted JSON parses");
            if !emit::json_equal(&saved, &fresh) {
                return fail("preparation already carries different tool evidence");
            }
        }
        Err(err) => return Err(err),
    }
    Ok(obj(vec![
        ("recorded", str_value(&pid)),
        ("waiting", StateValue::Bool(!missing.is_empty())),
    ]))
}

#[derive(Debug)]
pub struct Prestate {
    pub fields: ReqFields,
    pub tools: Vec<StateValue>,
}

/// Launch preconditions: known identity, recorded evidence, no missing
/// prerequisite, no launcher refusal, not finished.
pub fn start_prestate(ctx: &crate::Ctx, directory: &std::path::Path) -> Result<Prestate, Error> {
    let request = match fsx::read_json(ctx, &directory.join("request.json"), 4096) {
        Ok(value) => value,
        Err(Error::Missing) => return fail("unknown preparation identity"),
        Err(err) => return Err(err),
    };
    let tools_doc = match fsx::read_json(ctx, &directory.join("tools.json"), 65536) {
        Ok(value) => value,
        Err(Error::Missing) => return fail("tool evidence is not recorded"),
        Err(err) => return Err(err),
    };
    match tools_doc.get("missing").and_then(|v| v.as_str()) {
        Some("") => {}
        Some(missing) => return fail(format!("missing prerequisite bars setup: {missing}")),
        None => return fail("missing prerequisite bars setup"),
    }
    let verified = tools_doc.get("verified").unwrap_or(&StateValue::Null);
    let refusal = match validate::as_object(verified) {
        Some(_) => verified
            .get("refusal")
            .and_then(|v| v.as_str())
            .unwrap_or(""),
        None => return fail("unsupported launcher evidence"),
    };
    if !refusal.is_empty() {
        return fail(format!("launcher verification bars setup: {refusal}"));
    }
    if fsx::lexists(&directory.join("finished.json")) {
        return fail("preparation already finished");
    }
    let fields = ReqFields::from_stored(&request)?;
    let tools = match tools_doc.get("tools") {
        Some(StateValue::Array(items)) => items.clone(),
        _ => return Err(Error::fail("unsafe factory metadata")),
    };
    Ok(Prestate { fields, tools })
}

/// Bounded privileged log read; missing logs read empty.
pub fn read_log(ctx: &crate::Ctx, path: &Path) -> Result<String, Error> {
    let file = match fsx::open_ro(path, true) {
        Ok(file) => file,
        Err(Error::Missing) => return Ok(String::new()),
        Err(err) => return Err(err),
    };
    let meta = file.metadata().map_err(Error::classify)?;
    if !meta.is_file() || meta.uid() != ctx.priv_uid() {
        return fail(format!("unsafe preparation log: {}", fsx::file_name(path)));
    }
    let mut raw = Vec::new();
    use std::io::Read;
    file.take((crate::LOG_CAP as u64) + 1024)
        .read_to_end(&mut raw)
        .map_err(|err| Error::io("read", &err))?;
    Ok(String::from_utf8_lossy(&raw).into_owned())
}

/// `started.json` process group the helper recorded itself.
pub fn started_pgid(started: &StateValue) -> Result<i32, Error> {
    match validate::as_i64(started.get("pgid").unwrap_or(&StateValue::Null)) {
        Some(pgid) => i32::try_from(pgid).map_err(|_| Error::fail("unsafe factory metadata")),
        None => Err(Error::fail("unsafe factory metadata")),
    }
}

pub fn started_pid(started: &StateValue) -> Result<i32, Error> {
    match validate::as_i64(started.get("pid").unwrap_or(&StateValue::Null)) {
        Some(pid) => i32::try_from(pid).map_err(|_| Error::fail("unsafe factory metadata")),
        None => Err(Error::fail("unsafe factory metadata")),
    }
}

fn exit_is_zero(value: &StateValue, key: &str) -> bool {
    validate::as_int_text(value.get(key).unwrap_or(&StateValue::Null)).as_deref() == Some("0")
}

/// `(phase, finished?)`: completion decides `ready`/`failed` first, then
/// evidence, then supervisor liveness. A dead supervisor without
/// completion is `interrupted`, never resurrected.
pub fn phase_of(ctx: &crate::Ctx, directory: &Path) -> Result<(String, Option<StateValue>), Error> {
    let finished = match fsx::read_json(ctx, &directory.join("finished.json"), 1024) {
        Ok(value) => Some(value),
        Err(Error::Missing) => None,
        Err(err) => return Err(err),
    };
    if let Some(finished) = finished {
        let phase =
            if exit_is_zero(&finished, "setup_exit") && exit_is_zero(&finished, "check_exit") {
                "ready"
            } else {
                "failed"
            };
        return Ok((phase.to_string(), Some(finished)));
    }
    let tools = match fsx::read_json(ctx, &directory.join("tools.json"), 65536) {
        Ok(value) => value,
        Err(Error::Missing) => return Ok(("approved".to_string(), None)),
        Err(err) => return Err(err),
    };
    if tools.get("missing").and_then(|v| v.as_str()) != Some("") {
        return Ok(("waiting".to_string(), None));
    }
    let refusal = tools
        .get("verified")
        .and_then(|v| v.get("refusal"))
        .and_then(|v| v.as_str())
        .unwrap_or("");
    if !refusal.is_empty() {
        return Ok(("failed".to_string(), None));
    }
    let started = match fsx::read_json(ctx, &directory.join("started.json"), 1024) {
        Ok(value) => value,
        Err(Error::Missing) => return Ok(("approved".to_string(), None)),
        Err(err) => return Err(err),
    };
    if !group_alive(started_pgid(&started)?, Path::new("/proc")) {
        return Ok(("interrupted".to_string(), None));
    }
    Ok(("running".to_string(), None))
}

pub fn do_inspect(ctx: &crate::Ctx, data: &StateValue) -> Result<StateValue, Error> {
    let shape_ok = validate::as_object(data).is_some_and(|entries| {
        validate::key_set(entries, &["op"])
            || validate::key_set(entries, &["op", "id"])
            || validate::key_set(entries, &["op", "id", "deadline"])
    });
    if !shape_ok {
        return fail("unsupported inspect request");
    }
    fsx::ensure_layout(ctx)?;
    let hold = hold_state(ctx)?;
    if data.get("id").is_none() {
        return Ok(obj(vec![("hold", hold_json(&hold))]));
    }
    let pid = validate::check_id(data.get("id").unwrap_or(&StateValue::Null))?.to_string();
    let directory = fsx::prep_dir(ctx, &pid)?;
    if let Some(deadline) = data.get("deadline") {
        let expected = deadline
            .as_str()
            .filter(|text| soda_wire_time::parse_nanos(text).is_some())
            .ok_or_else(|| Error::fail("invalid preparation deadline"))?;
        match fsx::read_json(ctx, &directory.join("started.json"), 1024) {
            Ok(started)
                if started.get("deadline").and_then(|value| value.as_str()) == Some(expected) => {}
            Err(Error::Missing) => {}
            Ok(_) => return fail("preparation start deadline changed"),
            Err(error) => return Err(error),
        }
    }
    let request = match fsx::read_json(ctx, &directory.join("request.json"), 4096) {
        Ok(value) => value,
        Err(Error::Missing) => {
            return Ok(obj(vec![
                ("hold", hold_json(&hold)),
                ("known", StateValue::Bool(false)),
            ]));
        }
        Err(err) => return Err(err),
    };
    let (mut phase, finished) = phase_of(ctx, &directory)?;
    let tools = match fsx::read_json(ctx, &directory.join("tools.json"), 65536) {
        Ok(value) => value,
        Err(Error::Missing) => obj(vec![
            ("tools", StateValue::Array(Vec::new())),
            ("missing", str_value("")),
            ("verified", obj(vec![])),
        ]),
        Err(err) => return Err(err),
    };
    let stopped = fsx::lexists(&directory.join("stopped.json"));
    if stopped && ["running", "approved", "waiting", "interrupted"].contains(&phase.as_str()) {
        phase = "stopped".to_string();
    }
    let get_str = |key: &str| match request.get(key).and_then(|v| v.as_str()) {
        Some(text) => Ok(str_value(text)),
        None => Err(Error::fail("unsafe factory metadata")),
    };
    let mut result = vec![
        ("hold", hold_json(&hold)),
        ("known", StateValue::Bool(true)),
        ("phase", str_value(&phase)),
        ("role", get_str("role")?),
        ("setup_digest", get_str("setup_digest")?),
        ("source_commit", get_str("source_commit")?),
        (
            "tools",
            tools.get("tools").cloned().unwrap_or(StateValue::Null),
        ),
        (
            "missing",
            tools.get("missing").cloned().unwrap_or(StateValue::Null),
        ),
        (
            "verified",
            tools.get("verified").cloned().unwrap_or(StateValue::Null),
        ),
        ("stopped", StateValue::Bool(stopped)),
        ("ready", StateValue::Bool(phase == "ready")),
        (
            "setup_log",
            str_value(&read_log(ctx, &directory.join("setup.log"))?),
        ),
        (
            "check_log",
            str_value(&read_log(ctx, &directory.join("check.log"))?),
        ),
    ];
    if let Some(finished) = finished {
        result.push((
            "setup_exit",
            finished
                .get("setup_exit")
                .cloned()
                .unwrap_or(StateValue::Null),
        ));
        result.push((
            "check_exit",
            finished
                .get("check_exit")
                .cloned()
                .unwrap_or(StateValue::Null),
        ));
    }
    Ok(obj(result))
}

#[cfg(test)]
#[path = "records_tests.rs"]
mod tests;
