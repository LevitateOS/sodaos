use super::*;
use crate::state_json::StateValue;

fn val(text: &str) -> StateValue {
    StateValue::parse(text).expect("parse")
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
