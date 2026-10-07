use super::*;
use crate::key_lines::{canonical_key_ok, canonical_lines, KEY_FILE_LIMIT};
use crate::key_request::{decode_key_request, state_object, RevisionValue};

const KEY_A: &str = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqu7mMCw5R";
const KEY_B: &str = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY=";
const KEY_SK: &str = "sk-ssh-ed25519@openssh.com AAAAGnNrLXNzaC1lZDI1NTE5QG9wZW5zc2guY29tAAAAIA==";

#[test]
fn key_line_matrix() {
    for good in [
        KEY_A,
        KEY_B,
        KEY_SK,
        "ssh-rsa AAAAB3Nza+Z/",
        "ssh-dss AAAAB3Nza+CD/==",
        "ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjE=",
        "sk-ecdsa-sha2-nistp256@openssh.com AAAAGnNrLWVjZHNhLXNoYTItbmlzdHAyNTZAb3BlbnNzaC5jb20=",
        "ssh--x AAA", // subtype "-x" matches [\w-]+
    ] {
        assert!(canonical_key_ok(good), "{good}");
    }
    for bad in [
        "",
        "ssh-ed25519",
        "ssh-ed25519 ",
        " ssh-ed25519 AAA",
        "ssh-ed25519  AAA",
        "ssh-ed25519 AAA BBB",
        "ssh-ed25519 AAA\n",
        "ssh AAA",   // empty subtype
        "ecdsa AAA", // missing dash-subtype
        "sk AAA",    // missing dash-subtype
        "putty AAA",
        "ssh-rsaé AAA",
        "ssh-rsa AAAé",
        "ssh-rsa AAA-",
        "ssh-rsa AAA_",
        "ssh-rsa.AAA AAA",
        "SSH-RSA AAA",
    ] {
        assert!(!canonical_key_ok(bad), "{bad:?}");
    }
    // sk- class extras: @ . - allowed, but not space or plus.
    assert!(canonical_key_ok("sk-a@b.c-d_ AAA"));
    assert!(!canonical_key_ok("sk-a+b AAA"));
    assert!(!canonical_key_ok("ssh-a+b AAA"));
}

#[test]
fn canonical_lines_matrix() {
    assert_eq!(canonical_lines(b"").unwrap(), Vec::<String>::new());
    assert_eq!(
        canonical_lines(format!("{KEY_A}\n").as_bytes()).unwrap(),
        vec![KEY_A]
    );
    let two = format!("{KEY_A}\n{KEY_B}\n");
    assert_eq!(canonical_lines(two.as_bytes()).unwrap(), vec![KEY_A, KEY_B]);
    // Violations.
    assert!(canonical_lines(KEY_A.as_bytes()).is_err()); // no newline
    assert!(canonical_lines(format!("{KEY_A}\n{KEY_A}\n").as_bytes()).is_err()); // dup
    assert!(canonical_lines("not-a-key\n".as_bytes()).is_err());
    assert!(canonical_lines("ssh-rsa AAA\ntrailing".as_bytes()).is_err());
    assert!(canonical_lines("\n".as_bytes()).is_err()); // empty line
    assert!(canonical_lines("ssh-rsa AAAé\n".as_bytes()).is_err()); // non-ascii
    let big = vec![b'A'; KEY_FILE_LIMIT + 1];
    assert!(canonical_lines(&big).is_err()); // oversize
                                             // 33 unique keys rejected, 32 accepted.
    let many: Vec<String> = (0..33)
        .map(|i| format!("ssh-k{i} QUJDREVGR0U1NTE5{i:04}"))
        .collect();
    assert!(canonical_lines(format!("{}\n", many.join("\n")).as_bytes()).is_err());
    let ok32 = format!("{}\n", many[..32].join("\n"));
    assert_eq!(canonical_lines(ok32.as_bytes()).unwrap().len(), 32);
}

#[test]
fn truthiness_matrix() {
    let truthy = |raw: &str| RevisionValue::from_raw(raw).is_truthy();
    assert!(!truthy("null"));
    assert!(!truthy("false"));
    assert!(truthy("true"));
    assert!(!truthy("0"));
    assert!(!truthy("0.0"));
    assert!(truthy("1"));
    assert!(truthy("-2.5"));
    assert!(truthy("1e3"));
    assert!(truthy("99999999999999999999999"));
    assert!(!truthy("\"\""));
    assert!(truthy("\"0\""));
    assert!(!truthy("[]"));
    assert!(!truthy("{}"));
    assert!(truthy("[0]"));
    assert!(truthy("{\"a\":0}"));
    assert!(truthy("1e400"));
    let request = decode_key_request(
        br#"{"login":"op","identity":7,"apply":false,"revision":1e400,"keys":[]}"#,
    )
    .unwrap();
    assert!(request.revision.is_truthy());
    let nested_large = decode_key_request(
        br#"{"login":"op","identity":7,"apply":false,"revision":{"x":1e400},"keys":[]}"#,
    )
    .unwrap();
    assert!(nested_large.revision.is_truthy());
}

