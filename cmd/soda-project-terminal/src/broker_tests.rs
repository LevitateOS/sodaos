use super::*;
use crate::subscription_cgroup::parse_cgroup_events;
use crate::subscription_prepare::mount_argv;
use crate::subscription_retire::{is_mount, subscription_remove_model};
use crate::subscription_start::respawn_argv;
use crate::subscription_wire::{
    deadline_ok, empty_result, json_equal, json_int, lease_with_binding, native_binding,
    profile_object,
};

fn parse(text: &str) -> JsonValue {
    JsonValue::parse(text).unwrap()
}

#[test]
fn equality_matrix() {
    // Order-insensitive objects, nested.
    assert!(json_equal(
        &parse("{\"a\":1,\"b\":{\"x\":[1,2],\"y\":null}}"),
        &parse("{\"b\":{\"y\":null,\"x\":[1,2]},\"a\":1}")
    ));
    // Duplicate keys: last wins, like `json.loads`.
    assert!(json_equal(&parse("{\"a\":1,\"a\":2}"), &parse("{\"a\":2}")));
    // Numbers: int/int, int/float, exponent.
    assert!(json_equal(&parse("42"), &parse("42")));
    assert!(json_equal(&parse("42"), &parse("42.0")));
    assert!(json_equal(&parse("100"), &parse("1e2")));
    assert!(!json_equal(&parse("42"), &parse("43")));
    assert!(!json_equal(&parse("1.5"), &parse("1.0")));
    // Bool/number: True == 1, False == 0 only.
    assert!(json_equal(&parse("true"), &parse("1")));
    assert!(json_equal(&parse("false"), &parse("0")));
    assert!(json_equal(&parse("true"), &parse("1.0")));
    assert!(!json_equal(&parse("true"), &parse("2")));
    assert!(!json_equal(&parse("false"), &parse("0.5")));
    // Shapes.
    assert!(json_equal(&parse("[1,[2]]"), &parse("[1,[2]]")));
    assert!(!json_equal(&parse("[1,2]"), &parse("[2,1]"))); // arrays ordered
    assert!(!json_equal(
        &parse("{\"a\":1}"),
        &parse("{\"a\":1,\"b\":2}")
    ));
    assert!(!json_equal(&parse("{\"a\":1}"), &parse("[1]")));
    assert!(!json_equal(&parse("\"1\""), &parse("1")));
    assert!(!json_equal(&parse("null"), &parse("false")));
    assert!(json_equal(&parse("null"), &parse("null")));
}

#[test]
fn int_conversion_matrix() {
    assert_eq!(json_int(&parse("42")), Some(42));
    assert_eq!(json_int(&parse("-7")), Some(-7));
    assert_eq!(json_int(&parse("42.0")), Some(42));
    assert_eq!(json_int(&parse("1e3")), Some(1000));
    assert_eq!(json_int(&parse("4.5")), Some(4)); // int() truncates
    assert_eq!(json_int(&parse("-0.5")), Some(0));
    assert_eq!(json_int(&parse("\"42\"")), Some(42));
    assert_eq!(json_int(&parse("\"  +42  \"")), Some(42));
    assert_eq!(json_int(&parse("\"-7\"")), Some(-7));
    assert_eq!(json_int(&parse("\"4.0\"")), None);
    assert_eq!(json_int(&parse("\"\"")), None);
    assert_eq!(json_int(&parse("\"1_0\"")), None); // underscores rejected
    assert_eq!(json_int(&parse("true")), Some(1)); // int(True)
    assert_eq!(json_int(&parse("false")), Some(0)); // int(False)
    assert_eq!(json_int(&parse("null")), None);
    assert_eq!(json_int(&parse("[]")), None);
    assert_eq!(json_int(&parse("9223372036854775807")), Some(i64::MAX));
    assert_eq!(json_int(&parse("9223372036854775808")), None); // i64 overflow
}

#[test]
fn lease_binding_edit() {
    let lease = parse("{\"execution_id\":\"a\",\"actor_id\":7,\"binding\":null}");
    let native = parse("{\"kind\":\"terminal\"}");
    let bound = lease_with_binding(&lease, native.clone()).unwrap();
    // In-place replace keeps the key position.
    assert_eq!(
        pyemit::dumps(&bound),
        "{\"execution_id\":\"a\",\"actor_id\":7,\"binding\":{\"kind\":\"terminal\"}}"
    );
    let bare = parse("{\"execution_id\":\"a\"}");
    let bound = lease_with_binding(&bare, native).unwrap();
    assert_eq!(
        pyemit::dumps(&bound),
        "{\"execution_id\":\"a\",\"binding\":{\"kind\":\"terminal\"}}"
    );
    assert!(lease_with_binding(&parse("[1]"), parse("1")).is_none());
}

#[test]
fn deadline_window() {
    assert!(deadline_ok(101, 100));
    assert!(deadline_ok(100 + HORIZON_SECS, 100));
    assert!(!deadline_ok(100, 100)); // expired at equality
    assert!(!deadline_ok(99, 100));
    assert!(!deadline_ok(101 + HORIZON_SECS, 100));
}

