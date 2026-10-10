use super::listeners::{cockpit_bound, tls_material_paths};
use super::*;

#[test]
fn dumps_emits_equivalent_json_values() {
    let value = JsonValue::Object(vec![
        ("b".to_string(), JsonValue::Number("1".to_string())),
        (
            "a".to_string(),
            JsonValue::Array(vec![JsonValue::Bool(true), JsonValue::Null]),
        ),
    ]);
    let decoded: serde_json::Value = serde_json::from_str(&dumps(&value)).unwrap();
    assert_eq!(decoded["b"], 1);
    assert_eq!(decoded["a"], serde_json::json!([true, null]));
    let value = JsonValue::Object(vec![(
        "e".to_string(),
        JsonValue::Str("é\t\"q\"".to_string()),
    )]);
    assert_eq!(
        serde_json::from_str::<serde_json::Value>(&dumps(&value)).unwrap()["e"],
        "é\t\"q\""
    );
    let value = JsonValue::Str("<>&😀".to_string());
    assert_eq!(
        serde_json::from_str::<serde_json::Value>(&dumps(&value)).unwrap(),
        "<>&😀"
    );
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
fn cockpit_requires_only_wildcard_bindings() {
    let native = "LISTEN 0 4096 *:9090 *:*\n";
    cockpit_bound(&parse_listeners(native).unwrap()).unwrap();
    for address in ["0.0.0.0", "[::]"] {
        let rows = format!("LISTEN 0 4096 {address}:9090 *:*\n");
        cockpit_bound(&parse_listeners(&rows).unwrap()).unwrap();
    }
    for address in ["127.0.0.1", "10.0.2.15", "[::1]"] {
        let rows = format!("{native}LISTEN 0 4096 {address}:9090 *:*\n");
        assert_eq!(
            cockpit_bound(&parse_listeners(&rows).unwrap()).unwrap_err(),
            ProbeFailure::exit("unexpected additional service binding")
        );
    }
    assert_eq!(
        cockpit_bound(&parse_listeners("").unwrap()).unwrap_err(),
        ProbeFailure::exit("required service binding missing")
    );
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
fn tls_profiles_match_activation_outputs() {
    let local = parse_env_file(
        "FORGEJO_ORIGIN=https://10.0.2.15\nSODA_BIND=10.0.2.15\nSODA_TLS=internal\n",
    )
    .unwrap();
    let paths = tls_material_paths(local.get("SODA_TLS").map(String::as_str)).unwrap();
    assert!(paths.contains(&"/var/lib/soda/proxy/caddy/pki/authorities/local/root.crt"));
    assert!(paths.contains(&"/var/lib/soda/proxy/caddy/pki/authorities/local/root.key"));
    assert!(paths.contains(&"/var/lib/soda/proxy/caddy/pki/authorities/local/intermediate.key"));
    assert!(!paths.iter().any(|path| path.starts_with("/etc/soda/tls/")));

    let external = parse_env_file(
        "FORGEJO_ORIGIN=https://10.0.2.15\nSODA_BIND=10.0.2.15\nSODA_TLS=/etc/soda/tls/cert.pem /etc/soda/tls/key.pem\n",
    )
    .unwrap();
    assert_eq!(
        tls_material_paths(external.get("SODA_TLS").map(String::as_str)).unwrap(),
        ["/etc/soda/tls/cert.pem", "/etc/soda/tls/key.pem"]
    );
    for mode in [None, Some(""), Some("auto"), Some("/other/cert /other/key")] {
        assert_eq!(
            tls_material_paths(mode).unwrap_err(),
            ProbeFailure::exit("unsupported configured TLS mode")
        );
    }
}

#[test]
fn tailscale_summaries_shape() {
    let out =
        operator_tailscale(r#"{"BackendState": "Running", "Self": {"Expired": false}}"#).unwrap();
    assert!(out.ends_with('\n'));
    assert_eq!(
        serde_json::from_str::<serde_json::Value>(out.trim_end()).unwrap(),
        serde_json::json!({"BackendState": "Running", "Expired": false})
    );
    let out = operator_tailscale(r#"{"BackendState": "Stopped"}"#).unwrap();
    assert!(out.ends_with('\n'));
    assert_eq!(
        serde_json::from_str::<serde_json::Value>(out.trim_end()).unwrap(),
        serde_json::json!({"BackendState": "Stopped", "Expired": null})
    );
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
    assert!(out.ends_with('\n'));
    assert_eq!(
        serde_json::from_str::<serde_json::Value>(out.trim_end()).unwrap(),
        serde_json::json!([{
            "booted": true,
            "version": "41",
            "checksum": "abc",
            "requested-packages": ["x"]
        }])
    );
    assert!(host_deployments(r#"{"deployments": [7]}"#).is_err());
}
