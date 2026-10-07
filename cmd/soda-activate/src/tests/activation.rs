use std::fs;
use std::os::unix::fs::PermissionsExt;

use super::fixtures::*;

use crate::activation::activate;
use crate::cli::CliArgs;
use crate::system::{usage, ActivateError};

#[test]
fn first_activation_derives_provider_from_forgejo_origin() {
    for (origin, local_tls) in [
        ("https://forge.example.test:8443", false),
        ("https://[fd00::5]:8443/", false),
        ("https://192.168.2.100", true),
        ("https://[fd00::5]", true),
    ] {
        let fx = fixture(
            &dashboard(origin, "/run/soda/identity/admin.sock", "42"),
            "FORGEJO__ui__DEFAULT_THEME=soda-auto\nFORGEJO__server__SSH_DOMAIN=retained.example.test\n",
        );
        let address = if origin == "https://[fd00::5]" {
            "fd00::5"
        } else {
            "192.168.2.100"
        };
        // The non-local origins bind the same private address as the
        // Python suite's mapped run; only local-TLS binds the origin IP.
        let bind = if local_tls && address == "fd00::5" {
            "fd00::5"
        } else {
            "192.168.2.100"
        };
        let args = cli(bind, local_tls, &fx.temp);
        let mut sys = FakeSys::new();
        let mut stdout: Vec<u8> = Vec::new();
        activate(&args, &fx.paths, &mut sys, &mut stdout).expect("activate");
        let values = env_map(&fx.paths.root.join("forgejo.env"));
        assert_eq!(
            values["FORGEJO__picture__GRAVATAR_SOURCE"],
            format!("{}/-/soda/avatars/v1/", origin.trim_end_matches('/'))
        );
        assert_eq!(
            values["FORGEJO__server__SSH_DOMAIN"],
            "retained.example.test"
        );
        assert_eq!(values["FORGEJO__ui__DEFAULT_THEME"], "soda-auto");
        assert_eq!(values["FORGEJO__extensions__REQUIRED_IDS"], "soda");
        assert_eq!(
            values["FORGEJO__extensions__SERVICE_CALLBACK_PATH"],
            "/ipc/host.sock"
        );
        assert!(!values.keys().any(|k| k.contains("DISABLE_GRAVATAR")
            || k.contains("FEDERATED")
            || k.contains("OFFLINE_MODE")));
        // 3 activation phases + 3 is-active health probes.
        assert_eq!(sys.calls.len(), 6, "{origin}");
        let ext = fx.paths.var_lib.join("forgejo/gitea/extensions");
        assert_eq!(
            sys.chowns,
            vec![
                (fx.paths.root.clone(), 0, 2000),
                (fx.paths.root.join("dashboard.json"), 0, 2000),
                (fx.paths.root.join("grant-key"), 0, 2000),
                (ext.clone(), 1000, 1000),
                (ext.join(".data"), 1000, 1000),
                (ext.join(".data/soda"), 1000, 1000),
                (ext.join(".data/soda/operator-id"), 1000, 1000),
            ]
        );
        for name in ["dashboard.json", "grant-key"] {
            assert_eq!(
                fs::metadata(fx.paths.root.join(name))
                    .expect("m")
                    .permissions()
                    .mode()
                    & 0o777,
                0o640
            );
        }
        assert_eq!(
            fs::read_to_string(ext.join(".data/soda/operator-id")).expect("op"),
            "42\n"
        );
        assert_eq!(
            fs::metadata(ext.join(".data/soda/operator-id"))
                .expect("m")
                .permissions()
                .mode()
                & 0o777,
            0o600
        );
        assert_eq!(
            fs::metadata(&ext).expect("m").permissions().mode() & 0o777,
            0o700
        );
        let proxy = fs::read_to_string(fx.paths.root.join("proxy.env")).expect("proxy");
        assert!(proxy.contains(if local_tls {
            "SODA_TLS=internal\n"
        } else {
            "SODA_TLS=/etc/soda/tls/cert.pem /etc/soda/tls/key.pem\n"
        }));
        assert_eq!(fx.paths.root.join("tls/cert.pem").exists(), !local_tls);
    }
}

#[test]
fn tls_mode_rejects_missing_or_mixed_inputs_before_effects() {
    for (cert, key, local_tls) in [
        (false, false, false),
        (true, false, false),
        (true, false, true),
        (false, true, true),
    ] {
        let fx = fixture(
            &dashboard("https://192.168.1.5", "/run/soda/identity/admin.sock", "42"),
            "",
        );
        let args = CliArgs {
            bind_ip: "192.168.1.5".to_string(),
            certificate: cert.then(|| "/missing".to_string()),
            private_key: key.then(|| "/missing".to_string()),
            local_tls,
        };
        let mut sys = FakeSys::new();
        let mut stdout: Vec<u8> = Vec::new();
        let err = activate(&args, &fx.paths, &mut sys, &mut stdout).expect_err("rejected");
        assert!(matches!(err, ActivateError::Usage(_)), "{err:?}");
        assert!(sys.calls.is_empty(), "effects before validation");
        assert!(!fx.paths.root.join("proxy.env").exists());
    }
}

#[test]
fn empty_identity_socket_uses_standard_admin_socket() {
    let fx = fixture(
        &dashboard("https://192.168.2.100", "", "7"),
        "FORGEJO__ui__DEFAULT_THEME=soda-auto\n",
    );
    let args = cli("192.168.2.100", true, &fx.temp);
    let mut sys = FakeSys::new();
    let mut stdout: Vec<u8> = Vec::new();
    activate(&args, &fx.paths, &mut sys, &mut stdout).expect("activate");
    assert_eq!(
        fs::read_to_string(
            fx.paths
                .var_lib
                .join("forgejo/gitea/extensions/.data/soda/operator-id")
        )
        .expect("op"),
        "7\n"
    );
    assert_eq!(
        env_map(&fx.paths.root.join("forgejo.env"))["FORGEJO__extensions__SERVICE_BRIDGE_PEERS"],
        "2000:soda"
    );
}