fn request(login: &str, identity: &str, apply: &str, revision: &str, keys: &str) -> Vec<u8> {
    format!("{{\"login\":{login},\"identity\":{identity},\"apply\":{apply},\"revision\":{revision},\"keys\":{keys}}}")
        .into_bytes()
}

#[test]
fn decode_matrix() {
    let body = request("\"op\"", "7", "true", "\"rev\"", &format!("[\"{KEY_A}\"]"));
    let req = decode_key_request(&body).unwrap();
    assert_eq!(req.login, "op");
    assert_eq!(req.identity, 7);
    let negative_zero = request("\"op\"", "-0", "false", "null", "[]");
    assert_eq!(decode_key_request(&negative_zero).unwrap().identity, 0);
    assert!(req.apply);
    assert_eq!(req.desired, format!("{KEY_A}\n").into_bytes());
    // Empty set preview shape.
    let body = request("\"op\"", "7", "false", "\"\"", "[]");
    let req = decode_key_request(&body).unwrap();
    assert!(req.desired.is_empty());
    // Violations.
    assert!(decode_key_request(b"{}").is_err()); // missing keys
    assert!(decode_key_request(b"[]").is_err()); // not an object
    assert!(decode_key_request(b"null").is_err());
    assert!(decode_key_request(b"{oops").is_err());
    assert!(decode_key_request(&request("\"op\"", "7", "1", "\"\"", "[]")).is_err()); // apply not bool
    assert!(decode_key_request(&request("\"op\"", "7.0", "true", "\"\"", "[]")).is_err()); // identity not int
    assert!(decode_key_request(&request("\"op\"", "true", "true", "\"\"", "[]")).is_err());
    assert!(decode_key_request(&request("7", "7", "true", "\"\"", "[]")).is_err()); // login not str
    assert!(decode_key_request(&request("\"op\"", "7", "true", "\"\"", "[7]")).is_err()); // keys not strs
    assert!(decode_key_request(&request("\"op\"", "7", "true", "\"\"", "\"x\"")).is_err());
    let dup = format!("[\"{KEY_A}\",\"{KEY_A}\"]");
    assert!(decode_key_request(&request("\"op\"", "7", "true", "\"\"", &dup)).is_err()); // dup keys
    assert!(decode_key_request(&request("\"op\"", "7", "true", "\"\"", "[\"bad line\"]")).is_err());
    assert!(decode_key_request(&request("\"op\"", "7", "true", "\"\"", "[\"é\"]")).is_err()); // non-ascii
                                                                                              // Duplicate top-level field rejected (object_pairs_hook=unique).
    assert!(decode_key_request(b"{\"login\":\"op\",\"login\":\"x\",\"identity\":7,\"apply\":true,\"revision\":\"\",\"keys\":[]}").is_err());
    // Nested duplicates rejected too.
    assert!(decode_key_request(b"{\"login\":{\"a\":1,\"a\":2},\"identity\":7,\"apply\":true,\"revision\":\"\",\"keys\":[]}").is_err());
    let nested = |depth: usize| format!("{}0{}", "[".repeat(depth), "]".repeat(depth));
    let within_limit = format!(
        r#"{{"login":"op","identity":7,"apply":false,"revision":{},"keys":[]}}"#,
        nested(126)
    );
    assert!(decode_key_request(within_limit.as_bytes()).is_ok());
    let over_limit = format!(
        r#"{{"login":"op","identity":7,"apply":false,"revision":{},"keys":[]}}"#,
        nested(127)
    );
    assert!(decode_key_request(over_limit.as_bytes()).is_err());
    let deeper = format!(
        r#"{{"login":"op","identity":7,"apply":false,"revision":{},"keys":[]}}"#,
        nested(128)
    );
    assert!(decode_key_request(deeper.as_bytes()).is_err());
    // Extra field rejected (exact set).
    assert!(decode_key_request(
        b"{\"login\":\"op\",\"identity\":7,\"apply\":true,\"revision\":\"\",\"keys\":[],\"z\":1}"
    )
    .is_err());
}

#[test]
fn candidate_matrix() {
    assert_eq!(
        candidate_name("12345678-1234-5678-9abc-1234567890ab").as_deref(),
        Some(".soda-keys-12345678123456789abc1234567890ab")
    );
    assert!(candidate_name("12345678-1234-5678-9abc-1234567890AB").is_none()); // uppercase
    assert!(candidate_name("short").is_none());
    assert!(candidate_name("").is_none());
}

#[test]
fn state_shape_exact() {
    let value = state_object("abc123", &[KEY_A.to_string()]);
    assert_eq!(
        pyemit::dumps(&value),
        format!("{{\"revision\":\"abc123\",\"keys\":[\"{KEY_A}\"]}}")
    );
    assert_eq!(
        pyemit::dumps(&state_object("", &[])),
        "{\"revision\":\"\",\"keys\":[]}"
    );
}

#[test]
fn update_refuses_unprivileged() {
    if unsafe { libc::geteuid() } == 0 {
        return;
    }
    let req = decode_key_request(&request("\"op\"", "7", "false", "\"\"", "[]")).unwrap();
    assert_eq!(update(&req).unwrap_err(), "project-local root required");
}
