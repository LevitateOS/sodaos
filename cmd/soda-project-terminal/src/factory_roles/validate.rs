//! Request validators: exact ports of the `.py` `check_*` family plus the
//! duplicate-tolerant key-set checks (`json.loads` without a pairs hook is
//! last-wins, so duplicated keys collapse before the set comparison).

use crate::error::{fail, Error};
use crate::state_json::StateValue;

pub const ROLES: [&str; 2] = ["soda-coder", "soda-reviewer"];

fn is_lower_hex(bytes: &[u8]) -> bool {
    bytes.iter().all(|c| matches!(c, b'0'..=b'9' | b'a'..=b'f'))
}

/// `f[0-9a-f]{24}\Z`: exactly 25 chars, no trailing newline (`\Z`).
pub fn is_id(value: &str) -> bool {
    let bytes = value.as_bytes();
    bytes.len() == 25 && bytes[0] == b'f' && is_lower_hex(&bytes[1..])
}

pub fn is_role(value: &str) -> bool {
    ROLES.contains(&value)
}

/// `[A-Za-z0-9][A-Za-z0-9_.-]{0,63}\Z` plus the `..` refusal.
pub fn is_name(value: &str) -> bool {
    let bytes = value.as_bytes();
    if bytes.is_empty() || bytes.len() > 64 || value.contains("..") {
        return false;
    }
    if !bytes[0].is_ascii_alphanumeric() {
        return false;
    }
    bytes[1..]
        .iter()
        .all(|c| c.is_ascii_alphanumeric() || matches!(c, b'_' | b'.' | b'-'))
}

pub fn is_digest(value: &str) -> bool {
    value.len() == 64 && is_lower_hex(value.as_bytes())
}

pub fn is_commit(value: &str) -> bool {
    value.len() == 40 && is_lower_hex(value.as_bytes())
}

/// Plan-local task key (`[a-z][a-z0-9_-]{0,31}`): one child owns a task
/// until its terminal receipt, so keys are the child-control identity.
pub fn is_task_key(value: &str) -> bool {
    let bytes = value.as_bytes();
    if bytes.is_empty() || bytes.len() > 32 || !bytes[0].is_ascii_lowercase() {
        return false;
    }
    bytes[1..]
        .iter()
        .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || matches!(c, b'-' | b'_'))
}

/// Admitted reasoning-effort vocabulary in a staged execution profile.
/// Harness support for an effort rides the proved harness pin, never this
/// list alone.
pub fn is_effort(value: &str) -> bool {
    matches!(value, "low" | "medium" | "high" | "xhigh")
}

/// Admitted child-concurrency snapshot (`1..=4`): the lesser of policy and
/// qualified native capacity. The parent run is separate; nested child
/// spawning is refused.
pub fn is_child_limit(value: i64) -> bool {
    (1..=4).contains(&value)
}

/// Contributor mode is valid only for Muse; children inherit the exact
/// parent profile and account.
pub fn is_contributor_for_harness(contributor: bool, harness: &str) -> bool {
    !contributor || harness == "muse"
}

pub fn check_id(value: &StateValue) -> Result<&str, Error> {
    match value.as_str() {
        Some(text) if is_id(text) => Ok(text),
        _ => fail("unsupported preparation identity"),
    }
}

pub fn check_role(value: &StateValue) -> Result<&str, Error> {
    match value.as_str() {
        Some(text) if is_role(text) => Ok(text),
        _ => fail("unsupported factory role"),
    }
}

/// `role_record` and `write_snapshot` validate bare strings, not values.
pub fn check_role_str(login: &str) -> Result<&str, Error> {
    if is_role(login) {
        Ok(login)
    } else {
        fail("unsupported factory role")
    }
}

pub fn check_name_str(name: &str) -> Result<&str, Error> {
    if is_name(name) {
        Ok(name)
    } else {
        fail("unsupported approved file name")
    }
}

pub fn check_digest(value: &StateValue) -> Result<&str, Error> {
    match value.as_str() {
        Some(text) if is_digest(text) => Ok(text),
        _ => fail("unsupported approved digest"),
    }
}

pub fn check_commit(value: &StateValue) -> Result<&str, Error> {
    match value.as_str() {
        Some(text) if is_commit(text) => Ok(text),
        _ => fail("unsupported source commit"),
    }
}

/// Exact key set over unique keys (`set(data) == {...}` after last-wins).
pub fn key_set(entries: &[(String, StateValue)], keys: &[&str]) -> bool {
    if entries.iter().any(|(k, _)| !keys.contains(&k.as_str())) {
        return false;
    }
    keys.iter()
        .all(|want| entries.iter().any(|(have, _)| have == want))
}

pub fn as_object(value: &StateValue) -> Option<&Vec<(String, StateValue)>> {
    match value {
        StateValue::Object(entries) => Some(entries),
        _ => None,
    }
}

/// Strict JSON integer literal (`type(x) is int`: bool excluded, fraction
/// and exponent excluded) carried as normalized text so arbitrarily large
/// revisions echo exactly like CPython. `-0` normalizes to `0` because
/// `json.loads("-0")` is `0`.
pub fn as_int_text(value: &StateValue) -> Option<String> {
    match value {
        StateValue::Number(raw) => {
            let digits = raw.strip_prefix('-').unwrap_or(raw);
            if digits.is_empty() || !digits.bytes().all(|b| b.is_ascii_digit()) {
                return None;
            }
            if raw == "-0" {
                return Some("0".to_string());
            }
            Some(raw.clone())
        }
        _ => None,
    }
}

/// `started.json` / `finished.json` integers the helper wrote itself.
pub fn as_i64(value: &StateValue) -> Option<i64> {
    as_int_text(value).and_then(|text| text.parse().ok())
}
