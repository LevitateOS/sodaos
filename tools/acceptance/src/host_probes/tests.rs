use super::*;

#[test]
fn dumps_keeps_order_and_separators() {
    let value = JsonValue::Object(vec![
        ("b".to_string(), JsonValue::Number("1".to_string())),
        (
            "a".to_string(),
            JsonValue::Array(vec![JsonValue::Bool(true), JsonValue::Null]),
        ),
    ]);
    assert_eq!(dumps(&value), r#"{"b": 1, "a": [true, null]}"#);
    let value = JsonValue::Object(vec![(
        "e".to_string(),
        JsonValue::Str("é\t\"q\"".to_string()),
    )]);
    assert_eq!(dumps(&value), "{\"e\": \"\\u00e9\\t\\\"q\\\"\"}");
    let value = JsonValue::Str("<>&😀".to_string());
    assert_eq!(dumps(&value), "\"<>&\\ud83d\\ude00\"");
}

#[test]
fn origins_split_hosts_and_ports() {
    assert_eq!(
        split_origin("https://example.test").unwrap(),
        ("example.test".to_string(), 443)
    );
    assert_eq!(
        split_origin("https://Example.Test:8443/x").unwrap(),
        ("example.test".to_string(), 8443)
    );
    assert_eq!(
        split_origin("https://user@[::1]:2222").unwrap(),
        ("::1".to_string(), 2222)
    );
    assert_eq!(
        split_origin("https://[FE80::1]").unwrap(),
        ("fe80::1".to_string(), 443)
    );
    assert_eq!(
        split_origin("https://example.test:0").unwrap(),
        ("example.test".to_string(), 443)
    );
    assert!(split_origin("http://example.test").is_err());
    assert!(split_origin("https://").is_err());
    assert!(split_origin("https://host:notaport").is_err());
}

#[test]
fn private_binds_match_plain_lan_shapes() {
    for ip in [
        "10.1.2.3",
        "100.100.0.1",
        "172.20.0.1",
        "192.168.0.1",
        "fc00::1",
    ] {
        assert!(is_private_bind(ip.parse().unwrap()), "{ip}");
    }
    for ip in ["8.8.8.8", "0.0.0.0", "::", "127.0.0.1", "fe80::1"] {
        assert!(!is_private_bind(ip.parse().unwrap()), "{ip}");
    }
}

#[test]
fn listeners_parse_and_bind() {
    let rows = "LISTEN 0 128 127.0.0.1:9090 0.0.0.0:*\nLISTEN 0 128 [::1]:9090 [::]:*\n";
    let listeners = parse_listeners(rows).unwrap();
    assert_eq!(listeners.len(), 2);
    bound(&listeners, "127.0.0.1", 9090).unwrap();
    assert_eq!(
        bound(&listeners, "127.0.0.1", 3000).unwrap_err(),
        ProbeFailure::exit("required service binding missing")
    );
    let rows = "LISTEN 0 128 0.0.0.0:9090 0.0.0.0:*\n";
    let listeners = parse_listeners(rows).unwrap();
    assert_eq!(
        bound(&listeners, "127.0.0.1", 9090).unwrap_err(),
        ProbeFailure::exit("required service binding missing")
    );
    assert!(parse_listeners("short row\n").is_err());
}

#[test]
fn env_files_skip_empty_and_reject_bare_words() {
    let env = parse_env_file("A=1\n\nB=x=y\n").unwrap();
    assert_eq!(env.get("A").unwrap(), "1");
    assert_eq!(env.get("B").unwrap(), "x=y");
    assert_eq!(
        parse_env_file("BARE\n").unwrap_err(),
        ProbeFailure::failed("parse proxy.env")
    );
}

#[test]
fn tailscale_summaries_shape() {
    let out =
        operator_tailscale(r#"{"BackendState": "Running", "Self": {"Expired": false}}"#).unwrap();
    assert_eq!(out, "{\"BackendState\": \"Running\", \"Expired\": false}\n");
    let out = operator_tailscale(r#"{"BackendState": "Stopped"}"#).unwrap();
    assert_eq!(out, "{\"BackendState\": \"Stopped\", \"Expired\": null}\n");
    assert!(operator_tailscale(r#"{"BackendState": 7}"#).is_err());
    assert_eq!(
        forgejo_tailnet(r#"{"BackendState": "Running", "Self": {"Expired": false}}"#).unwrap(),
        ""
    );
    assert_eq!(
        forgejo_tailnet(r#"{"BackendState": "NoState"}"#).unwrap_err(),
        ProbeFailure::exit("approved running Tailnet required; no enrollment is performed")
    );
}

#[test]
fn deployments_summarize_in_order() {
    let out = host_deployments(r#"{"deployments": [{"booted": true, "version": "41", "checksum": "abc", "requested-packages": ["x"], "extra": 1}]}"#)
        .unwrap();
    assert_eq!(out, "[{\"booted\": true, \"version\": \"41\", \"checksum\": \"abc\", \"requested-packages\": [\"x\"]}]\n");
    assert!(host_deployments(r#"{"deployments": [7]}"#).is_err());
}
