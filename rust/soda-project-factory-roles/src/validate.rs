//! Request validators: exact ports of the `.py` `check_*` family plus the
//! duplicate-tolerant key-set checks (`json.loads` without a pairs hook is
//! last-wins, so duplicated keys collapse before the set comparison).

use crate::error::{fail, Error};
use soda_json::JsonValue;

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

pub fn check_id(value: &JsonValue) -> Result<&str, Error> {
    match value.as_str() {
        Some(text) if is_id(text) => Ok(text),
        _ => fail("unsupported preparation identity"),
    }
}

pub fn check_role(value: &JsonValue) -> Result<&str, Error> {
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

pub fn check_digest(value: &JsonValue) -> Result<&str, Error> {
    match value.as_str() {
        Some(text) if is_digest(text) => Ok(text),
        _ => fail("unsupported approved digest"),
    }
}

pub fn check_commit(value: &JsonValue) -> Result<&str, Error> {
    match value.as_str() {
        Some(text) if is_commit(text) => Ok(text),
        _ => fail("unsupported source commit"),
    }
}

/// Exact key set over unique keys (`set(data) == {...}` after last-wins).
pub fn key_set(entries: &[(String, JsonValue)], keys: &[&str]) -> bool {
    if entries.iter().any(|(k, _)| !keys.contains(&k.as_str())) {
        return false;
    }
    keys.iter()
        .all(|want| entries.iter().any(|(have, _)| have == want))
}

pub fn as_object(value: &JsonValue) -> Option<&Vec<(String, JsonValue)>> {
    match value {
        JsonValue::Object(entries) => Some(entries),
        _ => None,
    }
}

/// Strict JSON integer literal (`type(x) is int`: bool excluded, fraction
/// and exponent excluded) carried as normalized text so arbitrarily large
/// revisions echo exactly like CPython. `-0` normalizes to `0` because
/// `json.loads("-0")` is `0`.
pub fn as_int_text(value: &JsonValue) -> Option<String> {
    match value {
        JsonValue::Number(raw) => {
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
pub fn as_i64(value: &JsonValue) -> Option<i64> {
    as_int_text(value).and_then(|text| text.parse().ok())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn val(text: &str) -> JsonValue {
        JsonValue::parse(text).expect("parse")
    }

    #[test]
    fn id_matrix() {
        assert!(is_id("f0123456789abcdef01234567"));
        assert!(!is_id("../escape"));
        assert!(!is_id("f0123456789abcdef0123456")); // short
        assert!(!is_id("f0123456789abcdef012345678")); // long
        assert!(!is_id("g0123456789abcdef01234567")); // head
        assert!(!is_id("F0123456789ABCDEF01234567")); // upper
        assert!(!is_id("f0123456789abcdef01234567\n")); // \Z: no trailing NL
        assert!(!is_id(""));
        assert!(check_id(&val("\"f0123456789abcdef01234567\"")).is_ok());
        assert!(check_id(&val("5")).is_err());
    }

    #[test]
    fn role_matrix() {
        assert!(is_role("soda-coder"));
        assert!(is_role("soda-reviewer"));
        assert!(!is_role("root"));
        assert!(!is_role("soda-Coder"));
        assert!(!is_role(""));
    }

    #[test]
    fn name_matrix() {
        assert!(is_name("setup.sh"));
        assert!(is_name("a"));
        assert!(is_name(&"a".repeat(64)));
        assert!(!is_name(&"a".repeat(65)));
        assert!(!is_name(""));
        assert!(!is_name(".hidden")); // leading dot barred
        assert!(!is_name("-dash")); // leading dash barred
        assert!(!is_name("../x"));
        assert!(!is_name("a..b"));
        assert!(!is_name("a/b"));
        assert!(!is_name("a b"));
        assert!(is_name("a_B-c.d9"));
    }

    #[test]
    fn digest_commit_matrix() {
        assert!(is_digest(&"e".repeat(64)));
        assert!(!is_digest(&"e".repeat(63)));
        assert!(!is_digest(&"E".repeat(64)));
        assert!(!is_digest("zz"));
        assert!(is_commit(&"c".repeat(40)));
        assert!(!is_commit("short"));
        assert!(!is_commit(&"c".repeat(41)));
    }

    #[test]
    fn key_sets_collapse_duplicates() {
        let dup = val("{\"op\":\"ensure\",\"op\":\"ensure\"}");
        assert!(key_set(as_object(&dup).unwrap(), &["op"]));
        let two = val("{\"op\":\"x\",\"id\":\"y\"}");
        assert!(key_set(as_object(&two).unwrap(), &["op", "id"]));
        assert!(!key_set(as_object(&two).unwrap(), &["op"]));
        assert!(!key_set(as_object(&two).unwrap(), &["op", "id", "extra"]));
        // Last wins on lookup, like the `.py` decoder.
        assert_eq!(two.get("op").and_then(|v| v.as_str()), Some("x"));
    }

    #[test]
    fn int_text_matches_type_is_int() {
        assert_eq!(as_int_text(&val("42")), Some("42".to_string()));
        assert_eq!(as_int_text(&val("-1")), Some("-1".to_string()));
        assert_eq!(as_int_text(&val("-0")), Some("0".to_string()));
        assert_eq!(as_int_text(&val("0")), Some("0".to_string()));
        assert_eq!(as_int_text(&val("true")), None);
        assert_eq!(as_int_text(&val("4.0")), None);
        assert_eq!(as_int_text(&val("1e3")), None);
        assert_eq!(as_int_text(&val("\"42\"")), None);
        // Unbounded range like Python ints.
        let big = "123456789".repeat(30);
        assert_eq!(as_int_text(&val(&big)), Some("123456789".repeat(30)));
        assert_eq!(as_i64(&val("999")), Some(999));
        assert_eq!(as_i64(&val("-15")), Some(-15));
        assert_eq!(as_i64(&val("null")), None);
        assert_eq!(as_i64(&val(&big)), None); // out of i64, still a valid literal
    }
}
