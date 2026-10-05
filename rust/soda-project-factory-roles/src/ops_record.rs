//! `record` and `start`: tool-evidence intake plus supervised launch.

use crate::emit::{self, obj, str_value};
use crate::error::{fail, Error};
use crate::fsx;
use crate::ops_approve::ReqFields;
use crate::ops_inspect;
use crate::proc;
use crate::validate;
use soda_json::JsonValue;

pub const MAX_TOOLS: usize = 8;
pub const MAX_TOOL_TEXT: usize = 256;

fn char_len(text: &str) -> usize {
    text.chars().count()
}

pub fn check_tool_entry(tool: &JsonValue) -> Result<(), Error> {
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

pub fn check_verified(verified: &JsonValue) -> Result<(), Error> {
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

pub fn do_record(ctx: &crate::Ctx, data: &JsonValue) -> Result<JsonValue, Error> {
    if !validate::as_object(data)
        .is_some_and(|e| validate::key_set(e, &["op", "id", "tools", "missing", "verified"]))
    {
        return fail("unsupported record request");
    }
    let pid = validate::check_id(data.get("id").unwrap_or(&JsonValue::Null))?.to_string();
    let tools = data.get("tools").unwrap_or(&JsonValue::Null);
    let items = match tools {
        JsonValue::Array(items) if items.len() <= MAX_TOOLS => items,
        _ => return fail("unsupported tool evidence"),
    };
    for tool in items {
        check_tool_entry(tool)?;
    }
    let missing = match data.get("missing").and_then(|v| v.as_str()) {
        Some(text) if char_len(text) <= MAX_TOOL_TEXT => text,
        _ => return fail("unsupported missing requirement"),
    };
    let verified = data.get("verified").unwrap_or(&JsonValue::Null);
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
            let fresh = JsonValue::parse(&payload).expect("emitted JSON parses");
            if !emit::json_equal(&saved, &fresh) {
                return fail("preparation already carries different tool evidence");
            }
        }
        Err(err) => return Err(err),
    }
    Ok(obj(vec![
        ("recorded", str_value(&pid)),
        ("waiting", JsonValue::Bool(!missing.is_empty())),
    ]))
}

#[derive(Debug)]
pub struct Prestate {
    pub fields: ReqFields,
    pub tools: Vec<JsonValue>,
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
    let verified = tools_doc.get("verified").unwrap_or(&JsonValue::Null);
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
        Some(JsonValue::Array(items)) => items.clone(),
        _ => return Err(Error::fail("unsafe factory metadata")),
    };
    Ok(Prestate { fields, tools })
}

