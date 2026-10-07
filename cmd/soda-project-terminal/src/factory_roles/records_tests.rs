use super::*;
use crate::state_json::StateValue;
use crate::testutil::{approve_default, assert_fail, op_value, record_value, Scratch, PID, PID2};

fn tool(name: &str, path: &str, version: &str) -> StateValue {
    StateValue::Object(vec![
        ("name".to_string(), StateValue::Str(name.to_string())),
        ("path".to_string(), StateValue::Str(path.to_string())),
        ("version".to_string(), StateValue::Str(version.to_string())),
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
    assert!(check_tool_entry(&StateValue::Object(vec![
        ("name".to_string(), StateValue::Str("go".to_string())),
        ("path".to_string(), StateValue::Str("/bin/x".to_string())),
    ]))
    .is_err());
    assert!(check_tool_entry(&StateValue::Object(vec![
        ("name".to_string(), StateValue::Number("1".to_string())),
        ("path".to_string(), StateValue::Str("/bin/x".to_string())),
        ("version".to_string(), StateValue::Str("v".to_string())),
    ]))
    .is_err());
    assert!(check_tool_entry(&StateValue::Array(Vec::new())).is_err());
}

#[test]
fn verified_matrix() {
    let good = StateValue::parse("{\"uid\": \"1\", \"login\": \"a\", \"groups\": \"a\"}").unwrap();
    assert!(check_verified(&good).is_ok());
    let refused = StateValue::parse(
        "{\"uid\": \"1\", \"login\": \"a\", \"groups\": \"a\", \"refusal\": \"x\"}",
    )
    .unwrap();
    assert!(check_verified(&refused).is_ok());
    // Values are untyped except the refusal.
    let untyped = StateValue::parse("{\"uid\": 1, \"login\": null, \"groups\": [1]}").unwrap();
    assert!(check_verified(&untyped).is_ok());
    // Missing keys, extra keys, long/non-string refusals fail.
    assert!(
        check_verified(&StateValue::parse("{\"uid\": \"1\", \"login\": \"a\"}").unwrap()).is_err()
    );
    assert!(check_verified(
        &StateValue::parse("{\"uid\": \"1\", \"login\": \"a\", \"groups\": \"a\", \"zz\": 1}")
            .unwrap()
    )
    .is_err());
    assert!(check_verified(
        &StateValue::parse(&format!(
            "{{\"uid\": \"1\", \"login\": \"a\", \"groups\": \"a\", \"refusal\": \"{}\"}}",
            "r".repeat(257)
        ))
        .unwrap()
    )
    .is_err());
    assert!(check_verified(
        &StateValue::parse("{\"uid\": \"1\", \"login\": \"a\", \"groups\": \"a\", \"refusal\": 5}")
            .unwrap()
    )
    .is_err());
    assert!(check_verified(&StateValue::Array(Vec::new())).is_err());
}

#[test]
fn record_reports_waiting_and_locks_evidence() {
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    crate::ops_approve::do_approve(&ctx, &approve_default(PID)).unwrap();
    let result = do_record(&ctx, &record_value(PID, "node22", None)).unwrap();
    assert_eq!(result.get("recorded").and_then(|v| v.as_str()), Some(PID));
    assert_eq!(result.get("waiting").and_then(|v| v.as_bool()), Some(true));
    let ready = do_record(&ctx, &record_value(PID2, "", None));
    assert_fail(ready, "unknown preparation identity");
    // Identical repeat is accepted; changed evidence conflicts.
    assert!(do_record(&ctx, &record_value(PID, "node22", None)).is_ok());
    assert_fail(
        do_record(&ctx, &record_value(PID, "other-tool", None)),
        "preparation already carries different tool evidence",
    );
    // Nine tools are refused.
    let tools: Vec<StateValue> = (0..9)
        .map(|i| tool("t", &format!("/bin/t{i}"), "v"))
        .collect();
    let mut nine = record_value(PID, "", None);
    if let StateValue::Object(entries) = &mut nine {
        let slot = entries
            .iter_mut()
            .find(|(k, _)| k == "tools")
            .expect("tools");
        slot.1 = StateValue::Array(tools);
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
    do_record(&ctx, &record_value(PID, "node22", None)).unwrap();
    match start_prestate(&ctx, &directory) {
        Err(Error::Fail(text)) => assert_eq!(text, "missing prerequisite bars setup: node22"),
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