#[test]
fn reports_units_that_never_become_active() {
    let fx = fixture(
        &dashboard(
            "https://192.168.2.100",
            "/run/soda/identity/admin.sock",
            "7",
        ),
        "",
    );
    let args = cli("192.168.2.100", true, &fx.temp);
    let mut sys = FakeSys::new();
    sys.probe_code = 1;
    sys.now_values = vec![0.0, 61.0, 61.0];
    let mut stdout: Vec<u8> = Vec::new();
    let err = activate(&args, &fx.paths, &mut sys, &mut stdout).expect_err("inactive");
    assert_eq!(
        err,
        ActivateError::Usage(
            "activation started but not active: forgejo.service, soda-dashboard.service, soda-proxy.service; inspect journalctl -u forgejo.service".to_string()
        )
    );
}

#[test]
fn operator_identity_encodings() {
    for (operator_id, ok) in [
        ("42", true),
        ("1", true),
        ("9223372036854775807", true),
        ("0", false),
        ("-1", false),
        ("9223372036854775808", false),
        ("7.0", false),
        ("true", false),
        ("\"42\"", false),
        ("null", false),
    ] {
        let fx = fixture(
            &dashboard(
                "https://192.168.2.100",
                "/run/soda/identity/admin.sock",
                operator_id,
            ),
            "",
        );
        let args = cli("192.168.2.100", true, &fx.temp);
        let mut sys = FakeSys::new();
        let mut stdout: Vec<u8> = Vec::new();
        let result = activate(&args, &fx.paths, &mut sys, &mut stdout);
        assert_eq!(result.is_ok(), ok, "{operator_id}");
        if !ok {
            assert!(
                sys.calls.is_empty(),
                "invalid identity caused service calls"
            );
            assert!(
                sys.chowns.is_empty(),
                "invalid identity caused ownership changes"
            );
            assert!(!fx.paths.root.join("activated").exists());
        }
    }
    // public_url marks a legacy separate-origin configuration.
    let fx = fixture(
        "{\"forgejo_url\":\"https://192.168.2.100\",\"public_url\":\"https://x\",\"operator_id\":7}",
        "",
    );
    let args = cli("192.168.2.100", true, &fx.temp);
    let mut sys = FakeSys::new();
    let mut stdout: Vec<u8> = Vec::new();
    assert_eq!(
        activate(&args, &fx.paths, &mut sys, &mut stdout),
        Err(usage(
            "legacy separate-origin configuration; use rehearsed configuration maintenance"
        ))
    );
}

#[test]
fn dashboard_duplicate_fields_are_last_wins_and_unknown_fields_are_ignored() {
    let json = concat!(
        "{\"forgejo_url\":\"https://192.168.2.100\",",
        "\"forgejo_internal_url\":\"http://127.0.0.1:3000\",",
        "\"listen\":\"127.0.0.1:8080\",",
        "\"grant_key_file\":\"/etc/soda/grant-key\",",
        "\"host_socket\":\"/run/soda/host.sock\",",
        "\"identity_socket\":\"/run/soda/identity/admin.sock\",",
        "\"operator_id\":42,\"operator_id\":7,",
        "\"ignored\":{\"nested\":[1,true,null]}}"
    );
    let fx = fixture(json, "");
    let args = cli("192.168.2.100", true, &fx.temp);
    let mut sys = FakeSys::new();
    let mut stdout = Vec::new();
    activate(&args, &fx.paths, &mut sys, &mut stdout).expect("activate");
    assert_eq!(
        fs::read_to_string(
            fx.paths
                .var_lib
                .join("forgejo/gitea/extensions/.data/soda/operator-id")
        )
        .expect("operator ID"),
        "7\n"
    );
}

#[test]
fn public_url_null_is_present_and_trailing_values_are_rejected_before_mutation() {
    for (dashboard_json, expected) in [
        (
            "{\"forgejo_url\":\"https://192.168.2.100\",\"operator_id\":7,\"public_url\":null}",
            usage("legacy separate-origin configuration; use rehearsed configuration maintenance"),
        ),
        (
            "{\"forgejo_url\":\"https://192.168.2.100\",\"operator_id\":7} {}",
            ActivateError::Runtime("dashboard.json is not valid JSON".to_string()),
        ),
    ] {
        let fx = fixture(dashboard_json, "");
        let args = cli("192.168.2.100", true, &fx.temp);
        let mut sys = FakeSys::new();
        let mut stdout = Vec::new();
        assert_eq!(
            activate(&args, &fx.paths, &mut sys, &mut stdout),
            Err(expected)
        );
        assert!(sys.calls.is_empty());
        assert!(sys.chowns.is_empty());
        assert!(!fx.paths.root.join("activated").exists());
    }
}

#[test]
fn refuses_without_root_before_effects() {
    let fx = fixture(
        &dashboard(
            "https://192.168.2.100",
            "/run/soda/identity/admin.sock",
            "7",
        ),
        "",
    );
    let args = cli("192.168.2.100", true, &fx.temp);
    let mut sys = FakeSys::new();
    sys.euid = 1000;
    let mut stdout: Vec<u8> = Vec::new();
    assert_eq!(
        activate(&args, &fx.paths, &mut sys, &mut stdout),
        Err(usage("native host operator/root required"))
    );
    assert!(sys.calls.is_empty());
}