pub fn do_start(ctx: &crate::Ctx, data: &JsonValue) -> Result<JsonValue, Error> {
    if !validate::as_object(data).is_some_and(|e| validate::key_set(e, &["op", "id"])) {
        return fail("unsupported start request");
    }
    let pid = validate::check_id(data.get("id").unwrap_or(&JsonValue::Null))?.to_string();
    fsx::ensure_layout(ctx)?;
    let directory = fsx::prep_dir(ctx, &pid)?;
    ops_inspect::refuse_barred(ctx, &directory)?;
    let prestate = start_prestate(ctx, &directory)?;
    let started = match fsx::read_json(ctx, &directory.join("started.json"), 1024) {
        Ok(value) => Some(value),
        Err(Error::Missing) => None,
        Err(err) => return Err(err),
    };
    if let Some(started) = started {
        let pgid = ops_inspect::started_pgid(&started)?;
        if !ops_inspect::group_alive(pgid, std::path::Path::new("/proc")) {
            return fail("preparation supervisor is gone; stop and use a new identity");
        }
        return Ok(obj(vec![
            ("started", str_value(&pid)),
            ("repeated", JsonValue::Bool(true)),
        ]));
    }
    let started = proc::spawn_detached(ctx, &directory, &prestate.fields, &prestate.tools)?;
    let pgid = ops_inspect::started_pgid(&started)?;
    Ok(obj(vec![
        ("started", str_value(&pid)),
        ("repeated", JsonValue::Bool(false)),
        ("pgid", JsonValue::Number(pgid.to_string())),
    ]))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::testutil::{
        approve_default, assert_fail, op_value, record_value, Scratch, PID, PID2,
    };

    fn tool(name: &str, path: &str, version: &str) -> JsonValue {
        JsonValue::Object(vec![
            ("name".to_string(), JsonValue::Str(name.to_string())),
            ("path".to_string(), JsonValue::Str(path.to_string())),
            ("version".to_string(), JsonValue::Str(version.to_string())),
        ])
    }

    #[test]
    fn tool_entry_matrix() {
        assert!(check_tool_entry(&tool("go", "/usr/bin/go", "1.2")).is_ok());
        // Unbounded names are accepted; only path/version are capped.
        assert!(check_tool_entry(&tool(&"n".repeat(999), "/bin/x", "v")).is_ok());
        assert!(check_tool_entry(&tool("go", &"p".repeat(256), "v")).is_ok());
        assert!(check_tool_entry(&tool("go", &"p".repeat(257), "v")).is_err());
        assert!(check_tool_entry(&tool("go", "/bin/x", &"v".repeat(257))).is_err());
        // Non-ASCII counts in chars, not bytes.
        assert!(check_tool_entry(&tool("go", &"é".repeat(256), "v")).is_ok());
        assert!(check_tool_entry(&tool("go", &"é".repeat(257), "v")).is_err());
        assert!(check_tool_entry(&JsonValue::Object(vec![
            ("name".to_string(), JsonValue::Str("go".to_string())),
            ("path".to_string(), JsonValue::Str("/bin/x".to_string())),
        ]))
        .is_err());
        assert!(check_tool_entry(&JsonValue::Object(vec![
            ("name".to_string(), JsonValue::Number("1".to_string())),
            ("path".to_string(), JsonValue::Str("/bin/x".to_string())),
            ("version".to_string(), JsonValue::Str("v".to_string())),
        ]))
        .is_err());
        assert!(check_tool_entry(&JsonValue::Array(Vec::new())).is_err());
    }

    #[test]
    fn verified_matrix() {
        let good =
            JsonValue::parse("{\"uid\": \"1\", \"login\": \"a\", \"groups\": \"a\"}").unwrap();
        assert!(check_verified(&good).is_ok());
        let refused = JsonValue::parse(
            "{\"uid\": \"1\", \"login\": \"a\", \"groups\": \"a\", \"refusal\": \"x\"}",
        )
        .unwrap();
        assert!(check_verified(&refused).is_ok());
        // Values are untyped except the refusal.
        let untyped = JsonValue::parse("{\"uid\": 1, \"login\": null, \"groups\": [1]}").unwrap();
        assert!(check_verified(&untyped).is_ok());
        // Missing keys, extra keys, long/non-string refusals fail.
        assert!(
            check_verified(&JsonValue::parse("{\"uid\": \"1\", \"login\": \"a\"}").unwrap())
                .is_err()
        );
        assert!(check_verified(
            &JsonValue::parse("{\"uid\": \"1\", \"login\": \"a\", \"groups\": \"a\", \"zz\": 1}")
                .unwrap()
        )
        .is_err());
        assert!(check_verified(
            &JsonValue::parse(&format!(
                "{{\"uid\": \"1\", \"login\": \"a\", \"groups\": \"a\", \"refusal\": \"{}\"}}",
                "r".repeat(257)
            ))
            .unwrap()
        )
        .is_err());
        assert!(check_verified(
            &JsonValue::parse(
                "{\"uid\": \"1\", \"login\": \"a\", \"groups\": \"a\", \"refusal\": 5}"
            )
            .unwrap()
        )
        .is_err());
        assert!(check_verified(&JsonValue::Array(Vec::new())).is_err());
    }

    #[test]
    fn record_reports_waiting_and_locks_evidence() {
        let scratch = Scratch::fresh();
        let (git, _) = scratch.git_script("git-ok", 0);
        let ctx = scratch.ctx(&git);
        crate::ops_approve::do_approve(&ctx, &approve_default(PID)).unwrap();
        let result = do_record(&ctx, &record_value(PID, "python3", None)).unwrap();
        assert_eq!(result.get("recorded").and_then(|v| v.as_str()), Some(PID));
        assert_eq!(result.get("waiting").and_then(|v| v.as_bool()), Some(true));
        let ready = do_record(&ctx, &record_value(PID2, "", None));
        assert_fail(ready, "unknown preparation identity");
        // Identical repeat is accepted; changed evidence conflicts.
        assert!(do_record(&ctx, &record_value(PID, "python3", None)).is_ok());
        assert_fail(
            do_record(&ctx, &record_value(PID, "other-tool", None)),
            "preparation already carries different tool evidence",
        );
        // Nine tools are refused.
        let tools: Vec<JsonValue> = (0..9)
            .map(|i| tool("t", &format!("/bin/t{i}"), "v"))
            .collect();
        let mut nine = record_value(PID, "", None);
        if let JsonValue::Object(entries) = &mut nine {
            let slot = entries
                .iter_mut()
                .find(|(k, _)| k == "tools")
                .expect("tools");
            slot.1 = JsonValue::Array(tools);
        }
        assert_fail(do_record(&ctx, &nine), "unsupported tool evidence");
        assert_fail(
            do_record(&ctx, &op_value("record", Some(PID))),
            "unsupported record request",
        );
    }

    #[test]
    fn record_after_finish_is_refused() {
        let scratch = Scratch::fresh();
        let (git, _) = scratch.git_script("git-ok", 0);
        let ctx = scratch.ctx(&git);
        crate::ops_approve::do_approve(&ctx, &approve_default(PID)).unwrap();
        do_record(&ctx, &record_value(PID, "", None)).unwrap();
        let directory = ctx.preparations.join(PID);
        crate::fsx::write_new(
            &directory.join("finished.json"),
            b"{\"setup_exit\": 0, \"check_exit\": 0}",
            0o644,
        )
        .unwrap();
        assert_fail(
            do_record(&ctx, &record_value(PID, "", None)),
            "preparation already finished",
        );
    }

    #[test]
    fn prestate_bars_launch_blockers() {
        let scratch = Scratch::fresh();
        let (git, _) = scratch.git_script("git-ok", 0);
        let ctx = scratch.ctx(&git);
        crate::ops_approve::do_approve(&ctx, &approve_default(PID)).unwrap();
        let directory = ctx.preparations.join(PID);
        match start_prestate(&ctx, &directory) {
            Err(Error::Fail(text)) => assert_eq!(text, "tool evidence is not recorded"),
            other => panic!("unexpected {other:?}"),
        }
        do_record(&ctx, &record_value(PID, "python3", None)).unwrap();
        match start_prestate(&ctx, &directory) {
            Err(Error::Fail(text)) => assert_eq!(text, "missing prerequisite bars setup: python3"),
            other => panic!("unexpected {other:?}"),
        }
    }

    #[test]
    fn prestate_unknown_identity() {
        let scratch = Scratch::fresh();
        let (git, _) = scratch.git_script("git-ok", 0);
        let ctx = scratch.ctx(&git);
        crate::fsx::ensure_layout(&ctx).unwrap();
        let directory = ctx.preparations.join(PID);
        std::fs::create_dir_all(&directory).unwrap();
        match start_prestate(&ctx, &directory) {
            Err(Error::Fail(text)) => assert_eq!(text, "unknown preparation identity"),
            other => panic!("unexpected {other:?}"),
        }
    }
}