#[test]
fn request_decode_matrix() {
    assert!(decode_request(br#"{"action":"lookup"}"#).is_ok());
    assert!(decode_request(b"[1,2]").is_ok()); // no top shape check
    assert!(decode_request(b"5").is_ok());
    assert!(decode_request(b"").is_err());
    assert!(decode_request(b"{oops").is_err());
    assert!(decode_request(b"\xff\xfe").is_err());
    assert!(decode_request(&vec![b'x'; REQUEST_LIMIT + 1]).is_err());
    assert!(decode_request(&vec![b' '; REQUEST_LIMIT + 1]).is_err());
    // Exactly at the cap: padded valid JSON parses.
    let mut body = vec![b' '; REQUEST_LIMIT];
    body[0] = b'5';
    assert_eq!(
        decode_request(&body).unwrap(),
        JsonValue::Number("5".to_string())
    );
}

#[test]
fn result_shapes_exact() {
    let lease = parse("{\"execution_id\":\"a\",\"actor_id\":7}");
    // Default separators, byte-identical to `json.dumps`.
    assert_eq!(
        pyemit::dumps_default(&result_object(&lease, "")),
        "{\"lease\": {\"execution_id\": \"a\", \"actor_id\": 7}, \"credential\": \"\"}"
    );
    assert_eq!(
        pyemit::dumps_default(&empty_result()),
        "{\"lease\": {}, \"credential\": \"\"}"
    );
    assert_eq!(
        pyemit::dumps_default(&result_object(&lease, "eQ==")),
        "{\"lease\": {\"execution_id\": \"a\", \"actor_id\": 7}, \"credential\": \"eQ==\"}"
    );
    let native = native_binding("id", &parse("\"c\""), "op", &parse("3"));
    assert_eq!(
        pyemit::dumps(&native),
        "{\"kind\":\"terminal\",\"id\":\"id\",\"project\":\"c\",\"login\":\"op\",\"generation\":3}"
    );
    let profile = profile_object(&lease, &native, 1700000000, "scope");
    assert_eq!(
        pyemit::dumps(&profile),
        format!(
            "{{\"lease\":{},\"binding\":{},\"deadline\":1700000000,\"scope\":\"scope\"}}",
            pyemit::dumps(&lease),
            pyemit::dumps(&native)
        )
    );
}

#[test]
fn argv_constructors_exact() {
    assert_eq!(
        mount_argv("/m", "size=1g,nosuid,nodev,mode=755"),
        vec![
            "/usr/bin/mount",
            "-t",
            "tmpfs",
            "-o",
            "size=1g,nosuid,nodev,mode=755",
            "tmpfs",
            "/m"
        ]
    );
    let argv = respawn_argv("/run/soda-terminals/id", "/home/op");
    assert_eq!(
        &argv[..6],
        &["respawn-pane", "-k", "-t", "soda:0.0", "-c", "/home/op"]
    );
    assert_eq!(
        &argv[6..],
        &term::subscription_command("/run/soda-terminals/id")[..]
    );
}

#[test]
fn cgroup_events_matrix() {
    let rows = parse_cgroup_events(b"populated 0\nfrozen 1\n").unwrap();
    assert_eq!(
        rows,
        vec![
            ("populated".to_string(), "0".to_string()),
            ("frozen".to_string(), "1".to_string())
        ]
    );
    assert!(parse_cgroup_events(b"").unwrap().is_empty());
    assert!(parse_cgroup_events(b"populated\n").is_err()); // not a pair
    assert!(parse_cgroup_events(b"a b c\n").is_err());
    assert!(parse_cgroup_events(b"\xff\n").is_err());
}

#[test]
fn mount_probe_matrix() {
    assert!(is_mount("/")); // the root is always a mount
    assert!(!is_mount("/no-such-mount-soda-pt-xyz/auth"));
    assert!(!is_mount("relative/path/auth"));
    assert!(!is_mount(""));
}

static TEST_SEQ: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);

#[test]
fn remove_model_matrix() {
    let n = TEST_SEQ.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
    let dir = std::env::temp_dir().join(format!("soda-pt-broker-{n}-{}", std::process::id()));
    std::fs::create_dir_all(dir.join("model/auth")).unwrap();
    std::fs::create_dir_all(dir.join("model/harness")).unwrap();
    std::fs::write(dir.join("subscription"), b"{}").unwrap();
    std::fs::write(dir.join("subscription-started"), b"1").unwrap();
    let fd = std::fs::File::open(&dir).unwrap();
    subscription_remove_model(&fd).unwrap();
    assert!(!dir.join("subscription").exists());
    assert!(!dir.join("subscription-started").exists());
    assert!(!dir.join("model").exists());
    // Missing everything is a no-op.
    subscription_remove_model(&fd).unwrap();
    drop(fd);
    let _ = std::fs::remove_dir_all(&dir);
}

#[test]
fn dispatch_rejects_garbage_without_touching_fs() {
    // Unknown action with no delivery: fails before any privileged touch.
    assert!(subscription_dispatch(&parse("{\"action\":\"bogus\"}")).is_err());
    assert!(subscription_dispatch(&parse("[1]")).is_err());
    assert!(subscription_dispatch(&parse("{\"action\":5}")).is_err());
    assert!(subscription_dispatch(&parse("{}")).is_err());
    // Prepare without a lease shape fails at decode, before reserve.
    assert!(subscription_dispatch(&parse("{\"action\":\"prepare\",\"delivery\":{}}")).is_err());
}
